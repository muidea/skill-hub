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

	t.Run("Load skill from Markdown", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建测试技能目录
		skillID := "test-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		// 创建SKILL.md文件
		mdContent := `---
name: test-skill
description: A test skill for unit testing
compatibility:
  open_code: true
  cursor: true
  claude_code: true
metadata:
  version: 1.0.0
  author: Test Author
  tags: test,unit-test
---

# Test Skill Content

This is a test skill for unit testing.`

		mdPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write SKILL.md: %v", err)
		}

		// 加载技能
		skill, err := manager.LoadSkill(skillID)
		if err != nil {
			t.Fatalf("LoadSkill() error = %v", err)
		}

		if skill == nil {
			t.Fatal("LoadSkill() returned nil skill")
		}

		if skill.ID != skillID {
			t.Errorf("Skill.ID = %v, want %v", skill.ID, skillID)
		}

		if skill.Name != "test-skill" {
			t.Errorf("Skill.Name = %v, want %v", skill.Name, "test-skill")
		}

		if skill.Description != "A test skill for unit testing" {
			t.Errorf("Skill.Description = %v, want %v", skill.Description, "A test skill for unit testing")
		}

		if !skill.Compatibility.OpenCode {
			t.Error("Skill.Compatibility.OpenCode = false, want true")
		}

		if !skill.Compatibility.Cursor {
			t.Error("Skill.Compatibility.Cursor = false, want true")
		}

		if !skill.Compatibility.ClaudeCode {
			t.Error("Skill.Compatibility.ClaudeCode = false, want true")
		}
	})

	t.Run("Load non-existent skill", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		skill, err := manager.LoadSkill("non-existent-skill")
		if err == nil {
			t.Error("LoadSkill() should return error for non-existent skill")
		}
		if skill != nil {
			t.Error("LoadSkill() should return nil for non-existent skill")
		}
	})

	t.Run("Load skill without SKILL.md", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建空目录（没有SKILL.md文件）
		skillID := "empty-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		skill, err := manager.LoadSkill(skillID)
		if err == nil {
			t.Error("LoadSkill() should return error for skill without SKILL.md")
		}
		if skill != nil {
			t.Error("LoadSkill() should return nil for skill without SKILL.md")
		}
	})

	t.Run("Get skill prompt", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建测试技能目录
		skillID := "prompt-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		// 创建SKILL.md文件
		mdContent := `---
name: prompt-skill
description: A skill for prompt testing
compatibility:
  open_code: true
---

# Prompt Skill Content

This is the prompt content for testing.`

		mdPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write SKILL.md: %v", err)
		}

		// 获取提示词
		prompt, err := manager.GetSkillPrompt(skillID)
		if err != nil {
			t.Fatalf("GetSkillPrompt() error = %v", err)
		}

		if prompt != mdContent {
			t.Errorf("GetSkillPrompt() returned different content")
		}
	})

	t.Run("Check skill exists", func(t *testing.T) {
		manager := &SkillManager{skillsDir: skillsDir}

		// 创建测试技能目录
		skillID := "exists-skill"
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}

		// 创建SKILL.md文件
		mdContent := `---
name: exists-skill
description: A skill for exists testing
---
# Exists Skill`

		mdPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
			t.Fatalf("Failed to write SKILL.md: %v", err)
		}

		// 检查技能是否存在
		exists := manager.SkillExists(skillID)
		if !exists {
			t.Error("SkillExists() = false, want true")
		}

		// 检查不存在的技能
		notExists := manager.SkillExists("non-existent-skill")
		if notExists {
			t.Error("SkillExists() = true for non-existent skill, want false")
		}
	})

	t.Run("Load all skills", func(t *testing.T) {
		// 为这个测试创建独立的临时目录
		testSkillsDir := filepath.Join(t.TempDir(), "test-skills")
		if err := os.MkdirAll(testSkillsDir, 0755); err != nil {
			t.Fatalf("Failed to create test skills directory: %v", err)
		}

		manager := &SkillManager{skillsDir: testSkillsDir}

		// 创建多个测试技能
		skillIDs := []string{"skill-1", "skill-2", "skill-3"}
		for _, skillID := range skillIDs {
			skillDir := filepath.Join(testSkillsDir, skillID)
			if err := os.MkdirAll(skillDir, 0755); err != nil {
				t.Fatalf("Failed to create skill directory: %v", err)
			}

			mdContent := `---
name: ` + skillID + `
description: Test skill ` + skillID + `
---
# Content for ` + skillID

			mdPath := filepath.Join(skillDir, "SKILL.md")
			if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
				t.Fatalf("Failed to write SKILL.md: %v", err)
			}
		}

		// 加载所有技能
		skills, err := manager.LoadAllSkills()
		if err != nil {
			t.Fatalf("LoadAllSkills() error = %v", err)
		}

		if len(skills) != len(skillIDs) {
			t.Errorf("LoadAllSkills() returned %d skills, want %d", len(skills), len(skillIDs))
		}

		// 验证每个技能都被加载
		loadedIDs := make(map[string]bool)
		for _, skill := range skills {
			loadedIDs[skill.ID] = true
		}

		for _, skillID := range skillIDs {
			if !loadedIDs[skillID] {
				t.Errorf("Skill %s not loaded by LoadAllSkills()", skillID)
			}
		}
	})
}
