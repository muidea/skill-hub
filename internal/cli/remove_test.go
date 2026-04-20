package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestRunRemoveOpenCodeRemovesStateAndWorkspace(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, "workspace", "demo")
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

	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
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
					Variables:        map[string]string{},
				},
			},
		},
	}
	writeCLIState(t, homeDir, stateData)

	output := withWorkingDir(t, projectDir, func() string {
		return withStdin(t, "y\n", func() string {
			return captureStdout(t, func() {
				if err := runRemove("demo-skill"); err != nil {
					t.Fatalf("runRemove returned error: %v", err)
				}
			})
		})
	})

	if !strings.Contains(output, "技能移除完成") {
		t.Fatalf("unexpected output: %q", output)
	}
	if _, err := os.Stat(localSkillDir); !os.IsNotExist(err) {
		t.Fatalf("expected local skill dir removed, stat err=%v", err)
	}

	states := readCLIState(t, homeDir)
	if _, ok := states[projectDir].Skills["demo-skill"]; ok {
		t.Fatalf("expected demo-skill removed from state")
	}
}

func TestRunRemoveIgnoresLegacyCursorTargetAndRemovesWorkspace(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	projectDir := filepath.Join(homeDir, "workspace", "demo")
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

	localSkillDir := filepath.Join(projectDir, ".agents", "skills", "demo-skill")
	if err := os.MkdirAll(localSkillDir, 0755); err != nil {
		t.Fatalf("mkdir local skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkillDir, "SKILL.md"), skillContent, 0644); err != nil {
		t.Fatalf("write local skill: %v", err)
	}

	cursorConfig := filepath.Join(projectDir, ".cursorrules")
	cursorContent := "# === SKILL-HUB BEGIN: demo-skill ===\nhello\n# === SKILL-HUB END: demo-skill ===\n"
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(cursorConfig, []byte(cursorContent), 0644); err != nil {
		t.Fatalf("write cursor config: %v", err)
	}

	stateData := map[string]spec.ProjectState{
		projectDir: {
			ProjectPath:     projectDir,
			PreferredTarget: spec.TargetCursor,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:          "demo-skill",
					Version:          "1.0.0",
					SourceRepository: "main",
					Variables:        map[string]string{},
				},
			},
		},
	}
	writeCLIState(t, homeDir, stateData)

	withWorkingDir(t, projectDir, func() string {
		return withStdin(t, "y\n", func() string {
			return captureStdout(t, func() {
				if err := runRemove("demo-skill"); err != nil {
					t.Fatalf("runRemove returned error: %v", err)
				}
			})
		})
	})

	updatedCursorConfig, err := os.ReadFile(cursorConfig)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read cursor config: %v", err)
	}
	if !strings.Contains(string(updatedCursorConfig), "demo-skill") {
		t.Fatalf("expected legacy cursor config untouched, got %q", string(updatedCursorConfig))
	}
	if _, err := os.Stat(localSkillDir); !os.IsNotExist(err) {
		t.Fatalf("expected local skill dir removed, stat err=%v", err)
	}

	states := readCLIState(t, homeDir)
	if _, ok := states[projectDir].Skills["demo-skill"]; ok {
		t.Fatalf("expected demo-skill removed from state")
	}
}

func withStdin(t *testing.T, input string, fn func() string) string {
	t.Helper()

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdin: %v", err)
	}
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()

	return fn()
}

func writeCLIState(t *testing.T, homeDir string, stateData map[string]spec.ProjectState) {
	t.Helper()

	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(homeDir, "state.json"), payload, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

func readCLIState(t *testing.T, homeDir string) map[string]spec.ProjectState {
	t.Helper()

	payload, err := os.ReadFile(filepath.Join(homeDir, "state.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var states map[string]spec.ProjectState
	if err := json.Unmarshal(payload, &states); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	return states
}
