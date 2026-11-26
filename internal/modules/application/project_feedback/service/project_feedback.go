package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/muidea/skill-hub/internal/pkg/projectport"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
)

type PreviewResult = projectport.PreviewResult

type ApplyResult struct {
	Item *PreviewResult `json:"item"`
}

type ProjectFeedback struct {
	projectState projectstateport.ProjectState
	repository   repositoryport.ProjectSource
}

func New(projectState projectstateport.ProjectState, repository repositoryport.ProjectSource) *ProjectFeedback {
	return &ProjectFeedback{
		projectState: projectState,
		repository:   repository,
	}
}

func (p *ProjectFeedback) Preview(projectPath, skillID string) (*PreviewResult, error) {
	projectState, err := p.projectState.Find(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 查找项目状态失败")
	}
	if projectState == nil {
		return nil, errors.NewWithCode("Preview", errors.ErrProjectInvalid, "当前目录未在 skill-hub 中注册")
	}

	projectRoot := projectState.ProjectPath
	_, hasSkill := projectState.Skills[skillID]
	if projectRoot == "" {
		projectRoot = projectPath
	}
	absProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 获取项目绝对路径失败")
	}
	projectRoot = absProjectRoot

	if !hasSkill {
		return nil, errors.NewWithCodef("Preview", errors.ErrSkillNotFound, "技能 '%s' 未在项目工作区中启用", skillID)
	}

	projectSkillDir := filepath.Join(projectRoot, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return nil, errors.NewWithCodef("Preview", errors.ErrFileNotFound, "项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return nil, errors.WrapWithCode(err, "Preview", errors.ErrFileOperation, "读取项目工作区文件失败")
	}

	defaultRepo, err := p.repository.Default()
	if err != nil {
		return nil, errors.Wrap(err, "Preview: 获取默认仓库失败")
	}

	repoDir, err := p.repository.Path(defaultRepo)
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
	blocked := false
	blockReason := ""
	if skillExists && compareVersions(projectVersion, repoVersion) < 0 {
		blocked = true
		blockReason = fmt.Sprintf("项目工作区技能版本 %s 低于默认仓库版本 %s，请先执行 skill-hub apply --pattern %s 后再合并反馈", projectVersion, repoVersion, skillID)
	}
	if (hasContentChanges || len(changes) > 0) && compareVersions(projectVersion, repoVersion) <= 0 {
		resolvedVersion = bumpPatchVersion(repoVersion)
		needsVersionBump = true
	}

	return &PreviewResult{
		ProjectPath:       projectRoot,
		SkillID:           skillID,
		DefaultRepo:       defaultRepo,
		SkillExists:       skillExists,
		Changes:           changes,
		HasContentChanges: hasContentChanges,
		ProjectVersion:    projectVersion,
		RepoVersion:       repoVersion,
		ResolvedVersion:   resolvedVersion,
		NeedsVersionBump:  needsVersionBump,
		Blocked:           blocked,
		BlockReason:       blockReason,
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
	if preview.Blocked {
		return nil, errors.NewWithCode("Apply", errors.ErrValidation, preview.BlockReason)
	}

	projectSkillDir := filepath.Join(preview.ProjectPath, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	var originalContent []byte
	fileMode := os.FileMode(0644)
	if preview.NeedsVersionBump {
		if info, statErr := os.Stat(projectSkillPath); statErr == nil {
			fileMode = info.Mode()
		}
		originalContent, err = os.ReadFile(projectSkillPath)
		if err != nil {
			return nil, errors.Wrap(err, "Apply: 读取项目技能文件失败")
		}
		if err := updateSkillMdVersion(projectSkillPath, preview.ResolvedVersion); err != nil {
			return nil, errors.Wrap(err, "Apply: 更新版本号失败")
		}
	}

	if err := p.repository.Archive(skillID, projectSkillDir); err != nil {
		if preview.NeedsVersionBump {
			_ = os.WriteFile(projectSkillPath, originalContent, fileMode)
		}
		return nil, errors.Wrap(err, "Apply: 归档技能到默认仓库失败")
	}

	updatedPreview, err := p.Preview(preview.ProjectPath, skillID)
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
