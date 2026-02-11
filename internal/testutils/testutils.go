package testutils

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"skill-hub/pkg/logging"
)

// TempDir 创建临时目录并在测试后清理
func TempDir(t *testing.T, prefix string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// CopyTestData 复制测试数据到临时目录
func CopyTestData(t *testing.T, srcDir, dstDir string) {
	t.Helper()

	err := filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}

		// 目标路径
		dstPath := filepath.Join(dstDir, relPath)

		// 如果是目录，创建目录
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 如果是文件，复制文件
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})

	if err != nil {
		t.Fatalf("复制测试数据失败: %v", err)
	}
}

// CreateTestFile 创建测试文件
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	return filePath
}

// CreateTestSkill 创建测试技能目录结构
func CreateTestSkill(t *testing.T, baseDir, skillID, skillContent string) string {
	t.Helper()

	skillDir := filepath.Join(baseDir, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("创建SKILL.md失败: %v", err)
	}

	return skillDir
}

// CreateTestConfig 创建测试配置文件
func CreateTestConfig(t *testing.T, configDir string, repoPath string) string {
	t.Helper()

	configPath := filepath.Join(configDir, "config.yaml")
	configContent := `repo_path: ` + repoPath + `
skill_hub_home: ` + configDir + `
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	return configPath
}

// CreateTestState 创建测试状态文件
func CreateTestState(t *testing.T, stateDir string, projectPath, target string) string {
	t.Helper()

	statePath := filepath.Join(stateDir, "state.json")
	stateContent := `{
  "` + projectPath + `": {
    "project_path": "` + projectPath + `",
    "preferred_target": "` + target + `",
    "skills": {}
  }
}`

	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatalf("创建状态文件失败: %v", err)
	}

	return statePath
}

// CreateTestRegistry 创建测试注册表文件
func CreateTestRegistry(t *testing.T, registryDir string, skills []map[string]string) string {
	t.Helper()

	registryPath := filepath.Join(registryDir, "registry.json")

	// 构建技能数组
	skillsJSON := ""
	for i, skill := range skills {
		if i > 0 {
			skillsJSON += ",\n"
		}
		skillsJSON += `    {
      "id": "` + skill["id"] + `",
      "name": "` + skill["name"] + `",
      "version": "` + skill["version"] + `",
      "author": "` + skill["author"] + `",
      "description": "` + skill["description"] + `",
      "tags": null
    }`
	}

	registryContent := `{
  "version": "1.0.0",
  "skills": [
` + skillsJSON + `
  ]
}`

	if err := os.WriteFile(registryPath, []byte(registryContent), 0644); err != nil {
		t.Fatalf("创建注册表文件失败: %v", err)
	}

	return registryPath
}

// DiscardLogger 返回一个丢弃所有输出的logger，用于测试
func DiscardLogger() interface{} {
	// 使用logging包中的DiscardLogger
	return logging.DiscardLogger()
}

// ChangeToTempDir 切换到临时目录并返回原目录
func ChangeToTempDir(t *testing.T) (originalDir string) {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前目录失败: %v", err)
	}

	tempDir := TempDir(t, "test-chdir-")
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换到临时目录失败: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("切换回原目录失败: %v", err)
		}
	})

	return originalDir
}

// SetupTestSkillHub 设置完整的测试skill-hub环境
func SetupTestSkillHub(t *testing.T) (skillHubHome, repoDir, projectDir string) {
	t.Helper()

	// 创建skill-hub主目录
	skillHubHome = TempDir(t, "skill-hub-test-")

	// 创建仓库目录
	repoDir = filepath.Join(skillHubHome, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("创建仓库目录失败: %v", err)
	}

	// 创建技能目录
	skillsDir := filepath.Join(repoDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("创建技能目录失败: %v", err)
	}

	// 创建配置文件
	CreateTestConfig(t, skillHubHome, repoDir)

	// 创建状态文件
	CreateTestState(t, skillHubHome, "/tmp/test-project", "open_code")

	// 创建注册表文件
	CreateTestRegistry(t, skillHubHome, []map[string]string{
		{
			"id":          "test-skill-1",
			"name":        "test-skill-1",
			"version":     "1.0.0",
			"author":      "test",
			"description": "Test skill 1",
		},
		{
			"id":          "test-skill-2",
			"name":        "test-skill-2",
			"version":     "1.0.0",
			"author":      "test",
			"description": "Test skill 2",
		},
	})

	// 创建测试技能
	CreateTestSkill(t, skillsDir, "test-skill-1", `---
name: test-skill-1
description: Test skill 1
version: 1.0.0
compatibility: open_code
---
# Test Skill 1

This is a test skill.`)

	CreateTestSkill(t, skillsDir, "test-skill-2", `---
name: test-skill-2
description: Test skill 2
version: 1.0.0
compatibility: open_code
---
# Test Skill 2

This is another test skill.`)

	// 创建项目目录
	projectDir = TempDir(t, "test-project-")

	return skillHubHome, repoDir, projectDir
}
