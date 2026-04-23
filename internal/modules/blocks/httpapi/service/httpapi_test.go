package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestHTTPAPI_Health(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload httpapibiz.Response[httpapibiz.HealthData]
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Data.Status != "ok" {
		t.Fatalf("expected health status ok, got %q", payload.Data.Status)
	}
}

func TestHTTPAPI_ListProjects(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

	stateData := map[string]spec.ProjectState{
		"/tmp/project-one": {
			ProjectPath:     "/tmp/project-one",
			PreferredTarget: spec.TargetOpenCode,
			Skills: map[string]spec.SkillVars{
				"demo-skill": {
					SkillID:   "demo-skill",
					Version:   "1.2.3",
					Status:    spec.SkillStatusSynced,
					Variables: map[string]string{},
				},
			},
		},
	}

	payload, err := json.Marshal(stateData)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	statePath := filepath.Join(homeDir, "state.json")
	if err := os.WriteFile(statePath, payload, 0644); err != nil {
		t.Fatalf("write state file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp httpapibiz.Response[httpapibiz.ProjectListData]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Data.Items))
	}
	if resp.Data.Items[0].ProjectPath != "/tmp/project-one" {
		t.Fatalf("expected project path /tmp/project-one, got %q", resp.Data.Items[0].ProjectPath)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("preferred_target")) {
		t.Fatalf("project target should not be exposed in project list response: %s", rec.Body.String())
	}
}

func TestHTTPAPI_ListSkillsIncludesTotalAndShowsCompatibilityOnly(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)

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
`)
	if err := os.WriteFile(filepath.Join(homeDir, "config.yaml"), configContent, 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	skills := map[string]string{
		"open-code-skill": spec.TargetOpenCode,
		"cursor-skill":    spec.TargetCursor,
	}
	for id, target := range skills {
		skillDir := filepath.Join(homeDir, "repositories", "main", "skills", id)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		content := []byte(`---
name: ` + id + `
description: test skill
version: 1.0.0
compatibility: ` + target + `
---
`)
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()
	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp httpapibiz.Response[httpapibiz.SkillListData]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Data.Total)
	}
	if len(resp.Data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Data.Items))
	}
	ids := map[string]bool{}
	for _, item := range resp.Data.Items {
		ids[item.ID] = true
	}
	if !ids["open-code-skill"] || !ids["cursor-skill"] {
		t.Fatalf("expected compatibility metadata to be display-only, got %#v", ids)
	}
}

func TestHTTPAPI_ProjectStatusRequiresPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/project-status", nil)
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHTTPAPI_SearchRequiresKeyword(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHTTPAPI_UseSkillRequiresBodyFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/project-skills/use", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHTTPAPI_GlobalEndpoints(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)
	t.Setenv("CODEX_SKILLS_DIR", filepath.Join(homeDir, "codex", "skills"))

	writeHTTPAPITestConfig(t, homeDir)
	writeHTTPAPITestSkill(t, homeDir, "main", "demo-skill")

	useReq := httptest.NewRequest(http.MethodPost, "/api/v1/global-skills/use", bytes.NewBufferString(`{"skill_id":"demo-skill","repository":"main","agents":["codex"]}`))
	useRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(useRec, useReq)
	if useRec.Code != http.StatusOK {
		t.Fatalf("expected use global 200, got %d body=%s", useRec.Code, useRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/global-status?skill_id=demo-skill&agent=codex", nil)
	statusRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected global status 200, got %d body=%s", statusRec.Code, statusRec.Body.String())
	}
	var statusResp httpapibiz.Response[httpapibiz.GlobalStatusData]
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("unmarshal global status: %v", err)
	}
	if statusResp.Data.Item == nil || len(statusResp.Data.Item.Items) != 1 {
		t.Fatalf("unexpected global status payload: %+v", statusResp)
	}

	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/global-apply", bytes.NewBufferString(`{"skill_id":"demo-skill","agents":["codex"],"dry_run":true}`))
	applyRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected global apply 200, got %d body=%s", applyRec.Code, applyRec.Body.String())
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/api/v1/global-skills/demo-skill?agent=codex", nil)
	removeRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusOK {
		t.Fatalf("expected global remove 200, got %d body=%s", removeRec.Code, removeRec.Body.String())
	}
}

func TestHTTPAPI_GlobalExplicitUnknownSkillReturnsNotFound(t *testing.T) {
	config.ResetForTest()
	defer config.ResetForTest()

	homeDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", homeDir)
	writeHTTPAPITestConfig(t, homeDir)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/global-status?skill_id=missing-skill", nil)
	statusRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusNotFound {
		t.Fatalf("expected global status 404, got %d body=%s", statusRec.Code, statusRec.Body.String())
	}

	applyReq := httptest.NewRequest(http.MethodPost, "/api/v1/global-apply", bytes.NewBufferString(`{"skill_id":"missing-skill","dry_run":true}`))
	applyRec := httptest.NewRecorder()
	New().Handler().ServeHTTP(applyRec, applyReq)
	if applyRec.Code != http.StatusNotFound {
		t.Fatalf("expected global apply 404, got %d body=%s", applyRec.Code, applyRec.Body.String())
	}
}

func TestHTTPAPI_ApplyProjectRequiresPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/project-apply", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHTTPAPI_FeedbackRequiresFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/project-feedback/preview", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHTTPAPI_PushRequiresConfirm(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/skill-repository/push", bytes.NewBufferString(`{"message":"test"}`))
	rec := httptest.NewRecorder()

	New().Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("confirm=true")) {
		t.Fatalf("expected confirm error, got %s", rec.Body.String())
	}
}

func TestWriteWrappedErrorUsesAppErrorCode(t *testing.T) {
	rec := httptest.NewRecorder()

	writeWrappedError(rec, apperrors.NewWithCode("test", apperrors.ErrSkillNotFound, "技能不存在"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["code"] != string(apperrors.ErrSkillNotFound) {
		t.Fatalf("expected code %q, got %q", apperrors.ErrSkillNotFound, payload["code"])
	}
	if payload["message"] != "技能不存在" {
		t.Fatalf("expected app error message, got %q", payload["message"])
	}
}

func TestSkillRepositoryChangedFiles(t *testing.T) {
	status := "技能仓库状态:\n文件状态:\n M  skills/one/SKILL.md\n?? skills/two/SKILL.md\n D  skills/old/SKILL.md\n"
	files := skillRepositoryChangedFiles(status)
	want := []string{"skills/one/SKILL.md", "skills/two/SKILL.md", "skills/old/SKILL.md"}
	if !sameStringSet(files, want) {
		t.Fatalf("files = %#v, want %#v", files, want)
	}
}

func writeHTTPAPITestConfig(t *testing.T, homeDir string) {
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

func writeHTTPAPITestSkill(t *testing.T, homeDir, repoName, skillID string) {
	t.Helper()
	skillDir := filepath.Join(homeDir, "repositories", repoName, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := []byte("---\nname: Demo Skill\nversion: 1.0.0\n---\nHello\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
