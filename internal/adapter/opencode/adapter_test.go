package opencode

import (
	"testing"

	"skill-hub/internal/adapter/common"
)

func TestOpenCodeAdapterBasic(t *testing.T) {
	t.Run("Create adapter", func(t *testing.T) {
		adapter := NewOpenCodeAdapter()
		if adapter == nil {
			t.Error("NewOpenCodeAdapter() returned nil")
		}

		// 测试目标类型
		if adapter.GetTarget() != "open_code" {
			t.Errorf("Expected target 'open_code', got %s", adapter.GetTarget())
		}

		// 测试默认模式
		if adapter.GetMode() != "project" {
			t.Errorf("Expected default mode 'project', got %s", adapter.GetMode())
		}

		// 测试模式切换
		adapter.SetGlobalMode()
		if adapter.GetMode() != "global" {
			t.Errorf("Expected mode 'global' after SetGlobalMode, got %s", adapter.GetMode())
		}

		adapter.SetProjectMode()
		if adapter.GetMode() != "project" {
			t.Errorf("Expected mode 'project' after SetProjectMode, got %s", adapter.GetMode())
		}
	})

	t.Run("Functional Options pattern", func(t *testing.T) {
		// 测试使用选项创建适配器
		adapter := NewOpenCodeAdapterWithOptions(common.WithMode("global"))
		if adapter.GetMode() != "global" {
			t.Errorf("Expected mode 'global' with WithMode option, got %s", adapter.GetMode())
		}
	})
}
