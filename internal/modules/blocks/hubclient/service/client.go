package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Client struct {
	baseURL    string
	secretKey  string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return NewWithSecret(baseURL, "")
}

func NewWithSecret(baseURL, secretKey string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		secretKey: strings.TrimSpace(secretKey),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return false
	}
	c.attachAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *Client) ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error) {
	data, err := get[httpapibiz.RepoListData](ctx, c, "/api/v1/repos")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) AddRepo(ctx context.Context, repo config.RepositoryConfig) error {
	req := httpapibiz.AddRepoRequest{
		Name:        repo.Name,
		URL:         repo.URL,
		Branch:      repo.Branch,
		Type:        repo.Type,
		Description: repo.Description,
	}
	_, err := post[map[string]string](ctx, c, "/api/v1/repos", req)
	return err
}

func (c *Client) RemoveRepo(ctx context.Context, name string) error {
	_, err := del[map[string]string](ctx, c, "/api/v1/repos/"+url.PathEscape(name))
	return err
}

func (c *Client) SyncRepo(ctx context.Context, name string) error {
	_, err := post[map[string]string](ctx, c, "/api/v1/repos/"+url.PathEscape(name)+"/sync", nil)
	return err
}

func (c *Client) EnableRepo(ctx context.Context, name string) error {
	_, err := post[map[string]string](ctx, c, "/api/v1/repos/"+url.PathEscape(name)+"/enable", nil)
	return err
}

func (c *Client) DisableRepo(ctx context.Context, name string) error {
	_, err := post[map[string]string](ctx, c, "/api/v1/repos/"+url.PathEscape(name)+"/disable", nil)
	return err
}

func (c *Client) SetDefaultRepo(ctx context.Context, name string) error {
	_, err := post[map[string]string](ctx, c, "/api/v1/repos/"+url.PathEscape(name)+"/set-default", nil)
	return err
}

func (c *Client) SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
	data, err := get[httpapibiz.SkillRepositoryStatusData](ctx, c, "/api/v1/skill-repository/status")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error) {
	data, err := get[httpapibiz.SkillRepositoryCheckData](ctx, c, "/api/v1/skill-repository/sync-check")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
	data, err := post[httpapibiz.SyncSkillRepositoryData](ctx, c, "/api/v1/skill-repository/sync", nil)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error) {
	data, err := get[httpapibiz.PushSkillRepositoryPreviewData](ctx, c, "/api/v1/skill-repository/push-preview")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error) {
	data, err := post[httpapibiz.PushSkillRepositoryData](ctx, c, "/api/v1/skill-repository/push", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) ListSkills(ctx context.Context, repoNames []string, target string) ([]spec.SkillMetadata, error) {
	query := url.Values{}
	if len(repoNames) > 0 {
		query.Set("repo", strings.Join(repoNames, ","))
	}
	if target != "" {
		query.Set("target", target)
	}
	path := "/api/v1/skills"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	data, err := get[httpapibiz.SkillListData](ctx, c, path)
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

func (c *Client) SearchRemoteSkills(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error) {
	query := url.Values{}
	query.Set("keyword", keyword)
	if target != "" {
		query.Set("target", target)
	}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	data, err := get[httpapibiz.RemoteSearchData](ctx, c, "/api/v1/search?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

func (c *Client) GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
	query := url.Values{}
	query.Set("path", projectPath)
	if skillID != "" {
		query.Set("skill_id", skillID)
	}
	data, err := get[httpapibiz.ProjectStatusData](ctx, c, "/api/v1/project-status?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) SetProjectTarget(ctx context.Context, req httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error) {
	data, err := post[httpapibiz.SetProjectTargetData](ctx, c, "/api/v1/project-target", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error) {
	data, err := get[httpapibiz.SkillCandidateListData](ctx, c, "/api/v1/skills/"+url.PathEscape(skillID)+"/candidates")
	if err != nil {
		return nil, err
	}
	return data.Items, nil
}

func (c *Client) GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error) {
	query := url.Values{}
	query.Set("repo", repoName)
	data, err := get[httpapibiz.SkillDetailData](ctx, c, "/api/v1/skills/"+url.PathEscape(skillID)+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	return data.Item, nil
}

func (c *Client) UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
	data, err := post[httpapibiz.UseSkillData](ctx, c, "/api/v1/project-skills/use", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error) {
	data, err := post[httpapibiz.RegisterSkillData](ctx, c, "/api/v1/project-skills/register", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error) {
	data, err := post[httpapibiz.ImportSkillsData](ctx, c, "/api/v1/project-skills/import", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error) {
	data, err := post[httpapibiz.DedupeData](ctx, c, "/api/v1/project-skills/dedupe", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error) {
	data, err := post[httpapibiz.SyncCopiesData](ctx, c, "/api/v1/project-skills/sync-copies", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error) {
	data, err := post[httpapibiz.PathLintData](ctx, c, "/api/v1/project-skills/lint-paths", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error) {
	data, err := post[httpapibiz.ValidateProjectSkillsData](ctx, c, "/api/v1/project-skills/validate", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error) {
	data, err := post[httpapibiz.AuditProjectSkillsData](ctx, c, "/api/v1/project-skills/audit", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
	data, err := post[httpapibiz.ApplyProjectData](ctx, c, "/api/v1/project-apply", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	data, err := post[httpapibiz.FeedbackPreviewData](ctx, c, "/api/v1/project-feedback/preview", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	data, err := post[httpapibiz.FeedbackPreviewData](ctx, c, "/api/v1/project-feedback/apply", req)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func get[T any](ctx context.Context, c *Client, path string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return zero, err
	}
	c.attachAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	return decode[T](resp)
}

func post[T any](ctx context.Context, c *Client, path string, body any) (T, error) {
	var zero T
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return zero, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reader)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.attachAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	return decode[T](resp)
}

func del[T any](ctx context.Context, c *Client, path string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return zero, err
	}
	c.attachAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	return decode[T](resp)
}

func (c *Client) attachAuth(req *http.Request) {
	if c.secretKey != "" {
		req.Header.Set(httpapibiz.SecretKeyHeader, c.secretKey)
	}
}

func decode[T any](resp *http.Response) (T, error) {
	var zero T
	if resp.StatusCode >= 400 {
		return zero, decodeAPIError(resp, fmt.Sprintf("request failed with status %d", resp.StatusCode))
	}

	var payload httpapibiz.Response[T]
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return zero, err
	}
	if payload.Code != httpapibiz.CodeOK {
		return zero, apiError(payload.Code, payload.Message, "request failed")
	}
	return payload.Data, nil
}

func decodeAPIError(resp *http.Response, fallback string) error {
	var errResp httpapibiz.Response[json.RawMessage]
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
		return apiError(errResp.Code, errResp.Message, fallback)
	}
	return apiError("", "", fallback)
}

func apiError(code, message, fallback string) error {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	if message == "" {
		message = fallback
	}
	if code == "" {
		return fmt.Errorf("%s", message)
	}
	return apperrors.NewWithCode("hubclient", apperrors.ErrorCode(code), message)
}
