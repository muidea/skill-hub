package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	repositorybiz "github.com/muidea/skill-hub/internal/modules/blocks/repository/biz"
	repositoryservice "github.com/muidea/skill-hub/internal/modules/blocks/repository/service"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

func newGlobal(t *testing.T) *Global {
	t.Helper()
	hub := event.NewHub(8)
	background := task.NewBackgroundRoutine(2)
	repository := repositorybiz.New(hub, background, repositoryservice.New())
	t.Cleanup(func() {
		repository.Teardown()
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
	})
	return New(repositoryport.NewProjectSource(hub, "global_test"))
}

func TestGlobal_EnableApplyAndInspect(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir) // isolate from real claude/opencode commands
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	useResult, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, map[string]string{"env": "test"})
	if err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if len(useResult.Agents) != 1 || useResult.Agents[0] != "codex" {
		t.Fatalf("agents = %#v, want codex", useResult.Agents)
	}

	statusBeforeApply, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect before apply returned error: %v", err)
	}
	if got := statusBeforeApply.Items[0].Status; got != StatusMissingAgentDir {
		t.Fatalf("status before apply = %q, want %q", got, StatusMissingAgentDir)
	}
	if got := statusBeforeApply.Items[0].LegacyStatus; got != "missing_agent_dir" {
		t.Fatalf("legacy status before apply = %q, want missing_agent_dir", got)
	}
	if got := statusBeforeApply.Items[0].Reason; got != "agent_directory_missing" {
		t.Fatalf("reason before apply = %q, want agent_directory_missing", got)
	}

	applyResult, err := svc.Apply("", nil, false, false)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(applyResult.Items) != 1 || applyResult.Items[0].Status != StatusApplied {
		t.Fatalf("unexpected apply result: %+v", applyResult)
	}

	appliedSkill := filepath.Join(codexSkillsDir, "demo-skill", "SKILL.md")
	if content, err := os.ReadFile(appliedSkill); err != nil || string(content) != "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n" {
		t.Fatalf("applied skill content = %q err=%v", string(content), err)
	}
	if _, err := os.Stat(filepath.Join(codexSkillsDir, "demo-skill", ManifestFileName)); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}

	statusAfterApply, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect after apply returned error: %v", err)
	}
	if got := statusAfterApply.Items[0].Status; got != StatusOK {
		t.Fatalf("status after apply = %q, want %q", got, StatusOK)
	}
	if got := statusAfterApply.Items[0].LegacyStatus; got != "ok" {
		t.Fatalf("legacy status after apply = %q, want ok", got)
	}
}

func TestGlobal_EnableSkillPreservesVariablesWhenNotExplicitlyProvided(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir)
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", filepath.Join(homeDir, "codex", "skills"))
	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, map[string]string{"region": "cn"}); err != nil {
		t.Fatalf("initial EnableSkill: %v", err)
	}
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("re-enable without variables: %v", err)
	}
	state, err := loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if got := state.EnabledSkills["demo-skill"].Variables["region"]; got != "cn" {
		t.Fatalf("variables were cleared: %#v", state.EnabledSkills["demo-skill"].Variables)
	}
}

func TestGlobal_InspectModifiedAndConflict(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir) // isolate from real claude/opencode commands
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if _, err := svc.Apply("", nil, false, false); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	appliedSkill := filepath.Join(codexSkillsDir, "demo-skill", "SKILL.md")
	if err := os.WriteFile(appliedSkill, []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nChanged\n"), 0644); err != nil {
		t.Fatalf("modify applied skill: %v", err)
	}
	statusModified, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect modified returned error: %v", err)
	}
	if got := statusModified.Items[0].Status; got != StatusModified {
		t.Fatalf("status modified = %q, want %q", got, StatusModified)
	}

	if err := os.RemoveAll(filepath.Join(codexSkillsDir, "demo-skill")); err != nil {
		t.Fatalf("remove applied dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(codexSkillsDir, "demo-skill"), 0755); err != nil {
		t.Fatalf("mkdir conflict dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexSkillsDir, "demo-skill", "SKILL.md"), []byte("unmanaged"), 0644); err != nil {
		t.Fatalf("write conflict skill: %v", err)
	}
	statusConflict, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect conflict returned error: %v", err)
	}
	if got := statusConflict.Items[0].Status; got != StatusConflict {
		t.Fatalf("status conflict = %q, want %q", got, StatusConflict)
	}
}

func TestGlobal_ApplyRefreshesStoredVersion(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir)
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if _, err := svc.Apply("demo-skill", nil, false, false); err != nil {
		t.Fatalf("initial Apply returned error: %v", err)
	}

	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.1.0\n---\nUpdated\n")
	if _, err := svc.Apply("demo-skill", nil, false, false); err != nil {
		t.Fatalf("updated Apply returned error: %v", err)
	}

	status, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(status.Items) != 1 || status.Items[0].Version != "1.1.0" {
		t.Fatalf("status version = %+v, want 1.1.0", status.Items)
	}
}

func TestGlobal_ApplyDryRunAndForceConflict(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir) // isolate from real claude/opencode commands
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}

	dryRunResult, err := svc.Apply("", nil, true, false)
	if err != nil {
		t.Fatalf("dry-run Apply returned error: %v", err)
	}
	if len(dryRunResult.Items) != 2 {
		t.Fatalf("dry-run items = %d, want mirror + agent plan", len(dryRunResult.Items))
	}
	if _, err := os.Stat(filepath.Join(codexSkillsDir, "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create target dir, stat err=%v", err)
	}

	conflictDir := filepath.Join(codexSkillsDir, "demo-skill")
	if err := os.MkdirAll(conflictDir, 0755); err != nil {
		t.Fatalf("mkdir conflict dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conflictDir, "SKILL.md"), []byte("unmanaged"), 0644); err != nil {
		t.Fatalf("write unmanaged skill: %v", err)
	}

	conflictResult, err := svc.Apply("", nil, false, false)
	if err != nil {
		t.Fatalf("conflict Apply returned error: %v", err)
	}
	if len(conflictResult.Items) != 1 || conflictResult.Items[0].Status != StatusConflict {
		t.Fatalf("conflict result = %+v, want conflict", conflictResult)
	}
	content, err := os.ReadFile(filepath.Join(conflictDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read unmanaged skill: %v", err)
	}
	if string(content) != "unmanaged" {
		t.Fatalf("unmanaged content was overwritten without force: %q", string(content))
	}

	forcedResult, err := svc.Apply("", nil, false, true)
	if err != nil {
		t.Fatalf("forced Apply returned error: %v", err)
	}
	if len(forcedResult.Items) != 1 || forcedResult.Items[0].Status != StatusApplied {
		t.Fatalf("forced result = %+v, want applied", forcedResult)
	}
	if _, err := os.Stat(filepath.Join(conflictDir, ManifestFileName)); err != nil {
		t.Fatalf("forced apply should write manifest: %v", err)
	}
	inDirBackups, err := filepath.Glob(filepath.Join(codexSkillsDir, "*.skill-hub-backup.*"))
	if err != nil || len(inDirBackups) != 0 {
		t.Fatalf("in-dir backups = %#v err=%v, want none", inDirBackups, err)
	}
	backups, err := filepath.Glob(filepath.Join(homeDir, "backups", "global", "codex", "demo-skill.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("external backups = %#v err=%v, want one backup", backups, err)
	}
}

func TestGlobal_ExplicitUnknownSkillReturnsError(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", filepath.Join(homeDir, "codex", "skills"))

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}

	if _, err := svc.Inspect("missing-skill", []string{"codex"}); err == nil {
		t.Fatal("Inspect should reject an explicit skill id that is not globally enabled")
	}
	if _, err := svc.Apply("missing-skill", []string{"codex"}, true, false); err == nil {
		t.Fatal("Apply should reject an explicit skill id that is not globally enabled")
	}
}

func TestGlobal_ExplicitAgentFilterMismatchReturnsError(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", filepath.Join(homeDir, "codex", "skills"))
	t.Setenv("OPENCODE_SKILLS_DIR", filepath.Join(homeDir, "opencode", "skills"))

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}

	if _, err := svc.Inspect("demo-skill", []string{"opencode"}); err == nil || !strings.Contains(err.Error(), "opencode") {
		t.Fatalf("Inspect should reject an explicit agent mismatch, got %v", err)
	}
	if _, err := svc.Apply("demo-skill", []string{"opencode"}, true, false); err == nil || !strings.Contains(err.Error(), "opencode") {
		t.Fatalf("Apply should reject an explicit agent mismatch, got %v", err)
	}
}

func TestGlobal_AgentFilterSkipsUnmatchedSkills(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	opencodeSkillsDir := filepath.Join(homeDir, "opencode", "skills")
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)
	t.Setenv("OPENCODE_SKILLS_DIR", opencodeSkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "codex-skill", "---\nname: Codex Skill\nversion: 1.0.0\n---\nHello\n")
	writeRepoSkill(t, homeDir, "main", "opencode-skill", "---\nname: OpenCode Skill\nversion: 1.0.0\n---\nHello\n")

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("codex-skill", "main", []string{"codex"}, nil); err != nil {
		t.Fatalf("EnableSkill codex returned error: %v", err)
	}
	if _, err := svc.EnableSkill("opencode-skill", "main", []string{"opencode"}, nil); err != nil {
		t.Fatalf("EnableSkill opencode returned error: %v", err)
	}

	status, err := svc.Inspect("", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect with codex filter returned error: %v", err)
	}
	if status.SkillCount != 1 || len(status.Items) != 1 || status.Items[0].SkillID != "codex-skill" {
		t.Fatalf("filtered status = %+v, want only codex-skill", status)
	}

	applyResult, err := svc.Apply("", []string{"codex"}, false, false)
	if err != nil {
		t.Fatalf("Apply with codex filter returned error: %v", err)
	}
	if len(applyResult.Items) != 1 || applyResult.Items[0].SkillID != "codex-skill" || applyResult.Items[0].Status != StatusApplied {
		t.Fatalf("filtered apply result = %+v, want only codex-skill applied", applyResult)
	}
	if _, err := os.Stat(filepath.Join(codexSkillsDir, "codex-skill", "SKILL.md")); err != nil {
		t.Fatalf("codex skill should be applied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(opencodeSkillsDir, "opencode-skill")); !os.IsNotExist(err) {
		t.Fatalf("opencode skill should not be applied through codex filter, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(homeDir, "global", "skills", "opencode-skill")); !os.IsNotExist(err) {
		t.Fatalf("opencode global mirror should not be refreshed through codex filter, stat err=%v", err)
	}
}

func writeTestConfig(t *testing.T, homeDir string) {
	t.Helper()
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
}

func writeRepoSkill(t *testing.T, homeDir, repoName, skillID, content string) {
	t.Helper()
	repoSkillDir := filepath.Join(homeDir, "repositories", repoName, "skills", skillID)
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		t.Fatalf("mkdir repo skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoSkillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write repo skill: %v", err)
	}
}

func isolateGlobalAgentEnv(t *testing.T, homeDir string) {
	t.Helper()
	config.ResetForTest()
	t.Cleanup(config.ResetForTest)

	t.Setenv("HOME", homeDir)
	t.Setenv("PATH", homeDir)
	t.Setenv("SKILL_HUB_HOME", homeDir)
	for _, key := range []string{
		"CODEX_SKILLS_DIR", "CODEX_HOME",
		"OPENCODE_SKILLS_DIR", "OPENCODE_HOME",
		"CLAUDE_SKILLS_DIR",
		"AGY_SKILLS_DIR", "AGY_CONFIG_HOME", "AGY_HOME",
	} {
		t.Setenv(key, "")
	}
}

func TestResolveAgent_AgyDetectionGuardAndSignals(t *testing.T) {
	t.Run("does not detect agy from bare gemini directory or file at skills path", func(t *testing.T) {
		homeDir := t.TempDir()
		isolateGlobalAgentEnv(t, homeDir)
		writeTestConfig(t, homeDir)

		cfgDir := filepath.Join(homeDir, ".gemini", "config")
		if err := os.MkdirAll(cfgDir, 0755); err != nil {
			t.Fatalf("mkdir .gemini/config: %v", err)
		}

		info := ResolveAgent("agy")
		if info.Detected || info.Configured || info.Reason != "not_detected" {
			t.Fatalf("ResolveAgent(agy) = %+v, want undetected with not_detected", info)
		}
		for _, detected := range DetectAgents() {
			if detected.Name == "agy" {
				t.Fatalf("DetectAgents() unexpectedly included agy: %+v", detected)
			}
		}

		if err := os.WriteFile(filepath.Join(cfgDir, "skills"), []byte("not-a-dir"), 0644); err != nil {
			t.Fatalf("write file at skills path: %v", err)
		}
		info = ResolveAgent("agy")
		if info.Detected || info.Configured || info.Reason != "not_detected" {
			t.Fatalf("ResolveAgent(agy) with regular file at skills path = %+v, want undetected", info)
		}
	})

	t.Run("detects agy from specific markers and path", func(t *testing.T) {
		tests := []struct {
			name       string
			setup      func(t *testing.T, homeDir string)
			wantReason string
			wantConfig bool
		}{
			{
				name: "skills directory",
				setup: func(t *testing.T, homeDir string) {
					if err := os.MkdirAll(filepath.Join(homeDir, ".gemini", "config", "skills"), 0755); err != nil {
						t.Fatalf("mkdir skills: %v", err)
					}
				},
				wantReason: "agy_skills_dir",
				wantConfig: true,
			},
			{
				name: "migrated marker",
				setup: func(t *testing.T, homeDir string) {
					cfgDir := filepath.Join(homeDir, ".gemini", "config")
					if err := os.MkdirAll(cfgDir, 0755); err != nil {
						t.Fatalf("mkdir config: %v", err)
					}
					if err := os.WriteFile(filepath.Join(cfgDir, ".migrated"), []byte("1"), 0644); err != nil {
						t.Fatalf("write .migrated: %v", err)
					}
				},
				wantReason: "agy_config_dir",
			},
			{
				name: "config json marker",
				setup: func(t *testing.T, homeDir string) {
					cfgDir := filepath.Join(homeDir, ".gemini", "config")
					if err := os.MkdirAll(cfgDir, 0755); err != nil {
						t.Fatalf("mkdir config: %v", err)
					}
					if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte("{}"), 0644); err != nil {
						t.Fatalf("write config.json: %v", err)
					}
				},
				wantReason: "agy_config_dir",
			},
			{
				name: "skills json marker",
				setup: func(t *testing.T, homeDir string) {
					cfgDir := filepath.Join(homeDir, ".gemini", "config")
					if err := os.MkdirAll(cfgDir, 0755); err != nil {
						t.Fatalf("mkdir config: %v", err)
					}
					if err := os.WriteFile(filepath.Join(cfgDir, "skills.json"), []byte("{}"), 0644); err != nil {
						t.Fatalf("write skills.json: %v", err)
					}
				},
				wantReason: "agy_config_dir",
			},
			{
				name: "antigravity cli dir",
				setup: func(t *testing.T, homeDir string) {
					if err := os.MkdirAll(filepath.Join(homeDir, ".gemini", "antigravity-cli"), 0755); err != nil {
						t.Fatalf("mkdir antigravity-cli: %v", err)
					}
				},
				wantReason: "agy_cli_dir",
			},
			{
				name: "agy binary in PATH",
				setup: func(t *testing.T, homeDir string) {
					binPath := filepath.Join(homeDir, "agy")
					if err := os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
						t.Fatalf("write fake agy binary: %v", err)
					}
				},
				wantReason: "path:agy",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				homeDir := t.TempDir()
				isolateGlobalAgentEnv(t, homeDir)
				writeTestConfig(t, homeDir)
				tt.setup(t, homeDir)

				info := ResolveAgent("antigravity")
				wantSkillsDir := filepath.Join(homeDir, ".gemini", "config", "skills")
				if info.Name != "agy" || !info.Detected || info.Configured != tt.wantConfig || info.Reason != tt.wantReason || info.SkillsDir != wantSkillsDir {
					t.Fatalf("ResolveAgent(antigravity) = %+v, want name=agy detected=true configured=%v reason=%q skills_dir=%q", info, tt.wantConfig, tt.wantReason, wantSkillsDir)
				}
			})
		}
	})
}

func TestResolveAgent_AgyPriorityAndConfigPath(t *testing.T) {
	homeDir := t.TempDir()
	isolateGlobalAgentEnv(t, homeDir)

	cfgDir := filepath.Join(homeDir, "custom-agy-cfg")
	cfg := &config.Config{
		AgyConfigPath: cfgDir,
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

	// Config.AgyConfigPath applies when no AGY_* env vars are set, including ~/ and /skills suffix forms.
	info := ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(cfgDir, "skills") || info.Reason != "config:agy_config_path" || !info.Detected {
		t.Fatalf("config priority = %+v", info)
	}

	cfg.AgyConfigPath = "~/.gemini/config"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save tilde config: %v", err)
	}
	info = ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(homeDir, ".gemini", "config", "skills") || info.Reason != "config:agy_config_path" {
		t.Fatalf("tilde config priority = %+v", info)
	}

	explicitSkillsPath := filepath.Join(homeDir, "custom-agy", "skills")
	cfg.AgyConfigPath = explicitSkillsPath
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("save explicit skills config: %v", err)
	}
	info = ResolveAgent("agy")
	if info.SkillsDir != explicitSkillsPath || info.Reason != "config:agy_config_path" {
		t.Fatalf("explicit skills config priority = %+v, want %q", info, explicitSkillsPath)
	}

	// AGY_HOME with plain root -> <AGY_HOME>/skills; with existing config/ or .gemini -> <AGY_HOME>/config/skills.
	plainHome := filepath.Join(homeDir, "plain-agy-home")
	t.Setenv("AGY_HOME", plainHome)
	info = ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(plainHome, "skills") || info.Reason != "env:AGY_HOME" {
		t.Fatalf("AGY_HOME plain = %+v", info)
	}

	homeWithConfig := filepath.Join(homeDir, "custom-agy-root")
	if err := os.MkdirAll(filepath.Join(homeWithConfig, "config"), 0755); err != nil {
		t.Fatalf("mkdir custom-agy-root/config: %v", err)
	}
	t.Setenv("AGY_HOME", homeWithConfig)
	info = ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(homeWithConfig, "config", "skills") || info.Reason != "env:AGY_HOME" {
		t.Fatalf("AGY_HOME with existing config dir = %+v", info)
	}

	geminiHome := filepath.Join(homeDir, ".gemini")
	t.Setenv("AGY_HOME", geminiHome)
	info = ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(geminiHome, "config", "skills") || info.Reason != "env:AGY_HOME" {
		t.Fatalf("AGY_HOME .gemini = %+v", info)
	}

	// AGY_CONFIG_HOME overrides AGY_HOME and Config.AgyConfigPath.
	agyConfigHome := filepath.Join(homeDir, "agy-config-home")
	t.Setenv("AGY_CONFIG_HOME", agyConfigHome)
	info = ResolveAgent("agy")
	if info.SkillsDir != filepath.Join(agyConfigHome, "skills") || info.Reason != "env:AGY_CONFIG_HOME" {
		t.Fatalf("AGY_CONFIG_HOME = %+v", info)
	}

	// AGY_SKILLS_DIR overrides everything else.
	agySkillsDir := filepath.Join(homeDir, "agy-skills-direct")
	t.Setenv("AGY_SKILLS_DIR", agySkillsDir)
	info = ResolveAgent("agy")
	if info.SkillsDir != agySkillsDir || info.Reason != "env:AGY_SKILLS_DIR" {
		t.Fatalf("AGY_SKILLS_DIR = %+v", info)
	}
}

func TestGlobal_AgyOneLevelScanAndExternalBackup(t *testing.T) {
	homeDir := t.TempDir()
	isolateGlobalAgentEnv(t, homeDir)

	agySkillsDir := filepath.Join(homeDir, ".gemini", "config", "skills")
	t.Setenv("AGY_SKILLS_DIR", agySkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: demo-skill\nversion: 1.0.0\n---\nHello AGY\n")

	// Create unmanaged conflict directory, unrelated native skill directory, and stray root file.
	conflictDir := filepath.Join(agySkillsDir, "demo-skill")
	if err := os.MkdirAll(conflictDir, 0755); err != nil {
		t.Fatalf("mkdir conflictDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(conflictDir, "SKILL.md"), []byte("unmanaged agy skill"), 0644); err != nil {
		t.Fatalf("write unmanaged SKILL.md: %v", err)
	}
	nativeDir := filepath.Join(agySkillsDir, "native-agy-skill")
	if err := os.MkdirAll(nativeDir, 0755); err != nil {
		t.Fatalf("mkdir nativeDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "SKILL.md"), []byte("native skill"), 0644); err != nil {
		t.Fatalf("write native SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agySkillsDir, "skills.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	svc := newGlobal(t)
	if _, err := svc.EnableSkill("demo-skill", "main", []string{"antigravity"}, nil); err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}

	// Without --force, unmanaged conflict must be preserved.
	conflictResult, err := svc.Apply("", []string{"agy"}, false, false)
	if err != nil {
		t.Fatalf("Apply without force returned error: %v", err)
	}
	if len(conflictResult.Items) != 1 || conflictResult.Items[0].Status != StatusConflict {
		t.Fatalf("conflictResult = %+v, want conflict", conflictResult)
	}
	if raw, err := os.ReadFile(filepath.Join(conflictDir, "SKILL.md")); err != nil || string(raw) != "unmanaged agy skill" {
		t.Fatalf("unmanaged skill should remain intact, got %q err=%v", string(raw), err)
	}

	// With --force, backup is placed outside agySkillsDir so 1-level scan stays clean.
	forcedResult, err := svc.Apply("", []string{"agy"}, false, true)
	if err != nil {
		t.Fatalf("Apply with force returned error: %v", err)
	}
	if len(forcedResult.Items) != 1 || forcedResult.Items[0].Status != StatusApplied {
		t.Fatalf("forcedResult = %+v, want applied", forcedResult)
	}

	inDirBackups, err := filepath.Glob(filepath.Join(agySkillsDir, "*.skill-hub-backup.*"))
	if err != nil || len(inDirBackups) != 0 {
		t.Fatalf("agySkillsDir must not contain in-place backups, got %#v err=%v", inDirBackups, err)
	}
	extBackups, err := filepath.Glob(filepath.Join(homeDir, "backups", "global", "agy", "demo-skill.*"))
	if err != nil || len(extBackups) != 1 {
		t.Fatalf("expected one external backup under backups/global/agy, got %#v err=%v", extBackups, err)
	}
	if backedUp, err := os.ReadFile(filepath.Join(extBackups[0], "SKILL.md")); err != nil || string(backedUp) != "unmanaged agy skill" {
		t.Fatalf("backed up content = %q err=%v", string(backedUp), err)
	}

	status, err := svc.Inspect("demo-skill", []string{"agy"})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(status.Items) != 1 || status.Items[0].Status != StatusOK || len(status.Orphaned) != 0 {
		t.Fatalf("Inspect status = %+v, want synced and no orphaned items", status)
	}
	if _, err := os.Stat(filepath.Join(nativeDir, "SKILL.md")); err != nil {
		t.Fatalf("native unmanaged skill must remain untouched: %v", err)
	}
}

func TestGlobal_AgyMultiAgentModificationGuardAndRemove(t *testing.T) {
	homeDir := t.TempDir()
	isolateGlobalAgentEnv(t, homeDir)

	codexSkillsDir := filepath.Join(homeDir, "codex", "skills")
	opencodeSkillsDir := filepath.Join(homeDir, "opencode", "skills")
	claudeSkillsDir := filepath.Join(homeDir, "claude", "skills")
	agySkillsDir := filepath.Join(homeDir, "agy", "skills")
	t.Setenv("CODEX_SKILLS_DIR", codexSkillsDir)
	t.Setenv("OPENCODE_SKILLS_DIR", opencodeSkillsDir)
	t.Setenv("CLAUDE_SKILLS_DIR", claudeSkillsDir)
	t.Setenv("AGY_SKILLS_DIR", agySkillsDir)

	writeTestConfig(t, homeDir)
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: demo-skill\nversion: 1.0.0\n---\nHello All\n")

	svc := newGlobal(t)
	useRes, err := svc.EnableSkill("demo-skill", "main", nil, nil)
	if err != nil {
		t.Fatalf("EnableSkill returned error: %v", err)
	}
	if strings.Join(useRes.Agents, ",") != "agy,claude,codex,opencode" {
		t.Fatalf("enabled agents = %#v, want [agy claude codex opencode]", useRes.Agents)
	}
	if _, err := svc.Apply("", nil, false, false); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	// Modify AGY copy locally.
	agySkillFile := filepath.Join(agySkillsDir, "demo-skill", "SKILL.md")
	if err := os.WriteFile(agySkillFile, []byte("---\nname: demo-skill\nversion: 1.0.0\n---\nLocal AGY edit\n"), 0644); err != nil {
		t.Fatalf("modify AGY skill: %v", err)
	}

	status, err := svc.Inspect("demo-skill", nil)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	statusByAgent := map[string]string{}
	for _, item := range status.Items {
		statusByAgent[item.Agent] = item.Status
	}
	if statusByAgent["agy"] != StatusModified || statusByAgent["codex"] != StatusOK || statusByAgent["opencode"] != StatusOK || statusByAgent["claude"] != StatusOK {
		t.Fatalf("statusByAgent = %#v", statusByAgent)
	}

	// Apply without --force refuses to overwrite modified AGY copy.
	noForceRes, err := svc.Apply("demo-skill", []string{"agy"}, false, false)
	if err != nil {
		t.Fatalf("Apply without force returned error: %v", err)
	}
	if len(noForceRes.Items) != 1 || noForceRes.Items[0].Status != StatusConflict || noForceRes.Items[0].Reason != "target_modified" {
		t.Fatalf("noForceRes = %+v, want conflict/target_modified", noForceRes)
	}
	if content, err := os.ReadFile(agySkillFile); err != nil || !strings.Contains(string(content), "Local AGY edit") {
		t.Fatalf("local edit should be preserved without --force, got %q err=%v", string(content), err)
	}

	// Apply with --force backs up locally modified copy externally and resyncs.
	forceRes, err := svc.Apply("demo-skill", []string{"agy"}, false, true)
	if err != nil {
		t.Fatalf("Apply with force returned error: %v", err)
	}
	if len(forceRes.Items) != 1 || forceRes.Items[0].Status != StatusApplied {
		t.Fatalf("forceRes = %+v, want applied", forceRes)
	}
	if content, err := os.ReadFile(agySkillFile); err != nil || !strings.Contains(string(content), "Hello All") {
		t.Fatalf("AGY copy should be resynced after --force, got %q err=%v", string(content), err)
	}
	extBackups, err := filepath.Glob(filepath.Join(homeDir, "backups", "global", "agy", "demo-skill.*"))
	if err != nil || len(extBackups) != 1 {
		t.Fatalf("expected external backup of modified AGY copy, got %#v err=%v", extBackups, err)
	}

	// Repo update triggers StatusStale and refreshes cleanly without --force.
	writeRepoSkill(t, homeDir, "main", "demo-skill", "---\nname: demo-skill\nversion: 1.1.0\n---\nHello v1.1\n")
	staleStatus, err := svc.Inspect("demo-skill", []string{"agy"})
	if err != nil {
		t.Fatalf("Inspect stale returned error: %v", err)
	}
	if len(staleStatus.Items) != 1 || staleStatus.Items[0].Status != StatusStale {
		t.Fatalf("staleStatus = %+v, want outdated", staleStatus)
	}
	refreshRes, err := svc.Apply("demo-skill", []string{"agy"}, false, false)
	if err != nil {
		t.Fatalf("Apply refresh returned error: %v", err)
	}
	if len(refreshRes.Items) != 1 || refreshRes.Items[0].Status != StatusApplied {
		t.Fatalf("refreshRes = %+v, want applied", refreshRes)
	}

	// Orphaned manifest detection under AGY 1-level scan.
	orphanDir := filepath.Join(agySkillsDir, "orphan-skill")
	if err := os.MkdirAll(orphanDir, 0755); err != nil {
		t.Fatalf("mkdir orphanDir: %v", err)
	}
	if err := writeManifest(filepath.Join(orphanDir, ManifestFileName), Manifest{
		ManagedBy: "skill-hub",
		Scope:     "global",
		Agent:     "agy",
		SkillID:   "orphan-skill",
	}); err != nil {
		t.Fatalf("write orphan manifest: %v", err)
	}
	afterOrphan, err := svc.Inspect("demo-skill", []string{"agy"})
	if err != nil {
		t.Fatalf("Inspect orphan returned error: %v", err)
	}
	if len(afterOrphan.Orphaned) != 1 || afterOrphan.Orphaned[0].SkillID != "orphan-skill" {
		t.Fatalf("Orphaned = %+v, want [orphan-skill]", afterOrphan.Orphaned)
	}

	// Removing AGY removes only AGY directory and preserves other enabled agents in state.
	removeRes, err := svc.Remove("demo-skill", []string{"agy"}, false)
	if err != nil {
		t.Fatalf("Remove agy returned error: %v", err)
	}
	if len(removeRes.Items) != 1 || removeRes.Items[0].Status != StatusRemoved {
		t.Fatalf("removeRes = %+v, want removed", removeRes)
	}
	if _, err := os.Stat(filepath.Join(agySkillsDir, "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("AGY skill directory should be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(codexSkillsDir, "demo-skill", "SKILL.md")); err != nil {
		t.Fatalf("codex skill directory must remain intact: %v", err)
	}
	persistedState, err := loadState()
	if err != nil {
		t.Fatalf("loadState after removing agy: %v", err)
	}
	remaining := persistedState.EnabledSkills["demo-skill"].Agents
	if strings.Join(remaining, ",") != "claude,codex,opencode" {
		t.Fatalf("remaining enabled agents = %#v, want [claude codex opencode]", remaining)
	}
	codexStatus, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect codex after removing agy: %v", err)
	}
	if len(codexStatus.Items) != 1 || codexStatus.Items[0].Status != StatusStale {
		t.Fatalf("codexStatus = %+v, want outdated", codexStatus)
	}
	if _, err := svc.Apply("demo-skill", []string{"codex"}, false, false); err != nil {
		t.Fatalf("Apply codex refresh after removing agy: %v", err)
	}
	codexSynced, err := svc.Inspect("demo-skill", []string{"codex"})
	if err != nil {
		t.Fatalf("Inspect codex after refresh: %v", err)
	}
	if len(codexSynced.Items) != 1 || codexSynced.Items[0].Status != StatusOK {
		t.Fatalf("codexSynced = %+v, want synced", codexSynced)
	}
	if _, err := os.Stat(filepath.Join(agySkillsDir, "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("removed agy directory should not be recreated, stat err=%v", err)
	}
}

func TestCopyPath_PreservesFilesDirectoriesAndSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	nestedDir := filepath.Join(srcDir, "references")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("mkdir nestedDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "note.md"), []byte("note"), 0644); err != nil {
		t.Fatalf("write note.md: %v", err)
	}
	if err := os.Symlink("SKILL.md", filepath.Join(srcDir, "file-link")); err != nil {
		t.Fatalf("symlink file-link: %v", err)
	}
	if err := os.Symlink("references", filepath.Join(srcDir, "dir-link")); err != nil {
		t.Fatalf("symlink dir-link: %v", err)
	}
	if err := os.Symlink("does-not-exist", filepath.Join(srcDir, "dangling-link")); err != nil {
		t.Fatalf("symlink dangling-link: %v", err)
	}

	dstDir := filepath.Join(tempDir, "dst")
	if err := copyPath(srcDir, dstDir); err != nil {
		t.Fatalf("copyPath returned error: %v", err)
	}

	if content, err := os.ReadFile(filepath.Join(dstDir, "references", "note.md")); err != nil || string(content) != "note" {
		t.Fatalf("copied nested file = %q err=%v", string(content), err)
	}
	for linkName, wantTarget := range map[string]string{
		"file-link":     "SKILL.md",
		"dir-link":      "references",
		"dangling-link": "does-not-exist",
	} {
		gotTarget, err := os.Readlink(filepath.Join(dstDir, linkName))
		if err != nil {
			t.Fatalf("Readlink(%s) error = %v", linkName, err)
		}
		if gotTarget != wantTarget {
			t.Fatalf("Readlink(%s) = %q, want %q", linkName, gotTarget, wantTarget)
		}
	}
}
