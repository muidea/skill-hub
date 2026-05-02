package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
)

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

	svc := New()
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

	svc := New()
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

	svc := New()
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
	backups, err := filepath.Glob(filepath.Join(codexSkillsDir, "demo-skill.skill-hub-backup.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %#v err=%v, want one backup", backups, err)
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

	svc := New()
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

	svc := New()
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

	svc := New()
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
