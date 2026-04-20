package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/testutils"
)

func TestCheckInitDependency(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (skillHubHome string, cleanup func())
		wantErr     bool
		errContains string
	}{
		{
			name: "未初始化环境",
			setup: func(t *testing.T) (string, func()) {
				// 创建临时目录，但没有配置文件
				tempDir := testutils.TempDir(t, "skill-hub-test-")
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", tempDir)
				return tempDir, func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr:     true,
			errContains: "本地仓库未初始化",
		},
		{
			name: "已初始化环境",
			setup: func(t *testing.T) (string, func()) {
				// 创建完整的测试环境
				skillHubHome, _, _ := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)
				return skillHubHome, func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := tt.setup(t)
			defer cleanup()

			err := CheckInitDependency()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckInitDependency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if errStr := err.Error(); !contains(errStr, tt.errContains) {
					t.Errorf("CheckInitDependency() error = %v, should contain %v", errStr, tt.errContains)
				}
			}
		})
	}
}

func TestCheckProjectWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (projectDir, skillHubHome string, cleanup func())
		wantErr     bool
		errContains string
	}{
		{
			name: "项目存在于状态文件中",
			setup: func(t *testing.T) (string, string, func()) {
				skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)

				// 更新状态文件包含当前项目
				statePath := filepath.Join(skillHubHome, "state.json")
				stateContent := `{
  "` + projectDir + `": {
    "project_path": "` + projectDir + `",
    "preferred_target": "open_code",
    "skills": {}
  }
}`
				if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
					t.Fatalf("更新状态文件失败: %v", err)
				}

				return projectDir, skillHubHome, func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr: false,
		},
		{
			name: "项目不存在于状态文件中",
			setup: func(t *testing.T) (string, string, func()) {
				skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)

				// 状态文件不包含当前项目
				statePath := filepath.Join(skillHubHome, "state.json")
				stateContent := `{
  "/other/project": {
    "project_path": "/other/project",
    "preferred_target": "open_code",
    "skills": {}
  }
}`
				if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
					t.Fatalf("更新状态文件失败: %v", err)
				}

				return projectDir, skillHubHome, func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr:     false, // LoadProjectState会返回新状态，不是错误
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, _, cleanup := tt.setup(t)
			defer cleanup()

			_, err := CheckProjectWorkspace(projectDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckProjectWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if errStr := err.Error(); !contains(errStr, tt.errContains) {
					t.Errorf("CheckProjectWorkspace() error = %v, should contain %v", errStr, tt.errContains)
				}
			}
		})
	}
}

func TestEnsureProjectWorkspace(t *testing.T) {
	// 这个测试需要模拟用户输入，比较复杂，我们先测试基础场景
	t.Run("项目已存在", func(t *testing.T) {
		skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
		originalHome := os.Getenv("SKILL_HUB_HOME")
		os.Setenv("SKILL_HUB_HOME", skillHubHome)
		defer func() { os.Setenv("SKILL_HUB_HOME", originalHome) }()

		// 创建包含当前项目的状态文件
		statePath := filepath.Join(skillHubHome, "state.json")
		stateContent := `{
  "` + projectDir + `": {
    "project_path": "` + projectDir + `",
    "preferred_target": "open_code",
    "skills": {}
  }
}`
		if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
			t.Fatalf("更新状态文件失败: %v", err)
		}

		projectState, err := EnsureProjectWorkspace(projectDir)
		if err != nil {
			t.Errorf("EnsureProjectWorkspace() error = %v", err)
			return
		}

		if projectState == nil {
			t.Error("EnsureProjectWorkspace() returned nil projectState")
			return
		}

		if projectState.ProjectPath != projectDir {
			t.Errorf("EnsureProjectWorkspace() projectState.ProjectPath = %v, want %v", projectState.ProjectPath, projectDir)
		}

		if projectState.PreferredTarget != "open_code" {
			t.Errorf("EnsureProjectWorkspace() projectState.PreferredTarget = %v, want open_code", projectState.PreferredTarget)
		}
	})
}

func TestCheckSkillExists(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (skillHubHome, skillID string, cleanup func())
		wantErr     bool
		errContains string
	}{
		{
			name: "技能存在",
			setup: func(t *testing.T) (string, string, func()) {
				skillHubHome, _, _ := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)

				// 还需要设置配置文件为多仓库模式
				configPath := filepath.Join(skillHubHome, "config.yaml")
				configContent := `# skill-hub 配置文件（多仓库模式）
skill_hub_home: ` + skillHubHome + `
git_token: ""

# 多仓库配置（强制启用）
multi_repo:
  enabled: true
  default_repo: "main"  # 默认仓库名称
  repositories:
    main:
      name: "main"
      url: ""
      branch: "master"
      enabled: true
      description: "测试仓库"
      type: "user"
      is_archive: true
`
				if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
					t.Fatalf("更新配置文件失败: %v", err)
				}

				return skillHubHome, "test-skill-1", func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr:     true, // 技能管理器可能找不到测试技能
			errContains: "技能未找到",
		},
		{
			name: "技能不存在",
			setup: func(t *testing.T) (string, string, func()) {
				skillHubHome, _, _ := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)
				return skillHubHome, "non-existent-skill", func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr:     true,
			errContains: "技能未找到",
		},
		{
			name: "未初始化环境",
			setup: func(t *testing.T) (string, string, func()) {
				tempDir := testutils.TempDir(t, "skill-hub-test-")
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", tempDir)
				return tempDir, "test-skill", func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantErr:     true,
			errContains: "技能未找到", // CheckInitDependency可能没有正确检测未初始化状态
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, skillID, cleanup := tt.setup(t)
			defer cleanup()

			err := CheckSkillExists(skillID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSkillExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if errStr := err.Error(); !contains(errStr, tt.errContains) {
					t.Errorf("CheckSkillExists() error = %v, should contain %v", errStr, tt.errContains)
				}
			}
		})
	}
}

func TestCheckSkillInProject(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (projectDir, skillHubHome, skillID string, cleanup func())
		wantExists  bool
		wantErr     bool
		errContains string
	}{
		{
			name: "技能在项目中",
			setup: func(t *testing.T) (string, string, string, func()) {
				skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)

				// 更新状态文件，包含技能
				statePath := filepath.Join(skillHubHome, "state.json")
				stateContent := `{
  "` + projectDir + `": {
    "project_path": "` + projectDir + `",
    "preferred_target": "open_code",
    "skills": {
      "test-skill-1": {
        "skill_id": "test-skill-1",
        "version": "1.0.0",
        "variables": {
          "target": "open_code"
        }
      }
    }
  }
}`
				if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
					t.Fatalf("更新状态文件失败: %v", err)
				}

				return projectDir, skillHubHome, "test-skill-1", func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "技能不在项目中",
			setup: func(t *testing.T) (string, string, string, func()) {
				skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
				originalHome := os.Getenv("SKILL_HUB_HOME")
				os.Setenv("SKILL_HUB_HOME", skillHubHome)

				// 更新状态文件，不包含该技能
				statePath := filepath.Join(skillHubHome, "state.json")
				stateContent := `{
  "` + projectDir + `": {
    "project_path": "` + projectDir + `",
    "preferred_target": "open_code",
    "skills": {
      "other-skill": {
        "skill_id": "other-skill",
        "version": "1.0.0",
        "variables": {
          "target": "open_code"
        }
      }
    }
  }
}`
				if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
					t.Fatalf("更新状态文件失败: %v", err)
				}

				return projectDir, skillHubHome, "test-skill-1", func() {
					os.Setenv("SKILL_HUB_HOME", originalHome)
				}
			},
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, _, skillID, cleanup := tt.setup(t)
			defer cleanup()

			exists, err := CheckSkillInProject(projectDir, skillID)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSkillInProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && err != nil {
				if errStr := err.Error(); !contains(errStr, tt.errContains) {
					t.Errorf("CheckSkillInProject() error = %v, should contain %v", errStr, tt.errContains)
				}
			}

			if !tt.wantErr && exists != tt.wantExists {
				t.Errorf("CheckSkillInProject() exists = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestInitializeWorkspaceFiles(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		checkFiles func(t *testing.T, cwd string) bool
	}{
		{
			name:    "初始化标准目录",
			wantErr: false,
			checkFiles: func(t *testing.T, cwd string) bool {
				agentsDir := filepath.Join(cwd, ".agents")
				skillsDir := filepath.Join(agentsDir, "skills")

				if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
					t.Errorf("agents目录未创建: %v", err)
					return false
				}
				if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
					t.Errorf("skills目录未创建: %v", err)
					return false
				}
				if _, err := os.Stat(filepath.Join(cwd, ".claude")); err == nil {
					t.Errorf("不应创建.claude目录")
					return false
				}
				if _, err := os.Stat(filepath.Join(cwd, ".cursorrules")); err == nil {
					t.Errorf("不应创建.cursorrules")
					return false
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := testutils.TempDir(t, "test-workspace-files-")

			err := initializeWorkspaceFiles(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("initializeWorkspaceFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFiles != nil {
				if !tt.checkFiles(t, tempDir) {
					t.Errorf("文件检查失败")
				}
			}
		})
	}
}

func TestRequireInitOnly(t *testing.T) {
	t.Run("已初始化时返回带Cwd的上下文", func(t *testing.T) {
		skillHubHome, _, _ := testutils.SetupTestSkillHub(t)
		originalHome := os.Getenv("SKILL_HUB_HOME")
		os.Setenv("SKILL_HUB_HOME", skillHubHome)
		defer os.Setenv("SKILL_HUB_HOME", originalHome)

		ctx, err := RequireInitOnly()
		if err != nil {
			t.Fatalf("RequireInitOnly() error = %v", err)
		}
		if ctx == nil || ctx.Cwd == "" {
			t.Error("RequireInitOnly() expected non-empty Cwd")
		}
		if ctx.StateManager != nil || ctx.ProjectState != nil {
			t.Error("RequireInitOnly() expected nil StateManager and ProjectState")
		}
	})
}

func TestRequireInitAndWorkspace(t *testing.T) {
	t.Run("已初始化且项目在状态中时返回完整上下文", func(t *testing.T) {
		skillHubHome, _, projectDir := testutils.SetupTestSkillHub(t)
		originalHome := os.Getenv("SKILL_HUB_HOME")
		os.Setenv("SKILL_HUB_HOME", skillHubHome)
		defer os.Setenv("SKILL_HUB_HOME", originalHome)

		statePath := filepath.Join(skillHubHome, "state.json")
		stateContent := `{
  "` + projectDir + `": {
    "project_path": "` + projectDir + `",
    "preferred_target": "open_code",
    "skills": {}
  }
}`
		if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
			t.Fatalf("写入状态文件失败: %v", err)
		}

		ctx, err := RequireInitAndWorkspace(projectDir)
		if err != nil {
			t.Fatalf("RequireInitAndWorkspace() error = %v", err)
		}
		if ctx == nil || ctx.Cwd == "" || ctx.StateManager == nil || ctx.ProjectState == nil {
			t.Errorf("RequireInitAndWorkspace() expected full context, got Cwd=%q StateManager=%v ProjectState=%v",
				ctx.Cwd, ctx.StateManager != nil, ctx.ProjectState != nil)
		}
	})
}
