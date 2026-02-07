package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/validator"
)

var (
	validateAdapter string
	validateStrict  bool
)

var validateLocalCmd = &cobra.Command{
	Use:   "validate-local [skill-id]",
	Short: "在本地验证技能的有效性",
	Long: `验证技能在本地项目中的有效性。

检查技能格式、变量配置和适配器兼容性。
生成验证报告，帮助识别和修复问题。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidateLocal(args[0])
	},
}

func init() {
	validateLocalCmd.Flags().StringVar(&validateAdapter, "adapter", "auto", "适配器目标: cursor, claude, opencode, auto")
	validateLocalCmd.Flags().BoolVar(&validateStrict, "strict", false, "严格模式：警告也视为错误")
}

func runValidateLocal(skillID string) error {
	fmt.Printf("验证技能 '%s' 在本地项目中的有效性...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目是否启用了该技能
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return err
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在当前项目启用", skillID)
	}

	// 获取技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 检查技能是否存在
	if !skillManager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在", skillID)
	}

	// 加载技能详情
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 获取项目技能配置
	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	skillVars, exists := skills[skillID]
	if !exists {
		return fmt.Errorf("未找到技能变量配置")
	}

	// 开始验证
	fmt.Println("🔍 开始验证...")
	validationResult := &spec.ValidationResult{
		SkillID: skillID,
		IsValid: true,
	}

	// 验证1: 技能格式
	fmt.Println("1. 验证技能格式...")
	if err := validateSkillFormat(skillID, validationResult); err != nil {
		validationResult.Errors = append(validationResult.Errors, fmt.Sprintf("技能格式验证失败: %v", err))
		validationResult.IsValid = false
	} else {
		fmt.Println("   ✓ 技能格式正确")
	}

	// 验证2: 变量配置
	fmt.Println("2. 验证变量配置...")
	if err := validateVariables(skill, skillVars.Variables, validationResult); err != nil {
		validationResult.Errors = append(validationResult.Errors, fmt.Sprintf("变量验证失败: %v", err))
		validationResult.IsValid = false
	} else {
		fmt.Println("   ✓ 变量配置正确")
	}

	// 验证3: 适配器兼容性
	fmt.Println("3. 验证适配器兼容性...")
	if err := validateAdapterCompatibility(skill, validateAdapter, validationResult); err != nil {
		validationResult.Errors = append(validationResult.Errors, fmt.Sprintf("适配器兼容性验证失败: %v", err))
		validationResult.IsValid = false
	} else {
		fmt.Println("   ✓ 适配器兼容性正确")
	}

	// 验证4: 技能文件存在性
	fmt.Println("4. 验证技能文件...")
	if err := validateSkillFiles(skillID, validationResult); err != nil {
		validationResult.Errors = append(validationResult.Errors, fmt.Sprintf("技能文件验证失败: %v", err))
		validationResult.IsValid = false
	} else {
		fmt.Println("   ✓ 技能文件完整")
	}

	// 显示验证结果
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("验证结果:")
	fmt.Println(strings.Repeat("=", 50))

	if validationResult.IsValid {
		fmt.Println("✅ 验证通过！")
		fmt.Println("技能在本地项目中有效，可以正常使用。")
	} else {
		fmt.Println("❌ 验证失败！")
		fmt.Println("发现以下问题需要修复:")

		for i, err := range validationResult.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}

		for i, warning := range validationResult.Warnings {
			fmt.Printf("  ⚠️  %d. %s\n", len(validationResult.Errors)+i+1, warning)
		}

		fmt.Println("\n建议:")
		fmt.Println("1. 检查技能格式是否正确")
		fmt.Println("2. 验证变量配置是否完整")
		fmt.Println("3. 确保适配器兼容性")
		fmt.Println("4. 重新运行 'skill-hub apply' 应用修改")
	}

	// 如果启用了严格模式且存在警告，也视为失败
	if validateStrict && len(validationResult.Warnings) > 0 {
		fmt.Println("\n⚠️  严格模式：存在警告，验证失败")
		validationResult.IsValid = false
	}

	return nil
}

// validateSkillFormat 验证技能格式
func validateSkillFormat(skillID string, result *spec.ValidationResult) error {
	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return err
	}

	// 查找技能文件
	skillDir := filepath.Join(skillsDir, skillID)
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 如果不存在，尝试在 skills/skills/ 子目录中查找
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		skillsSubDir := filepath.Join(skillsDir, "skills", skillID)
		skillMdPath = filepath.Join(skillsSubDir, "SKILL.md")

		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			return fmt.Errorf("找不到SKILL.md文件")
		}
	}

	// 使用验证器验证技能格式
	validator := validator.NewValidator()
	validationResult, err := validator.ValidateFile(skillMdPath)
	if err != nil {
		return fmt.Errorf("验证技能文件失败: %w", err)
	}

	if !validationResult.IsValid {
		// 收集错误信息
		for _, err := range validationResult.Errors {
			result.Errors = append(result.Errors, fmt.Sprintf("格式错误: %s", err.Message))
		}
		for _, warning := range validationResult.Warnings {
			result.Warnings = append(result.Warnings, fmt.Sprintf("格式警告: %s", warning.Message))
		}
		return fmt.Errorf("技能格式验证失败")
	}

	return nil
}

// validateVariables 验证变量配置
func validateVariables(skill *spec.Skill, variables map[string]string, result *spec.ValidationResult) error {
	// 检查必需变量
	for _, variable := range skill.Variables {
		value, exists := variables[variable.Name]

		if !exists && variable.Default == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("缺少必需变量: %s", variable.Name))
			return fmt.Errorf("缺少必需变量")
		}

		if exists && value == "" && variable.Default == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("变量值为空: %s", variable.Name))
		}
	}

	// 检查未定义的变量
	for varName := range variables {
		found := false
		for _, variable := range skill.Variables {
			if variable.Name == varName {
				found = true
				break
			}
		}

		if !found {
			result.Warnings = append(result.Warnings, fmt.Sprintf("未定义的变量: %s", varName))
		}
	}

	return nil
}

// validateAdapterCompatibility 验证适配器兼容性
func validateAdapterCompatibility(skill *spec.Skill, adapterTarget string, result *spec.ValidationResult) error {
	// 获取技能兼容性描述
	compatLower := strings.ToLower(skill.Compatibility)

	// 确定要检查的适配器
	adaptersToCheck := []string{}

	switch adapterTarget {
	case "auto":
		// 自动检测：根据技能兼容性检查所有支持的适配器
		if strings.Contains(compatLower, "cursor") {
			adaptersToCheck = append(adaptersToCheck, "cursor")
		}
		if strings.Contains(compatLower, "claude") {
			adaptersToCheck = append(adaptersToCheck, "claude")
		}
		if strings.Contains(compatLower, "opencode") {
			adaptersToCheck = append(adaptersToCheck, "opencode")
		}

		// 如果没有明确指定，检查所有
		if len(adaptersToCheck) == 0 {
			adaptersToCheck = []string{"cursor", "claude", "opencode"}
			result.Warnings = append(result.Warnings, "技能未指定兼容性，将检查所有适配器")
		}

	case "cursor", "claude", "opencode":
		adaptersToCheck = append(adaptersToCheck, adapterTarget)

		// 检查技能是否支持该适配器
		supported := false
		for _, adapter := range adaptersToCheck {
			if strings.Contains(compatLower, adapter) {
				supported = true
				break
			}
		}

		if !supported {
			result.Errors = append(result.Errors,
				fmt.Sprintf("技能不支持 %s 适配器", adapterTarget))
			return fmt.Errorf("适配器不兼容")
		}
	}

	// 验证每个适配器
	for _, adapter := range adaptersToCheck {
		switch adapter {
		case "cursor":
			if !strings.Contains(compatLower, "cursor") {
				result.Warnings = append(result.Warnings, "技能可能不完全兼容 Cursor")
			}
		case "claude":
			if !strings.Contains(compatLower, "claude") {
				result.Warnings = append(result.Warnings, "技能可能不完全兼容 Claude Code")
			}
		case "opencode":
			if !strings.Contains(compatLower, "opencode") {
				result.Warnings = append(result.Warnings, "技能可能不完全兼容 OpenCode")
			}
		}
	}

	return nil
}

// validateSkillFiles 验证技能文件
func validateSkillFiles(skillID string, result *spec.ValidationResult) error {
	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return err
	}

	// 查找技能目录
	skillDir := filepath.Join(skillsDir, skillID)

	// 如果不存在，尝试在 skills/skills/ 子目录中查找
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		skillsSubDir := filepath.Join(skillsDir, "skills", skillID)
		skillDir = skillsSubDir

		if _, err := os.Stat(skillDir); os.IsNotExist(err) {
			return fmt.Errorf("找不到技能目录")
		}
	}

	// 检查必需文件
	requiredFiles := []string{"SKILL.md"}
	for _, filename := range requiredFiles {
		filePath := filepath.Join(skillDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("缺少必需文件: %s", filename))
			return fmt.Errorf("缺少必需文件")
		}
	}

	// 检查可选文件
	optionalFiles := []string{"prompt.md", "README.md", "examples/"}
	for _, filename := range optionalFiles {
		filePath := filepath.Join(skillDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("缺少可选文件: %s", filename))
		}
	}

	return nil
}
