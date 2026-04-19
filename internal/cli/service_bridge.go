package cli

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	hubclientmodule "github.com/muidea/skill-hub/internal/modules/blocks/hubclient"
	"github.com/muidea/skill-hub/pkg/spec"
)

const defaultServiceURL = "http://127.0.0.1:5525"

type serviceBridgeClient interface {
	Available(ctx context.Context) bool
	ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error)
	AddRepo(ctx context.Context, repo config.RepositoryConfig) error
	RemoveRepo(ctx context.Context, name string) error
	SyncRepo(ctx context.Context, name string) error
	EnableRepo(ctx context.Context, name string) error
	DisableRepo(ctx context.Context, name string) error
	SetDefaultRepo(ctx context.Context, name string) error
	SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error)
	CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error)
	SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error)
	PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error)
	PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error)
	ListSkills(ctx context.Context, repoNames []string, target string) ([]spec.SkillMetadata, error)
	SearchRemoteSkills(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error)
	GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error)
	SetProjectTarget(ctx context.Context, req httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error)
	FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error)
	GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error)
	UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
	RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error)
	ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error)
	DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error)
	SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error)
	LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error)
	ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error)
	AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error)
	ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
	PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
	ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
}

type hubBridgeClient struct {
	client *hubclientmodule.HubClient
}

func (h *hubBridgeClient) Available(ctx context.Context) bool {
	return h.client.Service().Available(ctx)
}

func (h *hubBridgeClient) ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error) {
	return h.client.Service().ListRepos(ctx)
}

func (h *hubBridgeClient) AddRepo(ctx context.Context, repo config.RepositoryConfig) error {
	return h.client.Service().AddRepo(ctx, repo)
}

func (h *hubBridgeClient) RemoveRepo(ctx context.Context, name string) error {
	return h.client.Service().RemoveRepo(ctx, name)
}

func (h *hubBridgeClient) SyncRepo(ctx context.Context, name string) error {
	return h.client.Service().SyncRepo(ctx, name)
}

func (h *hubBridgeClient) EnableRepo(ctx context.Context, name string) error {
	return h.client.Service().EnableRepo(ctx, name)
}

func (h *hubBridgeClient) DisableRepo(ctx context.Context, name string) error {
	return h.client.Service().DisableRepo(ctx, name)
}

func (h *hubBridgeClient) SetDefaultRepo(ctx context.Context, name string) error {
	return h.client.Service().SetDefaultRepo(ctx, name)
}

func (h *hubBridgeClient) SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
	return h.client.Service().SkillRepositoryStatus(ctx)
}

func (h *hubBridgeClient) CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error) {
	return h.client.Service().CheckSkillRepositoryUpdates(ctx)
}

func (h *hubBridgeClient) SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
	return h.client.Service().SyncSkillRepositoryAndRefresh(ctx)
}

func (h *hubBridgeClient) PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error) {
	return h.client.Service().PushSkillRepositoryPreview(ctx)
}

func (h *hubBridgeClient) PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error) {
	return h.client.Service().PushSkillRepositoryChanges(ctx, req)
}

func (h *hubBridgeClient) ListSkills(ctx context.Context, repoNames []string, target string) ([]spec.SkillMetadata, error) {
	return h.client.Service().ListSkills(ctx, repoNames, target)
}

func (h *hubBridgeClient) SearchRemoteSkills(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error) {
	return h.client.Service().SearchRemoteSkills(ctx, keyword, target, limit)
}

func (h *hubBridgeClient) GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
	return h.client.Service().GetProjectStatus(ctx, projectPath, skillID)
}

func (h *hubBridgeClient) SetProjectTarget(ctx context.Context, req httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error) {
	return h.client.Service().SetProjectTarget(ctx, req)
}

func (h *hubBridgeClient) FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error) {
	return h.client.Service().FindSkillCandidates(ctx, skillID)
}

func (h *hubBridgeClient) GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error) {
	return h.client.Service().GetSkillDetail(ctx, skillID, repoName)
}

func (h *hubBridgeClient) UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
	return h.client.Service().UseSkill(ctx, req)
}

func (h *hubBridgeClient) RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error) {
	return h.client.Service().RegisterSkill(ctx, req)
}

func (h *hubBridgeClient) ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error) {
	return h.client.Service().ImportSkills(ctx, req)
}

func (h *hubBridgeClient) DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error) {
	return h.client.Service().DedupeSkills(ctx, req)
}

func (h *hubBridgeClient) SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error) {
	return h.client.Service().SyncCopies(ctx, req)
}

func (h *hubBridgeClient) LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error) {
	return h.client.Service().LintPaths(ctx, req)
}

func (h *hubBridgeClient) ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error) {
	return h.client.Service().ValidateProjectSkills(ctx, req)
}

func (h *hubBridgeClient) AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error) {
	return h.client.Service().AuditProjectSkills(ctx, req)
}

func (h *hubBridgeClient) ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
	return h.client.Service().ApplyProject(ctx, req)
}

func (h *hubBridgeClient) PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	return h.client.Service().PreviewFeedback(ctx, req)
}

func (h *hubBridgeClient) ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	return h.client.Service().ApplyFeedback(ctx, req)
}

var serviceBridgeResolver = resolveServiceBridgeClient

func serviceBridgeEnabled() bool {
	value := strings.TrimSpace(os.Getenv("SKILL_HUB_DISABLE_SERVICE_BRIDGE"))
	return value == "" || value == "0" || strings.EqualFold(value, "false")
}

func serviceBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("SKILL_HUB_SERVICE_URL")); value != "" {
		return value
	}
	return defaultServiceURL
}

func serviceSecretKey() string {
	return strings.TrimSpace(os.Getenv("SKILL_HUB_SERVICE_SECRET_KEY"))
}

func hubClientIfAvailable() (serviceBridgeClient, bool) {
	return serviceBridgeResolver()
}

func resolveServiceBridgeClient() (serviceBridgeClient, bool) {
	if !serviceBridgeEnabled() {
		return nil, false
	}

	client := hubclientmodule.NewWithSecret(serviceBaseURL(), serviceSecretKey())
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	bridgeClient := &hubBridgeClient{client: client}
	if !bridgeClient.Available(ctx) {
		return nil, false
	}
	return bridgeClient, true
}
