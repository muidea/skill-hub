package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"
	projectfeedbackservice "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback/service"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	projectuseservice "github.com/muidea/skill-hub/internal/modules/kernel/project_use/service"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestRunListViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		listSkillsFn: func(ctx context.Context, repoNames []string, target string) ([]spec.SkillMetadata, error) {
			return []spec.SkillMetadata{{ID: "demo-skill", Name: "Demo Skill", Version: "1.0.0", Repository: "main", Compatibility: "open_code"}}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runList("", false, nil); err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
	})
	if !strings.Contains(output, "demo-skill") {
		t.Fatalf("expected output to contain skill id, got %q", output)
	}
}

func TestRunStatusViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		getProjectStatusFn: func(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
			return &httpapibiz.ProjectStatusData{
				Item: &projectstatusservice.ProjectStatusSummary{
					ProjectPath: projectPath,
					SkillCount:  1,
					Items: []projectstatusservice.SkillStatusItem{
						{SkillID: "demo-skill", Status: spec.SkillStatusSynced, LocalVersion: "1.0.0", RepoVersion: "1.0.0"},
					},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runStatus("", false); err != nil {
				t.Fatalf("runStatus returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "demo-skill") || !strings.Contains(output, "Synced") {
		t.Fatalf("unexpected status output: %q", output)
	}
}

func TestRunUseViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		findSkillCandidatesFn: func(ctx context.Context, skillID string) ([]spec.SkillMetadata, error) {
			return []spec.SkillMetadata{{ID: skillID, Name: "Demo Skill", Repository: "main"}}, nil
		},
		getSkillDetailFn: func(ctx context.Context, skillID, repoName string) (*spec.Skill, error) {
			return &spec.Skill{ID: skillID, Name: "Demo Skill", Repository: repoName, Version: "1.0.0"}, nil
		},
		getProjectStatusFn: func(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
			return &httpapibiz.ProjectStatusData{Item: &projectstatusservice.ProjectStatusSummary{ProjectPath: projectPath, SkillCount: 0}}, nil
		},
		useSkillFn: func(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
			return &httpapibiz.UseSkillData{Item: &projectuseservice.UseResult{ProjectPath: req.ProjectPath, SkillID: req.SkillID, Repository: "main", Target: req.Target}}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runUse("demo-skill", spec.TargetOpenCode); err != nil {
				t.Fatalf("runUse returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "已成功标记为使用") {
		t.Fatalf("unexpected use output: %q", output)
	}
}

func TestRunApplyViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		applyProjectFn: func(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
			return &httpapibiz.ApplyProjectData{
				Item: &projectapplyservice.ApplyResult{
					ProjectPath: req.ProjectPath,
					Target:      spec.TargetOpenCode,
					DryRun:      true,
					Items: []projectapplyservice.ApplyResultItem{
						{SkillID: "demo-skill", Status: "planned", Variables: 1},
					},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runApply(true, false); err != nil {
				t.Fatalf("runApply returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "演习模式") {
		t.Fatalf("unexpected apply output: %q", output)
	}
}

func TestRunFeedbackViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	feedbackForce = true
	t.Cleanup(func() { feedbackForce = false })

	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		previewFeedbackFn: func(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
			return &httpapibiz.FeedbackPreviewData{
				Item: &projectfeedbackservice.PreviewResult{
					ProjectPath:      req.ProjectPath,
					SkillID:          req.SkillID,
					DefaultRepo:      "main",
					ProjectVersion:   "1.0.0",
					RepoVersion:      "1.0.0",
					ResolvedVersion:  "1.0.1",
					NeedsVersionBump: true,
					Changes:          []string{"修改: SKILL.md"},
				},
			}, nil
		},
		applyFeedbackFn: func(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
			return &httpapibiz.FeedbackPreviewData{
				Item: &projectfeedbackservice.PreviewResult{
					ProjectPath: req.ProjectPath,
					SkillID:     req.SkillID,
					DefaultRepo: "main",
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runFeedback("demo-skill"); err != nil {
				t.Fatalf("runFeedback returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "反馈完成") {
		t.Fatalf("unexpected feedback output: %q", output)
	}
}

func TestRunSetTargetViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		setProjectTargetFn: func(ctx context.Context, req httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error) {
			return &httpapibiz.SetProjectTargetData{
				ProjectPath: req.ProjectPath,
				Target:      req.Target,
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runSetTarget(spec.TargetOpenCode); err != nil {
				t.Fatalf("runSetTarget returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "首选兼容目标设置为") {
		t.Fatalf("unexpected set-target output: %q", output)
	}
}

func TestRunSearchViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		searchRemoteSkillsFn: func(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error) {
			return []spec.RemoteSearchResult{
				{FullName: "demo/search-skill", Description: "demo", HTMLURL: "https://example.com/demo"},
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runSearch("demo", spec.TargetOpenCode, 5); err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
	})
	if !strings.Contains(output, "demo/search-skill") {
		t.Fatalf("unexpected search output: %q", output)
	}
}

func TestRunSearchViaServiceFailureIsGraceful(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		searchRemoteSkillsFn: func(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error) {
			return nil, errors.New("service unavailable")
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runSearch("demo", spec.TargetOpenCode, 5); err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
	})
	if !strings.Contains(output, "本地服务搜索失败") {
		t.Fatalf("unexpected search output: %q", output)
	}
}

func stubServiceBridge(t *testing.T, client serviceBridgeClient) func() {
	t.Helper()
	prev := serviceBridgeResolver
	serviceBridgeResolver = func() (serviceBridgeClient, bool) {
		return client, true
	}
	return func() {
		serviceBridgeResolver = prev
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func withWorkingDir(t *testing.T, dir string, fn func() string) string {
	t.Helper()
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(oldCwd) }()
	return fn()
}

type fakeServiceBridgeClient struct {
	listReposFn           func(context.Context) (*httpapibiz.RepoListData, error)
	addRepoFn             func(context.Context, config.RepositoryConfig) error
	removeRepoFn          func(context.Context, string) error
	syncRepoFn            func(context.Context, string) error
	enableRepoFn          func(context.Context, string) error
	disableRepoFn         func(context.Context, string) error
	setDefaultRepoFn      func(context.Context, string) error
	listSkillsFn          func(context.Context, []string, string) ([]spec.SkillMetadata, error)
	searchRemoteSkillsFn  func(context.Context, string, string, int) ([]spec.RemoteSearchResult, error)
	getProjectStatusFn    func(context.Context, string, string) (*httpapibiz.ProjectStatusData, error)
	setProjectTargetFn    func(context.Context, httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error)
	findSkillCandidatesFn func(context.Context, string) ([]spec.SkillMetadata, error)
	getSkillDetailFn      func(context.Context, string, string) (*spec.Skill, error)
	useSkillFn            func(context.Context, httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
	applyProjectFn        func(context.Context, httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
	previewFeedbackFn     func(context.Context, httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
	applyFeedbackFn       func(context.Context, httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
}

func (f *fakeServiceBridgeClient) Available(ctx context.Context) bool { return true }
func (f *fakeServiceBridgeClient) ListRepos(ctx context.Context) (*httpapibiz.RepoListData, error) {
	if f.listReposFn != nil {
		return f.listReposFn(ctx)
	}
	return &httpapibiz.RepoListData{}, nil
}
func (f *fakeServiceBridgeClient) AddRepo(ctx context.Context, repo config.RepositoryConfig) error {
	if f.addRepoFn != nil {
		return f.addRepoFn(ctx, repo)
	}
	return nil
}
func (f *fakeServiceBridgeClient) RemoveRepo(ctx context.Context, name string) error {
	if f.removeRepoFn != nil {
		return f.removeRepoFn(ctx, name)
	}
	return nil
}
func (f *fakeServiceBridgeClient) SyncRepo(ctx context.Context, name string) error {
	if f.syncRepoFn != nil {
		return f.syncRepoFn(ctx, name)
	}
	return nil
}
func (f *fakeServiceBridgeClient) EnableRepo(ctx context.Context, name string) error {
	if f.enableRepoFn != nil {
		return f.enableRepoFn(ctx, name)
	}
	return nil
}
func (f *fakeServiceBridgeClient) DisableRepo(ctx context.Context, name string) error {
	if f.disableRepoFn != nil {
		return f.disableRepoFn(ctx, name)
	}
	return nil
}
func (f *fakeServiceBridgeClient) SetDefaultRepo(ctx context.Context, name string) error {
	if f.setDefaultRepoFn != nil {
		return f.setDefaultRepoFn(ctx, name)
	}
	return nil
}
func (f *fakeServiceBridgeClient) ListSkills(ctx context.Context, repoNames []string, target string) ([]spec.SkillMetadata, error) {
	if f.listSkillsFn != nil {
		return f.listSkillsFn(ctx, repoNames, target)
	}
	return nil, nil
}
func (f *fakeServiceBridgeClient) SearchRemoteSkills(ctx context.Context, keyword, target string, limit int) ([]spec.RemoteSearchResult, error) {
	if f.searchRemoteSkillsFn != nil {
		return f.searchRemoteSkillsFn(ctx, keyword, target, limit)
	}
	return nil, nil
}
func (f *fakeServiceBridgeClient) GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
	if f.getProjectStatusFn != nil {
		return f.getProjectStatusFn(ctx, projectPath, skillID)
	}
	return &httpapibiz.ProjectStatusData{}, nil
}
func (f *fakeServiceBridgeClient) SetProjectTarget(ctx context.Context, req httpapibiz.SetProjectTargetRequest) (*httpapibiz.SetProjectTargetData, error) {
	if f.setProjectTargetFn != nil {
		return f.setProjectTargetFn(ctx, req)
	}
	return &httpapibiz.SetProjectTargetData{}, nil
}
func (f *fakeServiceBridgeClient) FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error) {
	if f.findSkillCandidatesFn != nil {
		return f.findSkillCandidatesFn(ctx, skillID)
	}
	return nil, nil
}
func (f *fakeServiceBridgeClient) GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error) {
	if f.getSkillDetailFn != nil {
		return f.getSkillDetailFn(ctx, skillID, repoName)
	}
	return &spec.Skill{}, nil
}
func (f *fakeServiceBridgeClient) UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error) {
	if f.useSkillFn != nil {
		return f.useSkillFn(ctx, req)
	}
	return &httpapibiz.UseSkillData{}, nil
}
func (f *fakeServiceBridgeClient) ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
	if f.applyProjectFn != nil {
		return f.applyProjectFn(ctx, req)
	}
	return &httpapibiz.ApplyProjectData{}, nil
}
func (f *fakeServiceBridgeClient) PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	if f.previewFeedbackFn != nil {
		return f.previewFeedbackFn(ctx, req)
	}
	return &httpapibiz.FeedbackPreviewData{}, nil
}
func (f *fakeServiceBridgeClient) ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error) {
	if f.applyFeedbackFn != nil {
		return f.applyFeedbackFn(ctx, req)
	}
	return &httpapibiz.FeedbackPreviewData{}, nil
}
