package service

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
)

type ValidateOptions struct {
	ProjectPath string `json:"project_path"`
	SkillID     string `json:"skill_id,omitempty"`
	All         bool   `json:"all"`
	Fix         bool   `json:"fix"`
	Links       bool   `json:"links"`
	CheckRemote bool   `json:"check_remote"`
	ProjectRoot string `json:"project_root,omitempty"`
}

type ValidateReport struct {
	ProjectPath    string              `json:"project_path"`
	SkillID        string              `json:"skill_id,omitempty"`
	All            bool                `json:"all"`
	Fix            bool                `json:"fix"`
	Links          bool                `json:"links"`
	CheckRemote    bool                `json:"check_remote"`
	ProjectRoot    string              `json:"project_root,omitempty"`
	Total          int                 `json:"total"`
	Passed         int                 `json:"passed"`
	Failed         int                 `json:"failed"`
	Repaired       int                 `json:"repaired"`
	LinkIssueCount int                 `json:"link_issue_count"`
	Items          []ValidateItem      `json:"items"`
	Failures       []ValidateFailure   `json:"failures,omitempty"`
	LinkIssues     []MarkdownLinkIssue `json:"link_issues,omitempty"`
}

type ValidateItem struct {
	SkillID    string              `json:"skill_id"`
	SkillDir   string              `json:"skill_dir"`
	SkillMd    string              `json:"skill_md"`
	Valid      bool                `json:"valid"`
	Repaired   bool                `json:"repaired"`
	BackupPath string              `json:"backup_path,omitempty"`
	Errors     []string            `json:"errors,omitempty"`
	LinkIssues []MarkdownLinkIssue `json:"link_issues,omitempty"`
}

type ValidateFailure struct {
	SkillID string `json:"skill_id"`
	Path    string `json:"path"`
	Error   string `json:"error"`
}

type MarkdownLinkIssue struct {
	SkillID      string `json:"skill_id"`
	SourceFile   string `json:"source_file"`
	Line         int    `json:"line"`
	Link         string `json:"link"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
}

var markdownLinkPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)\s]+)(?:\s+["'][^)]*["'])?\)`)

func (p *ProjectLifecycle) ValidateProjectSkills(opts ValidateOptions) (*ValidateReport, error) {
	if strings.TrimSpace(opts.ProjectPath) == "" {
		return nil, errors.NewWithCode("ValidateProjectSkills", errors.ErrInvalidInput, "项目路径不能为空")
	}
	absProjectPath, err := filepath.Abs(opts.ProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateProjectSkills: 获取项目绝对路径失败")
	}
	opts.ProjectPath = absProjectPath
	if opts.ProjectRoot != "" {
		opts.ProjectRoot, err = filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return nil, errors.Wrap(err, "ValidateProjectSkills: 解析project-root失败")
		}
	}

	stateManager, err := p.projectStateSvc.Service().Manager()
	if err != nil {
		return nil, errors.Wrap(err, "ValidateProjectSkills: 创建状态管理器失败")
	}
	projectState, err := stateManager.LoadProjectState(absProjectPath)
	if err != nil {
		return nil, errors.Wrap(err, "ValidateProjectSkills: 加载项目状态失败")
	}

	skillIDs, err := validateSkillIDs(projectState, opts)
	if err != nil {
		return nil, err
	}
	report := &ValidateReport{
		ProjectPath: absProjectPath,
		SkillID:     opts.SkillID,
		All:         opts.All,
		Fix:         opts.Fix,
		Links:       opts.Links,
		CheckRemote: opts.CheckRemote,
		ProjectRoot: opts.ProjectRoot,
		Total:       len(skillIDs),
	}

	for _, skillID := range skillIDs {
		item := p.validateOneProjectSkill(absProjectPath, projectState, skillID, opts)
		report.Items = append(report.Items, item)
		if item.Repaired {
			report.Repaired++
		}
		if item.Valid {
			report.Passed++
		} else {
			report.Failed++
			report.Failures = append(report.Failures, ValidateFailure{
				SkillID: skillID,
				Path:    item.SkillMd,
				Error:   strings.Join(item.Errors, "; "),
			})
		}
		if len(item.LinkIssues) > 0 {
			report.LinkIssueCount += len(item.LinkIssues)
			report.LinkIssues = append(report.LinkIssues, item.LinkIssues...)
		}
	}
	if report.Failed > 0 {
		return report, errors.NewWithCodef("ValidateProjectSkills", errors.ErrValidation, "%d 个技能验证失败", report.Failed)
	}
	return report, nil
}

func validateSkillIDs(projectState *spec.ProjectState, opts ValidateOptions) ([]string, error) {
	if projectState.Skills == nil {
		projectState.Skills = map[string]spec.SkillVars{}
	}
	if opts.All {
		ids := make([]string, 0, len(projectState.Skills))
		for skillID := range projectState.Skills {
			ids = append(ids, skillID)
		}
		sort.Strings(ids)
		return ids, nil
	}
	skillID := strings.TrimSpace(opts.SkillID)
	if skillID == "" {
		return nil, errors.NewWithCode("ValidateProjectSkills", errors.ErrInvalidInput, "缺少 skill_id")
	}
	if _, ok := projectState.Skills[skillID]; !ok {
		return nil, errors.NewWithCodef("ValidateProjectSkills", errors.ErrSkillNotFound, "技能 %s 未在state.json里项目工作区登记，该技能非法", skillID)
	}
	return []string{skillID}, nil
}

func (p *ProjectLifecycle) validateOneProjectSkill(projectPath string, projectState *spec.ProjectState, skillID string, opts ValidateOptions) ValidateItem {
	skillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	skillMd := filepath.Join(skillDir, "SKILL.md")
	item := ValidateItem{
		SkillID:  skillID,
		SkillDir: skillDir,
		SkillMd:  skillMd,
		Valid:    true,
	}

	if _, err := os.Stat(skillDir); err != nil {
		item.Valid = false
		item.Errors = append(item.Errors, fmt.Sprintf("项目本地工作区目录不存在: .agents/skills/%s", skillID))
		return item
	}
	if _, err := os.Stat(skillMd); err != nil {
		item.Valid = false
		item.Errors = append(item.Errors, fmt.Sprintf("SKILL.md文件不存在: %s", skillMd))
		return item
	}

	if opts.Fix {
		result, err := RepairSkillFrontmatter(skillID, skillMd)
		if err != nil {
			item.Valid = false
			item.Errors = append(item.Errors, err.Error())
			return item
		}
		item.Repaired = result.Changed
		item.BackupPath = result.BackupPath
	}

	content, err := os.ReadFile(skillMd)
	if err != nil {
		item.Valid = false
		item.Errors = append(item.Errors, fmt.Sprintf("读取SKILL.md失败: %v", err))
		return item
	}
	if err := skill.ValidateSkillFile(content); err != nil {
		item.Valid = false
		item.Errors = append(item.Errors, err.Error())
	}

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = projectRootFromSkillMetadata(content, projectPath)
	}
	if projectRoot == "" {
		projectRoot = projectPath
	}
	if opts.Links {
		linkIssues := validateMarkdownLinks(skillID, skillDir, projectRoot, opts.CheckRemote)
		item.LinkIssues = append(item.LinkIssues, linkIssues...)
		if len(linkIssues) > 0 {
			item.Valid = false
			for _, issue := range linkIssues {
				item.Errors = append(item.Errors, fmt.Sprintf("broken link %s in %s:%d", issue.Link, issue.SourceFile, issue.Line))
			}
		}
	}
	return item
}

func projectRootFromSkillMetadata(content []byte, projectPath string) string {
	frontmatter, err := skill.ParseFrontmatter(content)
	if err != nil {
		return ""
	}
	metadata, ok := frontmatter["metadata"].(map[string]interface{})
	if !ok || metadata == nil {
		return ""
	}
	value, ok := metadata["project_root"].(string)
	if !ok {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	if filepath.Base(projectPath) == value {
		return projectPath
	}
	candidates := []string{
		filepath.Join(projectPath, value),
		filepath.Join(filepath.Dir(projectPath), value),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join(projectPath, value)
}

func validateMarkdownLinks(skillID, skillDir, projectRoot string, checkRemote bool) []MarkdownLinkIssue {
	files := markdownFiles(skillDir)
	var issues []MarkdownLinkIssue
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil || !utf8.Valid(content) {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for _, match := range markdownLinkPattern.FindAllStringSubmatch(line, -1) {
				if len(match) < 2 {
					continue
				}
				if issue, ok := validateOneMarkdownLink(skillID, file, i+1, match[1], skillDir, projectRoot, checkRemote); ok {
					issues = append(issues, issue)
				}
			}
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].SourceFile != issues[j].SourceFile {
			return issues[i].SourceFile < issues[j].SourceFile
		}
		return issues[i].Line < issues[j].Line
	})
	return issues
}

func markdownFiles(skillDir string) []string {
	var files []string
	_ = filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func validateOneMarkdownLink(skillID, sourceFile string, line int, rawLink, skillDir, projectRoot string, checkRemote bool) (MarkdownLinkIssue, bool) {
	link := strings.Trim(strings.TrimSpace(rawLink), "<>")
	if link == "" || strings.HasPrefix(link, "#") {
		return MarkdownLinkIssue{}, false
	}
	parsed, err := url.Parse(link)
	if err == nil && parsed.Scheme != "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			if !checkRemote || remoteURLExists(link) {
				return MarkdownLinkIssue{}, false
			}
			return MarkdownLinkIssue{SkillID: skillID, SourceFile: sourceFile, Line: line, Link: rawLink, Status: "broken", Reason: "远端链接不可访问"}, true
		case "file":
			return validateLocalLinkPath(skillID, sourceFile, line, rawLink, parsed.Path, skillDir, projectRoot)
		default:
			return MarkdownLinkIssue{}, false
		}
	}

	linkPath := link
	if cut := strings.IndexAny(linkPath, "?#"); cut >= 0 {
		linkPath = linkPath[:cut]
	}
	if decoded, decodeErr := url.PathUnescape(linkPath); decodeErr == nil {
		linkPath = decoded
	}
	if linkPath == "" {
		return MarkdownLinkIssue{}, false
	}
	return validateLocalLinkPath(skillID, sourceFile, line, rawLink, linkPath, skillDir, projectRoot)
}

func validateLocalLinkPath(skillID, sourceFile string, line int, rawLink, linkPath, skillDir, projectRoot string) (MarkdownLinkIssue, bool) {
	candidates := candidateLinkPaths(sourceFile, linkPath, skillDir, projectRoot)
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return MarkdownLinkIssue{}, false
		}
	}
	resolved := ""
	if len(candidates) > 0 {
		resolved = candidates[len(candidates)-1]
	}
	return MarkdownLinkIssue{
		SkillID:      skillID,
		SourceFile:   sourceFile,
		Line:         line,
		Link:         rawLink,
		ResolvedPath: resolved,
		Status:       "broken",
		Reason:       "本地链接目标不存在",
	}, true
}

func candidateLinkPaths(sourceFile, linkPath, skillDir, projectRoot string) []string {
	if filepath.IsAbs(linkPath) {
		return []string{filepath.Clean(linkPath)}
	}
	candidates := []string{
		filepath.Join(filepath.Dir(sourceFile), linkPath),
		filepath.Join(skillDir, linkPath),
	}
	if projectRoot != "" {
		candidates = append(candidates, filepath.Join(projectRoot, linkPath))
	}
	out := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if !seen[candidate] {
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func remoteURLExists(rawURL string) bool {
	client := http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
