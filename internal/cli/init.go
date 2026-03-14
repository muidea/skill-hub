package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/logging"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

var initCmd = &cobra.Command{
	Use:   "init [git-url]",
	Short: "初始化Skill Hub工作区",
	Long: `初始化Skill Hub工作区，创建必要的配置文件和目录结构。

如果提供了Git仓库URL，会克隆远程仓库到本地。
如果没有提供URL，会创建一个空的本地仓库。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		return runInit(args, target)
	},
}

func init() {
	initCmd.Flags().String("target", "open_code", "技能目标环境，默认为 open_code")
}

func runInit(args []string, target string) error {
	// 获取日志记录器
	logger := logging.GetGlobalLogger().WithOperation("runInit")

	// 记录开始
	startTime := time.Now()
	logger.Info("开始初始化skill-hub",
		"args", args,
		"target", target,
		"timestamp", startTime.Format(time.RFC3339))

	// 支持通过环境变量指定skill-hub目录
	skillHubDir := os.Getenv("SKILL_HUB_HOME")
	if skillHubDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.WrapWithCode(err, "runInit", errors.ErrSystem, "获取用户主目录失败")
		}
		skillHubDir = filepath.Join(homeDir, ".skill-hub")
	}

	// 多仓库目录
	repositoriesDir := filepath.Join(skillHubDir, "repositories")

	// 检查是否提供了Git URL
	var gitURL string
	if len(args) > 0 {
		gitURL = args[0]
	}

	// 从Git URL提取仓库名称
	repoName := "main"
	if gitURL != "" {
		// 尝试从git URL提取仓库名称
		if strings.Contains(gitURL, "/") {
			parts := strings.Split(gitURL, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if strings.HasSuffix(lastPart, ".git") {
					repoName = lastPart[:len(lastPart)-4]
				} else {
					repoName = lastPart
				}
			}
		}
	}

	// 仓库路径
	repoPath := filepath.Join(repositoriesDir, repoName)

	// 检查是否已经初始化了相同的配置
	alreadyInitialized, err := checkAlreadyInitialized(skillHubDir, gitURL)
	if err != nil {
		return errors.WrapWithCode(err, "runInit", errors.ErrSystem, "检查初始化状态失败")
	}

	// 如果gitURL为空，检查是否有现有的git仓库需要更新配置
	if gitURL == "" && alreadyInitialized {
		// 尝试从默认仓库（main）获取远程URL
		defaultRepoPath := filepath.Join(repositoriesDir, "main")
		if remoteURL, err := getRemoteURLFromGit(defaultRepoPath); err == nil && remoteURL != "" {
			// 有git仓库且有远程URL，需要更新配置
			logger.Info("检测到现有git仓库，将更新配置中的远程URL", "remoteURL", remoteURL)
			gitURL = remoteURL
			alreadyInitialized = false // 强制重新初始化以更新配置
		}
	}

	if alreadyInitialized {
		logger.Info("skill-hub 已经初始化完成", "skillHubDir", skillHubDir, "gitURL", gitURL)
		fmt.Printf("✅ skill-hub 已经初始化完成！\n")
		fmt.Println("工作区位置:", skillHubDir)
		if gitURL != "" {
			fmt.Println("远程仓库:", gitURL)
		}
		fmt.Println("\n使用 'skill-hub list' 查看可用技能")

		// 记录初始化完成
		logger.Info("skill-hub已经初始化完成",
			"skill_hub_dir", skillHubDir,
			"git_url", gitURL,
			"already_initialized", true)
		return nil
	}

	logger.Info("正在初始化Skill Hub工作区", "skillHubDir", skillHubDir, "gitURL", gitURL)
	fmt.Printf("正在初始化Skill Hub工作区: %s\n", skillHubDir)
	if gitURL != "" {
		fmt.Printf("将克隆远程仓库: %s\n", gitURL)
	}

	// 创建基础目录结构（只创建多仓库结构）
	dirs := []string{
		skillHubDir,
		repositoriesDir,
	}

	for _, dir := range dirs {
		if err := utils.EnsureDir(dir); err != nil {
			return err
		}
		fmt.Printf("✓ 目录已就绪: %s\n", dir)
	}

	// 创建配置文件
	configPath := filepath.Join(skillHubDir, "config.yaml")

	// 如果gitURL为空，但默认仓库已存在且有远程URL，尝试从git配置读取
	if gitURL == "" {
		// 尝试从默认仓库（main）读取远程URL
		defaultRepoPath := filepath.Join(repositoriesDir, "main")
		if remoteURL, err := getRemoteURLFromGit(defaultRepoPath); err == nil && remoteURL != "" {
			gitURL = remoteURL
			fmt.Printf("✓ 从现有Git仓库读取远程URL: %s\n", gitURL)
		}
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建新配置文件（只支持多仓库模式）
		configContent := fmt.Sprintf(`# skill-hub 配置文件（多仓库模式）
claude_config_path: "~/.claude/config.json"
cursor_config_path: "~/.cursor/rules"
default_tool: "open_code"
git_token: ""

# 多仓库配置（强制启用）
multi_repo:
  enabled: true
  default_repo: "%s"  # 默认仓库名称
  repositories:
    %s:
      name: "%s"
      url: "%s"
      branch: "master"
      enabled: true
      description: "主技能仓库"
      type: "user"
      is_archive: true
`, repoName, repoName, repoName, gitURL)

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return errors.WrapWithCode(err, "runInit", errors.ErrFileOperation, "创建配置文件失败")
		}
		fmt.Printf("✓ 创建配置文件: %s\n", configPath)
	} else {
		// 配置文件已存在，更新git_remote_url字段
		// 首先检查是否需要从git配置读取远程URL
		if gitURL == "" {
			repoPath := filepath.Join(skillHubDir, "repo")
			if remoteURL, err := getRemoteURLFromGit(repoPath); err == nil && remoteURL != "" {
				gitURL = remoteURL
				fmt.Printf("✓ 从现有Git仓库读取远程URL: %s\n", gitURL)
			}
		}

		// 配置文件已存在，确保它是多仓库配置
		fmt.Printf("✓ 配置文件已存在: %s\n", configPath)
		// 注意：现有配置文件需要手动更新为多仓库格式
		// 用户可以通过重新运行 init 或手动编辑 config.yaml 来更新
	}

	// 创建状态文件（在根目录）
	statePath := filepath.Join(skillHubDir, "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		initialState := `{}`
		if err := os.WriteFile(statePath, []byte(initialState), 0644); err != nil {
			return errors.WrapWithCode(err, "runInit", errors.ErrFileOperation, "创建状态文件失败")
		}
		fmt.Printf("✓ 创建状态文件: %s\n", statePath)
	} else {
		fmt.Printf("✓ 状态文件已存在: %s\n", statePath)
	}

	// 根据是否提供git_url执行不同的初始化逻辑
	repoAlreadyValid := false

	if gitURL != "" {
		// 情况1：提供了git_url，克隆远程仓库到repo目录

		// 检查是否已经是相同的git仓库
		if isSameGitRepo(repoPath, gitURL) {
			fmt.Println("\n✅ 检测到相同的远程仓库，跳过克隆")
		} else {
			fmt.Println("\n正在克隆远程技能仓库...")

			// 如果repo目录已存在且非空，备份
			if entries, err := os.ReadDir(repoPath); err == nil && len(entries) > 0 {
				backupDir := repoPath + ".bak." + time.Now().Format("20060102-150405")
				fmt.Printf("备份现有仓库到: %s\n", backupDir)
				if err := os.Rename(repoPath, backupDir); err != nil {
					return errors.WrapWithCode(err, "runInit", errors.ErrFileOperation, "备份失败")
				}
				// 重新创建空目录
				if err := utils.EnsureDir(repoPath); err != nil {
					return err
				}
			}

			// 创建临时Repository对象用于克隆
			tempRepo, err := newGitRepository(repoPath)
			if err != nil {
				return errors.WrapWithCode(err, "runInit", errors.ErrGitOperation, "创建仓库对象失败")
			}

			// 克隆远程仓库
			if err := tempRepo.Clone(gitURL); err != nil {
				fmt.Printf("⚠️  克隆远程仓库失败: %v\n", err)
				fmt.Println("\n故障排除建议:")
				fmt.Println("1. 对于SSH URL (git@...):")
				fmt.Println("   - 确保SSH agent正在运行: eval $(ssh-agent) && ssh-add ~/.ssh/id_rsa")
				fmt.Println("   - 或使用HTTPS URL代替: https://github.com/user/repo.git")
				fmt.Println("2. 对于HTTPS URL:")
				fmt.Println("   - 确保网络连接正常")
				fmt.Println("   - 如果需要认证，设置Git token: skill-hub config set git_token YOUR_TOKEN")
				fmt.Println("3. 检查URL格式是否正确")
				fmt.Println("\n将创建本地空仓库")

				// 如果克隆失败，创建本地空仓库
				return initLocalEmptyRepository(repoPath)
			}

			fmt.Println("✅ 远程技能仓库克隆完成")
		}

		// 刷新技能索引
		fmt.Println("\n正在刷新技能索引...")
		if err := rebuildRepositoryIndex(repoName); err != nil {
			fmt.Printf("⚠️  刷新技能索引失败: %v\n", err)
		} else {
			fmt.Println("✓ 技能索引已刷新")
		}

	} else {
		// 情况2：没有提供git_url
		// 检查repo目录是否已存在且符合要求
		if isRepoDirectoryValid(repoPath) {
			fmt.Println("\n✅ 检测到有效的技能仓库，直接使用现有仓库")
			repoAlreadyValid = true
		} else {
			// 初始化新的本地空git仓库
			if err := initLocalEmptyRepository(repoPath); err != nil {
				return err
			}
		}
	}

	fmt.Println("\n✅ skill-hub 初始化完成！")
	fmt.Println("工作区位置:", skillHubDir)

	if gitURL != "" {
		fmt.Println("远程仓库:", gitURL)
		fmt.Println("使用 'skill-hub git sync' 同步最新技能")
	} else {
		if repoAlreadyValid {
			fmt.Println("使用现有技能仓库")
		} else {
			fmt.Println("本地空仓库已初始化")
		}
	}

	fmt.Println("\n使用 'skill-hub list' 查看可用技能")

	// 检查当前目录的项目状态，如果为空则设置目标
	if err := setDefaultTargetIfEmpty(target); err != nil {
		fmt.Printf("⚠️  设置默认目标失败: %v\n", err)
	}

	// 清理可能创建的备份目录
	if gitURL != "" {
		if err := cleanupTimestampedBackupDirs(repoPath); err != nil {
			fmt.Printf("⚠️  清理备份目录失败: %v\n", err)
			logger.Warn("清理备份目录失败", "error", err.Error())
		}
	}

	// 记录初始化成功
	logger.Info("skill-hub初始化成功完成",
		"skill_hub_dir", skillHubDir,
		"git_url", gitURL,
		"repo_dir", repoPath,
		"duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

// isRepoDirectoryValid 检查repo目录是否有效
// 有效的repo目录需要满足：
// 1. 目录存在
// 2. 是git仓库（包含.git目录）
// 3. 包含skills子目录
func isRepoDirectoryValid(repoPath string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return false
	}

	// 检查是否是git仓库
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}

	// 检查是否包含skills目录
	skillsDir := filepath.Join(repoPath, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return false
	}

	return true
}

// initLocalEmptyRepository 在repo目录初始化本地空git仓库
func initLocalEmptyRepository(repoPath string) error {
	fmt.Println("\n正在初始化本地空技能仓库...")

	// 创建必要的子目录
	dirs := []string{
		filepath.Join(repoPath, "skills"),
		filepath.Join(repoPath, "template"),
	}

	for _, dir := range dirs {
		if err := utils.EnsureDir(dir); err != nil {
			return err
		}
		fmt.Printf("✓ 创建目录: %s\n", dir)
	}

	// 初始化git仓库（NewRepository会自动初始化如果不存在）
	_, err := newGitRepository(repoPath)
	if err != nil {
		return errors.WrapWithCode(err, "initLocalEmptyRepository", errors.ErrGitOperation, "初始化git仓库失败")
	}
	fmt.Println("✓ 初始化git仓库")

	repoName := filepath.Base(repoPath)
	if err := rebuildRepositoryIndex(repoName); err != nil {
		return errors.WrapWithCode(err, "initLocalEmptyRepository", errors.ErrFileOperation, "创建技能索引失败")
	}
	fmt.Printf("✓ 创建仓库技能索引: %s\n", filepath.Join(repoPath, "registry.json"))

	return nil
}

// isSameGitRepo 检查repo目录是否已经是相同的git仓库
func isSameGitRepo(repoPath, gitURL string) bool {
	// 检查是否是git仓库
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}

	// 读取git配置检查远程URL
	configPath := filepath.Join(gitDir, "config")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	// 在配置文件中查找远程URL
	configStr := string(configContent)
	lines := strings.Split(configStr, "\n")

	// 查找[remote "origin"]部分
	inOriginSection := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == `[remote "origin"]` {
			inOriginSection = true
			continue
		}

		if inOriginSection && strings.HasPrefix(trimmedLine, "url = ") {
			remoteURL := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "url = "))
			return remoteURL == gitURL
		}

		// 如果遇到新的section，退出origin section
		if inOriginSection && strings.HasPrefix(trimmedLine, "[") {
			break
		}
	}

	return false
}

// checkAlreadyInitialized 检查是否已经初始化了相同的配置
func checkAlreadyInitialized(skillHubDir, gitURL string) (bool, error) {
	// 检查配置文件是否存在
	configPath := filepath.Join(skillHubDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	// 配置文件已存在
	// 在多仓库模式下，我们只检查配置文件是否存在
	// 如果用户想要更改仓库配置，他们需要手动编辑配置文件或重新运行init
	return true, nil
}

// setDefaultTargetIfEmpty 在init时检查当前目录的项目状态，如果状态文件不存在则设置目标
func setDefaultTargetIfEmpty(target string) error {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	// 创建状态管理器
	stateManager, err := newStateManager()
	if err != nil {
		return err
	}

	// 检查状态文件是否存在
	if _, err := os.Stat(stateManager.GetStatePath()); os.IsNotExist(err) {
		// 状态文件不存在，这是一个新项目，设置目标
		// 如果target为空，使用默认值open_code
		if target == "" {
			target = spec.TargetOpenCode
		}
		if err := stateManager.SetPreferredTarget(cwd, target); err != nil {
			return errors.Wrap(err, "设置默认目标失败")
		}
		fmt.Printf("✅ 已为当前项目设置默认目标: %s\n", target)
	}

	return nil
}

// getRemoteURLFromGit 从现有Git仓库读取远程URL
func getRemoteURLFromGit(repoPath string) (string, error) {
	// 检查.git目录是否存在
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", errors.NewWithCode("getRemoteURLFromGit", errors.ErrGitOperation, "Git仓库不存在")
	}

	// 使用git命令读取远程URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "读取Git远程URL失败")
	}

	return strings.TrimSpace(string(output)), nil
}
