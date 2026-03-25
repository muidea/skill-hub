package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestRunPruneRemovesInvalidProjectStates(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	validProject := filepath.Join(homeDir, "workspace", "valid")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("mkdir valid project: %v", err)
	}

	missingProject := filepath.Join(homeDir, "workspace", "missing")
	writeCLIState(t, homeDir, map[string]spec.ProjectState{
		validProject: {
			ProjectPath:     validProject,
			PreferredTarget: spec.TargetOpenCode,
			Skills:          map[string]spec.SkillVars{},
		},
		missingProject: {
			ProjectPath:     missingProject,
			PreferredTarget: spec.TargetCursor,
			Skills:          map[string]spec.SkillVars{},
		},
	})

	output := captureStdout(t, func() {
		if err := runPrune(); err != nil {
			t.Fatalf("runPrune returned error: %v", err)
		}
	})

	if !strings.Contains(output, "已清理 1 条失效项目记录") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, missingProject) {
		t.Fatalf("expected output to contain removed project path, got %q", output)
	}

	states := readCLIState(t, homeDir)
	if len(states) != 1 {
		t.Fatalf("expected 1 remaining state, got %d", len(states))
	}
	if _, ok := states[missingProject]; ok {
		t.Fatalf("expected missing project removed from state")
	}
	if _, ok := states[validProject]; !ok {
		t.Fatalf("expected valid project kept in state")
	}
}

func TestRunPruneWhenStateIsAlreadyClean(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	validProject := filepath.Join(homeDir, "workspace", "valid")
	if err := os.MkdirAll(validProject, 0755); err != nil {
		t.Fatalf("mkdir valid project: %v", err)
	}

	writeCLIState(t, homeDir, map[string]spec.ProjectState{
		validProject: {
			ProjectPath:     validProject,
			PreferredTarget: spec.TargetOpenCode,
			Skills:          map[string]spec.SkillVars{},
		},
	})

	output := captureStdout(t, func() {
		if err := runPrune(); err != nil {
			t.Fatalf("runPrune returned error: %v", err)
		}
	})

	if !strings.Contains(output, "未发现失效项目记录") {
		t.Fatalf("unexpected output: %q", output)
	}

	states := readCLIState(t, homeDir)
	if len(states) != 1 {
		t.Fatalf("expected clean state to remain unchanged, got %d entries", len(states))
	}
}
