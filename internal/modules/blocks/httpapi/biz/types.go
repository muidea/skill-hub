package biz

import (
	"github.com/muidea/skill-hub/internal/config"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"
	projectfeedbackservice "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback/service"
	projectinventoryservice "github.com/muidea/skill-hub/internal/modules/kernel/project_inventory/service"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	projectuseservice "github.com/muidea/skill-hub/internal/modules/kernel/project_use/service"
	"github.com/muidea/skill-hub/pkg/spec"
)

const CodeOK = "OK"

type Response[T any] struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
}

type HealthData struct {
	Status string `json:"status"`
}

type RepoListData struct {
	DefaultRepo string                    `json:"default_repo"`
	Items       []config.RepositoryConfig `json:"items"`
}

type AddRepoRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type SkillListData struct {
	Items []spec.SkillMetadata `json:"items"`
}

type RemoteSearchData struct {
	Items []spec.RemoteSearchResult `json:"items"`
}

type SkillCandidateListData struct {
	Items []spec.SkillMetadata `json:"items"`
}

type SkillDetailData struct {
	Item *spec.Skill `json:"item"`
}

type ProjectListData struct {
	Items []projectinventoryservice.ProjectSummary `json:"items"`
}

type ProjectDetailData struct {
	Item *projectinventoryservice.ProjectDetail `json:"item"`
}

type ProjectSkillListData struct {
	Items []projectinventoryservice.ProjectSkill `json:"items"`
}

type ProjectStatusData struct {
	Item *projectstatusservice.ProjectStatusSummary `json:"item"`
}

type SetProjectTargetRequest struct {
	ProjectPath string `json:"project_path"`
	Target      string `json:"target"`
}

type SetProjectTargetData struct {
	ProjectPath string `json:"project_path"`
	Target      string `json:"target"`
}

type UseSkillRequest struct {
	ProjectPath string            `json:"project_path"`
	SkillID     string            `json:"skill_id"`
	Repository  string            `json:"repository,omitempty"`
	Target      string            `json:"target,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type UseSkillData struct {
	Item *projectuseservice.UseResult `json:"item"`
}

type ApplyProjectRequest struct {
	ProjectPath string `json:"project_path"`
	DryRun      bool   `json:"dry_run"`
	Force       bool   `json:"force"`
}

type ApplyProjectData struct {
	Item *projectapplyservice.ApplyResult `json:"item"`
}

type FeedbackRequest struct {
	ProjectPath string `json:"project_path"`
	SkillID     string `json:"skill_id"`
}

type FeedbackPreviewData struct {
	Item *projectfeedbackservice.PreviewResult `json:"item"`
}
