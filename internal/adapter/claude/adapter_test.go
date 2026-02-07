package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestClaudeAdapter(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Create adapter", func(t *testing.T) {
		adapter := NewClaudeAdapter()
		if adapter == nil {
			t.Error("NewClaudeAdapter() returned nil")
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

	t.Run("Config file operations", func(t *testing.T) {
		adapter := NewClaudeAdapter().WithProjectMode()

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
		configPath, err := adapter.GetConfigPath()
		if err != nil {
			t.Errorf("GetConfigPath() error = %v", err)
		}

		expectedPath := filepath.Join(tmpDir, ".clauderc")
		if configPath != expectedPath {
			t.Errorf("GetConfigPath() = %v, want %v", configPath, expectedPath)
		}

		// 测试默认配置创建
		defaultConfig := adapter.createDefaultConfig()
		if defaultConfig == nil {
			t.Error("createDefaultConfig() returned nil")
		}

		// 检查默认配置结构
		if version, ok := defaultConfig["version"].(string); !ok || version != "1.0" {
			t.Errorf("Default config missing version field or incorrect value")
		}

		if settings, ok := defaultConfig["settings"].(map[string]interface{}); !ok {
			t.Errorf("Default config missing settings field")
		} else {
			if editor, ok := settings["editor"].(map[string]interface{}); !ok {
				t.Errorf("Default config missing editor field")
			} else {
				if theme, ok := editor["theme"].(string); !ok || theme != "dark" {
					t.Errorf("Default config editor theme incorrect")
				}
			}
		}

		// 测试配置读写
		adapter.configPath = configPath

		// 写入测试配置
		testConfig := map[string]interface{}{
			"version": "1.0",
			"settings": map[string]interface{}{
				"editor": map[string]interface{}{
					"theme":    "light",
					"fontSize": 12,
				},
			},
			"customInstructions": []interface{}{},
		}

		if err := adapter.writeConfig(testConfig); err != nil {
			t.Errorf("writeConfig() error = %v", err)
		}

		// 读取配置
		readConfig, err := adapter.readConfig()
		if err != nil {
			t.Errorf("readConfig() error = %v", err)
		}

		// 验证配置内容
		if readVersion, ok := readConfig["version"].(string); !ok || readVersion != "1.0" {
			t.Errorf("Read config version incorrect")
		}

		// 验证文件存在
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("Config file not created: %v", err)
		}
	})

	t.Run("Template rendering", func(t *testing.T) {
		adapter := NewClaudeAdapter()

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

	t.Run("Skill injection and extraction", func(t *testing.T) {
		adapter := NewClaudeAdapter()

		skillID := "test-skill"
		content := "test content\nwith multiple lines"

		// 创建测试配置
		configData := adapter.createDefaultConfig()

		// 注入技能
		if err := adapter.injectSkill(configData, skillID, content); err != nil {
			t.Errorf("injectSkill() error = %v", err)
		}

		// 验证技能被注入
		instructions, ok := configData["customInstructions"].([]interface{})
		if !ok || len(instructions) != 1 {
			t.Errorf("Skill not injected properly")
		}

		// 提取技能
		extracted, err := adapter.extractSkill(configData, skillID)
		if err != nil {
			t.Errorf("extractSkill() error = %v", err)
		}

		// 验证提取的内容
		if extracted != content {
			t.Errorf("extractSkill() = %v, want %v", extracted, content)
		}

		// 测试技能列表
		skillList := adapter.listSkills(configData)
		if len(skillList) != 1 || skillList[0] != skillID {
			t.Errorf("listSkills() = %v, want [%s]", skillList, skillID)
		}

		// 测试技能移除
		if err := adapter.removeSkill(configData, skillID); err != nil {
			t.Errorf("removeSkill() error = %v", err)
		}

		// 验证技能被移除
		instructionsAfter, _ := configData["customInstructions"].([]interface{})
		if len(instructionsAfter) != 0 {
			t.Errorf("Skill not removed properly")
		}

		// 测试提取不存在的技能
		_, err = adapter.extractSkill(configData, "non-existent")
		if err == nil {
			t.Error("Expected error when extracting non-existent skill")
		}
	})

	t.Run("Multiple skills management", func(t *testing.T) {
		adapter := NewClaudeAdapter()
		configData := adapter.createDefaultConfig()

		// 注入多个技能
		skills := []struct {
			id      string
			content string
		}{
			{"skill-1", "Content for skill 1"},
			{"skill-2", "Content for skill 2"},
			{"skill-3", "Content for skill 3"},
		}

		for _, skill := range skills {
			if err := adapter.injectSkill(configData, skill.id, skill.content); err != nil {
				t.Errorf("injectSkill(%s) error = %v", skill.id, err)
			}
		}

		// 验证技能数量
		skillList := adapter.listSkills(configData)
		if len(skillList) != 3 {
			t.Errorf("Expected 3 skills, got %d", len(skillList))
		}

		// 验证技能ID
		expectedIDs := map[string]bool{"skill-1": true, "skill-2": true, "skill-3": true}
		for _, id := range skillList {
			if !expectedIDs[id] {
				t.Errorf("Unexpected skill ID: %s", id)
			}
		}

		// 测试更新现有技能
		updatedContent := "Updated content for skill 2"
		if err := adapter.injectSkill(configData, "skill-2", updatedContent); err != nil {
			t.Errorf("injectSkill(update) error = %v", err)
		}

		// 验证更新
		extracted, err := adapter.extractSkill(configData, "skill-2")
		if err != nil {
			t.Errorf("extractSkill(updated) error = %v", err)
		}

		if extracted != updatedContent {
			t.Errorf("Skill not updated properly: got %v, want %v", extracted, updatedContent)
		}

		// 验证技能数量不变
		skillListAfterUpdate := adapter.listSkills(configData)
		if len(skillListAfterUpdate) != 3 {
			t.Errorf("Skill count changed after update: got %d, want 3", len(skillListAfterUpdate))
		}
	})

	t.Run("Supports check", func(t *testing.T) {
		adapter := NewClaudeAdapter()

		if !adapter.Supports() {
			t.Error("Supports() should return true for Claude adapter")
		}
	})

	t.Run("JSON serialization", func(t *testing.T) {
		adapter := NewClaudeAdapter()

		// 创建包含技能的配置
		configData := adapter.createDefaultConfig()

		// 注入测试技能
		if err := adapter.injectSkill(configData, "test-skill", "test content"); err != nil {
			t.Fatalf("Failed to inject skill: %v", err)
		}

		// 序列化配置
		data, err := json.MarshalIndent(configData, "", "  ")
		if err != nil {
			t.Errorf("JSON serialization error = %v", err)
		}

		// 反序列化验证
		var parsedConfig map[string]interface{}
		if err := json.Unmarshal(data, &parsedConfig); err != nil {
			t.Errorf("JSON deserialization error = %v", err)
		}

		// 验证结构
		if _, ok := parsedConfig["version"].(string); !ok {
			t.Error("Parsed config missing version field")
		}

		if _, ok := parsedConfig["customInstructions"].([]interface{}); !ok {
			t.Error("Parsed config missing customInstructions field")
		}
	})

	t.Run("Expand path", func(t *testing.T) {
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
	})

	t.Run("Error handling", func(t *testing.T) {
		adapter := NewClaudeAdapter()

		// 测试读取不存在的文件
		adapter.configPath = filepath.Join(tmpDir, "non-existent.json")
		_, err := adapter.readConfig()
		if err == nil {
			t.Error("Expected error when reading non-existent file")
		}

		// 测试无效的JSON
		invalidJSONPath := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(invalidJSONPath, []byte("{invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON file: %v", err)
		}

		adapter.configPath = invalidJSONPath
		_, err = adapter.readConfig()
		if err == nil {
			t.Error("Expected error when reading invalid JSON")
		}

		// 测试无效的配置结构
		configData := map[string]interface{}{
			"customInstructions": "not an array", // 应该是数组
		}

		// 测试提取技能时的错误
		_, err = adapter.extractSkill(configData, "test-skill")
		if err == nil {
			t.Error("Expected error when customInstructions is not an array")
		}

		// 测试移除技能时的错误
		err = adapter.removeSkill(configData, "test-skill")
		if err == nil {
			t.Error("Expected error when customInstructions is not an array")
		}
	})
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
