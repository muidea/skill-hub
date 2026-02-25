package cli

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "检查技能状态",
	Long: `对比项目本地工作区文件与技能仓库源文件的差异，显示技能状态：
- Synced: 本地与仓库一致
- Modified: 本地有未反馈的修改
- Outdated: 仓库版本领先于本地
- Missing: 技能已启用但本地文件缺失`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skillID := ""
		if len(args) > 0 {
			skillID = args[0]
		}
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runStatus(skillID, verbose)
	},
}

func init() {
	statusCmd.Flags().Bool("verbose", false, "显示详细差异信息")
}

func runStatus(skillID string, verbose bool) error {
	// 检查init依赖（规范4.9：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Println("检查技能状态...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	// 检查项目工作区状态（规范4.9：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, "")
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
	}

	// 加载项目状态
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 获取项目状态
	projectState, err := stateManager.LoadProjectState(cwd)
	if err != nil {
		return err
	}

	skills := projectState.Skills
	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		return nil
	}

	// 如果指定了skillID，只检查该技能
	if skillID != "" {
		if _, exists := skills[skillID]; !exists {
			return fmt.Errorf("技能 %s 未在当前项目中启用", skillID)
		}
		// 创建一个只包含指定技能的map
		singleSkill := map[string]spec.SkillVars{
			skillID: skills[skillID],
		}
		skills = singleSkill
	}

	// 显示项目信息
	fmt.Printf("项目路径: %s\n", cwd)
	fmt.Printf("启用技能数: %d\n", len(skills))
	if skillID != "" {
		fmt.Printf("检查特定技能: %s\n", skillID)
	}
	fmt.Println()

	// 检查项目本地工作区文件
	fmt.Println("检查项目本地工作区文件...")

	results := make(map[string]string) // skillID -> status

	for skillID, skillVars := range skills {
		// 检查.agents/skills/[skillID]目录
		agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
		skillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")

		// 检查本地文件是否存在
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			results[skillID] = spec.SkillStatusMissing
			// 更新状态到state.json
			updateSkillStatus(cwd, skillID, spec.SkillStatusMissing, skillVars.Version)
			continue
		}

		// 获取本地技能信息
		localVersion, localHash, err := getLocalSkillInfo(skillMdPath)
		if err != nil {
			// 如果获取本地技能信息失败，可能是文件格式错误或其他问题
			// 这种情况下，如果文件存在但无法读取，应该标记为Modified而不是Error
			fmt.Printf("⚠️  获取技能 %s 信息失败，标记为Modified: %v\n", skillID, err)
			results[skillID] = spec.SkillStatusModified
			updateSkillStatus(cwd, skillID, spec.SkillStatusModified, "unknown")
			continue
		}

		// 获取仓库技能信息
		repoVersion, repoHash, err := getRepoSkillInfo(skillID)
		if err != nil {
			// 如果仓库中不存在该技能，可能是本地创建的技能
			results[skillID] = spec.SkillStatusModified
			if verbose {
				fmt.Printf("  ℹ️  技能 %s 在仓库中不存在，标记为 Modified\n", skillID)
			}
			updateSkillStatus(cwd, skillID, spec.SkillStatusModified, localVersion)
			continue
		}

		// 比较版本和内容
		status := determineSkillStatus(localVersion, localHash, repoVersion, repoHash)
		results[skillID] = status

		// 更新状态到state.json
		updateSkillStatus(cwd, skillID, status, localVersion)
	}

	// 显示结果
	fmt.Println("\n=== 技能状态 ===")

	// 计算最大ID长度用于动态列宽
	maxIDLength := 2 // 至少"ID"的长度
	for skillID := range results {
		if len(skillID) > maxIDLength {
			maxIDLength = len(skillID)
		}
	}

	// 生成标题行
	fmt.Printf("%-*s 状态\n", maxIDLength, "ID")
	fmt.Println(strings.Repeat("-", maxIDLength+4)) // +4 为了" 状态"

	for skillID, status := range results {
		statusSymbol := "❓"
		switch status {
		case "Synced":
			statusSymbol = "✅"
		case "Modified":
			statusSymbol = "⚠️"
		case "Outdated":
			statusSymbol = "🔄"
		case "Missing":
			statusSymbol = "❌"
		}
		fmt.Printf("%-*s %s %s\n", maxIDLength, skillID, statusSymbol, status)
	}

	if verbose {
		fmt.Println("\n=== 详细差异信息 ===")
		fmt.Println("⚠️  详细差异检查功能暂未实现")
		fmt.Println("此功能将显示项目本地工作区文件与技能仓库源文件的具体差异")
	}

	fmt.Println("\n说明:")
	fmt.Println("✅ Synced: 本地与仓库一致")
	fmt.Println("⚠️  Modified: 本地有未反馈的修改")
	fmt.Println("🔄 Outdated: 仓库版本领先于本地")
	fmt.Println("❌ Missing: 技能已启用但本地文件缺失")

	if skillID == "" {
		fmt.Println("\n使用 'skill-hub status <id>' 检查特定技能状态")
		fmt.Println("使用 'skill-hub status --verbose' 显示详细差异")
	}

	return nil
}

// getLocalSkillInfo 获取本地技能信息（版本和文件哈希）
func getLocalSkillInfo(skillMdPath string) (string, string, error) {
	// 读取文件内容
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", "", utils.ReadFileErr(err, skillMdPath)
	}

	// 计算文件哈希
	hash := md5.Sum(content)
	hashStr := fmt.Sprintf("%x", hash)

	// 解析YAML frontmatter获取版本
	version := "1.0.0" // 默认版本
	lines := strings.Split(string(content), "\n")
	if len(lines) > 2 && lines[0] == "---" {
		var frontmatterLines []string
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				break
			}
			frontmatterLines = append(frontmatterLines, lines[i])
		}

		frontmatter := strings.Join(frontmatterLines, "\n")
		var skillData map[string]interface{}
		if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err == nil {
			if metadata, ok := skillData["metadata"].(map[string]interface{}); ok {
				if v, ok := metadata["version"].(string); ok {
					version = v
				}
			} else if v, ok := skillData["version"].(string); ok {
				// 兼容旧格式：version直接在根级别
				version = v
			}
		}
	}

	return version, hashStr, nil
}

// getRepoSkillInfo 获取仓库技能信息
func getRepoSkillInfo(skillID string) (string, string, error) {
	// 获取配置
	cfg, err := config.GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("获取配置失败: %w", err)
	}

	// 多仓库模式：获取默认仓库路径
	var repoPath string
	if cfg.MultiRepo != nil {
		rootDir, err := config.GetRootDir()
		if err != nil {
			return "", "", fmt.Errorf("获取根目录失败: %w", err)
		}
		repoPath = filepath.Join(rootDir, "repositories", cfg.MultiRepo.DefaultRepo)
	} else {
		return "", "", fmt.Errorf("多仓库配置未初始化")
	}

	// 检查仓库中是否存在该技能
	repoSkillPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")
	if _, err := os.Stat(repoSkillPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("技能在仓库中不存在")
	}

	// 获取仓库技能信息
	return getLocalSkillInfo(repoSkillPath)
}

// determineSkillStatus 根据版本和哈希确定技能状态
func determineSkillStatus(localVersion, localHash, repoVersion, repoHash string) string {
	// 首先比较文件内容哈希
	if localHash != repoHash {
		// 文件内容不同，需要进一步判断哪个版本更新
		if compareVersions(localVersion, repoVersion) < 0 {
			// 仓库版本更高
			return spec.SkillStatusOutdated
		} else {
			// 本地版本更高或相同，但内容不同，说明本地有修改
			return spec.SkillStatusModified
		}
	}

	// 文件内容相同，检查版本
	if compareVersions(localVersion, repoVersion) < 0 {
		// 虽然内容相同但版本号不同，可能是仓库有更新但内容没变
		return spec.SkillStatusOutdated
	}

	// 内容和版本都相同
	return spec.SkillStatusSynced
}

// compareVersions 比较版本号（简化实现）
func compareVersions(v1, v2 string) int {
	// 移除可能的引号
	v1 = strings.Trim(v1, `"`)
	v2 = strings.Trim(v2, `"`)

	// 简单字符串比较
	if v1 == v2 {
		return 0
	}

	// 尝试解析为数字比较
	// 这里简化处理，只比较主要版本号
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		// 尝试转换为数字比较
		num1 := 0
		num2 := 0
		fmt.Sscanf(v1Parts[i], "%d", &num1)
		fmt.Sscanf(v2Parts[i], "%d", &num2)

		if num1 > num2 {
			return 1
		} else if num1 < num2 {
			return -1
		}
	}

	// 如果前面的部分都相同，长度更长的版本号更大
	if len(v1Parts) > len(v2Parts) {
		return 1
	} else if len(v1Parts) < len(v2Parts) {
		return -1
	}

	// 作为最后的手段，使用字符串比较
	if v1 > v2 {
		return 1
	}
	return -1
}

// updateSkillStatus 更新技能状态到state.json
func updateSkillStatus(projectPath, skillID, status, version string) error {
	// 创建状态管理器
	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("创建状态管理器失败: %w", err)
	}

	// 加载当前项目状态
	projectState, err := stateManager.LoadProjectState(projectPath)
	if err != nil {
		return fmt.Errorf("加载项目状态失败: %w", err)
	}

	// 更新技能状态
	if skillVars, exists := projectState.Skills[skillID]; exists {
		skillVars.Status = status
		skillVars.Version = version
		projectState.Skills[skillID] = skillVars
	} else {
		// 技能不存在于状态中，添加它
		projectState.Skills[skillID] = spec.SkillVars{
			SkillID: skillID,
			Version: version,
			Status:  status,
			Variables: map[string]string{
				"target": "open_code", // 默认值
			},
		}
	}

	// 保存项目状态
	if err := stateManager.SaveProjectState(projectState); err != nil {
		return fmt.Errorf("保存项目状态失败: %w", err)
	}

	return nil
}
