package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("原始错误")
	context := "操作失败"

	wrappedErr := Wrap(originalErr, context)
	if wrappedErr == nil {
		t.Fatal("Wrap 应该返回错误")
	}

	// 新的错误消息格式包含错误代码
	expectedMsg := "操作失败: SYSTEM_ERROR - 原始错误 (原始错误)"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("错误消息不匹配: 期望 %q, 得到 %q", expectedMsg, wrappedErr.Error())
	}

	// 测试包装 nil 错误
	if Wrap(nil, context) != nil {
		t.Error("Wrap(nil) 应该返回 nil")
	}
}

func TestWrapf(t *testing.T) {
	originalErr := fmt.Errorf("原始错误")
	format := "操作 %s 失败"
	arg := "测试"

	wrappedErr := Wrapf(originalErr, format, arg)
	if wrappedErr == nil {
		t.Fatal("Wrapf 应该返回错误")
	}

	// 新的错误消息格式包含错误代码
	expectedMsg := "操作 测试 失败: SYSTEM_ERROR - 原始错误 (原始错误)"
	if wrappedErr.Error() != expectedMsg {
		t.Errorf("错误消息不匹配: 期望 %q, 得到 %q", expectedMsg, wrappedErr.Error())
	}

	// 测试包装 nil 错误
	if Wrapf(nil, format, arg) != nil {
		t.Error("Wrapf(nil) 应该返回 nil")
	}
}

func TestMultiError(t *testing.T) {
	multiErr := NewMultiError()
	if multiErr.HasErrors() {
		t.Error("新的 MultiError 不应该有错误")
	}

	// 添加错误
	err1 := fmt.Errorf("错误1")
	err2 := fmt.Errorf("错误2")
	multiErr.Add(err1)
	multiErr.Add(err2)
	multiErr.Add(nil) // 应该忽略 nil

	if !multiErr.HasErrors() {
		t.Error("MultiError 应该有错误")
	}

	if len(multiErr.Errors) != 2 {
		t.Errorf("错误数量不匹配: 期望 2, 得到 %d", len(multiErr.Errors))
	}

	// 测试错误消息
	expectedMsg := `多个错误发生:
  1. 错误1
  2. 错误2
`
	if multiErr.Error() != expectedMsg {
		t.Errorf("错误消息不匹配:\n期望:\n%q\n得到:\n%q", expectedMsg, multiErr.Error())
	}

	// 测试单个错误的情况
	singleErr := &MultiError{Errors: []error{err1}}
	if singleErr.Error() != err1.Error() {
		t.Errorf("单个错误的错误消息不匹配: 期望 %q, 得到 %q", err1.Error(), singleErr.Error())
	}

	// 测试空错误
	emptyErr := &MultiError{Errors: []error{}}
	if emptyErr.Error() != "" {
		t.Errorf("空错误的错误消息应该为空, 得到: %q", emptyErr.Error())
	}
}

func TestCombine(t *testing.T) {
	err1 := fmt.Errorf("错误1")
	err2 := fmt.Errorf("错误2")

	// 测试合并多个错误
	combined := Combine(err1, err2, nil)
	if combined == nil {
		t.Fatal("Combine 应该返回错误")
	}

	multiErr, ok := combined.(*MultiError)
	if !ok {
		t.Fatal("Combine 应该返回 MultiError")
	}

	if len(multiErr.Errors) != 2 {
		t.Errorf("错误数量不匹配: 期望 2, 得到 %d", len(multiErr.Errors))
	}

	// 测试合并单个错误
	singleCombined := Combine(err1, nil)
	if singleCombined != err1 {
		t.Errorf("单个错误合并应该返回原错误: 期望 %v, 得到 %v", err1, singleCombined)
	}

	// 测试合并 nil 错误
	if Combine(nil, nil) != nil {
		t.Error("Combine(nil, nil) 应该返回 nil")
	}

	// 测试合并空参数
	if Combine() != nil {
		t.Error("Combine() 应该返回 nil")
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "文件未找到错误",
			err:      fmt.Errorf("open /path/to/file: no such file or directory"),
			expected: true,
		},
		{
			name:     "文件不存在错误",
			err:      fmt.Errorf("file does not exist"),
			expected: true,
		},
		{
			name:     "中文文件未找到错误",
			err:      fmt.Errorf("找不到文件"),
			expected: true,
		},
		{
			name:     "其他错误",
			err:      fmt.Errorf("权限被拒绝"),
			expected: false,
		},
		{
			name:     "nil 错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, 期望 %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	originalErr := errors.New("原始错误")
	wrappedErr := Wrap(originalErr, "上下文")

	// 测试错误解包
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Wrap 后的错误应该可以解包到原始错误")
	}

	// 测试错误链
	unwrapped := errors.Unwrap(wrappedErr)
	if unwrapped != originalErr {
		t.Errorf("解包后的错误不匹配: 期望 %v, 得到 %v", originalErr, unwrapped)
	}
}
