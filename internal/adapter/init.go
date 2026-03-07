package adapter

import (
	"github.com/muidea/skill-hub/internal/adapter/claude"
	"github.com/muidea/skill-hub/internal/adapter/cursor"
	"github.com/muidea/skill-hub/internal/adapter/opencode"
	"github.com/muidea/skill-hub/pkg/spec"
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
}
