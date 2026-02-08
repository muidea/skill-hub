package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/adapter"
	"skill-hub/internal/git"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var initCmd = &cobra.Command{
	Use:   "init [git-url]",
	Short: "初始化Skill Hub工作区",
	Long: `初始化Skill Hub工作区，创建必要的配置文件和目录结构。

如果提供了Git仓库URL，会克隆远程仓库到本地。
如果没有提供URL，会创建一个空的本地仓库。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(args)
	},
}

func runInit(args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	skillHubDir := filepath.Join(homeDir, ".skill-hub")
	repoDir := filepath.Join(skillHubDir, "repo")

	fmt.Printf("正在初始化Skill Hub工作区: %s\n", skillHubDir)

	// 检查是否提供了Git URL
	var gitURL string
	if len(args) > 0 {
		gitURL = args[0]
		fmt.Printf("将克隆远程仓库: %s\n", gitURL)
	}

	// 创建基础目录结构
	dirs := []string{
		skillHubDir,
		repoDir,
		filepath.Join(homeDir, ".cursor"), // 全局Cursor配置目录
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		fmt.Printf("✓ 创建目录: %s\n", dir)
	}

	// 创建配置文件
	configPath := filepath.Join(skillHubDir, "config.yaml")
	configContent := fmt.Sprintf(`# Skill Hub 配置文件
repo_path: "~/.skill-hub/repo"
claude_config_path: "~/.claude/config.json"
cursor_config_path: "~/.cursor/rules"
default_tool: "cursor"
git_remote_url: "%s"
git_token: ""
git_branch: "main"
`, gitURL)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	fmt.Printf("✓ 创建配置文件: %s\n", configPath)

	// 创建状态文件（在根目录）
	statePath := filepath.Join(skillHubDir, "state.json")
	initialState := `{}`
	if err := os.WriteFile(statePath, []byte(initialState), 0644); err != nil {
		return fmt.Errorf("创建状态文件失败: %w", err)
	}
	fmt.Printf("✓ 创建状态文件: %s\n", statePath)

	// 根据是否提供git_url执行不同的初始化逻辑
	repoAlreadyValid := false

	if gitURL != "" {
		// 情况1：提供了git_url，克隆远程仓库到repo目录
		fmt.Println("\n正在克隆远程技能仓库...")

		// 如果repo目录已存在且非空，备份
		if entries, err := os.ReadDir(repoDir); err == nil && len(entries) > 0 {
			backupDir := repoDir + ".bak." + time.Now().Format("20060102-150405")
			fmt.Printf("备份现有仓库到: %s\n", backupDir)
			if err := os.Rename(repoDir, backupDir); err != nil {
				return fmt.Errorf("备份失败: %w", err)
			}
			// 重新创建空目录
			if err := os.MkdirAll(repoDir, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
		}

		// 创建临时Repository对象用于克隆
		tempRepo, err := git.NewRepository(repoDir)
		if err != nil {
			return fmt.Errorf("创建仓库对象失败: %w", err)
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
			return initLocalEmptyRepository(repoDir, skillHubDir)
		}

		fmt.Println("✅ 远程技能仓库克隆完成")

		// 刷新技能索引
		fmt.Println("\n正在刷新技能索引...")
		if err := refreshSkillRegistry(repoDir); err != nil {
			fmt.Printf("⚠️  刷新技能索引失败: %v\n", err)
		} else {
			fmt.Println("✓ 技能索引已刷新")
		}

	} else {
		// 情况2：没有提供git_url
		// 检查repo目录是否已存在且符合要求
		if isRepoDirectoryValid(repoDir) {
			fmt.Println("\n✅ 检测到有效的技能仓库，直接使用现有仓库")
			repoAlreadyValid = true
		} else {
			// 初始化新的本地空git仓库
			if err := initLocalEmptyRepository(repoDir, skillHubDir); err != nil {
				return err
			}
		}
	}

	fmt.Println("\n✅ Skill Hub 初始化完成！")
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

	// 检查当前目录的项目状态，如果为空则默认设置目标为 open_code
	if err := setDefaultTargetIfEmpty(); err != nil {
		fmt.Printf("⚠️  设置默认目标失败: %v\n", err)
	}

	// 清理可能创建的备份目录
	if gitURL != "" {
		if err := adapter.CleanupTimestampedBackupDirs(repoDir); err != nil {
			fmt.Printf("⚠️  清理备份目录失败: %v\n", err)
		}
	}

	return nil
}

// isRepoDirectoryValid 检查repo目录是否有效
// 有效的repo目录需要满足：
// 1. 目录存在
// 2. 是git仓库（包含.git目录）
// 3. 包含skills子目录
func isRepoDirectoryValid(repoDir string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return false
	}

	// 检查是否是git仓库
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}

	// 检查是否包含skills目录
	skillsDir := filepath.Join(repoDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return false
	}

	return true
}

// initLocalEmptyRepository 在repo目录初始化本地空git仓库
func initLocalEmptyRepository(repoDir, skillHubDir string) error {
	fmt.Println("\n正在初始化本地空技能仓库...")

	// 创建必要的子目录
	dirs := []string{
		filepath.Join(repoDir, "skills"),
		filepath.Join(repoDir, "template"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		fmt.Printf("✓ 创建目录: %s\n", dir)
	}

	// 初始化git仓库（NewRepository会自动初始化如果不存在）
	_, err := git.NewRepository(repoDir)
	if err != nil {
		return fmt.Errorf("初始化git仓库失败: %w", err)
	}
	fmt.Println("✓ 初始化git仓库")

	// 创建初始registry.json（空的技能索引）- 在根目录
	registryPath := filepath.Join(skillHubDir, "registry.json")
	if err := createInitialRegistry(registryPath); err != nil {
		return fmt.Errorf("创建技能索引失败: %w", err)
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
		return nil, fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
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
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
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

// setDefaultTargetIfEmpty 在init时检查当前目录的项目状态，如果状态文件不存在则默认设置目标为 open_code
func setDefaultTargetIfEmpty() error {
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
		// 状态文件不存在，这是一个新项目，设置默认目标为 open_code
		if err := stateManager.SetPreferredTarget(cwd, spec.TargetOpenCode); err != nil {
			return fmt.Errorf("设置默认目标失败: %w", err)
		}
		fmt.Printf("✅ 已为当前项目设置默认目标: %s\n", spec.TargetOpenCode)
	}

	return nil
}

// refreshSkillRegistry 刷新技能索引
func refreshSkillRegistry(repoDir string) error {
	// registry.json现在在根目录
	skillHubDir := filepath.Dir(repoDir)
	registryPath := filepath.Join(skillHubDir, "registry.json")
	skillsDir := filepath.Join(repoDir, "skills")

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
