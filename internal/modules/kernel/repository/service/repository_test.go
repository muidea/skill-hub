package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestSetDefaultRepositoryRefreshesRootRegistry(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	rootDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", rootDir)

	configContent := []byte(`
default_tool: open_code
multi_repo:
  enabled: true
  default_repo: main
  repositories:
    main:
      name: main
      enabled: true
      type: official
    community:
      name: community
      enabled: true
      type: community
`)

	configPath := filepath.Join(rootDir, "config.yaml")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}

	mainSkillDir := filepath.Join(rootDir, "repositories", "main", "skills", "main-skill")
	if err := os.MkdirAll(mainSkillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(mainSkillDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainSkillDir, "SKILL.md"), []byte(`---
name: Main Skill
description: main repo skill
version: 1.0.0
compatibility: open_code
---
`), 0644); err != nil {
		t.Fatalf("WriteFile(main skill) error = %v", err)
	}

	communitySkillDir := filepath.Join(rootDir, "repositories", "community", "skills", "community-skill")
	if err := os.MkdirAll(communitySkillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(communitySkillDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(communitySkillDir, "SKILL.md"), []byte(`---
name: Community Skill
description: community repo skill
version: 2.0.0
compatibility: cursor
---
`), 0644); err != nil {
		t.Fatalf("WriteFile(community skill) error = %v", err)
	}

	repoSvc := New()
	if err := repoSvc.RebuildRepositoryIndex("main"); err != nil {
		t.Fatalf("RebuildRepositoryIndex(main) error = %v", err)
	}

	if err := repoSvc.SetDefaultRepository("community"); err != nil {
		t.Fatalf("SetDefaultRepository(community) error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(rootDir, "registry.json"))
	if err != nil {
		t.Fatalf("ReadFile(registry.json) error = %v", err)
	}

	var registry spec.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("Unmarshal(registry.json) error = %v", err)
	}

	if len(registry.Skills) != 1 {
		t.Fatalf("registry skill count = %d, want 1", len(registry.Skills))
	}
	if registry.Skills[0].ID != "community-skill" {
		t.Fatalf("registry skill id = %q, want %q", registry.Skills[0].ID, "community-skill")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if cfg.MultiRepo.DefaultRepo != "community" {
		t.Fatalf("default repo = %q, want %q", cfg.MultiRepo.DefaultRepo, "community")
	}
}
