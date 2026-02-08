package validator

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Validator 技能校验器
type Validator struct {
	rules []Rule
}

// NewValidator 创建新的校验器
func NewValidator() *Validator {
	return &Validator{
		rules: []Rule{
			NewFrontmatterRule(),
			NewNameRule(),
			NewDescriptionRule(),
			NewCompatibilityRule(),
			NewMetadataRule(),
			NewLicenseRule(),
			NewAllowedToolsRule(),
		},
	}
}

// ValidateFile 校验技能文件
func (v *Validator) ValidateFile(skillPath string) (*ValidationResult, error) {
	result := NewValidationResult(skillPath)

	// 读取文件内容
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析文件
	if err := v.parseFile(content, result); err != nil {
		return nil, err
	}

	// 运行所有校验规则
	for _, rule := range v.rules {
		rule.Validate(result)
	}

	return result, nil
}

// parseFile 解析技能文件
func (v *Validator) parseFile(content []byte, result *ValidationResult) error {
	lines := strings.Split(string(content), "\n")

	// 检查是否有frontmatter
	if len(lines) < 2 || lines[0] != "---" {
		// 没有frontmatter，直接返回
		return nil
	}

	result.HasFrontmatter = true

	// 提取frontmatter
	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	if len(frontmatterLines) == 0 {
		return nil
	}

	// 解析YAML
	frontmatterContent := strings.Join(frontmatterLines, "\n")
	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterContent), &frontmatter); err != nil {
		result.AddError(NewError(ErrYamlParseFailed, "", false))
		return nil
	}

	result.Frontmatter = frontmatter
	return nil
}

// ValidateSkill 校验技能对象（用于已加载的技能）
func (v *Validator) ValidateSkill(skillName string, frontmatter map[string]interface{}) *ValidationResult {
	result := NewValidationResult("")
	result.SkillName = skillName
	result.HasFrontmatter = true
	result.Frontmatter = frontmatter

	// 运行所有校验规则
	for _, rule := range v.rules {
		rule.Validate(result)
	}

	return result
}

// AddRule 添加自定义规则
func (v *Validator) AddRule(rule Rule) {
	v.rules = append(v.rules, rule)
}

// GetRules 获取所有规则
func (v *Validator) GetRules() []Rule {
	return v.rules
}

// ValidateWithOptions 使用选项校验技能文件
func (v *Validator) ValidateWithOptions(skillPath string, options ValidationOptions) (*ValidationResult, error) {
	result, err := v.ValidateFile(skillPath)
	if err != nil {
		return nil, err
	}

	// 根据选项过滤结果
	if options.IgnoreWarnings {
		result.Warnings = []ValidationWarning{}
	}

	if options.StrictMode && result.HasWarnings() {
		result.IsValid = false
	}

	return result, nil
}

// ValidationOptions 校验选项
type ValidationOptions struct {
	IgnoreWarnings bool // 忽略警告
	StrictMode     bool // 严格模式：警告也视为错误
}
