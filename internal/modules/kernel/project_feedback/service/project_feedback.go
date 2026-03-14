package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
)

type PreviewResult struct {
	ProjectPath       string   `json:"project_path"`
	SkillID           string   `json:"skill_id"`
	DefaultRepo       string   `json:"default_repo"`
	SkillExists       bool     `json:"skill_exists"`
	Changes           []string `json:"changes"`
	HasContentChanges bool     `json:"has_content_changes"`
	ProjectVersion    string   `json:"project_version"`
	RepoVersion       string   `json:"repo_version"`
	ResolvedVersion   string   `json:"resolved_version"`
	NeedsVersionBump  bool     `json:"needs_version_bump"`
	NoChanges         bool     `json:"no_changes"`
}

type ApplyResult struct {
	Item *PreviewResult `json:"item"`
}

type ProjectFeedback struct {
	projectStateSvc *projectstatemodule.ProjectState
	repositorySvc   *repositorymodule.Repository
}

func New() *ProjectFeedback {
	return &ProjectFeedback{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
	}
}

func (p *ProjectFeedback) Preview(projectPath, skillID string) (*PreviewResult, error) {
	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 创建状态管理器失败")
	}

	hasSkill, err := stateManager.ProjectHasSkill(projectPath, skillID)
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 检查项目技能状态失败")
	}
	if !hasSkill {
		return nil, errors.NewWithCodef("Preview", errors.ErrSkillNotFound, "技能 '%s' 未在项目工作区中启用", skillID)
	}

	projectSkillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return nil, errors.NewWithCodef("Preview", errors.ErrFileNotFound, "项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return nil, errors.WrapWithCode(err, "Preview", errors.ErrFileOperation, "读取项目工作区文件失败")
	}

	defaultRepo, err := p.repositorySvc.Service().DefaultRepository()
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 获取默认仓库失败")
	}

	repoDir, err := p.repositorySvc.Service().Path(defaultRepo.Name)
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 获取仓库路径失败")
	}

	repoSkillDir := filepath.Join(repoDir, "skills", skillID)
	repoSkillPath := filepath.Join(repoSkillDir, "SKILL.md")
	repoContent := []byte{}
	skillExists := true
	if _, err := os.Stat(repoSkillPath); os.IsNotExist(err) {
		skillExists = false
	} else {
		repoContent, err = os.ReadFile(repoSkillPath)
		if err != nil {
			return nil, errors.WrapWithCode(err, "Preview", errors.ErrFileOperation, "读取本地仓库文件失败")
		}
	}

	projectStr := strings.TrimSpace(string(projectContent))
	repoStr := strings.TrimSpace(string(repoContent))

	changes, err := compareSkillDirectories(projectSkillDir, repoSkillDir, skillExists)
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 比较技能目录失败")
	}

	hasContentChanges := projectStr != repoStr
	projectVersion := normalizeVersionToXYZ(skill.ExtractVersion(projectContent))
	repoVersion := "0.0.0"
	if skillExists && len(repoContent) > 0 {
		repoVersion = normalizeVersionToXYZ(skill.ExtractVersion(repoContent))
	}

	resolvedVersion := projectVersion
	needsVersionBump := false
	if (hasContentChanges || len(changes) > 0) && compareVersions(projectVersion, repoVersion) <= 0 {
		resolvedVersion = bumpPatchVersion(repoVersion)
		needsVersionBump = true
	}

	return &PreviewResult{
		ProjectPath:       projectPath,
		SkillID:           skillID,
		DefaultRepo:       defaultRepo.Name,
		SkillExists:       skillExists,
		Changes:           changes,
		HasContentChanges: hasContentChanges,
		ProjectVersion:    projectVersion,
		RepoVersion:       repoVersion,
		ResolvedVersion:   resolvedVersion,
		NeedsVersionBump:  needsVersionBump,
		NoChanges:         skillExists && len(changes) == 0 && !hasContentChanges,
	}, nil
}

func (p *ProjectFeedback) Apply(projectPath, skillID string) (*PreviewResult, error) {
	preview, err := p.Preview(projectPath, skillID)
	if err != nil {
		return nil, err
	}
	if preview.NoChanges {
		return preview, nil
	}

	projectSkillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if preview.NeedsVersionBump {
		if err := updateSkillMdVersion(projectSkillPath, preview.ResolvedVersion); err != nil {
			return nil, errors.Wrap(err, "Apply: 更新版本号失败")
		}
	}

	if err := p.repositorySvc.Service().ArchiveToDefaultRepository(skillID, projectSkillDir); err != nil {
		return nil, errors.Wrap(err, "Apply: 归档技能到默认仓库失败")
	}

	updatedPreview, err := p.Preview(projectPath, skillID)
	if err != nil {
		return preview, nil
	}
	updatedPreview.NeedsVersionBump = preview.NeedsVersionBump
	return updatedPreview, nil
}

func normalizeVersionToXYZ(version string) string {
	version = strings.Trim(version, `" `)
	if version == "" {
		return "0.0.0"
	}
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		version = version[1:]
	}
	parts := strings.Split(version, ".")
	var major, minor, patch int
	for i := 0; i < 3; i++ {
		var numStr string
		if i < len(parts) {
			for _, c := range parts[i] {
				if c >= '0' && c <= '9' {
					numStr += string(c)
				} else {
					break
				}
			}
		}
		if numStr == "" {
			numStr = "0"
		}
		n, _ := strconv.Atoi(numStr)
		switch i {
		case 0:
			major = n
		case 1:
			minor = n
		case 2:
			patch = n
		}
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func bumpPatchVersion(version string) string {
	version = normalizeVersionToXYZ(version)
	parts := strings.Split(version, ".")
	patch, _ := strconv.Atoi(parts[2])
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts[:3], ".")
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

func updateSkillMdVersion(skillMdPath, newVersion string) error {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inFrontmatter := false
	inMetadata := false
	updated := false

	for i, line := range lines {
		if line == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "metadata:" || trimmedLine == "\"metadata\":" {
			inMetadata = true
			continue
		}
		if inMetadata {
			if strings.HasPrefix(trimmedLine, "version:") || strings.HasPrefix(trimmedLine, "\"version\":") {
				indent := ""
				for _, c := range line {
					if c == ' ' || c == '\t' {
						indent += string(c)
					} else {
						break
					}
				}
				lines[i] = fmt.Sprintf("%sversion: %s", indent, newVersion)
				updated = true
				break
			} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmedLine != "" {
				inMetadata = false
			}
		}
	}

	if !updated {
		inFrontmatter = false
		for i, line := range lines {
			if line == "---" {
				if inFrontmatter {
					break
				}
				inFrontmatter = true
				continue
			}
			if inFrontmatter {
				trimmedLine := strings.TrimSpace(line)
				if (strings.HasPrefix(trimmedLine, "version:") || strings.HasPrefix(trimmedLine, "\"version\":")) &&
					!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					lines[i] = fmt.Sprintf("version: %s", newVersion)
					updated = true
					break
				}
			}
		}
	}

	if !updated {
		closingFence := -1
		fenceCount := 0
		for i, line := range lines {
			if line == "---" {
				fenceCount++
				if fenceCount == 2 {
					closingFence = i
					break
				}
			}
		}
		if closingFence < 0 {
			return errors.NewWithCode("updateSkillMdVersion", errors.ErrSkillInvalid, "未找到版本号字段")
		}
		insertLine := fmt.Sprintf("version: %s", newVersion)
		lines = append(lines[:closingFence], append([]string{insertLine}, lines[closingFence:]...)...)
	}

	return os.WriteFile(skillMdPath, []byte(strings.Join(lines, "\n")), 0644)
}

func compareSkillDirectories(projectDir, repoDir string, repoExists bool) ([]string, error) {
	var changes []string

	if !repoExists {
		err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(projectDir, path)
				if err != nil {
					return err
				}
				changes = append(changes, fmt.Sprintf("新增: %s", relPath))
			}
			return nil
		})
		return changes, err
	}

	projectFiles := make(map[string]bool)
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(projectDir, path)
			if err != nil {
				return err
			}
			projectFiles[relPath] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(repoDir, path)
			if err != nil {
				return err
			}

			projectPath := filepath.Join(projectDir, relPath)
			repoPath := path

			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				changes = append(changes, fmt.Sprintf("删除: %s", relPath))
			} else {
				projectContent, err1 := os.ReadFile(projectPath)
				repoContent, err2 := os.ReadFile(repoPath)
				if err1 != nil || err2 != nil {
					changes = append(changes, fmt.Sprintf("修改: %s (读取错误)", relPath))
				} else if string(projectContent) != string(repoContent) {
					changes = append(changes, fmt.Sprintf("修改: %s", relPath))
				}
				delete(projectFiles, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for relPath := range projectFiles {
		changes = append(changes, fmt.Sprintf("新增: %s", relPath))
	}
	return changes, nil
}
