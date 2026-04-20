package service

import (
	"path/filepath"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
)

type UseResult struct {
	ProjectPath string `json:"project_path"`
	SkillID     string `json:"skill_id"`
	Version     string `json:"version"`
	Repository  string `json:"repository"`
}

type ProjectUse struct {
	projectStateSvc *projectstatemodule.ProjectState
	repositorySvc   *repositorymodule.Repository
}

func New() *ProjectUse {
	return &ProjectUse{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
	}
}

func (p *ProjectUse) EnableSkill(projectPath, skillID, repoName string, variables map[string]string) (*UseResult, error) {
	if projectPath == "" {
		return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "项目路径不能为空")
	}
	if skillID == "" {
		return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "技能 ID 不能为空")
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 获取项目绝对路径失败")
	}

	candidates, err := p.repositorySvc.Service().FindSkill(skillID)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 查找技能失败")
	}
	if len(candidates) == 0 {
		return nil, errors.SkillNotFound("EnableSkill", skillID)
	}

	selectedRepo := repoName
	if selectedRepo == "" {
		if len(candidates) != 1 {
			return nil, errors.NewWithCode("EnableSkill", errors.ErrInvalidInput, "技能存在多个候选仓库，必须指定 repository")
		}
		selectedRepo = candidates[0].Repository
	}

	fullSkill, err := p.repositorySvc.Service().LoadSkill(skillID, selectedRepo)
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 加载技能详情失败")
	}

	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 创建状态管理器失败")
	}

	if err := stateManager.AddSkillToProjectWithSource(absProjectPath, skillID, fullSkill.Version, selectedRepo, variables); err != nil {
		return nil, errors.Wrap(err, "EnableSkill: 保存项目状态失败")
	}

	return &UseResult{
		ProjectPath: absProjectPath,
		SkillID:     skillID,
		Version:     fullSkill.Version,
		Repository:  selectedRepo,
	}, nil
}
