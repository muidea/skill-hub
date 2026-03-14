package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestProjectStatus_InspectSyncedSkill(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")

	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill dir: %v", err)
	}

	skillContent := []byte("---\nname: Demo Skill\nversion: 1.2.3\n---\nHello\n")
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {
					Name:    "main",
					Enabled: true,
				},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:   "demo-skill",
					Version:   "1.2.3",
					Status:    spec.SkillStatusSynced,
					Variables: map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(statePath, payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	summary, err := New().Inspect(projectDir, "")
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}

	if summary.ProjectPath != projectDir {
		t.Fatalf("expected project path %q, got %q", projectDir, summary.ProjectPath)
	}
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 status item, got %d", len(summary.Items))
	}
	if summary.Items[0].Status != spec.SkillStatusSynced {
		t.Fatalf("expected synced status, got %q", summary.Items[0].Status)
	}
	if summary.Items[0].RepoVersion != "1.2.3" {
		t.Fatalf("expected repo version 1.2.3, got %q", summary.Items[0].RepoVersion)
	}
	if !strings.HasSuffix(summary.Items[0].RepoPath, filepath.Join("main", "skills", "demo-skill", "SKILL.md")) {
		t.Fatalf("unexpected repo path %q", summary.Items[0].RepoPath)
	}
}
