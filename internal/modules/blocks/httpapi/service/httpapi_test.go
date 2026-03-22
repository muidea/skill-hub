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
