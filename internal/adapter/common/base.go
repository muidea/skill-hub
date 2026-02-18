package common

import (
	"skill-hub/pkg/errors"
)

// BaseAdapter 提供适配器的公共基础功能
type BaseAdapter struct {
	mode string // "project" 或 "global"
}

// NewBaseAdapter 创建基础适配器
func NewBaseAdapter() *BaseAdapter {
	return &BaseAdapter{
		mode: "project", // 默认项目模式
	}
}

// SetProjectMode 设置为项目模式
func (b *BaseAdapter) SetProjectMode() {
	b.mode = "project"
}

// SetGlobalMode 设置为全局模式
func (b *BaseAdapter) SetGlobalMode() {
	b.mode = "global"
}

// GetMode 获取当前模式
func (b *BaseAdapter) GetMode() string {
	return b.mode
}

// WithProjectMode 设置为项目模式（链式调用）
func (b *BaseAdapter) WithProjectMode() *BaseAdapter {
	b.mode = "project"
	return b
}

// WithGlobalMode 设置为全局模式（链式调用）
func (b *BaseAdapter) WithGlobalMode() *BaseAdapter {
	b.mode = "global"
	return b
}

// ModeOption 模式配置选项
type ModeOption func(*BaseAdapter)

// WithMode 设置模式选项
func WithMode(mode string) ModeOption {
	return func(b *BaseAdapter) {
		b.mode = mode
	}
}

// NewBaseAdapterWithOptions 使用Functional Options模式创建基础适配器
func NewBaseAdapterWithOptions(opts ...ModeOption) *BaseAdapter {
	b := NewBaseAdapter()
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// ValidateMode 验证模式是否有效
func (b *BaseAdapter) ValidateMode() error {
	if b.mode != "project" && b.mode != "global" {
		return errors.NewWithCodef("ValidateMode", errors.ErrInvalidInput, "无效的模式: %s", b.mode)
	}
	return nil
}
