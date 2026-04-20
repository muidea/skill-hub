package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"
	projectfeedbackservice "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback/service"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	projectuseservice "github.com/muidea/skill-hub/internal/modules/kernel/project_use/service"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestClient_AvailableAndListEndpoints(t *testing.T) {
	client := New("http://127.0.0.1:5525")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/health":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.HealthData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.HealthData{Status: "ok"},
				}), nil
			case "/api/v1/repos":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.RepoListData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.RepoListData{
						DefaultRepo: "main",
						Items: []config.RepositoryConfig{
							{Name: "main", Enabled: true},
						},
					},
				}), nil
			case "/api/v1/skills":
				if got := req.URL.Query().Get("repo"); got != "main" {
					t.Fatalf("expected repo filter main, got %q", got)
				}
				if got := req.URL.Query().Get("target"); got != "" {
					t.Fatalf("expected no target filter, got %q", got)
				}
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.SkillListData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.SkillListData{
						Items: []spec.SkillMetadata{
							{ID: "demo", Name: "Demo", Repository: "main"},
						},
						Total: 1,
					},
				}), nil
			case "/api/v1/search":
				if got := req.URL.Query().Get("keyword"); got != "demo" {
					t.Fatalf("expected keyword demo, got %q", got)
				}
				if got := req.URL.Query().Get("target"); got != "" {
					t.Fatalf("expected no target query, got %q", got)
				}
				if got := req.URL.Query().Get("limit"); got != "5" {
					t.Fatalf("expected limit 5, got %q", got)
				}
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.RemoteSearchData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.RemoteSearchData{
						Items: []spec.RemoteSearchResult{
							{FullName: "demo/search-skill", HTMLURL: "https://example.com/demo"},
						},
					},
				}), nil
			case "/api/v1/project-status":
				if got := req.URL.Query().Get("path"); got != "/tmp/project" {
					t.Fatalf("expected project path /tmp/project, got %q", got)
				}
				if got := req.URL.Query().Get("skill_id"); got != "demo" {
					t.Fatalf("expected skill_id demo, got %q", got)
				}
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.ProjectStatusData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.ProjectStatusData{
						Item: &projectstatusservice.ProjectStatusSummary{
							ProjectPath: "/tmp/project",
							SkillCount:  1,
							Items: []projectstatusservice.SkillStatusItem{
								{SkillID: "demo", Status: spec.SkillStatusSynced, LocalVersion: "1.0.0"},
							},
						},
					},
				}), nil
			case "/api/v1/skills/demo/candidates":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.SkillCandidateListData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.SkillCandidateListData{
						Items: []spec.SkillMetadata{
							{ID: "demo", Name: "Demo", Repository: "main"},
						},
					},
				}), nil
			case "/api/v1/skills/demo":
				if got := req.URL.Query().Get("repo"); got != "main" {
					t.Fatalf("expected repo query main, got %q", got)
				}
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.SkillDetailData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.SkillDetailData{
						Item: &spec.Skill{ID: "demo", Name: "Demo", Repository: "main"},
					},
				}), nil
			case "/api/v1/project-skills/use":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.UseSkillData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.UseSkillData{
						Item: &projectuseservice.UseResult{
							ProjectPath: "/tmp/project",
							SkillID:     "demo",
							Repository:  "main",
						},
					},
				}), nil
			case "/api/v1/project-apply":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.ApplyProjectData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.ApplyProjectData{
						Item: &projectapplyservice.ApplyResult{
							ProjectPath: "/tmp/project",
							DryRun:      true,
							Items: []projectapplyservice.ApplyResultItem{
								{SkillID: "demo", Status: "planned", Variables: 1},
							},
						},
					},
				}), nil
			case "/api/v1/project-feedback/preview", "/api/v1/project-feedback/apply":
				return jsonResponse(http.StatusOK, httpapibiz.Response[httpapibiz.FeedbackPreviewData]{
					Code: httpapibiz.CodeOK,
					Data: httpapibiz.FeedbackPreviewData{
						Item: &projectfeedbackservice.PreviewResult{
							ProjectPath:      "/tmp/project",
							SkillID:          "demo",
							DefaultRepo:      "main",
							ProjectVersion:   "1.0.0",
							RepoVersion:      "1.0.0",
							ResolvedVersion:  "1.0.1",
							NeedsVersionBump: true,
							Changes:          []string{"修改: SKILL.md"},
						},
					},
				}), nil
			default:
				return jsonResponse(http.StatusNotFound, map[string]string{"message": "not found"}), nil
			}
		}),
	}

	ctx := context.Background()
	if !client.Available(ctx) {
		t.Fatal("expected client to report available")
	}

	repos, err := client.ListRepos(ctx)
	if err != nil {
		t.Fatalf("ListRepos returned error: %v", err)
	}
	if repos.DefaultRepo != "main" || len(repos.Items) != 1 {
		t.Fatalf("unexpected repo data: %+v", repos)
	}

	skills, err := client.ListSkills(ctx, []string{"main"})
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if len(skills) != 1 || skills[0].ID != "demo" {
		t.Fatalf("unexpected skills: %+v", skills)
	}

	searchResults, err := client.SearchRemoteSkills(ctx, "demo", 5)
	if err != nil {
		t.Fatalf("SearchRemoteSkills returned error: %v", err)
	}
	if len(searchResults) != 1 || searchResults[0].FullName != "demo/search-skill" {
		t.Fatalf("unexpected search results: %+v", searchResults)
	}

	projectStatus, err := client.GetProjectStatus(ctx, "/tmp/project", "demo")
	if err != nil {
		t.Fatalf("GetProjectStatus returned error: %v", err)
	}
	if projectStatus.Item == nil || len(projectStatus.Item.Items) != 1 {
		t.Fatalf("unexpected project status payload: %+v", projectStatus)
	}

	candidates, err := client.FindSkillCandidates(ctx, "demo")
	if err != nil {
		t.Fatalf("FindSkillCandidates returned error: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Repository != "main" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}

	detail, err := client.GetSkillDetail(ctx, "demo", "main")
	if err != nil {
		t.Fatalf("GetSkillDetail returned error: %v", err)
	}
	if detail == nil || detail.ID != "demo" {
		t.Fatalf("unexpected skill detail: %+v", detail)
	}

	useResp, err := client.UseSkill(ctx, httpapibiz.UseSkillRequest{
		ProjectPath: "/tmp/project",
		SkillID:     "demo",
		Repository:  "main",
	})
	if err != nil {
		t.Fatalf("UseSkill returned error: %v", err)
	}
	if useResp.Item == nil || useResp.Item.Repository != "main" {
		t.Fatalf("unexpected use response: %+v", useResp)
	}

	applyResp, err := client.ApplyProject(ctx, httpapibiz.ApplyProjectRequest{
		ProjectPath: "/tmp/project",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("ApplyProject returned error: %v", err)
	}
	if applyResp.Item == nil || len(applyResp.Item.Items) != 1 || applyResp.Item.Items[0].Status != "planned" {
		t.Fatalf("unexpected apply response: %+v", applyResp)
	}

	feedbackPreview, err := client.PreviewFeedback(ctx, httpapibiz.FeedbackRequest{
		ProjectPath: "/tmp/project",
		SkillID:     "demo",
	})
	if err != nil {
		t.Fatalf("PreviewFeedback returned error: %v", err)
	}
	if feedbackPreview.Item == nil || feedbackPreview.Item.DefaultRepo != "main" {
		t.Fatalf("unexpected feedback preview: %+v", feedbackPreview)
	}

	feedbackApply, err := client.ApplyFeedback(ctx, httpapibiz.FeedbackRequest{
		ProjectPath: "/tmp/project",
		SkillID:     "demo",
	})
	if err != nil {
		t.Fatalf("ApplyFeedback returned error: %v", err)
	}
	if feedbackApply.Item == nil || feedbackApply.Item.ResolvedVersion != "1.0.1" {
		t.Fatalf("unexpected feedback apply: %+v", feedbackApply)
	}
}

func TestClientSendsSecretKeyHeader(t *testing.T) {
	client := NewWithSecret("http://127.0.0.1:5525", "write-secret")
	seen := map[string]bool{}
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get(httpapibiz.SecretKeyHeader); got != "write-secret" {
				t.Fatalf("expected secret key header, got %q", got)
			}
			seen[req.URL.Path] = true
			return jsonResponse(http.StatusOK, httpapibiz.Response[map[string]string]{
				Code: httpapibiz.CodeOK,
				Data: map[string]string{"ok": "true"},
			}), nil
		}),
	}

	ctx := context.Background()
	if !client.Available(ctx) {
		t.Fatal("expected client to report available")
	}
	if err := client.SyncRepo(ctx, "main"); err != nil {
		t.Fatalf("SyncRepo returned error: %v", err)
	}
	if !seen["/api/v1/health"] || !seen["/api/v1/repos/main/sync"] {
		t.Fatalf("expected health and sync requests, got %+v", seen)
	}
}

func TestClientPreservesAPIErrorCode(t *testing.T) {
	client := New("http://127.0.0.1:5525")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusForbidden, httpapibiz.Response[map[string]string]{
				Code:    httpapibiz.CodeReadOnly,
				Message: "serve 未配置 secretKey，禁止将本地仓库推送至远端",
				Data:    map[string]string{},
			}), nil
		}),
	}

	_, err := client.PushSkillRepositoryChanges(context.Background(), httpapibiz.PushSkillRepositoryRequest{Confirm: true})
	if err == nil {
		t.Fatal("expected PushSkillRepositoryChanges to return error")
	}
	if got := apperrors.Code(err); got != apperrors.ErrorCode(httpapibiz.CodeReadOnly) {
		t.Fatalf("expected READ_ONLY code, got %q from %v", got, err)
	}
	if !strings.Contains(err.Error(), "serve 未配置 secretKey") {
		t.Fatalf("expected read-only message, got %v", err)
	}
}

func TestClientPreservesPayloadErrorCode(t *testing.T) {
	client := New("http://127.0.0.1:5525")
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, httpapibiz.Response[map[string]string]{
				Code:    httpapibiz.CodeUnauthorized,
				Message: "secretKey 无效或缺失",
				Data:    map[string]string{},
			}), nil
		}),
	}

	err := client.SyncRepo(context.Background(), "main")
	if err == nil {
		t.Fatal("expected SyncRepo to return error")
	}
	if got := apperrors.Code(err); got != apperrors.ErrorCode(httpapibiz.CodeUnauthorized) {
		t.Fatalf("expected UNAUTHORIZED code, got %q from %v", got, err)
	}
	if !strings.Contains(err.Error(), "secretKey 无效或缺失") {
		t.Fatalf("expected unauthorized message, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, payload any) *http.Response {
	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}
