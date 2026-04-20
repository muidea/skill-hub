package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/muidea/skill-hub/internal/engine"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Skill struct{}

type gitHubSearchResponse struct {
	TotalCount int                       `json:"total_count"`
	Items      []spec.RemoteSearchResult `json:"items"`
}

var remoteSearchHTTPClient = &http.Client{Timeout: 10 * time.Second}

func New() *Skill {
	return &Skill{}
}

func (s *Skill) Manager() (*engine.SkillManager, error) {
	return engine.NewSkillManager()
}

func (s *Skill) SkillsDir() (string, error) {
	return engine.GetSkillsDir()
}

func (s *Skill) SearchRemote(keyword string, limit int) ([]spec.RemoteSearchResult, error) {
	return searchGitHubRepositories(keyword, limit)
}

func searchGitHubRepositories(keyword string, limit int) ([]spec.RemoteSearchResult, error) {
	query := url.QueryEscape(keyword + " topic:agent-skills")
	requestURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=%d", query, limit)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("User-Agent", "skill-hub-cli")

	resp, err := remoteSearchHTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewWithCodef("searchGitHubRepositories", errors.ErrAPIRequest, "GitHub API返回错误: %s - %s", resp.Status, string(body))
	}

	var payload gitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, errors.Wrap(err, "解析响应失败")
	}

	return payload.Items, nil
}
