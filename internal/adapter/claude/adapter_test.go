package claude

import (
	"testing"

	"skill-hub/internal/adapter/common"
)

func TestClaudeAdapterBasic(t *testing.T) {
	t.Run("Create adapter", func(t *testing.T) {
		adapter := NewClaudeAdapter()
		if adapter == nil {
			t.Error("NewClaudeAdapter() returned nil")
		}

		// 测试目标类型
		if adapter.GetTarget() != "claude_code" {
			t.Errorf("Expected target 'claude_code', got %s", adapter.GetTarget())
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

		// 测试链式调用
		adapter.WithGlobalMode()
		if adapter.GetMode() != "global" {
			t.Errorf("Expected mode 'global' after WithGlobalMode, got %s", adapter.GetMode())
		}

		adapter.WithProjectMode()
		if adapter.GetMode() != "project" {
			t.Errorf("Expected mode 'project' after WithProjectMode, got %s", adapter.GetMode())
		}
	})

	t.Run("Functional Options pattern", func(t *testing.T) {
		// 测试使用选项创建适配器
		adapter := NewClaudeAdapterWithOptions(common.WithMode("global"))
		if adapter.GetMode() != "global" {
			t.Errorf("Expected mode 'global' with WithMode option, got %s", adapter.GetMode())
		}

		// 测试默认选项
		adapter2 := NewClaudeAdapter()
		if adapter2.GetMode() != "project" {
			t.Errorf("Expected default mode 'project', got %s", adapter2.GetMode())
		}
	})
}
