package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	projectstatebiz "github.com/muidea/skill-hub/internal/modules/blocks/project_state/biz"
	projectstateservice "github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"
	repositorybiz "github.com/muidea/skill-hub/internal/modules/blocks/repository/biz"
	repositoryservice "github.com/muidea/skill-hub/internal/modules/blocks/repository/service"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func newProjectFeedback(t *testing.T) *ProjectFeedback {
	t.Helper()
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(2)
	projectState := projectstatebiz.New(hub, background, projectstateservice.New())
	repository := repositorybiz.New(hub, background, repositoryservice.New())
	t.Cleanup(func() {
		projectState.Teardown()
		repository.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})
	return New(projectstateport.New(hub, "project_feedback_test"), repositoryport.NewProjectSource(hub, "project_feedback_test"))
}

func TestProjectFeedback_ApplyBumpsVersionAndArchives(t *testing.T) {
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
	projectSkillDir := filepath.Join(homeDir, "workspace", "demo", ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.MkdirAll(projectSkillDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	repoContent := "---\nname: Demo Skill\nversion: 1.0.0\n---\nRepo\n"
	projectContent := "---\nname: Demo Skill\nversion: 1.0.0\n---\nProject Changed\n"
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte(repoContent), 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{\n  \""+filepath.Join(homeDir, "workspace", "demo")+"\": {\n    \"project_path\": \""+filepath.Join(homeDir, "workspace", "demo")+"\",\n    \"preferred_target\": \"open_code\",\n    \"skills\": {\n      \"demo-skill\": {\n        \"skill_id\": \"demo-skill\",\n        \"version\": \"1.0.0\",\n        \"variables\": {}\n      }\n    }\n  }\n}\n"), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result, err := newProjectFeedback(t).Apply(filepath.Join(homeDir, "workspace", "demo"), "demo-skill")
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if result.DefaultRepo != "main" {
		t.Fatalf("expected default repo main, got %q", result.DefaultRepo)
	}

	archivedContent, err := os.ReadFile(filepath.Join(repoSkillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read archived skill: %v", err)
	}
	if !strings.Contains(string(archivedContent), "version: 1.0.1") {
		t.Fatalf("expected bumped version in archived skill, got %q", string(archivedContent))
	}
}

func TestProjectFeedback_ApplyBlocksOutdatedProjectSkill(t *testing.T) {
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
	projectDir := filepath.Join(homeDir, "workspace", "demo")
	projectSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.MkdirAll(projectSkillDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	repoContent := "---\nname: Demo Skill\nversion: 1.1.0\n---\nRepo Newer\n"
	projectContent := "---\nname: Demo Skill\nversion: 1.0.0\n---\nProject Changed\n"
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte(repoContent), 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{\n  \""+projectDir+"\": {\n    \"project_path\": \""+projectDir+"\",\n    \"skills\": {\n      \"demo-skill\": {\n        \"skill_id\": \"demo-skill\",\n        \"version\": \"1.0.0\",\n        \"variables\": {}\n      }\n    }\n  }\n}\n"), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	preview, err := newProjectFeedback(t).Preview(projectDir, "demo-skill")
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if !preview.Blocked {
		t.Fatalf("expected preview to be blocked: %+v", preview)
	}

	_, err = newProjectFeedback(t).Apply(projectDir, "demo-skill")
	if err == nil {
		t.Fatal("expected Apply to reject outdated project skill")
	}

	archivedContent, err := os.ReadFile(filepath.Join(repoSkillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read repo skill: %v", err)
	}
	if string(archivedContent) != repoContent {
		t.Fatalf("repo skill should not be overwritten, got %q", string(archivedContent))
	}
}

func TestProjectFeedback_ApplyAllowsCodeBlockHashCommentEdits(t *testing.T) {
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

	repoSkillDir := filepath.Join(homeDir, "repositories", "main", "skills", "remote-skill")
	projectDir := filepath.Join(homeDir, "workspace", "demo")
	projectSkillDir := filepath.Join(projectDir, ".agents", "skills", "remote-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.MkdirAll(projectSkillDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	repoContent := "---\nname: Remote Skill\ndescription: remote verification\nversion: 1.0.0\n---\n# Remote Skill\n\n## Workflow\n```sh\n# 2. 执行远端标准（或 --suite abnormal）功能回归：\npython3 -B run.py\n```\n"
	projectContent := "---\nname: Remote Skill\ndescription: remote verification\nversion: 1.0.0\n---\n# Remote Skill\n\n## Workflow\n```sh\n# 2. 执行远端标准（以及独立的 --suite abnormal）功能回归：\npython3 -B run.py\n```\n"
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte(repoContent), 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{\n  \""+projectDir+"\": {\n    \"project_path\": \""+projectDir+"\",\n    \"skills\": {\n      \"remote-skill\": {\n        \"skill_id\": \"remote-skill\",\n        \"version\": \"1.0.0\",\n        \"variables\": {}\n      }\n    }\n  }\n}\n"), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	if _, err := newProjectFeedback(t).Apply(projectDir, "remote-skill"); err != nil {
		t.Fatalf("Apply returned unexpected error: %v", err)
	}

	archivedContent, err := os.ReadFile(filepath.Join(repoSkillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read archived skill: %v", err)
	}
	if !strings.Contains(string(archivedContent), "version: 1.0.1") || !strings.Contains(string(archivedContent), "以及独立的 --suite abnormal") {
		t.Fatalf("unexpected archived skill content:\n%s", string(archivedContent))
	}
}

func TestProjectFeedback_ApplyRollsBackVersionOnArchiveFailure(t *testing.T) {
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
	projectDir := filepath.Join(homeDir, "workspace", "demo")
	projectSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}
	if err := os.MkdirAll(projectSkillDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	repoContent := "---\nname: Demo Skill\ndescription: demo\nversion: 1.0.0\n---\n# Demo Skill\n\n## Required Section\nkeep\n"
	projectContent := "---\nname: Demo Skill\ndescription: demo\nversion: 1.0.0\n---\n# Demo Skill\n"
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte(repoContent), 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if err := os.WriteFile(projectSkillPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project skill: %v", err)
	}

	statePath := filepath.Join(homeDir, "state.json")
	if err := os.WriteFile(statePath, []byte("{\n  \""+projectDir+"\": {\n    \"project_path\": \""+projectDir+"\",\n    \"skills\": {\n      \"demo-skill\": {\n        \"skill_id\": \"demo-skill\",\n        \"version\": \"1.0.0\",\n        \"variables\": {}\n      }\n    }\n  }\n}\n"), 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	_, err := newProjectFeedback(t).Apply(projectDir, "demo-skill")
	if err == nil {
		t.Fatal("expected Apply to fail when required section is removed")
	}

	gotProjectContent, readErr := os.ReadFile(projectSkillPath)
	if readErr != nil {
		t.Fatalf("read project skill: %v", readErr)
	}
	if string(gotProjectContent) != projectContent {
		t.Fatalf("project skill version was not rolled back:\ngot:\n%s\nwant:\n%s", string(gotProjectContent), projectContent)
	}
}
