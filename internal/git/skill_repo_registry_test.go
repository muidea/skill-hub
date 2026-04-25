package git

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestSkillRepository_UpdateRegistryBuildsFromLocalRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)
	defer config.ResetForTest()

	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `multi_repo:
  enabled: true
  default_repo: "main"
  repositories:
    main:
      name: "main"
      url: ""
      branch: "main"
      enabled: true
      type: "user"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	repoDir := filepath.Join(tmpDir, "repositories", "main")
	skillDir := filepath.Join(repoDir, "skills", "indexed-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}

	skillContent := `---
name: Indexed Skill
description: Skill from local repo
metadata:
  version: "1.2.3"
  author: tester
---
# Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	sr := &SkillRepository{repo: &Repository{path: repoDir}}
	if err := sr.UpdateRegistry(); err != nil {
		t.Fatalf("UpdateRegistry() error = %v", err)
	}

	repoRegistryPath := filepath.Join(repoDir, "registry.json")
	data, err := os.ReadFile(repoRegistryPath)
	if err != nil {
		t.Fatalf("read repo registry: %v", err)
	}

	var registry spec.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("unmarshal registry: %v", err)
	}

	if registry.Version != "1.0.0" {
		t.Fatalf("registry version = %q, want 1.0.0", registry.Version)
	}
	if len(registry.Skills) != 1 {
		t.Fatalf("registry skills = %d, want 1", len(registry.Skills))
	}
	if registry.Skills[0].ID != "indexed-skill" {
		t.Fatalf("registry skill id = %q, want indexed-skill", registry.Skills[0].ID)
	}
}

func TestHasRepositoryChanges(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "clean status text",
			status: "技能仓库状态:\n远程仓库: origin\n最新提交: abc123: msg\n工作区干净\n",
			want:   false,
		},
		{
			name:   "unstaged tracked file",
			status: " M skills/demo/SKILL.md\n",
			want:   true,
		},
		{
			name:   "staged tracked file",
			status: "M  skills/demo/SKILL.md\n",
			want:   true,
		},
		{
			name:   "mixed staged and unstaged file",
			status: "MM registry.json\n",
			want:   true,
		},
		{
			name:   "untracked file",
			status: "?? skills/demo/new.md\n",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasRepositoryChanges(tt.status); got != tt.want {
				t.Fatalf("hasRepositoryChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuggestedCommitMessage(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "single skill",
			files: []string{"skills/demo-skill/SKILL.md", "registry.json"},
			want:  "更新技能: demo-skill",
		},
		{
			name:  "multiple skills sorted",
			files: []string{"skills/zeta/SKILL.md", "skills/alpha/references/README.md"},
			want:  "更新技能: alpha, zeta",
		},
		{
			name:  "many skills",
			files: []string{"skills/d/SKILL.md", "skills/a/SKILL.md", "skills/c/SKILL.md", "skills/b/SKILL.md"},
			want:  "更新 4 个技能: a, b, c 等",
		},
		{
			name:  "registry only",
			files: []string{"registry.json"},
			want:  "更新技能索引",
		},
		{
			name:  "non skill repository file",
			files: []string{"README.md"},
			want:  "更新技能仓库",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SuggestedCommitMessage(tt.files); got != tt.want {
				t.Fatalf("SuggestedCommitMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSuggestedCommitMessageFromStatus(t *testing.T) {
	status := "技能仓库状态:\n文件状态:\n M  skills/demo/SKILL.md\n?? skills/other/references/a.md\n"
	if got, want := SuggestedCommitMessageFromStatus(status), "更新技能: demo, other"; got != want {
		t.Fatalf("SuggestedCommitMessageFromStatus() = %q, want %q", got, want)
	}
}
