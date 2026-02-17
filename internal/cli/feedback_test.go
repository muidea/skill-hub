package cli

import (
	"os"
	"path/filepath"
	"testing"

	"skill-hub/internal/testutils"
)

func TestCompareSkillDirectories(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (projectDir, repoDir string, repoExists bool, cleanup func())
		expected []string
		wantErr  bool
	}{
		{
			name: "新增文件",
			setup: func(t *testing.T) (string, string, bool, func()) {
				projectDir := testutils.TempDir(t, "test-project-")
				repoDir := testutils.TempDir(t, "test-repo-")

				// 项目目录有文件，仓库目录没有
				testutils.CreateTestFile(t, projectDir, "SKILL.md", "Test content")
				testutils.CreateTestFile(t, projectDir, "README.md", "Readme content")

				return projectDir, repoDir, false, func() {}
			},
			expected: []string{"新增: README.md", "新增: SKILL.md"},
			wantErr:  false,
		},
		{
			name: "修改文件",
			setup: func(t *testing.T) (string, string, bool, func()) {
				projectDir := testutils.TempDir(t, "test-project-")
				repoDir := testutils.TempDir(t, "test-repo-")

				// 两个目录都有文件，但内容不同
				testutils.CreateTestFile(t, projectDir, "SKILL.md", "Project content")
				testutils.CreateTestFile(t, repoDir, "SKILL.md", "Repo content")

				return projectDir, repoDir, true, func() {}
			},
			expected: []string{"修改: SKILL.md"},
			wantErr:  false,
		},
		{
			name: "删除文件",
			setup: func(t *testing.T) (string, string, bool, func()) {
				projectDir := testutils.TempDir(t, "test-project-")
				repoDir := testutils.TempDir(t, "test-repo-")

				// 仓库目录有文件，项目目录没有
				testutils.CreateTestFile(t, repoDir, "old-file.md", "Old content")

				return projectDir, repoDir, true, func() {}
			},
			expected: []string{"删除: old-file.md"},
			wantErr:  false,
		},
		{
			name: "相同文件",
			setup: func(t *testing.T) (string, string, bool, func()) {
				projectDir := testutils.TempDir(t, "test-project-")
				repoDir := testutils.TempDir(t, "test-repo-")

				// 两个目录有相同的文件
				testutils.CreateTestFile(t, projectDir, "SKILL.md", "Same content")
				testutils.CreateTestFile(t, repoDir, "SKILL.md", "Same content")

				return projectDir, repoDir, true, func() {}
			},
			expected: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, repoDir, repoExists, cleanup := tt.setup(t)
			defer cleanup()

			changes, err := compareSkillDirectories(projectDir, repoDir, repoExists)
			if (err != nil) != tt.wantErr {
				t.Errorf("compareSkillDirectories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查变化数量
				if len(changes) != len(tt.expected) {
					t.Errorf("compareSkillDirectories() changes count = %v, want %v. Changes: %v", len(changes), len(tt.expected), changes)
					return
				}

				// 检查具体变化
				for i, change := range changes {
					if i >= len(tt.expected) {
						t.Errorf("compareSkillDirectories() unexpected change: %v", change)
						continue
					}
					if change != tt.expected[i] {
						t.Errorf("compareSkillDirectories() change[%d] = %v, want %v", i, change, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestCopySkillDirectory(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (srcDir, dstDir string, cleanup func())
		verify  func(t *testing.T, srcDir, dstDir string) bool
		wantErr bool
	}{
		{
			name: "复制整个目录",
			setup: func(t *testing.T) (string, string, func()) {
				srcDir := testutils.TempDir(t, "test-src-")
				dstDir := testutils.TempDir(t, "test-dst-")

				// 创建源目录结构
				testutils.CreateTestFile(t, srcDir, "SKILL.md", "Skill content")
				testutils.CreateTestFile(t, srcDir, "README.md", "Readme content")
				subDir := filepath.Join(srcDir, "scripts")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("创建子目录失败: %v", err)
				}
				testutils.CreateTestFile(t, subDir, "setup.sh", "#!/bin/bash\necho setup")

				return srcDir, dstDir, func() {}
			},
			verify: func(t *testing.T, srcDir, dstDir string) bool {
				// 检查文件是否复制
				files := []string{
					"SKILL.md",
					"README.md",
					"scripts/setup.sh",
				}

				for _, file := range files {
					srcPath := filepath.Join(srcDir, file)
					dstPath := filepath.Join(dstDir, file)

					// 检查目标文件是否存在
					if _, err := os.Stat(dstPath); os.IsNotExist(err) {
						t.Errorf("文件未复制: %s", file)
						return false
					}

					// 检查内容是否相同
					srcContent, err1 := os.ReadFile(srcPath)
					dstContent, err2 := os.ReadFile(dstPath)

					if err1 != nil || err2 != nil {
						t.Errorf("读取文件失败: src=%v, dst=%v", err1, err2)
						return false
					}

					if string(srcContent) != string(dstContent) {
						t.Errorf("文件内容不同: %s", file)
						return false
					}
				}
				return true
			},
			wantErr: false,
		},
		{
			name: "同步删除操作",
			setup: func(t *testing.T) (string, string, func()) {
				srcDir := testutils.TempDir(t, "test-src-")
				dstDir := testutils.TempDir(t, "test-dst-")

				// 目标目录有额外文件
				testutils.CreateTestFile(t, dstDir, "old-file.md", "Old content")
				testutils.CreateTestFile(t, dstDir, "SKILL.md", "Old skill")

				// 源目录只有SKILL.md
				testutils.CreateTestFile(t, srcDir, "SKILL.md", "New skill content")

				return srcDir, dstDir, func() {}
			},
			verify: func(t *testing.T, srcDir, dstDir string) bool {
				// 检查旧文件是否被删除
				oldFilePath := filepath.Join(dstDir, "old-file.md")
				if _, err := os.Stat(oldFilePath); !os.IsNotExist(err) {
					t.Errorf("旧文件未被删除: %s", oldFilePath)
					return false
				}

				// 检查SKILL.md是否更新
				skillPath := filepath.Join(dstDir, "SKILL.md")
				content, err := os.ReadFile(skillPath)
				if err != nil {
					t.Errorf("读取SKILL.md失败: %v", err)
					return false
				}

				if string(content) != "New skill content" {
					t.Errorf("SKILL.md内容未更新: %s", content)
					return false
				}

				return true
			},
			wantErr: false,
		},
		{
			name: "空源目录",
			setup: func(t *testing.T) (string, string, func()) {
				srcDir := testutils.TempDir(t, "test-src-")
				dstDir := testutils.TempDir(t, "test-dst-")

				// 目标目录有文件
				testutils.CreateTestFile(t, dstDir, "file1.md", "Content 1")
				testutils.CreateTestFile(t, dstDir, "file2.md", "Content 2")

				return srcDir, dstDir, func() {}
			},
			verify: func(t *testing.T, srcDir, dstDir string) bool {
				// 检查目标目录是否为空
				entries, err := os.ReadDir(dstDir)
				if err != nil {
					t.Errorf("读取目标目录失败: %v", err)
					return false
				}

				// 应该删除所有文件
				if len(entries) > 0 {
					t.Errorf("目标目录应该为空，但有 %d 个文件", len(entries))
					return false
				}

				return true
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDir, dstDir, cleanup := tt.setup(t)
			defer cleanup()

			err := copySkillDirectory(srcDir, dstDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("copySkillDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				if !tt.verify(t, srcDir, dstDir) {
					t.Errorf("copySkillDirectory() 验证失败")
				}
			}
		})
	}
}
