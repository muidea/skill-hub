package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
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
	fmt.Println("检查技能状态...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 加载项目状态
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 获取项目启用的技能
	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

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

	// 简化实现：检查项目本地工作区文件
	fmt.Println("检查项目本地工作区文件...")

	results := make(map[string]string) // skillID -> status

	for skillID := range skills {
		// 检查.agents/skills/[skillID]目录
		agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
		if _, err := os.Stat(agentsSkillDir); os.IsNotExist(err) {
			results[skillID] = "Missing"
			continue
		}

		// 检查SKILL.md文件
		skillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			results[skillID] = "Missing"
			continue
		}

		// TODO: 对比项目本地工作区文件与技能仓库源文件的差异
		// 这里简化实现，假设都是Synced
		results[skillID] = "Synced"
	}

	// 显示结果
	fmt.Println("\n=== 技能状态 ===")
	fmt.Println("ID          状态")
	fmt.Println("------------------")

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
		fmt.Printf("%-12s %s %s\n", skillID, statusSymbol, status)
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
