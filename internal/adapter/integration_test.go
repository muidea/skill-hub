package adapter

import (
	"testing"

	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/common"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
)

func TestAdapterIntegration(t *testing.T) {
	t.Run("Adapter creation with Functional Options", func(t *testing.T) {
		// Test ClaudeAdapter with options
		claudeAdapter := claude.NewClaudeAdapterWithOptions(common.WithMode("global"))
		if claudeAdapter.GetMode() != "global" {
			t.Errorf("ClaudeAdapter: expected mode 'global', got %s", claudeAdapter.GetMode())
		}
		if claudeAdapter.GetTarget() != "claude_code" {
			t.Errorf("ClaudeAdapter: expected target 'claude_code', got %s", claudeAdapter.GetTarget())
		}

		// Test CursorAdapter with options
		cursorAdapter := cursor.NewCursorAdapterWithOptions(common.WithMode("project"))
		if cursorAdapter.GetMode() != "project" {
			t.Errorf("CursorAdapter: expected mode 'project', got %s", cursorAdapter.GetMode())
		}
		if cursorAdapter.GetTarget() != "cursor" {
			t.Errorf("CursorAdapter: expected target 'cursor', got %s", cursorAdapter.GetTarget())
		}

		// Test OpenCodeAdapter with options
		openCodeAdapter := opencode.NewOpenCodeAdapterWithOptions(common.WithMode("global"))
		if openCodeAdapter.GetMode() != "global" {
			t.Errorf("OpenCodeAdapter: expected mode 'global', got %s", openCodeAdapter.GetMode())
		}
		if openCodeAdapter.GetTarget() != "open_code" {
			t.Errorf("OpenCodeAdapter: expected target 'open_code', got %s", openCodeAdapter.GetTarget())
		}
	})

	t.Run("Adapter mode switching", func(t *testing.T) {
		adapter := claude.NewClaudeAdapter()

		// Test initial mode
		if adapter.GetMode() != "project" {
			t.Errorf("Initial mode should be 'project', got %s", adapter.GetMode())
		}

		// Switch to global
		adapter.SetGlobalMode()
		if adapter.GetMode() != "global" {
			t.Errorf("After SetGlobalMode should be 'global', got %s", adapter.GetMode())
		}

		// Switch back to project
		adapter.SetProjectMode()
		if adapter.GetMode() != "project" {
			t.Errorf("After SetProjectMode should be 'project', got %s", adapter.GetMode())
		}

		// Test chainable methods
		adapter.WithGlobalMode()
		if adapter.GetMode() != "global" {
			t.Errorf("After WithGlobalMode should be 'global', got %s", adapter.GetMode())
		}

		adapter.WithProjectMode()
		if adapter.GetMode() != "project" {
			t.Errorf("After WithProjectMode should be 'project', got %s", adapter.GetMode())
		}
	})

	t.Run("Adapter manager integration", func(t *testing.T) {
		// Use the global manager (already initialized by init())
		// Test getting adapters
		claudeAdapter, err := GetAdapterForTarget("claude_code")
		if err != nil {
			t.Errorf("Failed to get claude adapter: %v", err)
		}
		if claudeAdapter.GetTarget() != "claude_code" {
			t.Errorf("Expected target 'claude_code', got %s", claudeAdapter.GetTarget())
		}

		cursorAdapter, err := GetAdapterForTarget("cursor")
		if err != nil {
			t.Errorf("Failed to get cursor adapter: %v", err)
		}
		if cursorAdapter.GetTarget() != "cursor" {
			t.Errorf("Expected target 'cursor', got %s", cursorAdapter.GetTarget())
		}

		openCodeAdapter, err := GetAdapterForTarget("open_code")
		if err != nil {
			t.Errorf("Failed to get opencode adapter: %v", err)
		}
		if openCodeAdapter.GetTarget() != "open_code" {
			t.Errorf("Expected target 'open_code', got %s", openCodeAdapter.GetTarget())
		}

		// Test supported targets - check that at least the expected ones are present
		targets := GetSupportedTargets()
		t.Logf("Actual supported targets: %v (count: %d)", targets, len(targets))

		// We expect at least these three targets
		expectedTargets := []string{"claude_code", "cursor", "open_code"}
		for _, expected := range expectedTargets {
			found := false
			for _, actual := range targets {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected target %s not found in supported targets: %v", expected, targets)
			}
		}

		// Also verify we can get each adapter
		for _, target := range expectedTargets {
			adapter, err := GetAdapterForTarget(target)
			if err != nil {
				t.Errorf("Failed to get adapter for target %s: %v", target, err)
			}
			if adapter.GetTarget() != target {
				t.Errorf("Adapter target mismatch: expected %s, got %s", target, adapter.GetTarget())
			}
		}
	})
}
