package validator

import (
	"regexp"
	"strings"
)

// Rule 校验规则接口
type Rule interface {
	Validate(result *ValidationResult) bool
	Name() string
}

// BaseRule 基础规则
type BaseRule struct {
	name string
}

func (r *BaseRule) Name() string {
	return r.name
}

// FrontmatterRule 检查frontmatter规则
type FrontmatterRule struct {
	BaseRule
}

func NewFrontmatterRule() *FrontmatterRule {
	return &FrontmatterRule{BaseRule{name: "frontmatter"}}
}

func (r *FrontmatterRule) Validate(result *ValidationResult) bool {
	if !result.HasFrontmatter {
		result.AddError(NewError(ErrMissingFrontmatter, "", false))
		return false
	}
	if len(result.Frontmatter) == 0 {
		result.AddError(NewError(ErrEmptyFrontmatter, "", false))
		return false
	}
	return true
}

// NameRule 检查name字段规则
type NameRule struct {
	BaseRule
}

func NewNameRule() *NameRule {
	return &NameRule{BaseRule{name: "name"}}
}

func (r *NameRule) Validate(result *ValidationResult) bool {
	nameValue, ok := result.Frontmatter["name"]
	if !ok {
		result.AddError(NewError(ErrMissingName, "name", true))
		return false
	}

	name, ok := nameValue.(string)
	if !ok {
		result.AddError(NewError(ErrMissingName, "name", true))
		return false
	}

	result.SkillName = name

	// 检查长度
	if len(name) < 1 {
		result.AddError(NewError(ErrNameTooShort, "name", true))
	} else if len(name) > 64 {
		result.AddError(NewError(ErrNameTooLong, "name", true))
	}

	// 检查命名规范: ^[a-z0-9]+(-[a-z0-9]+)*$
	namePattern := `^[a-z0-9]+(-[a-z0-9]+)*$`
	matched, _ := regexp.MatchString(namePattern, name)
	if !matched {
		result.AddError(NewError(ErrNameInvalidFormat, "name", true))
	}

	// 检查不能以连字符开头或结尾
	if strings.HasPrefix(name, "-") {
		result.AddError(NewError(ErrNameStartsWithDash, "name", true))
	}
	if strings.HasSuffix(name, "-") {
		result.AddError(NewError(ErrNameEndsWithDash, "name", true))
	}

	// 检查不能有连续连字符
	if strings.Contains(name, "--") {
		result.AddError(NewError(ErrNameDoubleDash, "name", true))
	}

	// 检查目录名是否匹配
	if name != result.DirName {
		result.AddWarning(NewWarning(WarnDirectoryMismatch, "name", true))
	}

	return true
}

// DescriptionRule 检查description字段规则
type DescriptionRule struct {
	BaseRule
}

func NewDescriptionRule() *DescriptionRule {
	return &DescriptionRule{BaseRule{name: "description"}}
}

func (r *DescriptionRule) Validate(result *ValidationResult) bool {
	descValue, ok := result.Frontmatter["description"]
	if !ok {
		result.AddError(NewError(ErrMissingDescription, "description", true))
		return false
	}

	desc, ok := descValue.(string)
	if !ok {
		result.AddError(NewError(ErrMissingDescription, "description", true))
		return false
	}

	// 检查长度
	if len(desc) < 1 {
		result.AddError(NewError(ErrDescTooShort, "description", true))
	} else if len(desc) > 1024 {
		result.AddError(NewError(ErrDescTooLong, "description", true))
	}

	// 检查内容质量（启发式检查）
	if len(desc) < 20 {
		result.AddWarning(NewWarning(WarnDescTooShort, "description", true))
	}

	if strings.Count(desc, ".") < 1 {
		result.AddWarning(NewWarning(WarnDescNoSentence, "description", true))
	}

	return true
}

// CompatibilityRule 检查compatibility字段规则
type CompatibilityRule struct {
	BaseRule
}

func NewCompatibilityRule() *CompatibilityRule {
	return &CompatibilityRule{BaseRule{name: "compatibility"}}
}

func (r *CompatibilityRule) Validate(result *ValidationResult) bool {
	compatValue, ok := result.Frontmatter["compatibility"]
	if !ok {
		return true
	}

	if v, ok := compatValue.(string); ok && len(v) > 500 {
		result.AddError(NewError(ErrCompatTooLong, "compatibility", true))
	}

	return true
}

// MetadataRule 检查metadata字段规则
type MetadataRule struct {
	BaseRule
}

func NewMetadataRule() *MetadataRule {
	return &MetadataRule{BaseRule{name: "metadata"}}
}

func (r *MetadataRule) Validate(result *ValidationResult) bool {
	metadataValue, ok := result.Frontmatter["metadata"]
	if !ok {
		// metadata是可选的
		return true
	}

	switch v := metadataValue.(type) {
	case map[string]interface{}:
		// 检查键值类型
		for key, value := range v {
			switch value.(type) {
			case string:
				// 字符串值，符合规范
			default:
				result.AddWarning(NewWarning(WarnMetadataValueType, "metadata."+key, false))
			}
		}
	default:
		result.AddWarning(NewWarning(WarnMetadataWrongType, "metadata", false))
	}

	return true
}

// LicenseRule 检查license字段规则
type LicenseRule struct {
	BaseRule
}

func NewLicenseRule() *LicenseRule {
	return &LicenseRule{BaseRule{name: "license"}}
}

func (r *LicenseRule) Validate(result *ValidationResult) bool {
	licenseValue, ok := result.Frontmatter["license"]
	if !ok {
		// license是可选的
		return true
	}

	switch v := licenseValue.(type) {
	case string:
		if len(v) > 200 {
			result.AddWarning(NewWarning(WarnLicenseTooLong, "license", true))
		}
	default:
		result.AddWarning(NewWarning(WarnLicenseWrongType, "license", false))
	}

	return true
}

// AllowedToolsRule 检查allowed-tools字段规则
type AllowedToolsRule struct {
	BaseRule
}

func NewAllowedToolsRule() *AllowedToolsRule {
	return &AllowedToolsRule{BaseRule{name: "allowed-tools"}}
}

func (r *AllowedToolsRule) Validate(result *ValidationResult) bool {
	allowedToolsValue, ok := result.Frontmatter["allowed-tools"]
	if !ok {
		// allowed-tools是可选的
		return true
	}

	switch allowedToolsValue.(type) {
	case string:
		// 符合规范
	default:
		result.AddWarning(NewWarning(WarnAllowedToolsWrongType, "allowed-tools", false))
	}

	return true
}
