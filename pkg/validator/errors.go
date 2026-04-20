package validator

// ValidationError 表示校验错误
type ValidationError struct {
	Code    string // 错误代码
	Message string // 用户友好的错误信息
	Field   string // 相关字段
	Fixable bool   // 是否可自动修复
}

// ValidationWarning 表示校验警告
type ValidationWarning struct {
	Code    string // 警告代码
	Message string // 用户友好的警告信息
	Field   string // 相关字段
	Fixable bool   // 是否可自动修复
}

// 错误代码常量
const (
	// 文件格式错误
	ErrMissingFrontmatter = "MISSING_FRONTMATTER"
	ErrEmptyFrontmatter   = "EMPTY_FRONTMATTER"
	ErrYamlParseFailed    = "YAML_PARSE_FAILED"

	// 必需字段错误
	ErrMissingName        = "MISSING_NAME"
	ErrMissingDescription = "MISSING_DESCRIPTION"

	// name字段错误
	ErrNameTooShort       = "NAME_TOO_SHORT"
	ErrNameTooLong        = "NAME_TOO_LONG"
	ErrNameInvalidFormat  = "NAME_INVALID_FORMAT"
	ErrNameStartsWithDash = "NAME_STARTS_WITH_DASH"
	ErrNameEndsWithDash   = "NAME_ENDS_WITH_DASH"
	ErrNameDoubleDash     = "NAME_DOUBLE_DASH"

	// description字段错误
	ErrDescTooShort = "DESC_TOO_SHORT"
	ErrDescTooLong  = "DESC_TOO_LONG"

	// compatibility字段错误
	ErrCompatTooLong   = "COMPAT_TOO_LONG"
	ErrCompatWrongType = "COMPAT_WRONG_TYPE"

	// metadata字段错误
	ErrMetadataWrongType = "METADATA_WRONG_TYPE"
	ErrMetadataValueType = "METADATA_VALUE_TYPE"

	// license字段错误
	ErrLicenseWrongType = "LICENSE_WRONG_TYPE"
	ErrLicenseTooLong   = "LICENSE_TOO_LONG"

	// allowed-tools字段错误
	ErrAllowedToolsWrongType = "ALLOWED_TOOLS_WRONG_TYPE"

	// 目录结构错误
	ErrDirectoryMismatch = "DIRECTORY_MISMATCH"
)

// 警告代码常量
const (
	// description质量警告
	WarnDescTooShort   = "DESC_TOO_SHORT_WARNING"
	WarnDescNoSentence = "DESC_NO_SENTENCE"

	// legacy compatibility格式警告
	WarnCompatObjectFormat = "COMPAT_OBJECT_FORMAT"
	WarnCompatUnknownType  = "COMPAT_UNKNOWN_TYPE"

	// metadata警告
	WarnMetadataWrongType = "METADATA_WRONG_TYPE_WARNING"
	WarnMetadataValueType = "METADATA_VALUE_TYPE_WARNING"

	// license警告
	WarnLicenseWrongType = "LICENSE_WRONG_TYPE_WARNING"
	WarnLicenseTooLong   = "LICENSE_TOO_LONG_WARNING"

	// allowed-tools警告
	WarnAllowedToolsWrongType = "ALLOWED_TOOLS_WRONG_TYPE_WARNING"

	// 目录结构警告
	WarnDirectoryMismatch = "DIRECTORY_MISMATCH_WARNING"
)

// 错误消息映射
var errorMessages = map[string]string{
	ErrMissingFrontmatter:    "缺少YAML frontmatter（必须以---开头）",
	ErrEmptyFrontmatter:      "frontmatter为空",
	ErrYamlParseFailed:       "解析YAML失败",
	ErrMissingName:           "缺少必需字段: name",
	ErrMissingDescription:    "缺少必需字段: description",
	ErrNameTooShort:          "name长度无效: 必须至少1个字符",
	ErrNameTooLong:           "name长度无效: 不能超过64个字符",
	ErrNameInvalidFormat:     "name不符合规范: 必须小写字母数字，用连字符分隔",
	ErrNameStartsWithDash:    "name不能以连字符开头",
	ErrNameEndsWithDash:      "name不能以连字符结尾",
	ErrNameDoubleDash:        "name不能有连续连字符",
	ErrDescTooShort:          "description长度无效: 必须至少1个字符",
	ErrDescTooLong:           "description长度无效: 不能超过1024个字符",
	ErrCompatTooLong:         "compatibility太长: 不能超过500个字符",
	ErrCompatWrongType:       "compatibility字段类型不符合规范",
	ErrMetadataWrongType:     "metadata字段类型不符合规范",
	ErrMetadataValueType:     "metadata值类型不符合规范",
	ErrLicenseWrongType:      "license字段类型不符合规范",
	ErrLicenseTooLong:        "license字段建议保持简短",
	ErrAllowedToolsWrongType: "allowed-tools字段类型不符合规范",
	ErrDirectoryMismatch:     "name字段与目录名不匹配",
}

// 警告消息映射
var warningMessages = map[string]string{
	WarnDescTooShort:          "description可能太短，建议提供更详细的描述",
	WarnDescNoSentence:        "description应该包含完整的句子",
	WarnCompatObjectFormat:    "compatibility对象格式保留为历史兼容",
	WarnCompatUnknownType:     "compatibility字段类型为非标准说明",
	WarnMetadataWrongType:     "metadata字段类型可能不符合规范",
	WarnMetadataValueType:     "metadata值类型可能不符合规范",
	WarnLicenseWrongType:      "license字段类型可能不符合规范",
	WarnLicenseTooLong:        "license字段建议保持简短",
	WarnAllowedToolsWrongType: "allowed-tools字段类型可能不符合规范",
	WarnDirectoryMismatch:     "name字段与目录名不匹配",
}

// NewError 创建新的校验错误
func NewError(code, field string, fixable bool) ValidationError {
	message, ok := errorMessages[code]
	if !ok {
		message = "未知错误"
	}
	return ValidationError{
		Code:    code,
		Message: message,
		Field:   field,
		Fixable: fixable,
	}
}

// NewWarning 创建新的校验警告
func NewWarning(code, field string, fixable bool) ValidationWarning {
	message, ok := warningMessages[code]
	if !ok {
		message = "未知警告"
	}
	return ValidationWarning{
		Code:    code,
		Message: message,
		Field:   field,
		Fixable: fixable,
	}
}
