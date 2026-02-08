package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenCodeAdapter(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("Create adapter", func(t *testing.T) {
		adapter := NewOpenCodeAdapter()
		if adapter == nil {
			t.Error("NewOpenCodeAdapter() returned nil")
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

	t.Run("Directory operations", func(t *testing.T) {
		adapter := NewOpenCodeAdapter().WithProjectMode()

		// 模拟当前目录
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// 测试基础路径获取
		basePath, err := adapter.getBasePath()
		if err != nil {
			t.Errorf("getBasePath() error = %v", err)
		}

		expectedPath := filepath.Join(tmpDir, ".agents")
		if basePath != expectedPath {
			t.Errorf("getBasePath() = %v, want %v", basePath, expectedPath)
		}

		// 测试技能路径获取
		skillsPath, err := adapter.GetSkillsPath()
		if err != nil {
			t.Errorf("GetSkillsPath() error = %v", err)
		}

		expectedSkillsPath := filepath.Join(basePath, "skills")
		if skillsPath != expectedSkillsPath {
			t.Errorf("GetSkillsPath() = %v, want %v", skillsPath, expectedSkillsPath)
		}

		// 测试目录创建
		testDir := filepath.Join(tmpDir, "test-dir")
		if err := createSkillDirectory(testDir); err != nil {
			t.Errorf("createSkillDirectory() error = %v", err)
		}

		// 验证目录创建
		if _, err := os.Stat(testDir); err != nil {
			t.Errorf("Directory not created: %v", err)
		}

		// 测试目录已存在时的处理
		if err := createSkillDirectory(testDir); err != nil {
			t.Errorf("createSkillDirectory(existing) error = %v", err)
		}

		// 测试文件写入
		testFile := filepath.Join(testDir, "test.txt")
		testContent := "test content"
		if err := writeSkillMDFile(testFile, testContent); err != nil {
			t.Errorf("writeSkillMDFile() error = %v", err)
		}

		// 验证文件内容
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read test file: %v", err)
		}

		if string(data) != testContent {
			t.Errorf("File content = %v, want %v", string(data), testContent)
		}

		// 测试文件已存在时的写入
		newContent := "new content"
		if err := writeSkillMDFile(testFile, newContent); err != nil {
			t.Errorf("writeSkillMDFile(existing) error = %v", err)
		}

		// 验证文件更新
		data, err = os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read updated file: %v", err)
		}

		if string(data) != newContent {
			t.Errorf("Updated file content = %v, want %v", string(data), newContent)
		}
	})

	t.Run("Skill management", func(t *testing.T) {
		adapter := NewOpenCodeAdapter().WithProjectMode()

		// 模拟当前目录
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		skillID := "test-skill"
		content := "test content\nwith multiple lines"

		// 测试技能应用
		if err := adapter.Apply(skillID, content, map[string]string{}); err != nil {
			t.Errorf("Apply() error = %v", err)
		}

		// 验证技能目录创建
		basePath, _ := adapter.getBasePath()
		skillDir := filepath.Join(basePath, "skills", skillID)
		if _, err := os.Stat(skillDir); err != nil {
			t.Errorf("Skill directory not created: %v", err)
		}

		// 验证SKILL.md文件创建
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Errorf("SKILL.md file not created: %v", err)
		}

		// 测试技能提取
		extracted, err := adapter.Extract(skillID)
		if err != nil {
			t.Errorf("Extract() error = %v", err)
		}

		// 验证提取的内容（包含OpenCode格式的frontmatter）
		expectedContent := "---\ndescription: 'Skill: test-skill'\nmetadata:\n    source: skill-hub\nname: test-skill\n---\n" + content
		if extracted != expectedContent {
			t.Errorf("Extract() = %v, want %v", extracted, expectedContent)
		}

		// 测试技能列表
		skillList, err := adapter.List()
		if err != nil {
			t.Errorf("List() error = %v", err)
		}

		if len(skillList) != 1 || skillList[0] != skillID {
			t.Errorf("List() = %v, want [%s]", skillList, skillID)
		}

		// 测试技能移除
		if err := adapter.Remove(skillID); err != nil {
			t.Errorf("Remove() error = %v", err)
		}

		// 验证技能被移除
		if _, err := os.Stat(skillDir); err == nil {
			t.Error("Skill directory not removed")
		}

		// 验证技能列表为空
		skillListAfter, err := adapter.List()
		if err != nil {
			t.Errorf("List(after remove) error = %v", err)
		}

		if len(skillListAfter) != 0 {
			t.Errorf("List() after remove = %v, want []", skillListAfter)
		}

		// 测试提取不存在的技能
		extracted, err = adapter.Extract("non-existent")
		if err != nil {
			t.Errorf("Extract(non-existent) error = %v", err)
		}

		if extracted != "" {
			t.Errorf("Extract(non-existent) = %v, want empty string", extracted)
		}

		// 测试移除不存在的技能
		if err := adapter.Remove("non-existent"); err != nil {
			t.Errorf("Remove(non-existent) error = %v", err)
		}
	})

	t.Run("Multiple skills management", func(t *testing.T) {
		adapter := NewOpenCodeAdapter().WithProjectMode()

		// 模拟当前目录
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// 创建多个技能
		skills := []struct {
			id      string
			content string
		}{
			{"skill-1", "Content for skill 1"},
			{"skill-2", "Content for skill 2"},
			{"skill-3", "Content for skill 3"},
		}

		for _, skill := range skills {
			if err := adapter.Apply(skill.id, skill.content, map[string]string{}); err != nil {
				t.Errorf("Apply(%s) error = %v", skill.id, err)
			}
		}

		// 验证技能列表
		skillList, err := adapter.List()
		if err != nil {
			t.Errorf("List() error = %v", err)
		}

		if len(skillList) != 3 {
			t.Errorf("Expected 3 skills, got %d", len(skillList))
		}

		// 验证每个技能的内容
		for _, skill := range skills {
			extracted, err := adapter.Extract(skill.id)
			if err != nil {
				t.Errorf("Extract(%s) error = %v", skill.id, err)
			}

			expectedContent := fmt.Sprintf("---\ndescription: 'Skill: %s'\nmetadata:\n    source: skill-hub\nname: %s\n---\n%s", skill.id, skill.id, skill.content)
			if extracted != expectedContent {
				t.Errorf("Extract(%s) = %v, want %v", skill.id, extracted, expectedContent)
			}
		}

		// 测试更新现有技能
		updatedContent := "Updated content for skill 2"
		if err := adapter.Apply("skill-2", updatedContent, map[string]string{}); err != nil {
			t.Errorf("Apply(update) error = %v", err)
		}

		// 验证更新
		extracted, err := adapter.Extract("skill-2")
		if err != nil {
			t.Errorf("Extract(updated) error = %v", err)
		}

		expectedUpdatedContent := fmt.Sprintf("---\ndescription: 'Skill: skill-2'\nmetadata:\n    source: skill-hub\nname: skill-2\n---\n%s", updatedContent)
		if extracted != expectedUpdatedContent {
			t.Errorf("Skill not updated properly: got %v, want %v", extracted, expectedUpdatedContent)
		}

		// 验证技能数量不变
		skillListAfterUpdate, err := adapter.List()
		if err != nil {
			t.Errorf("List(after update) error = %v", err)
		}

		if len(skillListAfterUpdate) != 3 {
			t.Errorf("Skill count changed after update: got %d, want 3", len(skillListAfterUpdate))
		}

		// 移除一个技能
		if err := adapter.Remove("skill-1"); err != nil {
			t.Errorf("Remove(skill-1) error = %v", err)
		}

		// 验证剩余技能
		skillListAfterRemove, err := adapter.List()
		if err != nil {
			t.Errorf("List(after remove) error = %v", err)
		}

		if len(skillListAfterRemove) != 2 {
			t.Errorf("Expected 2 skills after remove, got %d", len(skillListAfterRemove))
		}

		// 验证正确的技能被移除
		remainingSkills := map[string]bool{"skill-2": true, "skill-3": true}
		for _, id := range skillListAfterRemove {
			if !remainingSkills[id] {
				t.Errorf("Unexpected skill in list: %s", id)
			}
		}
	})

	t.Run("Supports check", func(t *testing.T) {
		adapter := NewOpenCodeAdapter()

		if !adapter.Supports() {
			t.Error("Supports() should return true for OpenCode adapter")
		}
	})

	t.Run("Directory empty check", func(t *testing.T) {
		// 测试空目录
		emptyDir := filepath.Join(tmpDir, "empty-dir")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		isEmpty, err := isDirectoryEmpty(emptyDir)
		if err != nil {
			t.Errorf("isDirectoryEmpty(empty) error = %v", err)
		}

		if !isEmpty {
			t.Error("Empty directory should be reported as empty")
		}

		// 测试非空目录
		nonEmptyDir := filepath.Join(tmpDir, "non-empty-dir")
		if err := os.MkdirAll(nonEmptyDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		testFile := filepath.Join(nonEmptyDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		isEmpty, err = isDirectoryEmpty(nonEmptyDir)
		if err != nil {
			t.Errorf("isDirectoryEmpty(non-empty) error = %v", err)
		}

		if isEmpty {
			t.Error("Non-empty directory should not be reported as empty")
		}

		// 测试不存在的目录
		_, err = isDirectoryEmpty(filepath.Join(tmpDir, "non-existent"))
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})

	t.Run("Backup and restore", func(t *testing.T) {
		// 测试备份
		testDir := filepath.Join(tmpDir, "test-backup")
		if err := os.MkdirAll(testDir, 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		testFile := filepath.Join(testDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 创建备份
		if err := backupSkill(testDir); err != nil {
			t.Errorf("backupSkill() error = %v", err)
		}

		// 验证备份存在
		backupDir := testDir + ".bak"
		if _, err := os.Stat(backupDir); err != nil {
			t.Errorf("Backup directory not created: %v", err)
		}

		// 验证原始目录被重命名
		if _, err := os.Stat(testDir); err == nil {
			t.Error("Original directory should not exist after backup")
		}

		// 测试恢复备份
		if err := restoreBackup(testDir); err != nil {
			t.Errorf("restoreBackup() error = %v", err)
		}

		// 验证目录恢复
		if _, err := os.Stat(testDir); err != nil {
			t.Errorf("Directory not restored: %v", err)
		}

		// 验证备份被清理
		if _, err := os.Stat(backupDir); err == nil {
			t.Error("Backup directory should be removed after restore")
		}

		// 测试恢复不存在的备份
		if err := restoreBackup(filepath.Join(tmpDir, "no-backup")); err != nil {
			t.Errorf("restoreBackup(no backup) error = %v", err)
		}

		// 测试备份不存在的目录
		if err := backupSkill(filepath.Join(tmpDir, "non-existent")); err != nil {
			t.Errorf("backupSkill(non-existent) error = %v", err)
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
		adapter := NewOpenCodeAdapter().WithProjectMode()

		// 测试无效的技能ID
		invalidSkillID := "invalid/skill/name" // 包含斜杠，不符合命名规范
		if err := adapter.Apply(invalidSkillID, "content", map[string]string{}); err == nil {
			t.Error("Expected error for invalid skill ID")
		}

		// 测试只读目录（在Windows上权限处理不同）
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		// 在Windows上，0555权限可能不够严格，我们尝试设置只读属性
		if runtime.GOOS == "windows" {
			// 在Windows上，我们创建一个文件并设置为只读来模拟错误
			testFile := filepath.Join(readOnlyDir, "test.txt")
			if err := os.WriteFile(testFile, []byte("test"), 0444); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}

		// 模拟当前目录
		oldDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get current directory: %v", err)
		}
		defer os.Chdir(oldDir)

		if err := os.Chdir(readOnlyDir); err != nil {
			t.Fatalf("Failed to change directory: %v", err)
		}

		// 尝试在只读目录中创建技能（应该失败）
		if err := adapter.Apply("test-skill", "content", map[string]string{}); err == nil {
			t.Error("Expected error when creating skill in read-only directory")
		}
	})
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
