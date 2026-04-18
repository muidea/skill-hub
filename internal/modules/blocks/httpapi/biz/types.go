package biz

import (
	"github.com/muidea/skill-hub/internal/config"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"
	projectfeedbackservice "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback/service"
	projectinventoryservice "github.com/muidea/skill-hub/internal/modules/kernel/project_inventory/service"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
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

type SkillRepositoryStatusData struct {
	Status string `json:"status"`
}

type SyncSkillRepositoryData struct {
	Status     string `json:"status"`
	SkillCount int    `json:"skill_count"`
}

type SkillRepositoryCheckData struct {
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
	RemoteURL    string `json:"remote_url,omitempty"`
	LocalCommit  string `json:"local_commit,omitempty"`
	RemoteCommit string `json:"remote_commit,omitempty"`
	HasUpdates   bool   `json:"has_updates"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
}

type PushSkillRepositoryPreviewData struct {
	DefaultRepo      string   `json:"default_repo,omitempty"`
	RemoteURL        string   `json:"remote_url,omitempty"`
	HasChanges       bool     `json:"has_changes"`
	ChangedFiles     []string `json:"changed_files"`
	SuggestedMessage string   `json:"suggested_message"`
	RawStatus        string   `json:"raw_status"`
}

type PushSkillRepositoryRequest struct {
	Message              string   `json:"message,omitempty"`
	Confirm              bool     `json:"confirm"`
	ExpectedChangedFiles []string `json:"expected_changed_files,omitempty"`
}

type PushSkillRepositoryData struct {
	Status       string   `json:"status"`
	Message      string   `json:"message,omitempty"`
	ChangedFiles []string `json:"changed_files,omitempty"`
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

type RegisterSkillRequest struct {
	ProjectPath  string `json:"project_path"`
	SkillID      string `json:"skill_id"`
	Target       string `json:"target,omitempty"`
	SkipValidate bool   `json:"skip_validate"`
}

type RegisterSkillData struct {
	Item *projectlifecycleservice.RegisterResult `json:"item"`
}

type ImportSkillsRequest struct {
	ProjectPath string                                `json:"project_path"`
	SkillsDir   string                                `json:"skills_dir"`
	Options     projectlifecycleservice.ImportOptions `json:"options"`
}

type ImportSkillsData struct {
	Item *projectlifecycleservice.ImportSummary `json:"item"`
}

type DedupeRequest struct {
	Scope   string                                `json:"scope"`
	Options projectlifecycleservice.DedupeOptions `json:"options"`
}

type DedupeData struct {
	Item *projectlifecycleservice.DuplicateReport `json:"item"`
}

type SyncCopiesRequest struct {
	Options projectlifecycleservice.SyncCopiesOptions `json:"options"`
}

type SyncCopiesData struct {
	Item *projectlifecycleservice.SyncCopiesResult `json:"item"`
}

type PathLintRequest struct {
	Options projectlifecycleservice.PathLintOptions `json:"options"`
}

type PathLintData struct {
	Item *projectlifecycleservice.PathLintReport `json:"item"`
}

type ValidateProjectSkillsRequest struct {
	Options projectlifecycleservice.ValidateOptions `json:"options"`
}

type ValidateProjectSkillsData struct {
	Item *projectlifecycleservice.ValidateReport `json:"item"`
}

type AuditProjectSkillsRequest struct {
	Options projectlifecycleservice.AuditOptions `json:"options"`
}

type AuditProjectSkillsData struct {
	Item *projectlifecycleservice.AuditReport `json:"item"`
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
