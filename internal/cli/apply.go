package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var (
	dryRun bool
	target string
	mode   string
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "将已启用的技能应用到当前项目",
	Long: `将当前项目已启用的技能分发到目标工具配置文件。

使用 --dry-run 参数可以预览变更而不实际修改文件。
使用 --target 参数指定目标工具 (cursor/claude_code/all)。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply()
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览变更而不实际修改文件")
	applyCmd.Flags().StringVar(&target, "target", "", "目标工具: cursor, claude_code, all (为空时使用状态绑定的目标)")
	applyCmd.Flags().StringVar(&mode, "mode", "project", "配置模式: project (项目级), global (全局)")
}

func runApply() error {
	fmt.Println("正在应用技能到当前项目...")

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

	// 确定目标工具
	resolvedTarget := target
	if resolvedTarget == spec.TargetAll {
		// 如果指定了all，直接使用all
	} else if resolvedTarget == "" {
		// 如果没有指定target，尝试从状态获取
		projectState, err := stateMgr.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("查找项目状态失败: %w", err)
		}

		if projectState == nil || projectState.PreferredTarget == "" {
			// 未绑定项目
			fmt.Println("❌ 当前目录未关联目标")
			fmt.Println("请先执行以下操作之一:")
			fmt.Printf("  1. 使用 'skill-hub set-target [%s|%s]' 设置首选目标\n", spec.TargetCursor, spec.TargetClaudeCode)
			fmt.Printf("  2. 使用 'skill-hub use [skill-id] --target [%s|%s]' 启用技能并指定目标\n", spec.TargetCursor, spec.TargetClaudeCode)
			fmt.Printf("  3. 使用 'skill-hub apply --target [%s|%s|%s]' 显式指定目标\n", spec.TargetCursor, spec.TargetClaudeCode, spec.TargetAll)
			return nil
		}

		resolvedTarget = spec.NormalizeTarget(projectState.PreferredTarget)
		fmt.Printf("🔍 使用状态绑定的目标: %s\n", resolvedTarget)
	}

	fmt.Printf("当前项目: %s\n", cwd)
	fmt.Printf("目标工具: %s\n", resolvedTarget)

	skills, err := stateMgr.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		fmt.Println("使用 'skill-hub use <skill-id>' 启用技能")
		return nil
	}

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 检查技能与目标的兼容性（当使用状态绑定的目标时）
	if target == "" && resolvedTarget != spec.TargetAll {
		fmt.Println("\n🔍 检查技能与目标兼容性...")
		incompatibleSkills := []string{}

		for skillID := range skills {
			skill, err := skillManager.LoadSkill(skillID)
			if err != nil {
				continue
			}

			if resolvedTarget == spec.TargetCursor && !skill.Compatibility.Cursor {
				incompatibleSkills = append(incompatibleSkills, fmt.Sprintf("%s (仅支持 Claude Code)", skillID))
			} else if resolvedTarget == spec.TargetClaudeCode && !skill.Compatibility.ClaudeCode {
				incompatibleSkills = append(incompatibleSkills, fmt.Sprintf("%s (仅支持 Cursor)", skillID))
			}
		}

		if len(incompatibleSkills) > 0 {
			fmt.Println("⚠️  警告: 以下技能与项目首选目标不兼容:")
			for _, skill := range incompatibleSkills {
				fmt.Printf("   - %s\n", skill)
			}
			fmt.Println("   这些技能将不会被应用到目标工具")
			fmt.Println("   考虑: 1) 修改技能兼容性 2) 切换项目目标 3) 使用 --target all 应用所有兼容技能")
		}
	}

	// 根据目标选择适配器
	var adapters []adapter.Adapter

	if resolvedTarget == spec.TargetAll || resolvedTarget == spec.TargetCursor {
		cursorAdapter := cursor.NewCursorAdapter()
		if mode == "global" {
			cursorAdapter = cursorAdapter.WithGlobalMode()
		} else {
			cursorAdapter = cursorAdapter.WithProjectMode()
		}
		adapters = append(adapters, cursorAdapter)
	}

	if resolvedTarget == spec.TargetAll || resolvedTarget == spec.TargetClaudeCode {
		claudeAdapter := claude.NewClaudeAdapter()
		if mode == "global" {
			claudeAdapter = claudeAdapter.WithGlobalMode()
		} else {
			claudeAdapter = claudeAdapter.WithProjectMode()
		}
		adapters = append(adapters, claudeAdapter)
	}

	if len(adapters) == 0 {
		return fmt.Errorf("无效的目标工具: %s，可用选项: %s, %s, %s", resolvedTarget, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetAll)
	}

	// 应用每个技能到每个适配器
	totalApplied := 0

	for _, adapter := range adapters {
		adapterName := getAdapterName(adapter)
		fmt.Printf("\n=== 处理 %s 适配器 ===\n", adapterName)

		adapterApplied := 0
		for skillID, skillVars := range skills {
			fmt.Printf("\n处理技能: %s\n", skillID)

			// 加载技能详情
			skill, err := skillManager.LoadSkill(skillID)
			if err != nil {
				fmt.Printf("⚠️  跳过技能 %s: %v\n", skillID, err)
				continue
			}

			// 检查适配器支持
			if !adapterSupportsSkill(adapter, skill) {
				fmt.Printf("ℹ️  技能 %s 不支持 %s，跳过\n", skillID, adapterName)
				continue
			}

			// 获取提示词内容
			prompt, err := skillManager.GetSkillPrompt(skillID)
			if err != nil {
				fmt.Printf("⚠️  跳过技能 %s: %v\n", skillID, err)
				continue
			}

			if dryRun {
				fmt.Printf("🔍 DRY RUN - 将应用技能 %s 到 %s\n", skillID, adapterName)
				fmt.Printf("变量: %v\n", skillVars.Variables)
				adapterApplied++
				continue
			}

			// 实际应用技能
			if err := adapter.Apply(skillID, prompt, skillVars.Variables); err != nil {
				fmt.Printf("❌ 应用技能 %s 到 %s 失败: %v\n", skillID, adapterName, err)
				continue
			}

			fmt.Printf("✓ 成功应用技能 %s 到 %s\n", skillID, adapterName)
			adapterApplied++
		}

		if adapterApplied > 0 {
			fmt.Printf("\n✅ %s: 成功应用 %d 个技能\n", adapterName, adapterApplied)
			totalApplied += adapterApplied
		} else {
			fmt.Printf("\nℹ️  %s: 没有技能被应用\n", adapterName)
		}
	}

	if totalApplied > 0 {
		fmt.Printf("\n🎉 总计成功应用 %d 个技能\n", totalApplied)
		fmt.Println("使用 'skill-hub status' 检查技能状态")
	} else {
		fmt.Println("\nℹ️  没有技能被应用到任何适配器")
	}

	return nil
}

// getAdapterName 获取适配器名称
func getAdapterName(adpt adapter.Adapter) string {
	if _, ok := adpt.(*cursor.CursorAdapter); ok {
		return "Cursor"
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return "Claude"
	}
	return "Unknown"
}

// adapterSupportsSkill 检查适配器是否支持该技能
func adapterSupportsSkill(adpt adapter.Adapter, skill *spec.Skill) bool {
	// 使用类型断言
	if _, ok := adpt.(*cursor.CursorAdapter); ok {
		return skill.Compatibility.Cursor
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return skill.Compatibility.ClaudeCode
	}
	return false
}
