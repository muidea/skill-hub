package errors

import (
	"fmt"
	"strings"
)

// Wrap 包装错误并添加上下文信息
func Wrap(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// Wrapf 包装错误并添加格式化的上下文信息
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
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
