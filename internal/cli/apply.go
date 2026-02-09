package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"skill-hub/internal/state"
	"skill-hub/pkg/spec"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "应用技能到项目",
	Long: `根据 state.json 中的启用记录和目标环境设置，将技能物理分发到项目。具体行为取决于项目工作区设置的目标环境：
- cursor: 注入到 .cursorrules 文件
- claude: 更新 Claude 配置文件
- open_code: 创建 .skills/[id]/ 目录结构`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		return runApply(dryRun, force)
	},
}

func init() {
	applyCmd.Flags().Bool("dry-run", false, "演习模式，仅显示将要执行的变更，不实际修改文件")
	applyCmd.Flags().Bool("force", false, "强制应用，即使检测到冲突也继续执行")
}

func runApply(dryRun bool, force bool) error {
	fmt.Println("正在应用技能到项目...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 创建状态管理器
	stateMgr, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 获取项目状态
	projectState, err := stateMgr.FindProjectByPath(cwd)
	if err != nil {
		return fmt.Errorf("查找项目状态失败: %w", err)
	}

	if projectState == nil || projectState.PreferredTarget == "" {
		return fmt.Errorf("项目未设置目标环境，请先使用 'skill-hub set-target <value>' 设置目标环境")
	}

	target := spec.NormalizeTarget(projectState.PreferredTarget)
	fmt.Printf("项目目标环境: %s\n", target)
	fmt.Printf("项目路径: %s\n", cwd)

	// 获取项目启用的技能
	skills, err := stateMgr.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		fmt.Println("使用 'skill-hub use <skill-id>' 启用技能")
		return nil
	}

	fmt.Printf("启用技能数: %d\n", len(skills))

	if dryRun {
		fmt.Println("\n=== 演习模式 (dry-run) ===")
		fmt.Println("将显示将要执行的变更，不实际修改文件")
	}

	// 根据目标环境应用技能
	switch target {
	case spec.TargetCursor:
		return applyToCursor(cwd, skills, dryRun, force)
	case spec.TargetClaudeCode:
		return applyToClaude(cwd, skills, dryRun, force)
	case spec.TargetOpenCode:
		return applyToOpenCode(cwd, skills, dryRun, force)
	default:
		return fmt.Errorf("不支持的目标环境: %s", target)
	}
}

// applyToCursor 应用技能到Cursor
func applyToCursor(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到 Cursor ===")

	// 检查.cursorrules文件
	cursorRulesPath := filepath.Join(projectPath, ".cursorrules")

	if dryRun {
		fmt.Printf("将更新文件: %s\n", cursorRulesPath)
		fmt.Println("将注入以下技能:")
		for skillID := range skills {
			fmt.Printf("  - %s\n", skillID)
		}
		fmt.Println("\n注意: 实际实现需要将技能内容注入到.cursorrules文件中")
		return nil
	}

	// TODO: 实际实现 - 将技能内容注入到.cursorrules文件中
	fmt.Println("⚠️  Cursor适配器功能暂未完全实现")
	fmt.Println("将创建/更新.cursorrules文件并注入技能内容")

	return nil
}

// applyToClaude 应用技能到Claude
func applyToClaude(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到 Claude ===")

	// 检查Claude配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}
	claudeConfigPath := filepath.Join(homeDir, ".claude", "config.json")

	if dryRun {
		fmt.Printf("将更新配置文件: %s\n", claudeConfigPath)
		fmt.Println("将注入以下技能:")
		for skillID := range skills {
			fmt.Printf("  - %s\n", skillID)
		}
		fmt.Println("\n注意: 实际实现需要更新Claude配置文件")
		return nil
	}

	// TODO: 实际实现 - 更新Claude配置文件
	fmt.Println("⚠️  Claude适配器功能暂未完全实现")
	fmt.Println("将更新Claude配置文件以包含技能内容")

	return nil
}

// applyToOpenCode 应用技能到OpenCode
func applyToOpenCode(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到 OpenCode ===")

	// 创建.skills目录
	skillsDir := filepath.Join(projectPath, ".skills")

	if dryRun {
		fmt.Printf("将创建/更新目录: %s\n", skillsDir)
		fmt.Println("将创建以下技能目录:")
		for skillID := range skills {
			skillDir := filepath.Join(skillsDir, skillID)
			fmt.Printf("  - %s\n", skillDir)
		}
		fmt.Println("\n注意: 实际实现需要创建.skills/[id]/目录结构")
		return nil
	}

	// 创建.skills目录
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("创建.skills目录失败: %w", err)
	}
	fmt.Printf("✓ 创建目录: %s\n", skillsDir)

	// 为每个技能创建目录
	createdCount := 0
	for skillID := range skills {
		skillDir := filepath.Join(skillsDir, skillID)

		// 检查目录是否已存在
		if _, err := os.Stat(skillDir); err == nil {
			if !force {
				fmt.Printf("⚠️  技能目录已存在: %s (使用 --force 覆盖)\n", skillDir)
				continue
			}
			// 强制模式，删除现有目录
			if err := os.RemoveAll(skillDir); err != nil {
				fmt.Printf("⚠️  删除现有目录失败: %s: %v\n", skillDir, err)
				continue
			}
		}

		// 创建技能目录
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			fmt.Printf("⚠️  创建技能目录失败: %s: %v\n", skillDir, err)
			continue
		}

		// 创建SKILL.md文件（简化实现）
		skillMdPath := filepath.Join(skillDir, "SKILL.md")
		content := fmt.Sprintf(`# %s Skill

这是为OpenCode环境创建的技能目录。

技能ID: %s

注意: 这是自动生成的占位文件，实际技能内容应从.agents/skills/%s/SKILL.md复制。
`, skillID, skillID, skillID)

		if err := os.WriteFile(skillMdPath, []byte(content), 0644); err != nil {
			fmt.Printf("⚠️  创建SKILL.md失败: %s: %v\n", skillMdPath, err)
			continue
		}

		fmt.Printf("✓ 创建技能目录: %s\n", skillDir)
		createdCount++
	}

	fmt.Printf("\n✅ 成功创建 %d 个技能目录\n", createdCount)
	fmt.Println("技能已应用到OpenCode环境")
	fmt.Println("使用 'skill-hub status' 检查技能状态")

	return nil
}
