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
