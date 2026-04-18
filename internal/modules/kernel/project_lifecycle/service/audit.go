package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type AuditOptions struct {
	ProjectPath string `json:"project_path"`
	Scope       string `json:"scope"`
	Canonical   string `json:"canonical,omitempty"`
	ProjectRoot string `json:"project_root,omitempty"`
}

type AuditReport struct {
	GeneratedAt        string                                     `json:"generated_at"`
	ProjectPath        string                                     `json:"project_path"`
	Scope              string                                     `json:"scope"`
	Canonical          string                                     `json:"canonical,omitempty"`
	ProjectRoot        string                                     `json:"project_root,omitempty"`
	DefaultRepository  string                                     `json:"default_repository,omitempty"`
	TargetSkillCount   int                                        `json:"target_skill_count"`
	RegisteredCount    int                                        `json:"registered_count"`
	UnregisteredCount  int                                        `json:"unregistered_count"`
	UnregisteredSkills []string                                   `json:"unregistered_skills,omitempty"`
	Validation         *ValidateReport                            `json:"validation,omitempty"`
	StatusSummary      *projectstatusservice.ProjectStatusSummary `json:"status_summary,omitempty"`
	DuplicateReport    *DuplicateReport                           `json:"duplicate_report,omitempty"`
	PathLint           *PathLintReport                            `json:"path_lint,omitempty"`
	LinkIssueCount     int                                        `json:"link_issue_count"`
	DuplicateConflicts int                                        `json:"duplicate_conflicts"`
	AbsolutePathHits   int                                        `json:"absolute_path_hits"`
	RemotePush         RemotePushReport                           `json:"remote_push"`
	FeedbackSummary    AuditFeedbackSummary                       `json:"feedback_summary"`
	Errors             []AuditError                               `json:"errors,omitempty"`
}

type AuditFeedbackSummary struct {
	Synced   int `json:"synced"`
	Modified int `json:"modified"`
	Outdated int `json:"outdated"`
	Missing  int `json:"missing"`
}

type RemotePushReport struct {
	Performed       bool   `json:"performed"`
	Status          string `json:"status"`
	RepositoryPath  string `json:"repository_path,omitempty"`
	Dirty           bool   `json:"dirty"`
	UnpushedCommits int    `json:"unpushed_commits"`
	Message         string `json:"message,omitempty"`
}

type AuditError struct {
	Step  string `json:"step"`
	Error string `json:"error"`
}

func (p *ProjectLifecycle) Audit(opts AuditOptions) (*AuditReport, error) {
	if strings.TrimSpace(opts.ProjectPath) == "" {
		return nil, errors.NewWithCode("Audit", errors.ErrInvalidInput, "项目路径不能为空")
	}
	if strings.TrimSpace(opts.Scope) == "" {
		opts.Scope = ".agents/skills"
	}
	absProjectPath, err := filepath.Abs(opts.ProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Audit: 获取项目绝对路径失败")
	}
	absScope := opts.Scope
	if !filepath.IsAbs(absScope) {
		absScope = filepath.Join(absProjectPath, absScope)
	}
	absScope, err = filepath.Abs(absScope)
	if err != nil {
		return nil, errors.Wrap(err, "Audit: 解析scope失败")
	}
	absProjectRoot := opts.ProjectRoot
	if absProjectRoot != "" {
		if !filepath.IsAbs(absProjectRoot) {
			absProjectRoot = filepath.Join(absProjectPath, absProjectRoot)
		}
		absProjectRoot, err = filepath.Abs(absProjectRoot)
		if err != nil {
			return nil, errors.Wrap(err, "Audit: 解析project-root失败")
		}
	}
	absCanonical := opts.Canonical
	if absCanonical != "" {
		if !filepath.IsAbs(absCanonical) {
			absCanonical = filepath.Join(absProjectPath, absCanonical)
		}
		absCanonical, err = filepath.Abs(absCanonical)
		if err != nil {
			return nil, errors.Wrap(err, "Audit: 解析canonical失败")
		}
	}

	report := &AuditReport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		ProjectPath: absProjectPath,
		Scope:       absScope,
		Canonical:   absCanonical,
		ProjectRoot: absProjectRoot,
	}

	defaultRepo, err := p.repositorySvc.Service().DefaultRepository()
	if err != nil {
		report.Errors = append(report.Errors, AuditError{Step: "default-repository", Error: err.Error()})
	} else if defaultRepo != nil {
		report.DefaultRepository = defaultRepo.Name
	}

	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return report, errors.Wrap(err, "Audit: 创建状态管理器失败")
	}
	projectState, err := stateManager.LoadProjectState(absProjectPath)
	if err != nil {
		return report, errors.Wrap(err, "Audit: 加载项目状态失败")
	}

	skillIDs, scanErr := scanAuditSkillIDs(absScope)
	if scanErr != nil {
		report.Errors = append(report.Errors, AuditError{Step: "scan", Error: scanErr.Error()})
	}
	report.TargetSkillCount = len(skillIDs)
	for _, skillID := range skillIDs {
		if _, ok := projectState.Skills[skillID]; ok {
			report.RegisteredCount++
		} else {
			report.UnregisteredCount++
			report.UnregisteredSkills = append(report.UnregisteredSkills, skillID)
		}
	}

	validation, validationErr := p.ValidateProjectSkills(ValidateOptions{
		ProjectPath: absProjectPath,
		All:         true,
		Links:       true,
		ProjectRoot: absProjectRoot,
		CheckRemote: false,
	})
	if validationErr != nil && validation == nil {
		report.Errors = append(report.Errors, AuditError{Step: "validate", Error: validationErr.Error()})
	} else {
		report.Validation = validation
		if validationErr != nil {
			report.Errors = append(report.Errors, AuditError{Step: "validate", Error: validationErr.Error()})
		}
		if validation != nil {
			report.LinkIssueCount = validation.LinkIssueCount
		}
	}

	statusSummary, statusErr := projectstatusservice.New().Inspect(absProjectPath, "")
	if statusErr != nil {
		report.Errors = append(report.Errors, AuditError{Step: "status", Error: statusErr.Error()})
	} else {
		report.StatusSummary = statusSummary
		report.FeedbackSummary = summarizeFeedbackStatus(statusSummary)
	}

	dedupe, dedupeErr := p.Dedupe(absScope, DedupeOptions{Canonical: absCanonical, Strategy: "newest", Report: true})
	if dedupeErr != nil && dedupe == nil {
		report.Errors = append(report.Errors, AuditError{Step: "dedupe", Error: dedupeErr.Error()})
	} else {
		report.DuplicateReport = dedupe
		if dedupeErr != nil {
			report.Errors = append(report.Errors, AuditError{Step: "dedupe", Error: dedupeErr.Error()})
		}
		if dedupe != nil {
			report.DuplicateConflicts = dedupe.Conflicts
		}
	}

	pathLint, lintErr := p.LintPaths(PathLintOptions{Scope: absScope, ProjectRoot: absProjectRoot})
	if lintErr != nil && pathLint == nil {
		report.Errors = append(report.Errors, AuditError{Step: "lint-paths", Error: lintErr.Error()})
	} else {
		report.PathLint = pathLint
		if lintErr != nil {
			report.Errors = append(report.Errors, AuditError{Step: "lint-paths", Error: lintErr.Error()})
		}
		if pathLint != nil {
			report.AbsolutePathHits = pathLint.FindingCount
		}
	}

	report.RemotePush = p.inspectRemotePush(report.DefaultRepository)
	if len(report.Errors) > 0 {
		return report, errors.NewWithCodef("Audit", errors.ErrValidation, "审计完成但发现 %d 个问题", len(report.Errors))
	}
	return report, nil
}

func scanAuditSkillIDs(scope string) ([]string, error) {
	locations, err := scanSkillLocations(scope, "")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, location := range locations {
		seen[location.SkillID] = true
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func summarizeFeedbackStatus(summary *projectstatusservice.ProjectStatusSummary) AuditFeedbackSummary {
	out := AuditFeedbackSummary{}
	if summary == nil {
		return out
	}
	for _, item := range summary.Items {
		switch item.Status {
		case spec.SkillStatusSynced:
			out.Synced++
		case spec.SkillStatusModified:
			out.Modified++
		case spec.SkillStatusOutdated:
			out.Outdated++
		case spec.SkillStatusMissing:
			out.Missing++
		}
	}
	return out
}

func (p *ProjectLifecycle) inspectRemotePush(defaultRepoName string) RemotePushReport {
	report := RemotePushReport{Status: "unknown", Message: "默认仓库未配置或不是Git仓库"}
	if defaultRepoName == "" {
		return report
	}
	repoPath, err := p.repositorySvc.Service().Path(defaultRepoName)
	if err != nil {
		report.Message = err.Error()
		return report
	}
	report.RepositoryPath = repoPath
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		report.Message = "默认仓库不是Git仓库或缺少.git目录"
		return report
	}
	dirtyOutput, err := gitOutput(repoPath, "status", "--porcelain")
	if err != nil {
		report.Message = err.Error()
		return report
	}
	report.Dirty = strings.TrimSpace(dirtyOutput) != ""
	unpushedOutput, err := gitOutput(repoPath, "rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		report.Status = "unknown"
		report.Message = "无法读取上游分支，可能尚未配置remote tracking"
		return report
	}
	unpushed, _ := strconv.Atoi(strings.TrimSpace(unpushedOutput))
	report.UnpushedCommits = unpushed
	report.Performed = !report.Dirty && unpushed == 0
	if report.Performed {
		report.Status = "pushed"
		report.Message = "默认仓库无未提交更改且无未推送提交"
	} else {
		report.Status = "pending"
		report.Message = "默认仓库存在未提交更改或未推送提交"
	}
	return report
}

func gitOutput(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
