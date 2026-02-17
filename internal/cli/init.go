package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"skill-hub/internal/adapter"
	"skill-hub/internal/git"
	"skill-hub/internal/state"
	"skill-hub/pkg/errors"
	"skill-hub/pkg/logging"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/utils"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
			fmt.Printf("检测到现有git仓库，将更新配置中的远程URL: %s\n", remoteURL)
			gitURL = remoteURL
			alreadyInitialized = false // 强制重新初始化以更新配置
		}
	}

	if alreadyInitialized {
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
			tempRepo, err := git.NewRepository(repoPath)
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
				return initLocalEmptyRepository(repoPath, skillHubDir)
			}

			fmt.Println("✅ 远程技能仓库克隆完成")
		}

		// 刷新技能索引
		fmt.Println("\n正在刷新技能索引...")
		if err := refreshSkillRegistry(repoPath); err != nil {
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
			if err := initLocalEmptyRepository(repoPath, skillHubDir); err != nil {
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
		if err := adapter.CleanupTimestampedBackupDirs(repoPath); err != nil {
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
func initLocalEmptyRepository(repoPath, skillHubDir string) error {
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
	_, err := git.NewRepository(repoPath)
	if err != nil {
		return errors.WrapWithCode(err, "initLocalEmptyRepository", errors.ErrGitOperation, "初始化git仓库失败")
	}
	fmt.Println("✓ 初始化git仓库")

	// 创建初始registry.json（空的技能索引）- 在根目录
	registryPath := filepath.Join(skillHubDir, "registry.json")
	if err := createInitialRegistry(registryPath); err != nil {
		return errors.WrapWithCode(err, "initLocalEmptyRepository", errors.ErrFileOperation, "创建技能索引失败")
	}
	fmt.Printf("✓ 创建技能索引: %s\n", registryPath)

	return nil
}

// createInitialRegistry 创建初始技能索引
func createInitialRegistry(registryPath string) error {
	registryContent := `{
  "version": "1.0.0",
  "skills": []
}
`

	return os.WriteFile(registryPath, []byte(registryContent), 0644)
}

// parseSkillMetadata 从SKILL.md文件解析技能元数据
func parseSkillMetadata(mdPath, skillID string) (*spec.SkillMetadata, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, errors.WrapWithCode(err, "parseSkillMetadata", errors.ErrFileOperation, "读取SKILL.md失败")
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, errors.NewWithCode("parseSkillMetadata", errors.ErrSkillInvalid, "无效的SKILL.md格式: 缺少frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML frontmatter
	var skillData map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err != nil {
		return nil, errors.WrapWithCode(err, "parseSkillMetadata", errors.ErrSkillInvalid, "解析frontmatter失败")
	}

	// 创建技能元数据对象
	skillMeta := &spec.SkillMetadata{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skillMeta.Name = name
	} else {
		skillMeta.Name = skillID
	}

	// 设置描述
	if desc, ok := skillData["description"].(string); ok {
		skillMeta.Description = desc
	}

	// 设置版本
	skillMeta.Version = "1.0.0"
	if version, ok := skillData["version"].(string); ok {
		skillMeta.Version = version
	}

	// 设置作者
	if author, ok := skillData["author"].(string); ok {
		skillMeta.Author = author
	} else if source, ok := skillData["source"].(string); ok {
		skillMeta.Author = source
	} else {
		skillMeta.Author = "unknown"
	}

	// 设置标签
	if tagsStr, ok := skillData["tags"].(string); ok {
		skillMeta.Tags = strings.Split(tagsStr, ",")
		for i, tag := range skillMeta.Tags {
			skillMeta.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 设置兼容性
	if compatData, ok := skillData["compatibility"]; ok {
		switch v := compatData.(type) {
		case string:
			skillMeta.Compatibility = v
		case map[string]interface{}:
			// 向后兼容：将对象格式转换为字符串
			var compatList []string
			if cursorVal, ok := v["cursor"].(bool); ok && cursorVal {
				compatList = append(compatList, "Cursor")
			}
			if claudeVal, ok := v["claude_code"].(bool); ok && claudeVal {
				compatList = append(compatList, "Claude Code")
			}
			if openCodeVal, ok := v["open_code"].(bool); ok && openCodeVal {
				compatList = append(compatList, "OpenCode")
			}
			if shellVal, ok := v["shell"].(bool); ok && shellVal {
				compatList = append(compatList, "Shell")
			}
			if len(compatList) > 0 {
				skillMeta.Compatibility = "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
			}
		}
	}

	return skillMeta, nil
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
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager()
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
			return fmt.Errorf("设置默认目标失败: %w", err)
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
		return "", fmt.Errorf("Git仓库不存在")
	}

	// 使用git命令读取远程URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("读取Git远程URL失败: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// refreshSkillRegistry 刷新技能索引
func refreshSkillRegistry(repoPath string) error {
	// registry.json现在在根目录
	skillHubDir := filepath.Dir(repoPath)
	registryPath := filepath.Join(skillHubDir, "registry.json")
	skillsDir := filepath.Join(repoPath, "skills")

	// 检查skills目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		// 如果skills目录不存在，创建空的registry.json
		registryContent := `{
  "version": "1.0.0",
  "skills": []
}`
		return os.WriteFile(registryPath, []byte(registryContent), 0644)
	}

	// 扫描skills目录下的所有子目录
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("读取skills目录失败: %w", err)
	}

	var skills []spec.SkillMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(skillsDir, skillID)
		skillMdPath := filepath.Join(skillDir, "SKILL.md")

		// 检查是否存在SKILL.md文件
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			continue
		}

		// 解析SKILL.md文件
		skillMeta, err := parseSkillMetadata(skillMdPath, skillID)
		if err != nil {
			fmt.Printf("⚠️  解析技能 %s 失败: %v\n", skillID, err)
			continue
		}

		skills = append(skills, *skillMeta)
	}

	// 创建registry对象
	registry := spec.Registry{
		Version: "1.0.0",
		Skills:  skills,
	}

	// 转换为JSON
	registryJSON, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化registry失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(registryPath, registryJSON, 0644); err != nil {
		return fmt.Errorf("写入registry.json失败: %w", err)
	}

	fmt.Printf("✓ 已索引 %d 个技能\n", len(skills))
	return nil
}
