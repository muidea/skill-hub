package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillManager(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试技能目录结构
	skillsDir := filepath.Join(tmpDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("Failed to create skills directory: %v", err)
	}

	t.Run("Create skill manager", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		if manager == nil {
			t.Error("SkillManager creation returned nil")
		}

		if manager.skillsDir != skillsDir {
			t.Errorf("Skills directory = %v, want %v", manager.skillsDir, skillsDir)
		}
	})

	t.Run("Load skill from YAML", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建测试技能目录
		skillID := "test-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		// 创建skill.yaml文件（使用实际支持的格式）
		yamlContent := `id: "test-skill"
name: "Test Skill"
version: "1.0.0"
description: "A test skill for unit testing"
author: "Test Author"
tags: ["test", "unit-test"]
compatibility:
  cursor: true
  claude_code: true
  open_code: true
variables:
  - name: "project_name"
    default: "{{ .ProjectName }}"
    description: "Project name"
  - name: "language"
    default: "{{ .Language }}"
    description: "Programming language"`

		yamlPath := filepath.Join(skillDir, "skill.yaml")
		if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write skill.yaml: %v", err)
		}

		// 加载技能
		skill, err := manager.loadSkillFromYAML(yamlPath, skillID)
		if err != nil {
			t.Errorf("loadSkillFromYAML() error = %v", err)
		}

		if skill == nil {
			t.Error("loadSkillFromYAML() returned nil")
		}

		// 验证技能属性
		if skill.ID != skillID {
			t.Errorf("Skill ID = %v, want %v", skill.ID, skillID)
		}

		if skill.Name != "Test Skill" {
			t.Errorf("Skill name = %v, want Test Skill", skill.Name)
		}

		if skill.Version != "1.0.0" {
			t.Errorf("Skill version = %v, want 1.0.0", skill.Version)
		}

		if skill.Description != "A test skill for unit testing" {
			t.Errorf("Skill description = %v, want A test skill for unit testing", skill.Description)
		}

		if skill.Author != "Test Author" {
			t.Errorf("Skill author = %v, want Test Author", skill.Author)
		}

		if len(skill.Tags) != 2 || skill.Tags[0] != "test" || skill.Tags[1] != "unit-test" {
			t.Errorf("Skill tags = %v, want [test unit-test]", skill.Tags)
		}

		// 验证兼容性
		if !skill.Compatibility.Cursor {
			t.Error("Cursor compatibility should be true")
		}

		if !skill.Compatibility.ClaudeCode {
			t.Error("ClaudeCode compatibility should be true")
		}

		if !skill.Compatibility.OpenCode {
			t.Error("OpenCode compatibility should be true")
		}

		// 验证变量
		if len(skill.Variables) != 2 {
			t.Errorf("Variables count = %d, want 2", len(skill.Variables))
		}

		// 验证变量内容
		foundProjectName := false
		foundLanguage := false
		for _, variable := range skill.Variables {
			if variable.Name == "project_name" {
				foundProjectName = true
				if variable.Default != "{{ .ProjectName }}" {
					t.Errorf("project_name default = %v, want {{ .ProjectName }}", variable.Default)
				}
			}
			if variable.Name == "language" {
				foundLanguage = true
				if variable.Default != "{{ .Language }}" {
					t.Errorf("language default = %v, want {{ .Language }}", variable.Default)
				}
			}
		}

		if !foundProjectName {
			t.Error("project_name variable not found")
		}

		if !foundLanguage {
			t.Error("language variable not found")
		}
	})

	t.Run("Load skill from directory", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		skillID := "directory-skill"

		// 测试加载不存在的技能
		_, err := manager.loadSkillFromDirectory(filepath.Join(skillsDir, "non-existent"), skillID)
		if err == nil {
			t.Error("Expected error when loading non-existent skill")
		}

		// 测试目录存在但没有技能文件
		emptyDir := filepath.Join(skillsDir, "empty-skill")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		_, err = manager.loadSkillFromDirectory(emptyDir, "empty-skill")
		if err == nil {
			t.Error("Expected error when directory has no skill files")
		}
	})

	t.Run("Skill loading errors", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 测试不存在的技能
		_, err := manager.LoadSkill("non-existent-skill")
		if err == nil {
			t.Error("Expected error when loading non-existent skill")
		}

		// 测试无效的YAML文件
		invalidSkillDir := filepath.Join(skillsDir, "invalid-yaml")
		if err := os.MkdirAll(invalidSkillDir, 0755); err != nil {
			t.Fatalf("Failed to create invalid skill directory: %v", err)
		}

		invalidYamlPath := filepath.Join(invalidSkillDir, "skill.yaml")
		if err := os.WriteFile(invalidYamlPath, []byte("invalid: yaml: content"), 0644); err != nil {
			t.Fatalf("Failed to write invalid YAML: %v", err)
		}

		_, err = manager.loadSkillFromYAML(invalidYamlPath, "invalid-yaml")
		if err == nil {
			t.Error("Expected error when parsing invalid YAML")
		}

		// 测试不存在的YAML文件
		_, err = manager.loadSkillFromYAML(filepath.Join(skillsDir, "non-existent.yaml"), "test")
		if err == nil {
			t.Error("Expected error when YAML file doesn't exist")
		}
	})

	t.Run("Skill compatibility parsing", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		testCases := []struct {
			name         string
			yamlContent  string
			expectCursor bool
			expectClaude bool
			expectOpen   bool
		}{
			{
				name: "All targets enabled",
				yamlContent: `id: "test-skill"
name: "Test"
compatibility:
  cursor: true
  claude_code: true
  open_code: true`,
				expectCursor: true,
				expectClaude: true,
				expectOpen:   true,
			},
			{
				name: "Only cursor enabled",
				yamlContent: `id: "test-skill"
name: "Test"
compatibility:
  cursor: true
  claude_code: false
  open_code: false`,
				expectCursor: true,
				expectClaude: false,
				expectOpen:   false,
			},
			{
				name: "Mixed compatibility",
				yamlContent: `id: "test-skill"
name: "Test"
compatibility:
  cursor: false
  claude_code: true
  open_code: true`,
				expectCursor: false,
				expectClaude: true,
				expectOpen:   true,
			},
			{
				name: "No compatibility specified",
				yamlContent: `id: "test-skill"
name: "Test"`,
				expectCursor: false,
				expectClaude: false,
				expectOpen:   false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// 创建临时YAML文件
				tempDir := t.TempDir()
				yamlPath := filepath.Join(tempDir, "test.yaml")
				if err := os.WriteFile(yamlPath, []byte(tc.yamlContent), 0644); err != nil {
					t.Fatalf("Failed to write test YAML: %v", err)
				}

				skill, err := manager.loadSkillFromYAML(yamlPath, "test-skill")
				if err != nil {
					t.Errorf("loadSkillFromYAML() error = %v", err)
					return
				}

				if skill.Compatibility.Cursor != tc.expectCursor {
					t.Errorf("Cursor compatibility = %v, want %v", skill.Compatibility.Cursor, tc.expectCursor)
				}

				if skill.Compatibility.ClaudeCode != tc.expectClaude {
					t.Errorf("ClaudeCode compatibility = %v, want %v", skill.Compatibility.ClaudeCode, tc.expectClaude)
				}

				if skill.Compatibility.OpenCode != tc.expectOpen {
					t.Errorf("OpenCode compatibility = %v, want %v", skill.Compatibility.OpenCode, tc.expectOpen)
				}
			})
		}
	})

	t.Run("Skill variable parsing", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		yamlContent := `id: "variable-skill"
name: "Variable Test"
version: "1.0.0"
description: "Test skill with variables"
author: "Test Author"
tags: ["test", "variables"]
compatibility:
  cursor: true
  claude_code: true
  open_code: true
variables:
  - name: "project_name"
    default: "{{ .ProjectName }}"
    description: "Project name"
  - name: "language"
    default: "{{ .Language }}"
    description: "Programming language"
  - name: "version"
    default: "1.0.0"
    description: "Version"
  - name: "debug"
    default: "{{ .Debug }}"
    description: "Debug mode"`

		tempDir := t.TempDir()
		yamlPath := filepath.Join(tempDir, "test.yaml")
		if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write test YAML: %v", err)
		}

		skill, err := manager.loadSkillFromYAML(yamlPath, "variable-skill")
		if err != nil {
			t.Errorf("loadSkillFromYAML() error = %v", err)
			return
		}

		if len(skill.Variables) != 4 {
			t.Errorf("Variables count = %d, want 4", len(skill.Variables))
		}

		// 验证变量存在
		foundVars := map[string]bool{
			"project_name": false,
			"language":     false,
			"version":      false,
			"debug":        false,
		}

		for _, variable := range skill.Variables {
			if _, exists := foundVars[variable.Name]; exists {
				foundVars[variable.Name] = true
			}
		}

		for name, found := range foundVars {
			if !found {
				t.Errorf("Variable %s not found", name)
			}
		}
	})

	t.Run("Get skill prompt", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建测试技能
		skillID := "prompt-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		// 创建skill.yaml文件
		yamlContent := `id: "prompt-skill"
name: "Prompt Skill"
version: "1.0.0"
description: "A skill with prompt"
author: "Test Author"
tags: ["test", "prompt"]
compatibility:
  cursor: true
  claude_code: true
  open_code: true`

		yamlPath := filepath.Join(skillDir, "skill.yaml")
		if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to write skill.yaml: %v", err)
		}

		// 创建prompt.md文件
		promptContent := `# Prompt Skill
This is a prompt skill.

Variables:
- Project: {{.project_name}}
- Language: {{.language}}`

		promptPath := filepath.Join(skillDir, "prompt.md")
		if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
			t.Fatalf("Failed to write prompt.md: %v", err)
		}

		// 测试GetSkillPrompt方法
		prompt, err := manager.GetSkillPrompt(skillID)
		if err != nil {
			t.Errorf("GetSkillPrompt() error = %v", err)
			return
		}

		// 验证技能内容
		expectedContent := "# Prompt Skill\nThis is a prompt skill.\n\nVariables:\n- Project: {{.project_name}}\n- Language: {{.language}}"
		if prompt != expectedContent {
			t.Errorf("Skill prompt = %v, want %v", prompt, expectedContent)
		}
	})
}
