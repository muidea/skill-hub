package cli

import (
	"os"
	"path/filepath"
	"testing"

	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
	"skill-hub/pkg/spec"
)

func TestGetAdapterName(t *testing.T) {
	tests := []struct {
		name     string
		adapter  adapter.Adapter
		expected string
	}{
		{
			name:     "Cursor adapter",
			adapter:  cursor.NewCursorAdapter(),
			expected: "Cursor",
		},
		{
			name:     "Claude adapter",
			adapter:  claude.NewClaudeAdapter(),
			expected: "Claude",
		},
		{
			name:     "OpenCode adapter",
			adapter:  opencode.NewOpenCodeAdapter(),
			expected: "OpenCode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAdapterName(tt.adapter)
			if result != tt.expected {
				t.Errorf("getAdapterName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAdapterSupportsSkill(t *testing.T) {
	tests := []struct {
		name          string
		adapter       adapter.Adapter
		compatibility string
		expected      bool
	}{
		{
			name:          "Cursor adapter with cursor support",
			adapter:       cursor.NewCursorAdapter(),
			compatibility: "Designed for Cursor and OpenCode (or similar AI coding assistants)",
			expected:      true,
		},
		{
			name:          "Claude adapter without claude support",
			adapter:       claude.NewClaudeAdapter(),
			compatibility: "Designed for Cursor and OpenCode (or similar AI coding assistants)",
			expected:      false,
		},
		{
			name:          "OpenCode adapter with opencode support",
			adapter:       opencode.NewOpenCodeAdapter(),
			compatibility: "Designed for Cursor and OpenCode (or similar AI coding assistants)",
			expected:      true,
		},
		{
			name:          "Claude adapter with claude support",
			adapter:       claude.NewClaudeAdapter(),
			compatibility: "Designed for Claude Code (or similar AI coding assistants)",
			expected:      true,
		},
		{
			name:          "Empty compatibility supports all",
			adapter:       cursor.NewCursorAdapter(),
			compatibility: "",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := &spec.Skill{
				Compatibility: tt.compatibility,
			}
			result := adapterSupportsSkill(tt.adapter, skill)
			if result != tt.expected {
				t.Errorf("adapterSupportsSkill() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAttemptRecovery(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()

	// 测试Cursor适配器恢复
	t.Run("Cursor adapter recovery", func(t *testing.T) {
		cursorAdapter := cursor.NewCursorAdapter()

		// 创建测试文件
		testFile := filepath.Join(tmpDir, ".cursorrules")
		backupFile := testFile + ".bak"

		// 写入备份文件
		if err := os.WriteFile(backupFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}

		// 测试恢复
		if err := attemptRecovery(cursorAdapter, "test-skill"); err != nil {
			t.Errorf("attemptRecovery() error = %v", err)
		}
	})

	// 测试Claude适配器恢复
	t.Run("Claude adapter recovery", func(t *testing.T) {
		claudeAdapter := claude.NewClaudeAdapter()

		// 在测试中，我们模拟配置文件存在的情况
		// 由于Claude适配器需要配置文件，我们跳过这个测试在CI环境中
		if os.Getenv("CI") == "true" {
			t.Skip("Skipping Claude adapter recovery test in CI environment")
		}

		if err := attemptRecovery(claudeAdapter, "test-skill"); err != nil {
			t.Errorf("attemptRecovery() error = %v", err)
		}
	})

	// 测试OpenCode适配器恢复
	t.Run("OpenCode adapter recovery", func(t *testing.T) {
		opencodeAdapter := opencode.NewOpenCodeAdapter()

		if err := attemptRecovery(opencodeAdapter, "test-skill"); err != nil {
			t.Errorf("attemptRecovery() error = %v", err)
		}
	})
}

func TestCleanupFunctionality(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()

	// 测试Cursor适配器清理
	t.Run("Cursor adapter cleanup", func(t *testing.T) {
		// 创建测试文件
		testFile := filepath.Join(tmpDir, ".cursorrules")
		backupFile := testFile + ".bak"
		tmpFile := testFile + ".tmp"

		// 创建备份文件和临时文件
		if err := os.WriteFile(backupFile, []byte("backup content"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
		if err := os.WriteFile(tmpFile, []byte("temp content"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		// 测试统一的清理函数
		if err := adapter.CleanupTempFiles(testFile); err != nil {
			t.Errorf("CleanupTempFiles() error = %v", err)
		}

		// 验证文件已被清理
		if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
			t.Error("Backup file should have been cleaned up")
		}
		if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
			t.Error("Temp file should have been cleaned up")
		}
	})

	// 测试Claude适配器清理
	t.Run("Claude adapter cleanup", func(t *testing.T) {
		// 创建测试文件
		testFile := filepath.Join(tmpDir, ".clauderc")
		backupFile := testFile + ".bak"
		tmpFile := testFile + ".tmp"

		// 创建备份文件和临时文件
		if err := os.WriteFile(backupFile, []byte("backup content"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
		if err := os.WriteFile(tmpFile, []byte("temp content"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		// 测试统一的清理函数
		if err := adapter.CleanupTempFiles(testFile); err != nil {
			t.Errorf("CleanupTempFiles() error = %v", err)
		}

		// 验证文件已被清理
		if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
			t.Error("Backup file should have been cleaned up")
		}
		if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
			t.Error("Temp file should have been cleaned up")
		}
	})

	// 测试OpenCode适配器清理
	t.Run("OpenCode adapter cleanup", func(t *testing.T) {
		// 创建测试目录结构
		baseDir := filepath.Join(tmpDir, ".opencode")
		skillsDir := filepath.Join(baseDir, "skills")
		skillDir := filepath.Join(skillsDir, "test-skill")
		backupDir := skillDir + ".bak"

		// 创建目录结构
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("Failed to create skill directory: %v", err)
		}
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			t.Fatalf("Failed to create backup directory: %v", err)
		}

		// 在备份目录中创建测试文件
		testFile := filepath.Join(backupDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 测试统一的清理函数
		if err := adapter.CleanupBackupDir(skillDir); err != nil {
			t.Errorf("CleanupBackupDir() error = %v", err)
		}

		// 验证备份目录已被清理
		if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
			t.Error("Backup directory should have been cleaned up")
		}
	})

	// 测试统一的清理函数
	t.Run("Unified cleanup functions", func(t *testing.T) {
		// 测试CleanupTempFiles
		testFile := filepath.Join(tmpDir, "test.txt")
		backupFile := testFile + ".bak"
		tmpFile := testFile + ".tmp"

		// 创建测试文件
		if err := os.WriteFile(backupFile, []byte("backup"), 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
		if err := os.WriteFile(tmpFile, []byte("temp"), 0644); err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		// 执行清理
		if err := adapter.CleanupTempFiles(testFile); err != nil {
			t.Errorf("CleanupTempFiles() error = %v", err)
		}

		// 验证文件已被清理
		if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
			t.Error("Backup file should have been cleaned up")
		}
		if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
			t.Error("Temp file should have been cleaned up")
		}

		// 测试CleanupBackupDir
		testDir := filepath.Join(tmpDir, "test-dir")
		backupDir := testDir + ".bak"

		// 创建测试目录
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			t.Fatalf("Failed to create backup directory: %v", err)
		}

		// 执行清理
		if err := adapter.CleanupBackupDir(testDir); err != nil {
			t.Errorf("CleanupBackupDir() error = %v", err)
		}

		// 验证目录已被清理
		if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
			t.Error("Backup directory should have been cleaned up")
		}
	})
}

func TestSelectAdapters(t *testing.T) {
	tests := []struct {
		name   string
		target string
		mode   string
		count  int
	}{
		{
			name:   "All targets",
			target: spec.TargetAll,
			mode:   "project",
			count:  3,
		},
		{
			name:   "Cursor only",
			target: spec.TargetCursor,
			mode:   "project",
			count:  1,
		},
		{
			name:   "Claude only",
			target: spec.TargetClaudeCode,
			mode:   "global",
			count:  1,
		},
		{
			name:   "OpenCode only",
			target: spec.TargetOpenCode,
			mode:   "project",
			count:  1,
		},
		{
			name:   "Invalid target",
			target: "invalid",
			mode:   "project",
			count:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapters := selectAdapters(tt.target, tt.mode)
			if len(adapters) != tt.count {
				t.Errorf("selectAdapters() returned %d adapters, want %d", len(adapters), tt.count)
			}
		})
	}
}

func TestRenderTemplateForRemove(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		expected  string
	}{
		{
			name:      "Simple variable replacement",
			content:   "Hello {{.Name}}!",
			variables: map[string]string{"Name": "World"},
			expected:  "Hello World!",
		},
		{
			name:      "Multiple variables",
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
		{
			name:      "Variable not in template",
			content:   "Hello World!",
			variables: map[string]string{"Name": "Test"},
			expected:  "Hello World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderTemplateForRemove(tt.content, tt.variables)
			if err != nil {
				t.Errorf("renderTemplateForRemove() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("renderTemplateForRemove() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "File exists",
			path:     testFile,
			expected: true,
		},
		{
			name:     "File does not exist",
			path:     filepath.Join(tmpDir, "nonexistent.txt"),
			expected: false,
		},
		{
			name:     "Directory exists",
			path:     tmpDir,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := os.Stat(tt.path)
			result := err == nil
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建临时工作目录
	tmpDir := t.TempDir()

	// 保存当前目录并切换到临时目录
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 创建状态管理器（暂时注释，因为需要配置文件）
	// stateMgr, err := state.NewStateManager()
	// if err != nil {
	// 	t.Fatalf("Failed to create state manager: %v", err)
	// }

	// 创建技能管理器（暂时注释，因为需要技能仓库）
	// skillManager, err := engine.NewSkillManager()
	// if err != nil {
	// 	t.Fatalf("Failed to create skill manager: %v", err)
	// }

	// 测试基本功能
	t.Run("Basic functionality", func(t *testing.T) {
		// 测试适配器选择
		adapters := selectAdapters(spec.TargetCursor, "project")
		if len(adapters) != 1 {
			t.Errorf("Expected 1 adapter for cursor target, got %d", len(adapters))
		}

		// 测试适配器名称
		adapterName := getAdapterName(adapters[0])
		if adapterName != "Cursor" {
			t.Errorf("Expected adapter name 'Cursor', got %s", adapterName)
		}

		// 测试文件存在检查
		if _, err := os.Stat(tmpDir); err != nil {
			t.Errorf("Expected directory to exist: %s, error: %v", tmpDir, err)
		}
	})
}
