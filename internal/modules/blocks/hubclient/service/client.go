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
	"github.com/muidea/skill-hub/pkg/spec"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 1200 * time.Millisecond,
		},
	}
}

func (c *Client) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/health", nil)
	if err != nil {
		return false
	}
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
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	return decode[T](resp)
}

func decode[T any](resp *http.Response) (T, error) {
	var zero T
	if resp.StatusCode >= 400 {
		var errResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if msg, ok := errResp["message"].(string); ok && msg != "" {
				return zero, fmt.Errorf("%s", msg)
			}
		}
		return zero, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var payload httpapibiz.Response[T]
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return zero, err
	}
	if payload.Code != httpapibiz.CodeOK {
		if payload.Message != "" {
			return zero, fmt.Errorf("%s", payload.Message)
		}
		return zero, fmt.Errorf("request failed")
	}
	return payload.Data, nil
}
