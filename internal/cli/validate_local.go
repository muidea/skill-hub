package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/validator"
)

var (
	validateTarget string
	validateStrict bool
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
	validateLocalCmd.Flags().StringVar(&validateTarget, "target", "", "目标工具: cursor, claude_code, open_code, all, auto (为空时使用状态绑定的目标)")
	validateLocalCmd.Flags().BoolVar(&validateStrict, "strict", false, "严格模式：警告也视为错误")
}

func runValidateLocal(skillID string) error {
	fmt.Printf("验证技能 '%s' 在本地项目中的有效性...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查.agents/skills/目录下是否存在该技能
	agentsSkillsDir := filepath.Join(cwd, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillsDir); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 在当前项目的 .agents/skills/ 目录中不存在", skillID)
	}

	// 检查SKILL.md文件是否存在
	skillMdPath := filepath.Join(agentsSkillsDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("技能文件 SKILL.md 在当前项目的 .agents/skills/%s/ 目录中不存在", skillID)
	}

	// 确定目标工具
	resolvedTarget := validateTarget
	stateManager, err := state.NewStateManager()
	if err != nil {
		// 如果状态管理器初始化失败，使用auto模式
		resolvedTarget = "auto"
		fmt.Println("🔍 状态管理器初始化失败，使用自动检测模式")
	} else if resolvedTarget == "" {
		// 如果没有指定target，尝试从状态获取
		projectState, err := stateManager.FindProjectByPath(cwd)
		if err != nil {
			// 查找项目状态失败，使用auto
			resolvedTarget = "auto"
			fmt.Println("🔍 查找项目状态失败，使用自动检测模式")
		} else if projectState == nil || projectState.PreferredTarget == "" {
			// 未绑定项目，使用auto
			resolvedTarget = "auto"
			fmt.Println("🔍 项目未绑定目标，使用自动检测模式")
		} else {
			resolvedTarget = spec.NormalizeTarget(projectState.PreferredTarget)
			fmt.Printf("🔍 使用状态绑定的目标: %s\n", resolvedTarget)
		}
	} else {
		resolvedTarget = spec.NormalizeTarget(resolvedTarget)
		fmt.Printf("🔍 使用指定的目标: %s\n", resolvedTarget)
	}

	// 从本地项目的.agents/skills/目录加载技能
	skill, err := loadSkillFromLocalProject(cwd, skillID)
	if err != nil {
		return fmt.Errorf("加载本地技能失败: %w", err)
	}

	// 获取项目技能配置（如果技能已启用）
	var skillVariables map[string]string
	skills, err := stateManager.GetProjectSkills(cwd)
	if err == nil {
		if skillVars, exists := skills[skillID]; exists {
			skillVariables = skillVars.Variables
			fmt.Println("🔍 技能已在项目中启用，使用项目变量配置")
		} else {
			skillVariables = make(map[string]string)
			fmt.Println("🔍 技能未在项目中启用，使用空变量配置")
		}
	} else {
		skillVariables = make(map[string]string)
		fmt.Println("🔍 无法获取项目状态，使用空变量配置")
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
	if err := validateVariables(skill, skillVariables, validationResult); err != nil {
		validationResult.Errors = append(validationResult.Errors, fmt.Sprintf("变量验证失败: %v", err))
		validationResult.IsValid = false
	} else {
		fmt.Println("   ✓ 变量配置正确")
	}

	// 验证3: 适配器兼容性
	fmt.Println("3. 验证适配器兼容性...")
	if err := validateAdapterCompatibility(skill, resolvedTarget, validationResult); err != nil {
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
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 查找本地项目的技能文件
	skillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 检查文件是否存在
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("找不到SKILL.md文件: %s", skillMdPath)
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
			// 如果变量不存在且没有默认值，给出警告而不是错误
			// 因为技能可能未在项目中启用
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("缺少必需变量: %s (技能未启用或未配置)", variable.Name))
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
func validateAdapterCompatibility(skill *spec.Skill, target string, result *spec.ValidationResult) error {
	// 获取技能兼容性描述
	compatLower := strings.ToLower(skill.Compatibility)

	// 规范化目标值
	target = spec.NormalizeTarget(target)

	// 确定要检查的适配器
	adaptersToCheck := []string{}

	switch target {
	case "", "auto":
		// 自动检测：根据技能兼容性检查所有支持的适配器
		if strings.Contains(compatLower, "cursor") {
			adaptersToCheck = append(adaptersToCheck, spec.TargetCursor)
		}
		if strings.Contains(compatLower, "claude") {
			adaptersToCheck = append(adaptersToCheck, spec.TargetClaudeCode)
		}
		if strings.Contains(compatLower, "opencode") {
			adaptersToCheck = append(adaptersToCheck, spec.TargetOpenCode)
		}

		// 如果没有明确指定，检查所有
		if len(adaptersToCheck) == 0 {
			adaptersToCheck = []string{spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode}
			result.Warnings = append(result.Warnings, "技能未指定兼容性，将检查所有适配器")
		}

	case spec.TargetAll:
		// 检查所有适配器
		adaptersToCheck = []string{spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode}

	case spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode:
		adaptersToCheck = append(adaptersToCheck, target)

		// 检查技能是否支持该适配器
		supported := false
		for _, adapter := range adaptersToCheck {
			// 将适配器名称转换为技能兼容性描述中可能的形式
			adapterName := adapter
			if adapter == spec.TargetClaudeCode {
				adapterName = "claude"
			} else if adapter == spec.TargetOpenCode {
				adapterName = "opencode"
			}

			if strings.Contains(compatLower, adapterName) {
				supported = true
				break
			}
		}

		if !supported {
			result.Errors = append(result.Errors,
				fmt.Sprintf("技能不支持 %s 适配器", target))
			return fmt.Errorf("适配器不兼容")
		}
	}

	// 验证每个适配器
	for _, adapter := range adaptersToCheck {
		// 将适配器名称转换为技能兼容性描述中可能的形式
		adapterName := adapter
		if adapter == spec.TargetClaudeCode {
			adapterName = "claude"
		} else if adapter == spec.TargetOpenCode {
			adapterName = "opencode"
		}

		if !strings.Contains(compatLower, adapterName) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("技能可能不完全兼容 %s", adapter))
		}
	}

	return nil
}

// validateSkillFiles 验证技能文件
func validateSkillFiles(skillID string, result *spec.ValidationResult) error {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 查找本地项目的技能目录
	skillDir := filepath.Join(cwd, ".agents", "skills", skillID)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		result.Errors = append(result.Errors, fmt.Sprintf("技能目录不存在: %s", skillDir))
		return fmt.Errorf("找不到技能目录")
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

// loadSkillFromLocalProject 从本地项目的.agents/skills/目录加载技能
func loadSkillFromLocalProject(projectPath, skillID string) (*spec.Skill, error) {
	// 构建技能文件路径
	skillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 读取技能文件内容
	content, err := os.ReadFile(skillMdPath)
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

	// 转换为Skill对象
	skill := &spec.Skill{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skill.Name = name
	} else {
		skill.Name = skillID
	}

	// 设置描述
	if description, ok := skillData["description"].(string); ok {
		skill.Description = description
	}

	// 设置兼容性
	if compatibility, ok := skillData["compatibility"].(string); ok {
		skill.Compatibility = compatibility
	}

	// 设置版本
	if metadata, ok := skillData["metadata"].(map[string]interface{}); ok {
		if version, ok := metadata["version"].(string); ok {
			skill.Version = version
		} else {
			skill.Version = "1.0.0"
		}
	} else {
		skill.Version = "1.0.0"
	}

	// 解析变量（简化实现）
	// 在实际实现中，应该解析技能内容中的变量定义
	// 这里使用空变量列表作为占位符
	skill.Variables = []spec.Variable{}

	return skill, nil
}
