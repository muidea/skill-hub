package service

import (
	"os"
	"path/filepath"
	"sort"
	"sync"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
)

var projectApplyCwdMu sync.Mutex

type ApplyResult struct {
	ProjectPath string            `json:"project_path"`
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
}

func New() *ProjectApply {
	return &ProjectApply{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
	}
}

func (p *ProjectApply) Apply(projectPath, skillID string, dryRun, force bool) (*ApplyResult, error) {
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
	skills := projectState.Skills
	if skillID != "" {
		skillVars, ok := skills[skillID]
		if !ok {
			return nil, errors.NewWithCodef("Apply", errors.ErrSkillNotFound, "技能 %s 未在当前项目中启用", skillID)
		}
		skills = map[string]spec.SkillVars{skillID: skillVars}
	}
	if len(skills) == 0 {
		return &ApplyResult{
			ProjectPath: projectState.ProjectPath,
			DryRun:      dryRun,
			Items:       []ApplyResultItem{},
		}, nil
	}

	skillIDs := make([]string, 0, len(skills))
	for skillID := range skills {
		skillIDs = append(skillIDs, skillID)
	}
	sort.Strings(skillIDs)

	result := &ApplyResult{
		ProjectPath: projectState.ProjectPath,
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

		if dryRun {
			item.Status = "planned"
			item.Message = "dry-run"
			result.Items = append(result.Items, item)
			continue
		}

		appliedVersion, err := p.copyRepositorySkillToProject(projectState.ProjectPath, repoName, skillID)
		if err != nil {
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

		existing := projectState.Skills[skillID]
		existing.Status = spec.SkillStatusSynced
		if appliedVersion != "" {
			existing.Version = appliedVersion
		}
		projectState.Skills[skillID] = existing
	}

	if !dryRun {
		if err := stateManager.SaveProjectState(projectState); err != nil {
			return nil, errors.Wrap(err, "Apply: 保存项目状态失败")
		}
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

func (p *ProjectApply) copyRepositorySkillToProject(projectPath, repoName, skillID string) (string, error) {
	projectApplyCwdMu.Lock()
	defer projectApplyCwdMu.Unlock()

	repoPath, err := p.repositorySvc.Service().Path(repoName)
	if err != nil {
		return "", errors.Wrap(err, "copyRepositorySkillToProject: 获取仓库路径失败")
	}
	srcDir := filepath.Join(repoPath, "skills", skillID)
	skillMDPath := filepath.Join(srcDir, "SKILL.md")
	if _, err := os.Stat(skillMDPath); err != nil {
		if os.IsNotExist(err) {
			return "", errors.NewWithCodef("copyRepositorySkillToProject", errors.ErrFileNotFound, "技能文件在仓库中不存在: %s", srcDir)
		}
		return "", errors.Wrap(err, "copyRepositorySkillToProject: 检查仓库技能失败")
	}
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return "", errors.Wrap(err, "copyRepositorySkillToProject: 读取仓库技能失败")
	}
	version := skill.ExtractVersion(content)
	dstDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	if err := syncSkillDirectory(srcDir, dstDir); err != nil {
		return "", err
	}
	return version, nil
}

func syncSkillDirectory(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return errors.WrapWithCode(err, "syncSkillDirectory", errors.ErrFileOperation, "创建目标目录失败")
	}

	srcFiles := make(map[string]bool)
	if err := filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(srcDir, srcPath)
			if err != nil {
				return err
			}
			srcFiles[relPath] = true
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "syncSkillDirectory: 遍历源目录失败")
	}

	dstFiles := make(map[string]bool)
	if err := filepath.Walk(dstDir, func(dstPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(dstDir, dstPath)
			if err != nil {
				return err
			}
			dstFiles[relPath] = true
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "syncSkillDirectory: 遍历目标目录失败")
	}

	for relPath := range srcFiles {
		srcPath := filepath.Join(srcDir, relPath)
		dstPath := filepath.Join(dstDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return errors.Wrapf(err, "syncSkillDirectory: 创建目录失败 %s", filepath.Dir(dstPath))
		}
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return errors.Wrap(err, "syncSkillDirectory: 读取源文件失败")
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			return errors.Wrap(err, "syncSkillDirectory: 获取源文件权限失败")
		}
		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return errors.Wrap(err, "syncSkillDirectory: 写入目标文件失败")
		}
		delete(dstFiles, relPath)
	}

	for relPath := range dstFiles {
		if err := os.Remove(filepath.Join(dstDir, relPath)); err != nil {
			return errors.Wrap(err, "syncSkillDirectory: 删除目标多余文件失败")
		}
	}

	return nil
}
