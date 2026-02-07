package cli

import (
	"fmt"
	"os"
	"strings"

	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/converter"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/validator"

	"github.com/spf13/cobra"
)

var (
	dryRun         bool
	target         string
	mode           string
	autoFix        bool
	skipValidation bool
	strictMode     bool
	interactive    bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "将已启用的技能应用到当前项目",
	Long: `将当前项目已启用的技能分发到目标工具配置文件。

使用 --dry-run 参数可以预览变更而不实际修改文件。
使用 --target 参数指定目标工具 (cursor/claude_code/open_code/all)。

技能标准校验选项:
  --auto-fix        自动修复不符合标准的技能
  --skip-validation 跳过技能标准校验
  --strict          严格模式：发现不合规技能立即失败
  --interactive     交互式模式：询问用户确认修复`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply()
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览变更而不实际修改文件")
	applyCmd.Flags().StringVar(&target, "target", "", "目标工具: cursor, claude_code, open_code, all (为空时使用状态绑定的目标)")
	applyCmd.Flags().StringVar(&mode, "mode", "project", "配置模式: project (项目级), global (全局)")
	applyCmd.Flags().BoolVar(&autoFix, "auto-fix", false, "自动修复不符合标准的技能")
	applyCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "跳过技能标准校验")
	applyCmd.Flags().BoolVar(&strictMode, "strict", false, "严格模式：发现不合规技能立即失败")
	applyCmd.Flags().BoolVar(&interactive, "interactive", false, "交互式模式：询问用户确认修复")
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
	switch resolvedTarget {
	case spec.TargetAll:
		// 如果指定了all，直接使用all
	case "":
		// 如果没有指定target，尝试从状态获取
		projectState, err := stateMgr.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("查找项目状态失败: %w", err)
		}

		if projectState == nil {
			// 项目状态不存在，使用LoadProjectState创建默认状态
			projectState, err = stateMgr.LoadProjectState(cwd)
			if err != nil {
				return fmt.Errorf("加载项目状态失败: %w", err)
			}
			// 保存新创建的状态
			if err := stateMgr.SaveProjectState(projectState); err != nil {
				return fmt.Errorf("保存项目状态失败: %w", err)
			}
		}

		if projectState.PreferredTarget == "" {
			// 未绑定项目
			fmt.Println("❌ 当前目录未关联目标")
			fmt.Println("请先执行以下操作之一:")
			fmt.Printf("  1. 使用 'skill-hub set-target [%s|%s|%s]' 设置首选目标\n", spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode)
			fmt.Printf("  2. 使用 'skill-hub use [skill-id] --target [%s|%s|%s]' 启用技能并指定目标\n", spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode)
			fmt.Printf("  3. 使用 'skill-hub apply --target [%s|%s|%s|%s]' 显式指定目标\n", spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll)
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

			// 检查技能是否兼容当前目标
			isCompatible := false
			if skill.Compatibility != "" {
				compatLower := strings.ToLower(skill.Compatibility)
				targetLower := strings.ToLower(resolvedTarget)

				// 检查兼容性字符串中是否包含目标名称
				if strings.Contains(compatLower, targetLower) {
					isCompatible = true
				} else if resolvedTarget == spec.TargetOpenCode && strings.Contains(compatLower, "opencode") {
					isCompatible = true
				} else if resolvedTarget == spec.TargetClaudeCode && (strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code")) {
					isCompatible = true
				}
			} else {
				// 如果没有指定兼容性，假设兼容所有
				isCompatible = true
			}

			if !isCompatible {
				incompatibleSkills = append(incompatibleSkills, fmt.Sprintf("%s (不兼容 %s)", skillID, resolvedTarget))
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

	if resolvedTarget == spec.TargetAll || resolvedTarget == spec.TargetOpenCode {
		opencodeAdapter := opencode.NewOpenCodeAdapter()
		if mode == "global" {
			opencodeAdapter = opencodeAdapter.WithGlobalMode()
		} else {
			opencodeAdapter = opencodeAdapter.WithProjectMode()
		}
		adapters = append(adapters, opencodeAdapter)
	}

	if len(adapters) == 0 {
		return fmt.Errorf("无效的目标工具: %s，可用选项: %s, %s, %s, %s", resolvedTarget, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll)
	}

	// 应用每个技能到每个适配器
	totalApplied := 0

	for _, adapter := range adapters {
		adapterName := getAdapterName(adapter)
		fmt.Printf("\n=== 处理 %s 适配器 ===\n", adapterName)

		adapterApplied := 0
		for skillID, skillVars := range skills {
			fmt.Printf("\n处理技能: %s\n", skillID)

			// 获取技能文件路径
			skillPath, err := getSkillFilePath(skillManager, skillID)
			if err != nil {
				fmt.Printf("⚠️  跳过技能 %s: %v\n", skillID, err)
				continue
			}

			// 验证并修复技能
			if !skipValidation {
				valid, issues, err := validateAndFixSkill(skillPath, skillID, autoFix, skipValidation, strictMode, interactive)
				if err != nil {
					fmt.Printf("⚠️  技能验证失败 %s: %v\n", skillID, err)
					if strictMode {
						return fmt.Errorf("严格模式下验证失败: %s", skillID)
					}
					continue
				}

				if !valid {
					fmt.Printf("❌ 技能不符合标准: %s\n", skillID)
					for _, issue := range issues {
						fmt.Printf("  %s\n", issue)
					}

					if strictMode {
						return fmt.Errorf("严格模式下发现不合规技能: %s", skillID)
					}

					if !autoFix {
						fmt.Println("  使用 --auto-fix 自动修复或 --skip-validation 跳过验证")
						continue
					}
				}
			}

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
				// 尝试恢复操作
				if recoveryErr := attemptRecovery(adapter, skillID); recoveryErr != nil {
					fmt.Printf("⚠️  恢复操作失败: %v\n", recoveryErr)
				}
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

// validateAndFixSkill 验证并修复技能文件
func validateAndFixSkill(skillPath string, skillID string, autoFix, skipValidation, strictMode, interactive bool) (bool, []string, error) {
	if skipValidation {
		return true, nil, nil
	}

	// Create validator
	v := validator.NewValidator()
	options := validator.ValidationOptions{
		IgnoreWarnings: false,
		StrictMode:     strictMode,
	}

	// Validate the skill
	result, err := v.ValidateWithOptions(skillPath, options)
	if err != nil {
		return false, nil, fmt.Errorf("验证技能失败: %w", err)
	}

	// Check if skill is valid
	if result.IsValid && (!result.HasWarnings() || !strictMode) {
		return true, nil, nil
	}

	// Collect issues
	var issues []string
	if result.HasErrors() {
		for _, err := range result.Errors {
			issues = append(issues, fmt.Sprintf("❌ [%s] %s", err.Code, err.Message))
		}
	}
	if result.HasWarnings() {
		for _, warn := range result.Warnings {
			issues = append(issues, fmt.Sprintf("⚠️  [%s] %s", warn.Code, warn.Message))
		}
	}

	// If not auto-fixing, return issues
	if !autoFix {
		return false, issues, nil
	}

	// Auto-fix the skill
	fmt.Printf("\n🔧 正在自动修复技能: %s\n", skillID)

	// Create converter
	converter, err := converter.NewConverter()
	if err != nil {
		return false, issues, fmt.Errorf("创建转换器失败: %w", err)
	}

	// Preview conversion first
	preview, err := converter.PreviewConversion(skillPath, options)
	if err != nil {
		return false, issues, fmt.Errorf("预览修复失败: %w", err)
	}

	if len(preview.AppliedFixes) == 0 {
		fmt.Println("ℹ️  无需修复")
		return true, nil, nil
	}

	// Show what will be fixed
	fmt.Println("将应用以下修复:")
	for _, fix := range preview.AppliedFixes {
		fmt.Printf("  - %s\n", fix)
	}

	// If interactive mode, ask for confirmation
	if interactive {
		fmt.Print("\n是否应用这些修复? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("跳过修复")
			return false, issues, nil
		}
	}

	// Apply the fixes
	conversionResult, err := converter.ConvertSkill(skillPath, options)
	if err != nil {
		return false, issues, fmt.Errorf("应用修复失败: %w", err)
	}

	// Show results
	fmt.Printf("✅ 成功应用 %d 个修复\n", len(conversionResult.AppliedFixes))
	if len(conversionResult.Errors) > 0 {
		fmt.Println("修复后仍存在的错误:")
		for _, err := range conversionResult.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
	if len(conversionResult.Warnings) > 0 {
		fmt.Println("修复后仍存在的警告:")
		for _, warn := range conversionResult.Warnings {
			fmt.Printf("  - %s\n", warn)
		}
	}

	// Validate again after fixing
	result, err = v.ValidateWithOptions(skillPath, options)
	if err != nil {
		return false, issues, fmt.Errorf("重新验证失败: %w", err)
	}

	return result.IsValid && (!result.HasWarnings() || !strictMode), nil, nil
}

// attemptRecovery 尝试恢复失败的技能应用
func attemptRecovery(adpt adapter.Adapter, skillID string) error {
	// 尝试从适配器移除残留内容
	if err := adpt.Remove(skillID); err != nil {
		return fmt.Errorf("移除残留内容失败: %w", err)
	}

	// 检查适配器是否支持备份恢复
	if cursorAdapter, ok := adpt.(*cursor.CursorAdapter); ok {
		// 对于Cursor适配器，检查备份文件
		filePath, err := cursorAdapter.GetFilePath()
		if err != nil {
			return err
		}

		backupPath := filePath + ".bak"
		if _, err := os.Stat(backupPath); err == nil {
			// 备份文件存在，尝试恢复
			if err := os.Rename(backupPath, filePath); err != nil {
				return fmt.Errorf("恢复备份失败: %w", err)
			}
			return nil
		}
	}

	return nil
}

// getSkillFilePath 获取技能文件路径
func getSkillFilePath(skillManager *engine.SkillManager, skillID string) (string, error) {
	// Try to get skills directory
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return "", fmt.Errorf("获取技能目录失败: %w", err)
	}

	// Only use standard structure: skills/skillID
	skillDir := fmt.Sprintf("%s/%s", skillsDir, skillID)
	skillPath := fmt.Sprintf("%s/SKILL.md", skillDir)
	if _, err := os.Stat(skillPath); err == nil {
		return skillPath, nil
	}

	return "", fmt.Errorf("找不到技能文件: %s", skillID)
}

// getAdapterName 获取适配器名称
func getAdapterName(adpt adapter.Adapter) string {
	if _, ok := adpt.(*cursor.CursorAdapter); ok {
		return "Cursor"
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return "Claude"
	}
	if _, ok := adpt.(*opencode.OpenCodeAdapter); ok {
		return "OpenCode"
	}
	return "Unknown"
}

// adapterSupportsSkill 检查适配器是否支持该技能
func adapterSupportsSkill(adpt adapter.Adapter, skill *spec.Skill) bool {
	// 如果没有指定兼容性，假设兼容所有
	if skill.Compatibility == "" {
		return true
	}

	compatLower := strings.ToLower(skill.Compatibility)

	// 使用类型断言检查适配器类型
	if _, ok := adpt.(*cursor.CursorAdapter); ok {
		return strings.Contains(compatLower, "cursor")
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code")
	}
	if _, ok := adpt.(*opencode.OpenCodeAdapter); ok {
		return strings.Contains(compatLower, "opencode")
	}
	return false
}
