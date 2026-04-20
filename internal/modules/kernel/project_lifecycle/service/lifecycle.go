package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
)

type ProjectLifecycle struct {
	projectStateSvc *projectstatemodule.ProjectState
	repositorySvc   *repositorymodule.Repository
}

type RegisterResult struct {
	ProjectPath string `json:"project_path"`
	SkillID     string `json:"skill_id"`
	Version     string `json:"version"`
	Target      string `json:"target"`
	LocalPath   string `json:"local_path"`
	Registered  bool   `json:"registered"`
}

type ImportOptions struct {
	Target         string `json:"target"`
	Archive        bool   `json:"archive"`
	Force          bool   `json:"force"`
	DryRun         bool   `json:"dry_run"`
	FailFast       bool   `json:"fail_fast"`
	FixFrontmatter bool   `json:"fix_frontmatter"`
}

type ImportSummary struct {
	ProjectPath string          `json:"project_path"`
	SkillsDir   string          `json:"skills_dir"`
	Discovered  int             `json:"discovered"`
	Registered  int             `json:"registered"`
	Valid       int             `json:"valid"`
	Archived    int             `json:"archived"`
	Unchanged   int             `json:"unchanged"`
	Failed      int             `json:"failed"`
	Failures    []ImportFailure `json:"failures,omitempty"`
}

type ImportFailure struct {
	SkillID string `json:"skill_id"`
	Command string `json:"command"`
	Path    string `json:"path"`
	Error   string `json:"error"`
}

type RepairResult struct {
	Changed    bool   `json:"changed"`
	BackupPath string `json:"backup_path,omitempty"`
}

type importSkillItem struct {
	ID      string
	Dir     string
	SkillMd string
}

func New() *ProjectLifecycle {
	return &ProjectLifecycle{
		projectStateSvc: projectstatemodule.New(),
		repositorySvc:   repositorymodule.New(),
	}
}

func (p *ProjectLifecycle) Register(projectPath, skillID, target string, skipValidate bool) (*RegisterResult, error) {
	if projectPath == "" {
		return nil, errors.NewWithCode("Register", errors.ErrInvalidInput, "项目路径不能为空")
	}
	if !isValidSkillName(skillID) {
		return nil, errors.NewWithCodef("Register", errors.ErrValidation, "技能ID '%s' 格式无效。应使用小写字母、数字和连字符，例如：my-logic-skill", skillID)
	}
	target = normalizeTargetOrDefault(target)
	if !isValidTarget(target) {
		return nil, errors.NewWithCodef("Register", errors.ErrInvalidInput, "无效的项目目标: %s。可用选项: cursor, claude, claude_code, open_code", target)
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Register: 获取项目绝对路径失败")
	}

	skillDir := filepath.Join(absProjectPath, ".agents", "skills", skillID)
	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	content, err := os.ReadFile(skillFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewWithCodef("Register", errors.ErrFileNotFound, "未找到已有技能文件: %s", skillFilePath)
		}
		return nil, errors.WrapWithCode(err, "Register", errors.ErrFileOperation, "读取SKILL.md失败")
	}
	if !skipValidate {
		if err := skill.ValidateSkillFile(content); err != nil {
			return nil, errors.Wrap(err, "技能文件验证失败，可先运行 'skill-hub validate <id> --fix' 或使用 --skip-validate")
		}
	}

	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "Register: 创建状态管理器失败")
	}
	projectState, err := stateManager.LoadProjectState(absProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Register: 加载项目状态失败")
	}
	if projectState.Skills == nil {
		projectState.Skills = make(map[string]spec.SkillVars)
	}

	_, existed := projectState.Skills[skillID]
	existing := projectState.Skills[skillID]
	existing.SkillID = skillID
	existing.Version = skill.ExtractVersion(content)
	if existing.Variables == nil {
		existing.Variables = map[string]string{}
	}
	existing.Variables["target"] = target
	projectState.Skills[skillID] = existing
	if projectState.PreferredTarget == "" {
		projectState.PreferredTarget = target
	}

	if err := stateManager.SaveProjectState(projectState); err != nil {
		return nil, errors.Wrap(err, "Register: 保存项目状态失败")
	}

	return &RegisterResult{
		ProjectPath: absProjectPath,
		SkillID:     skillID,
		Version:     existing.Version,
		Target:      target,
		LocalPath:   skillFilePath,
		Registered:  !existed,
	}, nil
}

func (p *ProjectLifecycle) Import(projectPath, skillsDir string, opts ImportOptions) (*ImportSummary, error) {
	if projectPath == "" {
		return nil, errors.NewWithCode("Import", errors.ErrInvalidInput, "项目路径不能为空")
	}
	opts.Target = normalizeTargetOrDefault(opts.Target)
	if !isValidTarget(opts.Target) {
		return nil, errors.NewWithCodef("Import", errors.ErrInvalidInput, "无效的项目目标: %s。可用选项: cursor, claude, claude_code, open_code", opts.Target)
	}

	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Import: 获取项目绝对路径失败")
	}
	absSkillsDir := skillsDir
	if !filepath.IsAbs(absSkillsDir) {
		absSkillsDir = filepath.Join(absProjectPath, skillsDir)
	}
	absSkillsDir, err = filepath.Abs(absSkillsDir)
	if err != nil {
		return nil, errors.WrapWithCode(err, "Import", errors.ErrInvalidInput, "解析导入目录失败")
	}

	items, err := scanImportSkills(absSkillsDir)
	if err != nil {
		return nil, err
	}
	summary := &ImportSummary{
		ProjectPath: absProjectPath,
		SkillsDir:   absSkillsDir,
		Discovered:  len(items),
	}

	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "Import: 创建状态管理器失败")
	}
	projectState, err := stateManager.LoadProjectState(absProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "Import: 加载项目状态失败")
	}
	if projectState.Skills == nil {
		projectState.Skills = make(map[string]spec.SkillVars)
	}

	for _, item := range items {
		err := p.importOneSkill(projectState, stateManager, item, opts, summary)
		if err != nil && opts.FailFast {
			return summary, err
		}
	}
	if summary.Failed > 0 {
		return summary, errors.NewWithCodef("Import", errors.ErrValidation, "%d 个技能导入失败", summary.Failed)
	}
	return summary, nil
}

func (p *ProjectLifecycle) importOneSkill(projectState *spec.ProjectState, stateManager interface {
	SaveProjectState(*spec.ProjectState) error
}, item importSkillItem, opts ImportOptions, summary *ImportSummary) error {
	if !isValidSkillName(item.ID) {
		err := errors.NewWithCodef("importOneSkill", errors.ErrValidation, "技能ID格式无效: %s", item.ID)
		recordImportFailure(summary, item, "scan", err)
		return err
	}

	content, err := os.ReadFile(item.SkillMd)
	if err != nil {
		err = errors.WrapWithCode(err, "importOneSkill", errors.ErrFileOperation, "读取SKILL.md失败")
		recordImportFailure(summary, item, "read", err)
		return err
	}

	if opts.FixFrontmatter {
		repaired, changed, repairErr := buildRepairedSkillContent(item.ID, content, opts.Target)
		if repairErr != nil {
			recordImportFailure(summary, item, "fix-frontmatter", repairErr)
			return repairErr
		}
		if changed {
			if opts.DryRun {
				content = repaired
			} else {
				if _, repairErr := RepairSkillFrontmatter(item.ID, item.SkillMd, opts.Target); repairErr != nil {
					recordImportFailure(summary, item, "fix-frontmatter", repairErr)
					return repairErr
				}
				content, _ = os.ReadFile(item.SkillMd)
			}
		}
	}

	if err := skill.ValidateSkillFile(content); err != nil {
		err = errors.Wrap(err, "技能验证失败")
		recordImportFailure(summary, item, "validate", err)
		return err
	}
	summary.Valid++

	_, alreadyRegistered := projectState.Skills[item.ID]
	if !alreadyRegistered {
		if !opts.DryRun {
			existing := spec.SkillVars{
				SkillID: item.ID,
				Version: skill.ExtractVersion(content),
				Variables: map[string]string{
					"target": opts.Target,
				},
			}
			projectState.Skills[item.ID] = existing
			if projectState.PreferredTarget == "" {
				projectState.PreferredTarget = opts.Target
			}
			if err := stateManager.SaveProjectState(projectState); err != nil {
				recordImportFailure(summary, item, "register", err)
				return err
			}
		}
		summary.Registered++
	}

	if !opts.Archive {
		if alreadyRegistered {
			summary.Unchanged++
		}
		return nil
	}

	if p.equalToRepo(item.ID, item.Dir) {
		summary.Unchanged++
		return nil
	}
	if opts.DryRun {
		return nil
	}
	if err := p.repositorySvc.Service().ArchiveToDefaultRepository(item.ID, item.Dir); err != nil {
		recordImportFailure(summary, item, "archive", err)
		return err
	}
	summary.Archived++
	return nil
}

func RepairSkillFrontmatter(skillID, skillMdPath, target string) (*RepairResult, error) {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, errors.WrapWithCode(err, "RepairSkillFrontmatter", errors.ErrFileOperation, "读取SKILL.md失败")
	}
	repaired, changed, err := buildRepairedSkillContent(skillID, content, target)
	if err != nil {
		return nil, err
	}
	if !changed {
		return &RepairResult{Changed: false}, nil
	}

	backupPath := fmt.Sprintf("%s.bak.%s", skillMdPath, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return nil, errors.WrapWithCode(err, "RepairSkillFrontmatter", errors.ErrFileOperation, "创建SKILL.md备份失败")
	}
	if err := os.WriteFile(skillMdPath, repaired, 0644); err != nil {
		return nil, errors.WrapWithCode(err, "RepairSkillFrontmatter", errors.ErrFileOperation, "写入修复后的SKILL.md失败")
	}
	return &RepairResult{Changed: true, BackupPath: backupPath}, nil
}

func scanImportSkills(skillsDir string) ([]importSkillItem, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, errors.WrapWithCode(err, "scanImportSkills", errors.ErrFileNotFound, "读取导入目录失败")
	}

	var items []importSkillItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillID := entry.Name()
		skillDir := filepath.Join(skillsDir, skillID)
		skillMd := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillMd); err == nil {
			items = append(items, importSkillItem{ID: skillID, Dir: skillDir, SkillMd: skillMd})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (p *ProjectLifecycle) equalToRepo(skillID, skillDir string) bool {
	defaultRepo, err := p.repositorySvc.Service().DefaultRepository()
	if err != nil {
		return false
	}
	repoPath, err := p.repositorySvc.Service().Path(defaultRepo.Name)
	if err != nil {
		return false
	}
	repoSkillDir := filepath.Join(repoPath, "skills", skillID)
	equal, err := skillDirsEqual(skillDir, repoSkillDir)
	return err == nil && equal
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

func recordImportFailure(summary *ImportSummary, item importSkillItem, command string, err error) {
	summary.Failed++
	summary.Failures = append(summary.Failures, ImportFailure{
		SkillID: item.ID,
		Command: command,
		Path:    item.SkillMd,
		Error:   err.Error(),
	})
}

func buildRepairedSkillContent(skillID string, content []byte, target string) ([]byte, bool, error) {
	frontmatterBytes, body, hasFrontmatter, validFence := splitSkillFrontmatter(content)

	frontmatter := map[string]interface{}{}
	changed := !hasFrontmatter || !validFence
	if validFence {
		if err := yaml.Unmarshal(frontmatterBytes, &frontmatter); err != nil {
			frontmatter = map[string]interface{}{}
			changed = true
		}
	}
	if frontmatter == nil {
		frontmatter = map[string]interface{}{}
		changed = true
	}

	if str, ok := frontmatter["name"].(string); !ok || strings.TrimSpace(str) == "" {
		frontmatter["name"] = skillID
		changed = true
	}
	if str, ok := frontmatter["description"].(string); !ok || strings.TrimSpace(str) == "" {
		frontmatter["description"] = inferSkillDescription(body, skillID)
		changed = true
	}
	metadata, ok := frontmatter["metadata"].(map[string]interface{})
	if !ok || metadata == nil {
		metadata = map[string]interface{}{}
		frontmatter["metadata"] = metadata
		changed = true
	}
	if str, ok := metadata["version"].(string); !ok || strings.TrimSpace(str) == "" {
		metadata["version"] = "1.0.0"
		changed = true
	}
	if str, ok := metadata["author"].(string); !ok || strings.TrimSpace(str) == "" {
		metadata["author"] = currentAuthor()
		changed = true
	}

	if !changed {
		return content, false, nil
	}

	marshaled, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, false, errors.WrapWithCode(err, "buildRepairedSkillContent", errors.ErrValidation, "生成frontmatter失败")
	}

	var out bytes.Buffer
	out.WriteString("---\n")
	out.Write(marshaled)
	out.WriteString("---\n")
	out.Write(body)
	return out.Bytes(), true, nil
}

func splitSkillFrontmatter(content []byte) ([]byte, []byte, bool, bool) {
	firstLineLen := 0
	switch {
	case bytes.HasPrefix(content, []byte("---\n")):
		firstLineLen = len("---\n")
	case bytes.HasPrefix(content, []byte("---\r\n")):
		firstLineLen = len("---\r\n")
	default:
		return nil, content, false, false
	}

	pos := firstLineLen
	for pos <= len(content) {
		lineStart := pos
		nextNewline := bytes.IndexByte(content[pos:], '\n')
		lineEnd := len(content)
		next := len(content)
		if nextNewline >= 0 {
			lineEnd = pos + nextNewline
			next = lineEnd + 1
		}

		line := bytes.TrimSuffix(content[lineStart:lineEnd], []byte("\r"))
		if bytes.Equal(line, []byte("---")) {
			return content[firstLineLen:lineStart], content[next:], true, true
		}
		if next == len(content) {
			break
		}
		pos = next
	}
	return nil, content, true, false
}

func inferSkillDescription(body []byte, skillID string) string {
	inCodeBlock := false
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock || trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, ">") {
			continue
		}
		if len(trimmed) > 200 {
			trimmed = trimmed[:200]
		}
		return trimmed
	}
	return fmt.Sprintf("Imported legacy skill %s.", skillID)
}

func currentAuthor() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

func normalizeTargetOrDefault(target string) string {
	if strings.TrimSpace(target) == "" {
		return spec.TargetOpenCode
	}
	return spec.NormalizeTarget(strings.TrimSpace(target))
}

func isValidTarget(target string) bool {
	switch target {
	case spec.TargetCursor, spec.TargetClaude, spec.TargetClaudeCode, spec.TargetOpenCode, "opencode":
		return true
	default:
		return false
	}
}

func isValidSkillName(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}
