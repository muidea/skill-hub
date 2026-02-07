package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidSkillName(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		expected  bool
	}{
		{"valid name", "my-skill", true},
		{"valid with numbers", "skill-123", true},
		{"empty name", "", false},
		{"uppercase letters", "My-Skill", false},
		{"starts with hyphen", "-skill", false},
		{"ends with hyphen", "skill-", false},
		{"double hyphen", "skill--name", false},
		{"underscore", "skill_name", false},
		{"space", "skill name", false},
		{"special chars", "skill@name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSkillName(tt.skillName)
			if result != tt.expected {
				t.Errorf("isValidSkillName(%q) = %v, want %v", tt.skillName, result, tt.expected)
			}
		})
	}
}

func TestIsValidCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		compatibility string
		expected      bool
	}{
		{"cursor", "cursor", true},
		{"claude", "claude", true},
		{"opencode", "opencode", true},
		{"all", "all", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
		{"mixed case", "Cursor", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCompatibility(tt.compatibility)
			if result != tt.expected {
				t.Errorf("isValidCompatibility(%q) = %v, want %v", tt.compatibility, result, tt.expected)
			}
		})
	}
}

func TestGenerateCompatibilityDescription(t *testing.T) {
	tests := []struct {
		name          string
		compatibility string
		expected      string
	}{
		{"cursor", "cursor", "Designed for Cursor (or similar AI coding assistants)"},
		{"claude", "claude", "Designed for Claude Code (or similar AI coding assistants)"},
		{"opencode", "opencode", "Designed for OpenCode (or similar AI coding assistants)"},
		{"all", "all", "Designed for Cursor, Claude Code, and OpenCode (or similar AI coding assistants)"},
		{"invalid", "invalid", "Designed for AI coding assistants"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCompatibilityDescription(tt.compatibility)
			if result != tt.expected {
				t.Errorf("generateCompatibilityDescription(%q) = %q, want %q", tt.compatibility, result, tt.expected)
			}
		})
	}
}

func TestGenerateSkillContent(t *testing.T) {
	tests := []struct {
		name          string
		skillName     string
		description   string
		compatibility string
		expectError   bool
	}{
		{"valid skill", "test-skill", "Test description", "all", false},
		{"cursor skill", "cursor-skill", "Cursor skill", "cursor", false},
		{"claude skill", "claude-skill", "Claude skill", "claude", false},
		{"opencode skill", "opencode-skill", "OpenCode skill", "opencode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := generateSkillContent(tt.skillName, tt.description, tt.compatibility)

			if tt.expectError {
				if err == nil {
					t.Errorf("generateSkillContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("generateSkillContent() unexpected error: %v", err)
				return
			}

			// 检查基本内容
			if content == "" {
				t.Error("generateSkillContent() returned empty content")
			}

			// 检查是否包含技能名称
			if !contains(content, tt.skillName) {
				t.Errorf("generateSkillContent() content doesn't contain skill name: %s", tt.skillName)
			}

			// 检查是否包含描述
			if !contains(content, tt.description) {
				t.Errorf("generateSkillContent() content doesn't contain description: %s", tt.description)
			}

			// 检查是否包含frontmatter
			if !contains(content, "---") {
				t.Error("generateSkillContent() content doesn't contain frontmatter")
			}

			// 检查是否包含版本号
			if !contains(content, "version: \"1.0.0\"") {
				t.Error("generateSkillContent() content doesn't contain version")
			}
		})
	}
}

func TestCreateCommandIntegration(t *testing.T) {
	// 创建临时目录进行测试
	tempDir := t.TempDir()

	// 切换到临时目录
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// 测试创建技能
	skillName := "test-integration-skill"

	// 模拟运行create命令
	err = runCreate(skillName)
	if err != nil {
		t.Errorf("runCreate() failed: %v", err)
	}

	// 检查文件是否创建
	skillFilePath := filepath.Join(tempDir, "SKILL.md")
	if _, err := os.Stat(skillFilePath); os.IsNotExist(err) {
		t.Error("SKILL.md file was not created")
	}

	// 读取文件内容
	content, err := os.ReadFile(skillFilePath)
	if err != nil {
		t.Errorf("Failed to read SKILL.md: %v", err)
	}

	contentStr := string(content)

	// 验证文件内容
	if !contains(contentStr, skillName) {
		t.Errorf("SKILL.md doesn't contain skill name: %s", skillName)
	}

	if !contains(contentStr, "version: \"1.0.0\"") {
		t.Error("SKILL.md doesn't contain version")
	}

	// 测试覆盖现有文件
	err = runCreate(skillName)
	if err != nil {
		t.Errorf("runCreate() failed on second run: %v", err)
	}

	// 清理
	os.Remove(skillFilePath)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr)) &&
		(s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
