package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

type ProjectStatusSummary struct {
	ProjectPath string            `json:"project_path"`
	SkillCount  int               `json:"skill_count"`
	Items       []SkillStatusItem `json:"items"`
}

type SkillStatusItem struct {
	SkillID          string `json:"skill_id"`
	Status           string `json:"status"`
	SourceRepository string `json:"source_repository,omitempty"`
	LocalVersion     string `json:"local_version,omitempty"`
	RepoVersion      string `json:"repo_version,omitempty"`
	LocalPath        string `json:"local_path,omitempty"`
	RepoPath         string `json:"repo_path,omitempty"`
}

type ProjectStatus struct {
	projectStateSvc *projectstatemodule.ProjectState
	repositorySvc   *repositorymodule.Repository
}

func New() *ProjectStatus {
	return &ProjectStatus{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
	}
}

func (p *ProjectStatus) Inspect(projectPath, skillID string) (*ProjectStatusSummary, error) {
	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "Inspect: 创建状态管理器失败")
	}

	projectState, err := stateManager.FindProjectByPath(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Inspect: 查找项目状态失败")
	}
	if projectState == nil {
		return nil, errors.NewWithCode("Inspect", errors.ErrFileNotFound, "当前目录未在 skill-hub 中注册")
	}

	skills := projectState.Skills
	if skillID != "" {
		skillVars, ok := skills[skillID]
		if !ok {
			return nil, errors.NewWithCodef("Inspect", errors.ErrSkillNotFound, "技能 %s 未在当前项目中启用", skillID)
		}
		skills = map[string]spec.SkillVars{skillID: skillVars}
	}

	items := make([]SkillStatusItem, 0, len(skills))
	for currentSkillID, skillVars := range skills {
		item, err := p.inspectSkill(projectState.ProjectPath, currentSkillID, skillVars)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)

		existing := projectState.Skills[currentSkillID]
		existing.Status = item.Status
		if item.LocalVersion != "" {
			existing.Version = item.LocalVersion
		}
		projectState.Skills[currentSkillID] = existing
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SkillID < items[j].SkillID
	})

	if err := stateManager.SaveProjectState(projectState); err != nil {
		return nil, errors.Wrap(err, "Inspect: 保存项目状态失败")
	}

	return &ProjectStatusSummary{
		ProjectPath: projectState.ProjectPath,
		SkillCount:  len(items),
		Items:       items,
	}, nil
}

func (p *ProjectStatus) inspectSkill(projectPath, skillID string, skillVars spec.SkillVars) (*SkillStatusItem, error) {
	agentsSkillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	localSkillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")

	item := &SkillStatusItem{
		SkillID:          skillID,
		SourceRepository: skillVars.SourceRepository,
		LocalPath:        localSkillMdPath,
	}

	repoSkillPath, repoVersion, repoHash, repoExists, err := p.getRepoSkillInfo(skillID, skillVars.SourceRepository)
	if err != nil {
		return nil, err
	}
	if repoExists {
		item.RepoPath = repoSkillPath
		item.RepoVersion = repoVersion
	}

	if _, err := os.Stat(localSkillMdPath); os.IsNotExist(err) {
		item.Status = spec.SkillStatusMissing
		item.LocalVersion = "—"
		return item, nil
	}

	repoSkillDir := filepath.Dir(repoSkillPath)
	if repoExists {
		equal, eqErr := skillDirsEqual(agentsSkillDir, repoSkillDir)
		if eqErr == nil && !equal {
			localVersion, _, localErr := getLocalSkillInfo(localSkillMdPath)
			if localErr != nil {
				item.Status = spec.SkillStatusModified
				item.LocalVersion = "unknown"
				return item, nil
			}
			item.LocalVersion = localVersion
			if compareVersions(repoVersion, localVersion) > 0 {
				item.Status = spec.SkillStatusOutdated
			} else {
				item.Status = spec.SkillStatusModified
			}
			return item, nil
		}
	}

	localVersion, localHash, err := getLocalSkillInfo(localSkillMdPath)
	if err != nil {
		return nil, err
	}
	item.LocalVersion = localVersion

	if !repoExists {
		item.Status = spec.SkillStatusModified
		return item, nil
	}

	item.Status = determineSkillStatus(localVersion, localHash, repoVersion, repoHash)
	if item.Status == "" {
		item.Status = skillVars.Status
		if item.Status == "" {
			item.Status = spec.SkillStatusModified
		}
	}
	return item, nil
}

func (p *ProjectStatus) getRepoSkillInfo(skillID, sourceRepository string) (string, string, string, bool, error) {
	repoName, err := p.resolveSourceRepository(sourceRepository)
	if err != nil {
		return "", "", "", false, errors.Wrap(err, "getRepoSkillInfo: 获取来源仓库失败")
	}

	repoPath, err := p.repositorySvc.Service().Path(repoName)
	if err != nil {
		return "", "", "", false, errors.Wrap(err, "getRepoSkillInfo: 获取仓库路径失败")
	}

	repoSkillDir := filepath.Join(repoPath, "skills", skillID)
	repoSkillPath := filepath.Join(repoSkillDir, "SKILL.md")
	if _, err := os.Stat(repoSkillPath); os.IsNotExist(err) {
		return repoSkillPath, "", "", false, nil
	}

	version, hash, err := getLocalSkillInfo(repoSkillPath)
	if err != nil {
		return "", "", "", false, err
	}
	return repoSkillPath, version, hash, true, nil
}

func (p *ProjectStatus) resolveSourceRepository(sourceRepository string) (string, error) {
	if sourceRepository != "" {
		return sourceRepository, nil
	}

	defaultRepo, err := p.repositorySvc.Service().DefaultRepository()
	if err != nil {
		return "", errors.Wrap(err, "resolveSourceRepository: 获取默认仓库失败")
	}
	return defaultRepo.Name, nil
}

func getLocalSkillInfo(skillMdPath string) (string, string, error) {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", "", utils.ReadFileErr(err, skillMdPath)
	}

	version := skill.ExtractVersion(content)
	hashStr := skill.ContentHash(content)
	return version, hashStr, nil
}

func skillDirsEqual(dirA, dirB string) (bool, error) {
	manifestA, err := buildSkillDirManifest(dirA)
	if err != nil {
		return false, err
	}
	manifestB, err := buildSkillDirManifest(dirB)
	if err != nil {
		return false, err
	}
	if len(manifestA) != len(manifestB) {
		return false, nil
	}
	for relPath, hashA := range manifestA {
		if hashB, ok := manifestB[relPath]; !ok || hashA != hashB {
			return false, nil
		}
	}
	return true, nil
}

func buildSkillDirManifest(dir string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[relPath] = skill.ContentHash(content)
		return nil
	})
	return out, err
}

func determineSkillStatus(localVersion, localHash, repoVersion, repoHash string) string {
	if localHash != repoHash {
		if compareVersions(localVersion, repoVersion) < 0 {
			return spec.SkillStatusOutdated
		}
		return spec.SkillStatusModified
	}

	if compareVersions(localVersion, repoVersion) < 0 {
		return spec.SkillStatusOutdated
	}
	return spec.SkillStatusSynced
}

func compareVersions(v1, v2 string) int {
	v1 = strings.Trim(v1, `"`)
	v2 = strings.Trim(v2, `" `)

	if v1 == v2 {
		return 0
	}

	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")
	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		num1 := 0
		num2 := 0
		fmt.Sscanf(v1Parts[i], "%d", &num1)
		fmt.Sscanf(v2Parts[i], "%d", &num2)
		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	if len(v1Parts) > len(v2Parts) {
		return 1
	}
	if len(v1Parts) < len(v2Parts) {
		return -1
	}

	if v1 > v2 {
		return 1
	}
	return -1
}
