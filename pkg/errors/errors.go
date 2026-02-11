package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode 定义错误代码
type ErrorCode string

const (
	// 配置相关错误
	ErrConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"
	ErrConfigInvalid  ErrorCode = "CONFIG_INVALID"

	// 技能相关错误
	ErrSkillNotFound ErrorCode = "SKILL_NOT_FOUND"
	ErrSkillInvalid  ErrorCode = "SKILL_INVALID"
	ErrSkillExists   ErrorCode = "SKILL_EXISTS"

	// 项目相关错误
	ErrProjectNotFound ErrorCode = "PROJECT_NOT_FOUND"
	ErrProjectInvalid  ErrorCode = "PROJECT_INVALID"

	// 文件操作错误
	ErrFileOperation  ErrorCode = "FILE_OPERATION_FAILED"
	ErrFileNotFound   ErrorCode = "FILE_NOT_FOUND"
	ErrFilePermission ErrorCode = "FILE_PERMISSION_DENIED"

	// Git操作错误
	ErrGitOperation ErrorCode = "GIT_OPERATION_FAILED"
	ErrGitRemote    ErrorCode = "GIT_REMOTE_ERROR"

	// 网络错误
	ErrNetwork    ErrorCode = "NETWORK_ERROR"
	ErrAPIRequest ErrorCode = "API_REQUEST_FAILED"

	// 验证错误
	ErrValidation   ErrorCode = "VALIDATION_FAILED"
	ErrInvalidInput ErrorCode = "INVALID_INPUT"

	// 系统错误
	ErrSystem         ErrorCode = "SYSTEM_ERROR"
	ErrNotImplemented ErrorCode = "NOT_IMPLEMENTED"

	// 用户错误
	ErrUserCancel ErrorCode = "USER_CANCELLED"
	ErrUserInput  ErrorCode = "USER_INPUT_ERROR"
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode
	Message string
	Op      string
	Err     error
	Details map[string]interface{}
}

// Error 实现error接口
func (e *AppError) Error() string {
	var sb strings.Builder

	sb.WriteString(e.Op)
	sb.WriteString(": ")
	sb.WriteString(string(e.Code))

	if e.Message != "" {
		sb.WriteString(" - ")
		sb.WriteString(e.Message)
	}

	if e.Err != nil {
		sb.WriteString(" (")
		sb.WriteString(e.Err.Error())
		sb.WriteString(")")
	}

	return sb.String()
}

// Unwrap 实现错误链
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// Is 检查错误是否匹配（实现errors.Is接口）
func (e *AppError) Is(target error) bool {
	if target == nil {
		return false
	}

	// 检查是否是相同类型的错误
	if other, ok := target.(*AppError); ok {
		return e.Code == other.Code
	}

	return false
}

// IsCode 检查错误代码是否匹配（向后兼容）
func (e *AppError) IsCode(code ErrorCode) bool {
	return e.Code == code
}

// Wrap 包装错误
func Wrap(err error, context string) error {
	if err == nil {
		return nil
	}

	// 如果是AppError，保持原有Code
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:    appErr.Code,
			Message: appErr.Message,
			Op:      context,
			Err:     appErr.Err,
			Details: appErr.Details,
		}
	}

	// 否则创建新的AppError
	return &AppError{
		Code:    ErrSystem,
		Message: err.Error(),
		Op:      context,
		Err:     err,
	}
}

// Wrapf 包装错误并添加格式化的上下文信息
func Wrapf(err error, format string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// WrapWithCode 使用特定错误代码包装错误
func WrapWithCode(err error, op string, code ErrorCode, msg string) error {
	if err == nil {
		return nil
	}

	return &AppError{
		Code:    code,
		Message: msg,
		Op:      op,
		Err:     err,
	}
}

// NewWithCode 创建新错误
func NewWithCode(op string, code ErrorCode, msg string) error {
	return &AppError{
		Code:    code,
		Message: msg,
		Op:      op,
	}
}

// NewWithCodef 使用格式创建新错误
func NewWithCodef(op string, code ErrorCode, format string, args ...interface{}) error {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Op:      op,
	}
}

// IsCode 检查错误是否是指定类型
func IsCode(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}

	// 检查错误链
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return IsCode(unwrapped, code)
	}

	return false
}

// Code 获取错误代码
func Code(err error) ErrorCode {
	if err == nil {
		return ""
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}

	// 检查错误链
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return Code(unwrapped)
	}

	return ErrSystem
}

// Message 获取错误消息
func Message(err error) string {
	if err == nil {
		return ""
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}

	return err.Error()
}

// Operation 获取操作名称
func Operation(err error) string {
	if err == nil {
		return ""
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Op
	}

	return ""
}

// Details 获取错误详情
func Details(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.Details
	}

	return nil
}

// MultiError 表示多个错误
type MultiError struct {
	Errors []error
}

// Error 返回错误字符串
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return ""
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString("多个错误发生:\n")
	for i, err := range m.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Add 添加错误到多错误集合
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// HasErrors 检查是否有错误
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// NewMultiError 创建新的多错误
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]error, 0),
	}
}

// Combine 合并多个错误（兼容Go 1.20+的errors.Join）
func Combine(errs ...error) error {
	// 使用标准库的errors.Join如果可用
	// 否则使用自定义实现
	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	if len(nonNilErrs) == 0 {
		return nil
	}
	if len(nonNilErrs) == 1 {
		return nonNilErrs[0]
	}

	// 如果Go版本>=1.20，使用errors.Join
	// 这里使用自定义实现保持兼容性
	return &MultiError{Errors: nonNilErrs}
}

// JoinErrors 使用errors.Join合并错误（Go 1.20+）
func JoinErrors(errs ...error) error {
	// 在实际使用中，可以替换为标准的errors.Join
	// 这里提供兼容性包装
	return Combine(errs...)
}

// New 创建新错误（兼容标准库）
func New(text string) error {
	return errors.New(text)
}

// Is 检查错误是否匹配（兼容标准库）
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 检查错误类型（兼容标准库）
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// 预定义错误构造函数
func ConfigNotFound(op, path string) error {
	return NewWithCode(op, ErrConfigNotFound, fmt.Sprintf("配置文件未找到: %s", path))
}

func SkillNotFound(op, skillName string) error {
	return NewWithCode(op, ErrSkillNotFound, fmt.Sprintf("技能未找到: %s", skillName))
}

func ProjectNotFound(op, projectPath string) error {
	return NewWithCode(op, ErrProjectNotFound, fmt.Sprintf("项目未找到: %s", projectPath))
}

func FileNotFound(op, filePath string) error {
	return NewWithCode(op, ErrFileNotFound, fmt.Sprintf("文件未找到: %s", filePath))
}

func ValidationFailed(op, reason string) error {
	return NewWithCode(op, ErrValidation, fmt.Sprintf("验证失败: %s", reason))
}

func InvalidInput(op, field string) error {
	return NewWithCode(op, ErrInvalidInput, fmt.Sprintf("无效输入: %s", field))
}

// IsNotFound 检查是否是文件未找到错误
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such file or directory") ||
		strings.Contains(err.Error(), "file does not exist") ||
		strings.Contains(err.Error(), "找不到文件") ||
		strings.Contains(err.Error(), "目录不存在") ||
		strings.Contains(err.Error(), "不存在")
}

// SkillError 自定义技能错误类型
type SkillError struct {
	Type    string
	Message string
	Err     error
}

// Error 实现error接口
func (e *SkillError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap 支持错误链
func (e *SkillError) Unwrap() error {
	return e.Err
}

// NewSkillError 创建新的技能错误
func NewSkillError(errType, message string) *SkillError {
	return &SkillError{
		Type:    errType,
		Message: message,
	}
}

// WrapSkillError 包装现有错误为技能错误
func WrapSkillError(errType, message string, err error) *SkillError {
	return &SkillError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// IsSkillError 检查错误是否为技能错误
func IsSkillError(err error) bool {
	_, ok := err.(*SkillError)
	return ok
}
