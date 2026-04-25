package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestProjectApply_ApplyOpenCodeProject(t *testing.T) {
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
	skillContent := []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "1.0.0",
					SourceRepository: "main",
					Variables: map[string]string{
						"env": "test",
					},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, "state.json"), payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result, err := New().Apply(projectDir, "", false, false)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != "applied" {
		t.Fatalf("unexpected apply result: %+v", result)
	}

	appliedPath := filepath.Join(projectDir, ".agents", "skills", "demo-skill", "SKILL.md")
	appliedContent, err := os.ReadFile(appliedPath)
	if err != nil {
		t.Fatalf("read applied skill: %v", err)
	}
	if string(appliedContent) != string(skillContent) {
		t.Fatalf("expected applied content to match repo content")
	}
}

func TestProjectApply_ApplyUsesSourceRepository(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main":      {Name: "main", Enabled: true},
				"community": {Name: "community", Enabled: true},
			},
		},
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	mainSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "demo-skill")
	if err := os.MkdirAll(mainSkillDir, 0755); err != nil {
		t.Fatalf("mkdir main skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mainSkillDir, "SKILL.md"), []byte("---\nname: Main Demo Skill\nversion: 1.0.0\n---\nFROM_MAIN\n"), 0644); err != nil {
		t.Fatalf("write main skill: %v", err)
	}

	communitySkillDir := filepath.Join(homeDir, "repositories", "community", "skills", "demo-skill")
	if err := os.MkdirAll(communitySkillDir, 0755); err != nil {
		t.Fatalf("mkdir community skill dir: %v", err)
	}
	communityContent := []byte("---\nname: Community Demo Skill\nversion: 2.0.0\n---\nFROM_COMMUNITY\n")
	if err := os.WriteFile(filepath.Join(communitySkillDir, "SKILL.md"), communityContent, 0644); err != nil {
		t.Fatalf("write community skill: %v", err)
	}

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "2.0.0",
					SourceRepository: "community",
					Variables:        map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, "state.json"), payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result, err := New().Apply(projectDir, "", false, false)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Status != "applied" {
		t.Fatalf("unexpected apply result: %+v", result)
	}

	appliedPath := filepath.Join(projectDir, ".agents", "skills", "demo-skill", "SKILL.md")
	appliedContent, err := os.ReadFile(appliedPath)
	if err != nil {
		t.Fatalf("read applied skill: %v\nresult=%+v\ntree=%s", err, result, debugTree(t, filepath.Join(projectDir, ".agents")))
	}
	if string(appliedContent) != string(communityContent) {
		t.Fatalf("expected applied content from community repository, got %q", string(appliedContent))
	}
}

func TestProjectApply_ApplySpecificOutdatedSkillRefreshesFromRepository(t *testing.T) {
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
	repoContent := []byte("---\nname: Demo Skill\nversion: 1.1.0\n---\nrepo version\n")
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), repoContent, 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}

	projectDir := filepath.Join(homeDir, "workspace", "demo")
	projectSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(projectSkillDir, 0755); err != nil {
		t.Fatalf("mkdir project skill dir: %v", err)
	}
	localContent := []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nlocal old version\n")
	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), localContent, 0644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}

	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath: projectDir,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "1.0.0",
					Status:           spec.SkillStatusOutdated,
					SourceRepository: "main",
					Variables:        map[string]string{},
				},
				"other-skill": {
					SkillID:          "other-skill",
					Version:          "1.0.0",
					SourceRepository: "main",
					Variables:        map[string]string{},
				},
			},
		},
	}
	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, "state.json"), payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result, err := New().Apply(projectDir, "demo-skill", false, false)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].SkillID != "demo-skill" || result.Items[0].Status != "applied" {
		t.Fatalf("unexpected apply result: %+v", result)
	}

	appliedContent, err := os.ReadFile(filepath.Join(projectSkillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read applied skill: %v", err)
	}
	if string(appliedContent) != string(repoContent) {
		t.Fatalf("project skill was not refreshed from repository, got %q", string(appliedContent))
	}

	var savedState map[string]spec.ProjectState
	savedData, err := os.ReadFile(filepath.Join(homeDir, "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if err := json.Unmarshal(savedData, &savedState); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	savedSkill := savedState[projectDir].Skills["demo-skill"]
	if savedSkill.Version != "1.1.0" || savedSkill.Status != spec.SkillStatusSynced {
		t.Fatalf("state skill = %+v, want version 1.1.0 and Synced", savedSkill)
	}

	status, err := projectstatusservice.New().Inspect(projectDir, "demo-skill")
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(status.Items) != 1 || status.Items[0].Status != spec.SkillStatusSynced {
		t.Fatalf("status after apply = %+v, want Synced", status.Items)
	}
}

func debugTree(t *testing.T, root string) string {
	t.Helper()

	var lines []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			lines = append(lines, path+": ERR "+err.Error())
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		lines = append(lines, rel)
		return nil
	})
	if err != nil {
		return err.Error()
	}
	if len(lines) == 0 {
		return "<empty>"
	}
	return strings.Join(lines, "\n")
}
