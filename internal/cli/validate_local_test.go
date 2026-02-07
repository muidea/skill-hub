package cli

import (
	"os"
	"path/filepath"
	"testing"

	"skill-hub/pkg/spec"
)

func TestValidateLocalCommand(t *testing.T) {
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

	// 创建测试技能文件
	skillName := "test-validation-skill"
	skillDir := filepath.Join(tempDir, ".skill-hub", "repo", "skills", skillName)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill directory: %v", err)
	}

	// 创建SKILL.md文件
	skillContent := `---
name: test-validation-skill
description: Test skill for validation
compatibility: Designed for Cursor, Claude Code, and OpenCode (or similar AI coding assistants)
metadata:
  version: "1.0.0"
  author: "test"
  created_at: "2024-01-01T00:00:00Z"
---
# Test Validation Skill

Test skill for validation testing.

## Variables

- ` + "`" + `PROJECT_NAME` + "`" + `: Project name {{.PROJECT_NAME}}
- ` + "`" + `LANGUAGE` + "`" + `: Programming language {{.LANGUAGE}}
`

	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillFilePath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}

	// 创建项目状态文件
	stateDir := filepath.Join(tempDir, ".skill-hub", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Failed to create state directory: %v", err)
	}

	// 注意：实际测试需要更完整的模拟环境
	// 这里只是验证函数逻辑，不运行完整命令

	t.Run("validate variables", func(t *testing.T) {
		// 测试变量验证逻辑
		skill := &spec.Skill{
			ID:   "test-skill",
			Name: "Test Skill",
			Variables: []spec.Variable{
				{Name: "PROJECT_NAME", Default: "test-project", Description: "Project name"},
				{Name: "LANGUAGE", Default: "Go", Description: "Programming language"},
			},
		}

		variables := map[string]string{
			"PROJECT_NAME": "my-project",
			"LANGUAGE":     "Go",
		}

		result := &spec.ValidationResult{
			SkillID: "test-skill",
			IsValid: true,
		}

		err := validateVariables(skill, variables, result)
		if err != nil {
			t.Errorf("validateVariables() failed: %v", err)
		}

		if !result.IsValid {
			t.Error("validateVariables() should be valid")
		}

		if len(result.Errors) > 0 {
			t.Errorf("validateVariables() should have no errors, got: %v", result.Errors)
		}
	})

	t.Run("validate missing required variable", func(t *testing.T) {
		skill := &spec.Skill{
			ID:   "test-skill",
			Name: "Test Skill",
			Variables: []spec.Variable{
				{Name: "REQUIRED_VAR", Default: ""},
			},
		}

		variables := map[string]string{}

		result := &spec.ValidationResult{
			SkillID: "test-skill",
			IsValid: true,
		}

		err := validateVariables(skill, variables, result)
		if err != nil {
			t.Errorf("validateVariables() should not fail for missing required variable (now warning): %v", err)
		}

		// 现在缺少必需变量是警告而不是错误
		if len(result.Errors) > 0 {
			t.Error("validateVariables() should not have errors for missing required variable (should be warning)")
		}

		if len(result.Warnings) == 0 {
			t.Error("validateVariables() should have warnings for missing required variable")
		}
	})

	t.Run("validate adapter compatibility", func(t *testing.T) {
		skill := &spec.Skill{
			ID:            "test-skill",
			Name:          "Test Skill",
			Compatibility: "Designed for Cursor and Claude Code",
		}

		result := &spec.ValidationResult{
			SkillID: "test-skill",
			IsValid: true,
		}

		// 测试支持的适配器
		err := validateAdapterCompatibility(skill, "cursor", result)
		if err != nil {
			t.Errorf("validateAdapterCompatibility() failed for cursor: %v", err)
		}

		// 测试不支持的适配器
		result2 := &spec.ValidationResult{
			SkillID: "test-skill",
			IsValid: true,
		}

		err = validateAdapterCompatibility(skill, "opencode", result2)
		if err == nil {
			t.Error("validateAdapterCompatibility() should fail for unsupported adapter")
		}

		if len(result2.Errors) == 0 {
			t.Error("validateAdapterCompatibility() should have errors for unsupported adapter")
		}
	})

	t.Run("validate auto adapter detection", func(t *testing.T) {
		skill := &spec.Skill{
			ID:            "test-skill",
			Name:          "Test Skill",
			Compatibility: "Designed for Cursor",
		}

		result := &spec.ValidationResult{
			SkillID: "test-skill",
			IsValid: true,
		}

		err := validateAdapterCompatibility(skill, "auto", result)
		if err != nil {
			t.Errorf("validateAdapterCompatibility() failed for auto: %v", err)
		}

		// 应该只检查cursor，不检查其他
		if len(result.Warnings) > 1 {
			t.Errorf("validateAdapterCompatibility() should have minimal warnings for auto mode, got: %v", result.Warnings)
		}
	})
}

func TestValidationResultStructure(t *testing.T) {
	// 测试ValidationResult结构
	result := &spec.ValidationResult{
		SkillID:  "test-skill",
		IsValid:  false,
		Errors:   []string{"Error 1", "Error 2"},
		Warnings: []string{"Warning 1"},
	}

	if result.SkillID != "test-skill" {
		t.Errorf("SkillID = %s, want test-skill", result.SkillID)
	}

	if result.IsValid {
		t.Error("IsValid should be false")
	}

	if len(result.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(result.Errors))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("Warnings length = %d, want 1", len(result.Warnings))
	}

	// 测试有效的结果
	validResult := &spec.ValidationResult{
		SkillID: "valid-skill",
		IsValid: true,
	}

	if !validResult.IsValid {
		t.Error("IsValid should be true for valid result")
	}

	if len(validResult.Errors) != 0 {
		t.Errorf("Valid result should have no errors, got: %v", validResult.Errors)
	}
}
