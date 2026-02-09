package adapter

import (
	"fmt"

	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
	"skill-hub/pkg/spec"
)

// init 初始化默认的Adapter注册
func init() {
	// 注册OpenCode Adapter
	openCodeAdapter := opencode.NewOpenCodeAdapter()
	RegisterAdapter(spec.TargetOpenCode, openCodeAdapter)

	// 注册Claude Adapter
	claudeAdapter := claude.NewClaudeAdapter()
	RegisterAdapter(spec.TargetClaudeCode, claudeAdapter)

	// 注册Cursor Adapter
	cursorAdapter := cursor.NewCursorAdapter()
	RegisterAdapter(spec.TargetCursor, cursorAdapter)

	fmt.Printf("已注册适配器: %v\n", GetSupportedTargets())
}
