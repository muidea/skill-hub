package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/internal/config"
)

func TestCompleter_WithEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)
	defer config.ResetForTest()
	resetCompletionCache()

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
    community:
      name: "community"
      url: ""
      branch: "main"
      enabled: true
      type: "community"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	reposDir := filepath.Join(tmpDir, "repositories", "main", "skills", "go-refactor")
	if err := os.MkdirAll(reposDir, 0755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	skillContent := `---
name: go-refactor
description: Go refactoring
metadata:
  version: "1.0.0"
---
# Skill
`
	if err := os.WriteFile(filepath.Join(reposDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	absTmp, _ := filepath.Abs(tmpDir)
	statePath := filepath.Join(tmpDir, "state.json")
	stateContent := `{"` + absTmp + `":{"project_path":"` + absTmp + `","skills":{"go-refactor":{"skill_id":"go-refactor","version":"1.0.0","variables":{}}}}}
`
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	origWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	gotRepos, dir := completeRepoNames(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("completeRepoNames directive = %v", dir)
	}
	if len(gotRepos) != 2 {
		t.Errorf("completeRepoNames: got %v, want 2 repo names", gotRepos)
	}

	gotSkills, dir := completeEnabledSkillIDsForCwd(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("completeEnabledSkillIDsForCwd directive = %v", dir)
	}
	if len(gotSkills) != 1 || gotSkills[0] != "go-refactor" {
		t.Errorf("completeEnabledSkillIDsForCwd: got %v, want [go-refactor]", gotSkills)
	}

	gotUse, dir := completeSkillIDs(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("completeSkillIDs directive = %v", dir)
	}
	if len(gotUse) < 1 {
		t.Errorf("completeSkillIDs: got %v, want at least main/go-refactor", gotUse)
	}
}
