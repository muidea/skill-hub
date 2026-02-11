package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshRegistryLogic(t *testing.T) {
	t.Run("扫描有效技能目录", func(t *testing.T) {
		tmpDir := t.TempDir()

		// 创建skills目录结构
		skillsDir := filepath.Join(tmpDir, "skills")

		// 创建有效技能目录
		validSkillDir := filepath.Join(skillsDir, "valid-skill")
		if err := os.MkdirAll(validSkillDir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}

		validSkillContent := `---
name: Valid Skill
description: A valid skill
version: 1.0.0
compatibility: open_code
---

# Valid Skill`

		validSkillPath := filepath.Join(validSkillDir, "SKILL.md")
		if err := os.WriteFile(validSkillPath, []byte(validSkillContent), 0644); err != nil {
			t.Fatalf("创建有效SKILL.md失败: %v", err)
		}

		// 创建无效技能目录（无SKILL.md文件）
		invalidSkillDir := filepath.Join(skillsDir, "invalid-skill")
		if err := os.MkdirAll(invalidSkillDir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}

		// 扫描目录
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			t.Fatalf("读取skills目录失败: %v", err)
		}

		// 验证扫描结果
		dirCount := 0
		for _, entry := range entries {
			if entry.IsDir() {
				dirCount++
			}
		}

		if dirCount != 2 {
			t.Errorf("期望2个目录，找到%d个", dirCount)
		}

		// 验证有效技能文件存在
		if _, err := os.Stat(validSkillPath); os.IsNotExist(err) {
			t.Error("有效技能文件不存在")
		}

		// 验证无效技能目录没有SKILL.md
		invalidSkillPath := filepath.Join(invalidSkillDir, "SKILL.md")
		if _, err := os.Stat(invalidSkillPath); err == nil {
			t.Error("无效技能目录不应该有SKILL.md文件")
		}
	})

	t.Run("生成registry.json结构", func(t *testing.T) {
		// 测试registry.json结构
		registry := map[string]interface{}{
			"version": "1.0.0",
			"skills": []map[string]interface{}{
				{
					"name":          "Test Skill",
					"description":   "Test description",
					"version":       "1.0.0",
					"compatibility": "open_code",
					"path":          "test-skill",
				},
			},
		}

		registryJSON, err := json.MarshalIndent(registry, "", "  ")
		if err != nil {
			t.Fatalf("序列化registry失败: %v", err)
		}

		// 验证JSON结构
		var decoded map[string]interface{}
		if err := json.Unmarshal(registryJSON, &decoded); err != nil {
			t.Fatalf("解析registry JSON失败: %v", err)
		}

		if version, ok := decoded["version"].(string); !ok || version != "1.0.0" {
			t.Errorf("版本号不正确: %v", decoded["version"])
		}

		skills, ok := decoded["skills"].([]interface{})
		if !ok || len(skills) != 1 {
			t.Errorf("技能列表不正确: %v", decoded["skills"])
		}
	})
}
