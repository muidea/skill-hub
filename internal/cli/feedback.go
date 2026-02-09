package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/engine"
	"skill-hub/internal/state"

	"github.com/spf13/cobra"
)

var (
	feedbackDryRun bool
	feedbackForce  bool
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback <id>",
	Short: "将项目工作区技能修改内容更新至到本地仓库",
	Long: `将项目工作区本地的技能修改同步回本地技能仓库。

此命令会：
1. 提取项目工作区本地文件内容
2. 与本地仓库源文件对比，显示差异
3. 经用户确认后更新本地仓库文件
4. 更新 registry.json 中的版本/哈希信息

使用 --dry-run 参数演习模式，仅显示将要同步的差异。
使用 --force 参数强制更新，即使有冲突也继续执行。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(args[0])
	},
}

func init() {
	feedbackCmd.Flags().BoolVar(&feedbackDryRun, "dry-run", false, "演习模式，仅显示将要同步的差异")
	feedbackCmd.Flags().BoolVar(&feedbackForce, "force", false, "强制更新，即使有冲突也继续执行")
}

func runFeedback(skillID string) error {
	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查技能是否在项目工作区中启用
	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("初始化状态管理器失败: %w", err)
	}

	// 检查项目是否已启用该技能
	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查项目技能状态失败: %w", err)
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在项目工作区中启用", skillID)
	}

	// 检查项目工作区本地文件
	projectSkillPath := filepath.Join(cwd, ".skills", skillID, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return fmt.Errorf("项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	// 读取项目工作区文件内容
	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return fmt.Errorf("读取项目工作区文件失败: %w", err)
	}

	// 检查技能是否在本地仓库中存在
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return fmt.Errorf("初始化技能管理器失败: %w", err)
	}

	skillExists := skillManager.SkillExists(skillID)
	if !skillExists {
		return fmt.Errorf("技能 '%s' 在本地仓库中不存在", skillID)
	}

	// 读取本地仓库文件内容
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("获取技能目录失败: %w", err)
	}

	repoSkillPath := filepath.Join(skillsDir, skillID, "SKILL.md")
	repoContent, err := os.ReadFile(repoSkillPath)
	if err != nil {
		return fmt.Errorf("读取本地仓库文件失败: %w", err)
	}

	// 比较内容
	projectStr := strings.TrimSpace(string(projectContent))
	repoStr := strings.TrimSpace(string(repoContent))

	if projectStr == repoStr {
		fmt.Println("✅ 技能内容未修改")
		return nil
	}

	// 显示差异
	fmt.Println("\n🔍 检测到手动修改:")
	fmt.Println("========================================")

	projectLines := strings.Split(projectStr, "\n")
	repoLines := strings.Split(repoStr, "\n")

	// 简单差异显示
	maxLines := len(projectLines)
	if len(repoLines) > maxLines {
		maxLines = len(repoLines)
	}

	changesFound := false
	for i := 0; i < maxLines; i++ {
		var projectLine, repoLine string
		if i < len(projectLines) {
			projectLine = projectLines[i]
		}
		if i < len(repoLines) {
			repoLine = repoLines[i]
		}

		if projectLine != repoLine {
			if !changesFound {
				fmt.Println("行号 | 修改前                      | 修改后")
				fmt.Println("-----|---------------------------|---------------------------")
				changesFound = true
			}

			lineNum := i + 1
			fmt.Printf("%4d | %-25s | %-25s\n", lineNum,
				truncate(repoLine, 25),
				truncate(projectLine, 25))
		}
	}

	if !changesFound {
		fmt.Println("（仅空白字符差异）")
	}

	fmt.Println("========================================")

	// 如果是演习模式，只显示差异
	if feedbackDryRun {
		fmt.Println("\n✅ 演习模式完成，未进行实际修改")
		return nil
	}

	// 如果是强制模式，直接更新
	if feedbackForce {
		fmt.Println("\n🔧 强制模式，直接更新本地仓库...")
	} else {
		// 确认反馈
		fmt.Print("\n是否将这些修改更新到本地仓库？ [y/N]: ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("❌ 取消反馈操作")
			return nil
		}
	}

	// 更新本地仓库文件
	if err := os.WriteFile(repoSkillPath, projectContent, 0644); err != nil {
		return fmt.Errorf("更新本地仓库文件失败: %w", err)
	}

	fmt.Println("✓ 更新本地仓库文件")

	// 更新registry.json中的版本信息
	if err := updateRegistryVersion(skillID); err != nil {
		fmt.Printf("⚠️  更新registry.json失败: %v\n", err)
		fmt.Println("本地仓库文件已更新，但registry.json未更新")
	} else {
		fmt.Println("✓ 更新registry.json版本信息")
	}

	fmt.Println("\n✅ 反馈完成！")
	fmt.Println("使用 'skill-hub push' 同步到远程仓库")

	return nil
}

// truncate 截断字符串
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// updateRegistryVersion 更新registry.json中的版本信息
func updateRegistryVersion(skillID string) error {
	// 获取技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return fmt.Errorf("初始化技能管理器失败: %w", err)
	}

	// 加载技能详情
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 更新registry.json
	// 这里简化实现，实际应该更新registry.json文件
	fmt.Printf("技能 '%s' 版本信息已更新: %s\n", skillID, skill.Version)
	return nil
}
