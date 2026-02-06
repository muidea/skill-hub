package cursor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCursorAdapter(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Create adapter", func(t *testing.T) {
		adapter := NewCursorAdapter()
		if adapter == nil {
			t.Error("NewCursorAdapter() returned nil")
		}

		// 测试项目模式
		projectAdapter := adapter.WithProjectMode()
		if projectAdapter == nil {
			t.Error("WithProjectMode() returned nil")
		}

		// 测试全局模式
		globalAdapter := adapter.WithGlobalMode()
		if globalAdapter == nil {
			t.Error("WithGlobalMode() returned nil")
		}
	})

	t.Run("File operations", func(t *testing.T) {
		adapter := NewCursorAdapter().WithProjectMode()

		// 模拟当前目录
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// 测试文件路径获取
		filePath, err := adapter.GetFilePath()
		if err != nil {
			t.Errorf("GetFilePath() error = %v", err)
		}

		expectedPath := filepath.Join(tmpDir, ".cursorrules")
		if filePath != expectedPath {
			t.Errorf("GetFilePath() = %v, want %v", filePath, expectedPath)
		}

		// 直接测试文件读写（不通过适配器）
		testContent := "test content"
		if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// 验证文件存在
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}

		if string(data) != testContent {
			t.Errorf("File content = %v, want %v", string(data), testContent)
		}

		// 测试文件写入（直接测试writeFile方法）
		adapter.filePath = filePath
		newContent := "new content"
		if err := adapter.writeFile(newContent); err != nil {
			t.Errorf("writeFile() error = %v", err)
		}

		// 验证写入结果
		data, err = os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file after write: %v", err)
		}

		if string(data) != newContent {
			t.Errorf("File content = %v, want %v", string(data), newContent)
		}
	})

	t.Run("Template rendering", func(t *testing.T) {
		adapter := NewCursorAdapter()

		tests := []struct {
			name      string
			content   string
			variables map[string]string
			expected  string
		}{
			{
				name:      "Simple replacement",
				content:   "Hello {{.Name}}",
				variables: map[string]string{"Name": "World"},
				expected:  "Hello World",
			},
			{
				name:      "Multiple replacements",
				content:   "Project: {{.Project}}, Port: {{.Port}}",
				variables: map[string]string{"Project": "test", "Port": "8080"},
				expected:  "Project: test, Port: 8080",
			},
			{
				name:      "No variables",
				content:   "Static content",
				variables: map[string]string{},
				expected:  "Static content",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := adapter.renderTemplate(tt.content, tt.variables)
				if err != nil {
					t.Errorf("renderTemplate() error = %v", err)
					return
				}

				if result != tt.expected {
					t.Errorf("renderTemplate() = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("Marker block operations", func(t *testing.T) {
		adapter := NewCursorAdapter()

		skillID := "test-skill"
		content := "test content"

		// 测试标记块创建
		markerBlock := adapter.createMarkerBlock(skillID, content)
		expectedBegin := "# === SKILL-HUB BEGIN: test-skill ==="
		expectedEnd := "# === SKILL-HUB END: test-skill ==="

		if !contains(markerBlock, expectedBegin) {
			t.Errorf("Marker block missing begin marker: %s", expectedBegin)
		}

		if !contains(markerBlock, expectedEnd) {
			t.Errorf("Marker block missing end marker: %s", expectedEnd)
		}

		if !contains(markerBlock, content) {
			t.Errorf("Marker block missing content: %s", content)
		}

		// 测试标记块替换
		existingContent := "# === SKILL-HUB BEGIN: test-skill ===\nold content\n# === SKILL-HUB END: test-skill ==="
		newContent := adapter.replaceOrAddMarker(existingContent, skillID, markerBlock)

		if !contains(newContent, content) {
			t.Errorf("Replaced content missing new content: %s", content)
		}

		if contains(newContent, "old content") {
			t.Errorf("Replaced content still contains old content")
		}

		// 测试标记块添加（当不存在时）
		emptyContent := ""
		addedContent := adapter.replaceOrAddMarker(emptyContent, skillID, markerBlock)

		if addedContent != markerBlock {
			t.Errorf("Added content = %v, want %v", addedContent, markerBlock)
		}
	})

	t.Run("Extract marked content", func(t *testing.T) {
		adapter := NewCursorAdapter()

		skillID := "test-skill"
		content := "test content\nwith multiple lines"
		fullContent := "# === SKILL-HUB BEGIN: test-skill ===\n" + content + "\n# === SKILL-HUB END: test-skill ==="

		extracted, err := adapter.extractMarkedContent(fullContent, skillID)
		if err != nil {
			t.Errorf("extractMarkedContent() error = %v", err)
			return
		}

		if extracted != content {
			t.Errorf("extractMarkedContent() = %v, want %v", extracted, content)
		}

		// 测试找不到标记的情况
		_, err = adapter.extractMarkedContent("no markers here", skillID)
		if err == nil {
			t.Error("Expected error when no markers found")
		}
	})

	t.Run("Supports check", func(t *testing.T) {
		adapter := NewCursorAdapter()

		if !adapter.Supports() {
			t.Error("Supports() should return true for Cursor adapter")
		}
	})
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Path with ~",
			path:     "~/test/path",
			expected: "", // 具体值取决于用户主目录
		},
		{
			name:     "Path without ~",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "Relative path",
			path:     "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path)

			if tt.path == "~/test/path" {
				// 检查是否展开了~（应该包含用户主目录）
				homeDir, err := os.UserHomeDir()
				if err == nil && !contains(result, homeDir) {
					t.Errorf("expandPath() did not expand ~: %v", result)
				}
			} else if result != tt.expected {
				t.Errorf("expandPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
