package service

import (
	"os"
	"sort"
	"sync"

	adaptermodule "github.com/muidea/skill-hub/internal/modules/blocks/adapter"
	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

var projectApplyCwdMu sync.Mutex

type ApplyResult struct {
	ProjectPath string            `json:"project_path"`
	Target      string            `json:"target"`
	DryRun      bool              `json:"dry_run"`
	Items       []ApplyResultItem `json:"items"`
}

type ApplyResultItem struct {
	SkillID   string `json:"skill_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Variables int    `json:"variables"`
}

type ProjectApply struct {
	projectStateSvc *projectstatemodule.ProjectState
	repositorySvc   *repositorymodule.Repository
	adapterSvc      *adaptermodule.Adapter
}

const sourceRepositoryVariable = "_skill_hub_source_repository"

func New() *ProjectApply {
	return &ProjectApply{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
		adapterSvc:      adaptermodule.New(),
	}
}

func (p *ProjectApply) Apply(projectPath string, dryRun, force bool) (*ApplyResult, error) {
	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "Apply: 创建状态管理器失败")
	}

	projectState, err := stateManager.FindProjectByPath(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Apply: 查找项目状态失败")
	}
	if projectState == nil {
		return nil, errors.NewWithCode("Apply", errors.ErrProjectInvalid, "当前目录未在 skill-hub 中注册")
	}
	if projectState.PreferredTarget == "" {
		return nil, errors.NewWithCode("Apply", errors.ErrProjectInvalid, "项目未设置目标环境，请先使用 'skill-hub set-target <value>' 设置目标环境")
	}

	target := spec.NormalizeTarget(projectState.PreferredTarget)
	skills := projectState.Skills
	if len(skills) == 0 {
		return &ApplyResult{
			ProjectPath: projectState.ProjectPath,
			Target:      target,
			DryRun:      dryRun,
			Items:       []ApplyResultItem{},
		}, nil
	}

	adapter, err := p.adapterSvc.Service().ForTarget(target)
	if err != nil {
		return nil, errors.WrapWithCode(err, "Apply", errors.ErrSystem, "获取适配器失败")
	}
	adapter.SetProjectMode()

	skillIDs := make([]string, 0, len(skills))
	for skillID := range skills {
		skillIDs = append(skillIDs, skillID)
	}
	sort.Strings(skillIDs)

	result := &ApplyResult{
		ProjectPath: projectState.ProjectPath,
		Target:      target,
		DryRun:      dryRun,
		Items:       make([]ApplyResultItem, 0, len(skillIDs)),
	}

	for _, skillID := range skillIDs {
		skillVars := skills[skillID]
		item := ApplyResultItem{
			SkillID:   skillID,
			Variables: len(skillVars.Variables),
		}

		repoName, err := p.resolveSourceRepository(skillVars)
		if err != nil {
			item.Status = "error"
			item.Message = err.Error()
			result.Items = append(result.Items, item)
			continue
		}

		content, err := p.repositorySvc.Service().ReadSkillContent(repoName, skillID)
		if err != nil {
			item.Status = "error"
			item.Message = err.Error()
			result.Items = append(result.Items, item)
			continue
		}

		if dryRun {
			item.Status = "planned"
			item.Message = "dry-run"
			result.Items = append(result.Items, item)
			continue
		}

		applyVariables := cloneApplyVariables(skillVars.Variables, repoName)
		if err := p.applyInProjectDir(projectState.ProjectPath, func() error {
			return adapter.Apply(skillID, content, applyVariables)
		}); err != nil {
			item.Status = "error"
			item.Message = err.Error()
			result.Items = append(result.Items, item)
			if !force {
				continue
			}
			continue
		}

		item.Status = "applied"
		result.Items = append(result.Items, item)
	}

	return result, nil
}

func (p *ProjectApply) resolveSourceRepository(skillVars spec.SkillVars) (string, error) {
	if skillVars.SourceRepository != "" {
		return skillVars.SourceRepository, nil
	}

	defaultRepo, err := p.repositorySvc.Service().DefaultRepository()
	if err != nil {
		return "", errors.Wrap(err, "resolveSourceRepository: 获取默认仓库失败")
	}
	return defaultRepo.Name, nil
}

func cloneApplyVariables(variables map[string]string, repoName string) map[string]string {
	if len(variables) == 0 && repoName == "" {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(variables)+1)
	for key, value := range variables {
		cloned[key] = value
	}
	if repoName != "" {
		cloned[sourceRepositoryVariable] = repoName
	}
	return cloned
}

func (p *ProjectApply) applyInProjectDir(projectPath string, fn func() error) error {
	projectApplyCwdMu.Lock()
	defer projectApplyCwdMu.Unlock()

	originalCwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "applyInProjectDir: 获取当前目录失败")
	}
	if err := os.Chdir(projectPath); err != nil {
		return errors.Wrap(err, "applyInProjectDir: 切换项目目录失败")
	}
	defer func() {
		_ = os.Chdir(originalCwd)
	}()

	return fn()
}
