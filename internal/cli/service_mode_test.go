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
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	projectuseservice "github.com/muidea/skill-hub/internal/modules/kernel/project_use/service"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestRunListViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		listSkillsFn: func(ctx context.Context, repoNames []string) ([]spec.SkillMetadata, error) {
			return []spec.SkillMetadata{{ID: "demo-skill", Name: "Demo Skill", Version: "1.0.0", Repository: "main", Compatibility: "open_code"}}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runList(false, nil); err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
	})
	if !strings.Contains(output, "demo-skill") {
		t.Fatalf("expected output to contain skill id, got %q", output)
	}
}

func TestRunRepoListJSONViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		listReposFn: func(ctx context.Context) (*httpapibiz.RepoListData, error) {
			return &httpapibiz.RepoListData{
				DefaultRepo: "main",
				Items: []config.RepositoryConfig{
					{Name: "main", Type: "user", Enabled: true, IsArchive: true},
				},
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runRepoList(true); err != nil {
			t.Fatalf("runRepoList returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"default_repo": "main"`) || !strings.Contains(output, `"name": "main"`) {
		t.Fatalf("unexpected repo list json output: %q", output)
	}
}

func TestRunRepoSyncJSONViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		listReposFn: func(ctx context.Context) (*httpapibiz.RepoListData, error) {
			return &httpapibiz.RepoListData{
				DefaultRepo: "main",
				Items: []config.RepositoryConfig{
					{Name: "main", Enabled: true},
					{Name: "disabled", Enabled: false},
				},
			}, nil
		},
		syncRepoFn: func(ctx context.Context, name string) error {
			if name == "main" {
				return nil
			}
			return errors.New("unexpected repo sync")
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runRepoSync(nil, false, true); err != nil {
			t.Fatalf("runRepoSync returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"synced": 1`) || !strings.Contains(output, `"skipped": 1`) {
		t.Fatalf("unexpected repo sync json output: %q", output)
	}
}

func TestRunPushDryRunJSONViaServiceWithoutLocalConfig(t *testing.T) {
	pushDryRun = true
	pushJSON = true
	t.Cleanup(func() {
		pushDryRun = false
		pushJSON = false
		pushForce = false
		pushMessage = ""
	})

	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		skillRepositoryStatusFn: func(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
			return &httpapibiz.SkillRepositoryStatusData{
				Status: "技能仓库状态:\n文件状态:\n?? skills/demo/SKILL.md\n",
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runPush(); err != nil {
			t.Fatalf("runPush returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"status": "planned"`) || !strings.Contains(output, "skills/demo/SKILL.md") {
		t.Fatalf("unexpected push json output: %q", output)
	}
}

func TestRunPushJSONViaServiceRequiresConfirmedRequest(t *testing.T) {
	pushForce = true
	pushJSON = true
	t.Cleanup(func() {
		pushDryRun = false
		pushJSON = false
		pushForce = false
		pushMessage = ""
	})

	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		skillRepositoryStatusFn: func(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
			return &httpapibiz.SkillRepositoryStatusData{
				Status: "技能仓库状态:\n文件状态:\n?? skills/demo/SKILL.md\n",
			}, nil
		},
		pushSkillRepositoryFn: func(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error) {
			if !req.Confirm {
				t.Fatalf("expected confirmed push request")
			}
			if len(req.ExpectedChangedFiles) != 1 || req.ExpectedChangedFiles[0] != "skills/demo/SKILL.md" {
				t.Fatalf("unexpected expected changed files: %#v", req.ExpectedChangedFiles)
			}
			return &httpapibiz.PushSkillRepositoryData{Status: "pushed", Message: req.Message, ChangedFiles: req.ExpectedChangedFiles}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runPush(); err != nil {
			t.Fatalf("runPush returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"status": "pushed"`) {
		t.Fatalf("unexpected push json output: %q", output)
	}
}

func TestRunGitStatusJSONViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		skillRepositoryStatusFn: func(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
			return &httpapibiz.SkillRepositoryStatusData{
				Status: "技能仓库状态:\n文件状态:\n M  skills/demo/SKILL.md\n",
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runGitStatusWithOptions(true); err != nil {
			t.Fatalf("runGitStatusWithOptions returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"state": "dirty"`) || !strings.Contains(output, "skills/demo/SKILL.md") {
		t.Fatalf("unexpected git status json output: %q", output)
	}
}

func TestRunGitSyncJSONViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		syncSkillRepositoryFn: func(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
			return &httpapibiz.SyncSkillRepositoryData{Status: "synced", SkillCount: 3}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runGitSyncWithOptions(true); err != nil {
			t.Fatalf("runGitSyncWithOptions returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"command": "sync"`) || !strings.Contains(output, `"skill_count": 3`) {
		t.Fatalf("unexpected git sync json output: %q", output)
	}
}

func TestRunGitPullJSONViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		syncSkillRepositoryFn: func(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
			return &httpapibiz.SyncSkillRepositoryData{Status: "synced", SkillCount: 4}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runGitPullWithOptions(true); err != nil {
			t.Fatalf("runGitPullWithOptions returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"command": "pull"`) || !strings.Contains(output, `"skill_count": 4`) {
		t.Fatalf("unexpected git pull json output: %q", output)
	}
}

func TestRunPullCheckJSONViaServiceWithoutLocalConfig(t *testing.T) {
	pullCheck = true
	pullJSON = true
	t.Cleanup(func() {
		pullCheck = false
		pullJSON = false
		pullForce = false
	})

	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		checkSkillRepositoryFn: func(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error) {
			return &httpapibiz.SkillRepositoryCheckData{
				Status:       "updates_available",
				Message:      "远程存在可拉取更新",
				RemoteURL:    "https://example.com/skills.git",
				LocalCommit:  "1111111111111111111111111111111111111111",
				RemoteCommit: "2222222222222222222222222222222222222222",
				HasUpdates:   true,
				Behind:       1,
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runPull(); err != nil {
			t.Fatalf("runPull returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"status": "updates_available"`) || !strings.Contains(output, `"behind": 1`) {
		t.Fatalf("unexpected pull check json output: %q", output)
	}
}

func TestRunPullJSONViaServiceWithoutLocalConfig(t *testing.T) {
	pullJSON = true
	t.Cleanup(func() {
		pullCheck = false
		pullJSON = false
		pullForce = false
	})

	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		syncSkillRepositoryFn: func(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
			return &httpapibiz.SyncSkillRepositoryData{Status: "synced", SkillCount: 2}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runPull(); err != nil {
			t.Fatalf("runPull returned error: %v", err)
		}
	})
	if !strings.Contains(output, `"status": "synced"`) || !strings.Contains(output, `"skill_count": 2`) {
		t.Fatalf("unexpected pull json output: %q", output)
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
			return &httpapibiz.UseSkillData{Item: &projectuseservice.UseResult{ProjectPath: req.ProjectPath, SkillID: req.SkillID, Repository: "main"}}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runUse("demo-skill"); err != nil {
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
			if req.SkillID != "demo-skill" {
				t.Fatalf("SkillID = %q, want demo-skill", req.SkillID)
			}
			return &httpapibiz.ApplyProjectData{
				Item: &projectapplyservice.ApplyResult{
					ProjectPath: req.ProjectPath,
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
			if err := runApply("demo-skill", true, false); err != nil {
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

func TestRunRegisterViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		registerSkillFn: func(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error) {
			return &httpapibiz.RegisterSkillData{
				Item: &projectlifecycleservice.RegisterResult{
					ProjectPath: req.ProjectPath,
					SkillID:     req.SkillID,
					Version:     "1.0.0",
					Registered:  true,
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			if err := runRegister("demo-skill", false); err != nil {
				t.Fatalf("runRegister returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "已登记到项目状态") {
		t.Fatalf("unexpected register output: %q", output)
	}
}

func TestRunImportViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		importSkillsFn: func(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error) {
			return &httpapibiz.ImportSkillsData{
				Item: &projectlifecycleservice.ImportSummary{
					ProjectPath: req.ProjectPath,
					SkillsDir:   req.SkillsDir,
					Discovered:  2,
					Registered:  2,
					Valid:       2,
					Archived:    2,
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runImport(".agents/skills", projectlifecycleservice.ImportOptions{
				Archive: true,
				Force:   true,
			})
			if err != nil {
				t.Fatalf("runImport returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "discovered: 2") || !strings.Contains(output, "archived:   2") {
		t.Fatalf("unexpected import output: %q", output)
	}
}

func TestRunDedupeViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		dedupeSkillsFn: func(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error) {
			return &httpapibiz.DedupeData{
				Item: &projectlifecycleservice.DuplicateReport{
					Scope:      req.Scope,
					Canonical:  req.Options.Canonical,
					Strategy:   req.Options.Strategy,
					SkillCount: 1,
					Groups: []projectlifecycleservice.DuplicateGroup{
						{SkillID: "demo-skill", ContentDiffers: true},
					},
					Conflicts: 1,
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runDedupe(".", projectlifecycleservice.DedupeOptions{
				Canonical: ".agents/skills",
				Strategy:  "newest",
			}, false)
			if err != nil {
				t.Fatalf("runDedupe returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "demo-skill") || !strings.Contains(output, "冲突数: 1") {
		t.Fatalf("unexpected dedupe output: %q", output)
	}
}

func TestRunSyncCopiesViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		syncCopiesFn: func(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error) {
			return &httpapibiz.SyncCopiesData{
				Item: &projectlifecycleservice.SyncCopiesResult{
					Scope:     req.Options.Scope,
					Canonical: req.Options.Canonical,
					Synced:    1,
					Items: []projectlifecycleservice.SyncCopiesItem{
						{SkillID: "demo-skill", SourceDir: "canonical", TargetDir: "copy", Status: "synced"},
					},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runSyncCopies(projectlifecycleservice.SyncCopiesOptions{
				Scope:     ".",
				Canonical: ".agents/skills",
			}, false)
			if err != nil {
				t.Fatalf("runSyncCopies returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "synced:    1") || !strings.Contains(output, "[synced]") {
		t.Fatalf("unexpected sync-copies output: %q", output)
	}
}

func TestRunLintPathsViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		lintPathsFn: func(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error) {
			return &httpapibiz.PathLintData{
				Item: &projectlifecycleservice.PathLintReport{
					Scope:        req.Options.Scope,
					ProjectRoot:  req.Options.ProjectRoot,
					FilesScanned: 1,
					FindingCount: 1,
					Findings: []projectlifecycleservice.PathLintFinding{
						{File: req.Options.Scope + "/.agents/skills/demo/SKILL.md", Line: 8, Kind: "absolute-path", Value: "/home/tester/workspace/docs/a.md", Replacement: "docs/a.md", Status: "fixable"},
					},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runLintPaths(projectlifecycleservice.PathLintOptions{
				Scope:       ".",
				ProjectRoot: "/home/tester/workspace",
			}, false)
			if err != nil {
				t.Fatalf("runLintPaths returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "findings:      1") || !strings.Contains(output, "docs/a.md") {
		t.Fatalf("unexpected lint output: %q", output)
	}
}

func TestRunValidateViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		validateProjectSkillsFn: func(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error) {
			return &httpapibiz.ValidateProjectSkillsData{
				Item: &projectlifecycleservice.ValidateReport{
					ProjectPath: req.Options.ProjectPath,
					SkillID:     req.Options.SkillID,
					Links:       req.Options.Links,
					Total:       1,
					Passed:      1,
					Items: []projectlifecycleservice.ValidateItem{
						{SkillID: req.Options.SkillID, SkillDir: req.Options.ProjectPath + "/.agents/skills/demo", SkillMd: req.Options.ProjectPath + "/.agents/skills/demo/SKILL.md", Valid: true},
					},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runValidateWithOptions("demo-skill", validateCLIOptions{Links: true})
			if err != nil {
				t.Fatalf("runValidateWithOptions returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "验证技能合规性: demo-skill") || !strings.Contains(output, "link issues: 0") {
		t.Fatalf("unexpected validate output: %q", output)
	}
}

func TestRunAuditViaServiceWithoutLocalConfig(t *testing.T) {
	projectDir := t.TempDir()
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		auditProjectSkillsFn: func(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error) {
			return &httpapibiz.AuditProjectSkillsData{
				Item: &projectlifecycleservice.AuditReport{
					ProjectPath:       req.Options.ProjectPath,
					Scope:             req.Options.Scope,
					DefaultRepository: "main",
					TargetSkillCount:  1,
					RegisteredCount:   1,
					Validation:        &projectlifecycleservice.ValidateReport{Total: 1, Passed: 1},
					FeedbackSummary:   projectlifecycleservice.AuditFeedbackSummary{Synced: 1},
					RemotePush:        projectlifecycleservice.RemotePushReport{Status: "unknown"},
				},
			}, nil
		},
	})
	defer reset()

	output := withWorkingDir(t, projectDir, func() string {
		return captureStdout(t, func() {
			err := runAudit(projectlifecycleservice.AuditOptions{Scope: ".agents/skills"}, "markdown", "")
			if err != nil {
				t.Fatalf("runAudit returned error: %v", err)
			}
		})
	})
	if !strings.Contains(output, "# Skill Hub Audit Report") || !strings.Contains(output, "| Target Skills | 1 |") {
		t.Fatalf("unexpected audit output: %q", output)
	}
}

func TestRunSearchViaServiceWithoutLocalConfig(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		searchRemoteSkillsFn: func(ctx context.Context, keyword string, limit int) ([]spec.RemoteSearchResult, error) {
			return []spec.RemoteSearchResult{
				{FullName: "demo/search-skill", Description: "demo", HTMLURL: "https://example.com/demo"},
			}, nil
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runSearch("demo", 5); err != nil {
			t.Fatalf("runSearch returned error: %v", err)
		}
	})
	if !strings.Contains(output, "demo/search-skill") {
		t.Fatalf("unexpected search output: %q", output)
	}
}

func TestRunSearchViaServiceFailureIsGraceful(t *testing.T) {
	reset := stubServiceBridge(t, &fakeServiceBridgeClient{
		searchRemoteSkillsFn: func(ctx context.Context, keyword string, limit int) ([]spec.RemoteSearchResult, error) {
			return nil, errors.New("service unavailable")
		},
	})
	defer reset()

	output := captureStdout(t, func() {
		if err := runSearch("demo", 5); err != nil {
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
	listReposFn             func(context.Context) (*httpapibiz.RepoListData, error)
	addRepoFn               func(context.Context, config.RepositoryConfig) error
	removeRepoFn            func(context.Context, string) error
	syncRepoFn              func(context.Context, string) error
	enableRepoFn            func(context.Context, string) error
	disableRepoFn           func(context.Context, string) error
	setDefaultRepoFn        func(context.Context, string) error
	skillRepositoryStatusFn func(context.Context) (*httpapibiz.SkillRepositoryStatusData, error)
	checkSkillRepositoryFn  func(context.Context) (*httpapibiz.SkillRepositoryCheckData, error)
	syncSkillRepositoryFn   func(context.Context) (*httpapibiz.SyncSkillRepositoryData, error)
	pushPreviewFn           func(context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error)
	pushSkillRepositoryFn   func(context.Context, httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error)
	listSkillsFn            func(context.Context, []string) ([]spec.SkillMetadata, error)
	searchRemoteSkillsFn    func(context.Context, string, int) ([]spec.RemoteSearchResult, error)
	getProjectStatusFn      func(context.Context, string, string) (*httpapibiz.ProjectStatusData, error)
	findSkillCandidatesFn   func(context.Context, string) ([]spec.SkillMetadata, error)
	getSkillDetailFn        func(context.Context, string, string) (*spec.Skill, error)
	useSkillFn              func(context.Context, httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
	useGlobalSkillFn        func(context.Context, httpapibiz.UseGlobalSkillRequest) (*httpapibiz.UseGlobalSkillData, error)
	registerSkillFn         func(context.Context, httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error)
	importSkillsFn          func(context.Context, httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error)
	dedupeSkillsFn          func(context.Context, httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error)
	syncCopiesFn            func(context.Context, httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error)
	lintPathsFn             func(context.Context, httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error)
	validateProjectSkillsFn func(context.Context, httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error)
	auditProjectSkillsFn    func(context.Context, httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error)
	applyProjectFn          func(context.Context, httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
	getGlobalStatusFn       func(context.Context, string, []string) (*httpapibiz.GlobalStatusData, error)
	applyGlobalFn           func(context.Context, httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error)
	removeGlobalSkillFn     func(context.Context, string, []string, bool) (*httpapibiz.RemoveGlobalSkillData, error)
	previewFeedbackFn       func(context.Context, httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
	applyFeedbackFn         func(context.Context, httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
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
func (f *fakeServiceBridgeClient) SkillRepositoryStatus(ctx context.Context) (*httpapibiz.SkillRepositoryStatusData, error) {
	if f.skillRepositoryStatusFn != nil {
		return f.skillRepositoryStatusFn(ctx)
	}
	return &httpapibiz.SkillRepositoryStatusData{}, nil
}
func (f *fakeServiceBridgeClient) CheckSkillRepositoryUpdates(ctx context.Context) (*httpapibiz.SkillRepositoryCheckData, error) {
	if f.checkSkillRepositoryFn != nil {
		return f.checkSkillRepositoryFn(ctx)
	}
	return &httpapibiz.SkillRepositoryCheckData{Status: "no_remote", Message: "未设置远程仓库URL"}, nil
}
func (f *fakeServiceBridgeClient) SyncSkillRepositoryAndRefresh(ctx context.Context) (*httpapibiz.SyncSkillRepositoryData, error) {
	if f.syncSkillRepositoryFn != nil {
		return f.syncSkillRepositoryFn(ctx)
	}
	return &httpapibiz.SyncSkillRepositoryData{Status: "synced"}, nil
}
func (f *fakeServiceBridgeClient) PushSkillRepositoryPreview(ctx context.Context) (*httpapibiz.PushSkillRepositoryPreviewData, error) {
	if f.pushPreviewFn != nil {
		return f.pushPreviewFn(ctx)
	}
	return &httpapibiz.PushSkillRepositoryPreviewData{}, nil
}
func (f *fakeServiceBridgeClient) PushSkillRepositoryChanges(ctx context.Context, req httpapibiz.PushSkillRepositoryRequest) (*httpapibiz.PushSkillRepositoryData, error) {
	if f.pushSkillRepositoryFn != nil {
		return f.pushSkillRepositoryFn(ctx, req)
	}
	return &httpapibiz.PushSkillRepositoryData{Status: "pushed"}, nil
}
func (f *fakeServiceBridgeClient) ListSkills(ctx context.Context, repoNames []string) ([]spec.SkillMetadata, error) {
	if f.listSkillsFn != nil {
		return f.listSkillsFn(ctx, repoNames)
	}
	return nil, nil
}
func (f *fakeServiceBridgeClient) SearchRemoteSkills(ctx context.Context, keyword string, limit int) ([]spec.RemoteSearchResult, error) {
	if f.searchRemoteSkillsFn != nil {
		return f.searchRemoteSkillsFn(ctx, keyword, limit)
	}
	return nil, nil
}
func (f *fakeServiceBridgeClient) GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error) {
	if f.getProjectStatusFn != nil {
		return f.getProjectStatusFn(ctx, projectPath, skillID)
	}
	return &httpapibiz.ProjectStatusData{}, nil
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
func (f *fakeServiceBridgeClient) UseGlobalSkill(ctx context.Context, req httpapibiz.UseGlobalSkillRequest) (*httpapibiz.UseGlobalSkillData, error) {
	if f.useGlobalSkillFn != nil {
		return f.useGlobalSkillFn(ctx, req)
	}
	return &httpapibiz.UseGlobalSkillData{}, nil
}
func (f *fakeServiceBridgeClient) RegisterSkill(ctx context.Context, req httpapibiz.RegisterSkillRequest) (*httpapibiz.RegisterSkillData, error) {
	if f.registerSkillFn != nil {
		return f.registerSkillFn(ctx, req)
	}
	return &httpapibiz.RegisterSkillData{}, nil
}
func (f *fakeServiceBridgeClient) ImportSkills(ctx context.Context, req httpapibiz.ImportSkillsRequest) (*httpapibiz.ImportSkillsData, error) {
	if f.importSkillsFn != nil {
		return f.importSkillsFn(ctx, req)
	}
	return &httpapibiz.ImportSkillsData{}, nil
}
func (f *fakeServiceBridgeClient) DedupeSkills(ctx context.Context, req httpapibiz.DedupeRequest) (*httpapibiz.DedupeData, error) {
	if f.dedupeSkillsFn != nil {
		return f.dedupeSkillsFn(ctx, req)
	}
	return &httpapibiz.DedupeData{}, nil
}
func (f *fakeServiceBridgeClient) SyncCopies(ctx context.Context, req httpapibiz.SyncCopiesRequest) (*httpapibiz.SyncCopiesData, error) {
	if f.syncCopiesFn != nil {
		return f.syncCopiesFn(ctx, req)
	}
	return &httpapibiz.SyncCopiesData{}, nil
}
func (f *fakeServiceBridgeClient) LintPaths(ctx context.Context, req httpapibiz.PathLintRequest) (*httpapibiz.PathLintData, error) {
	if f.lintPathsFn != nil {
		return f.lintPathsFn(ctx, req)
	}
	return &httpapibiz.PathLintData{}, nil
}
func (f *fakeServiceBridgeClient) ValidateProjectSkills(ctx context.Context, req httpapibiz.ValidateProjectSkillsRequest) (*httpapibiz.ValidateProjectSkillsData, error) {
	if f.validateProjectSkillsFn != nil {
		return f.validateProjectSkillsFn(ctx, req)
	}
	return &httpapibiz.ValidateProjectSkillsData{}, nil
}
func (f *fakeServiceBridgeClient) AuditProjectSkills(ctx context.Context, req httpapibiz.AuditProjectSkillsRequest) (*httpapibiz.AuditProjectSkillsData, error) {
	if f.auditProjectSkillsFn != nil {
		return f.auditProjectSkillsFn(ctx, req)
	}
	return &httpapibiz.AuditProjectSkillsData{}, nil
}
func (f *fakeServiceBridgeClient) ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error) {
	if f.applyProjectFn != nil {
		return f.applyProjectFn(ctx, req)
	}
	return &httpapibiz.ApplyProjectData{}, nil
}
func (f *fakeServiceBridgeClient) GetGlobalStatus(ctx context.Context, skillID string, agents []string) (*httpapibiz.GlobalStatusData, error) {
	if f.getGlobalStatusFn != nil {
		return f.getGlobalStatusFn(ctx, skillID, agents)
	}
	return &httpapibiz.GlobalStatusData{}, nil
}
func (f *fakeServiceBridgeClient) ApplyGlobal(ctx context.Context, req httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error) {
	if f.applyGlobalFn != nil {
		return f.applyGlobalFn(ctx, req)
	}
	return &httpapibiz.ApplyGlobalData{}, nil
}
func (f *fakeServiceBridgeClient) RemoveGlobalSkill(ctx context.Context, skillID string, agents []string, force bool) (*httpapibiz.RemoveGlobalSkillData, error) {
	if f.removeGlobalSkillFn != nil {
		return f.removeGlobalSkillFn(ctx, skillID, agents, force)
	}
	return &httpapibiz.RemoveGlobalSkillData{}, nil
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
