package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestProjectUse_EnableSkill(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {Name: "main", Enabled: true},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n"), 0644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	result, err := New().EnableSkill(projectDir, "demo-skill", "main", spec.TargetOpenCode, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if result.SkillID != "demo-skill" || result.Repository != "main" {
		t.Fatalf("unexpected result: %+v", result)
	}

	statePayload, err := os.ReadFile(filepath.Join(homeDir, "state.json"))
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}

	var states map[string]spec.ProjectState
	if err := json.Unmarshal(statePayload, &states); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	projectState, ok := states[result.ProjectPath]
	if !ok {
		t.Fatalf("expected project state for %q", result.ProjectPath)
	}
	if projectState.Skills["demo-skill"].Version != "1.0.0" {
		t.Fatalf("expected stored version 1.0.0, got %q", projectState.Skills["demo-skill"].Version)
	}
	if projectState.Skills["demo-skill"].Variables["env"] != "test" {
		t.Fatalf("expected stored variable env=test, got %+v", projectState.Skills["demo-skill"].Variables)
	}
}
