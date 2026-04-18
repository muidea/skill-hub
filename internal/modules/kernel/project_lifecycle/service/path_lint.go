package service

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/muidea/skill-hub/pkg/errors"
)

type PathLintOptions struct {
	Scope       string `json:"scope"`
	ProjectRoot string `json:"project_root,omitempty"`
	Fix         bool   `json:"fix"`
	DryRun      bool   `json:"dry_run"`
	NoBackup    bool   `json:"no_backup"`
}

type PathLintReport struct {
	Scope        string            `json:"scope"`
	ProjectRoot  string            `json:"project_root,omitempty"`
	Fix          bool              `json:"fix"`
	DryRun       bool              `json:"dry_run"`
	FilesScanned int               `json:"files_scanned"`
	Findings     []PathLintFinding `json:"findings"`
	FindingCount int               `json:"finding_count"`
	Rewritten    int               `json:"rewritten"`
	ManualReview int               `json:"manual_review"`
	Backups      []string          `json:"backups,omitempty"`
}

type PathLintFinding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Kind        string `json:"kind"`
	Value       string `json:"value"`
	Replacement string `json:"replacement,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
}

type pathReplacement struct {
	old string
	new string
}

var localPathPattern = regexp.MustCompile(`file://[^\s\]\)'"<>]+|vscode://[^\s\]\)'"<>]+|/(?:home|Users)/[^\s\]\)'"<>]+`)

func (p *ProjectLifecycle) LintPaths(opts PathLintOptions) (*PathLintReport, error) {
	scope := strings.TrimSpace(opts.Scope)
	if scope == "" {
		scope = "."
	}
	absScope, err := filepath.Abs(scope)
	if err != nil {
		return nil, errors.Wrap(err, "LintPaths: 解析scope失败")
	}

	projectRoot := strings.TrimSpace(opts.ProjectRoot)
	if projectRoot != "" {
		if !filepath.IsAbs(projectRoot) {
			projectRoot = filepath.Join(absScope, projectRoot)
		}
		projectRoot, err = filepath.Abs(projectRoot)
		if err != nil {
			return nil, errors.Wrap(err, "LintPaths: 解析project-root失败")
		}
	}

	files, err := scanSkillTextFiles(absScope)
	if err != nil {
		return nil, err
	}
	report := &PathLintReport{
		Scope:       absScope,
		ProjectRoot: projectRoot,
		Fix:         opts.Fix,
		DryRun:      opts.DryRun,
	}

	for _, file := range files {
		content, readErr := os.ReadFile(file)
		if readErr != nil || !utf8.Valid(content) {
			continue
		}
		report.FilesScanned++
		ownerRoot := projectRoot
		if ownerRoot == "" {
			ownerRoot = inferProjectRootFromSkillFile(file)
		}
		findings, replacements := lintPathContent(file, string(content), ownerRoot, opts.Fix, opts.DryRun)
		report.Findings = append(report.Findings, findings...)
		for _, finding := range findings {
			switch finding.Status {
			case "rewritten", "would-rewrite":
				report.Rewritten++
			case "manual-review":
				report.ManualReview++
			}
		}
		if opts.Fix && len(replacements) > 0 && !opts.DryRun {
			backupPath := ""
			if !opts.NoBackup {
				backupPath = fmt.Sprintf("%s.bak.%s", file, time.Now().Format("20060102-150405"))
				if err := os.WriteFile(backupPath, content, 0644); err != nil {
					return report, errors.WrapWithCode(err, "LintPaths", errors.ErrFileOperation, "创建路径修复备份失败")
				}
				report.Backups = append(report.Backups, backupPath)
			}
			updated := applyPathReplacements(string(content), replacements)
			info, statErr := os.Stat(file)
			mode := os.FileMode(0644)
			if statErr == nil {
				mode = info.Mode()
			}
			if err := os.WriteFile(file, []byte(updated), mode); err != nil {
				return report, errors.WrapWithCode(err, "LintPaths", errors.ErrFileOperation, "写入路径修复结果失败")
			}
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].File != report.Findings[j].File {
			return report.Findings[i].File < report.Findings[j].File
		}
		return report.Findings[i].Line < report.Findings[j].Line
	})
	report.FindingCount = len(report.Findings)
	return report, nil
}

func scanSkillTextFiles(scope string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(scope, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", ".pytest_cache", "__pycache__":
				return filepath.SkipDir
			}
			return nil
		}
		skillDir := nearestSkillDir(path)
		if skillDir == "" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files, err
}

func nearestSkillDir(path string) string {
	dir := filepath.Dir(path)
	for {
		if filepath.Base(filepath.Dir(dir)) == "skills" {
			if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func inferProjectRootFromSkillFile(file string) string {
	dir := filepath.Dir(file)
	for {
		if filepath.Base(dir) == ".agents" {
			return filepath.Dir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func lintPathContent(file, content, projectRoot string, fix, dryRun bool) ([]PathLintFinding, []pathReplacement) {
	var findings []PathLintFinding
	replacementsByOld := map[string]string{}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := localPathPattern.FindAllString(line, -1)
		for _, raw := range matches {
			kind := classifyPathFinding(raw)
			replacement, ok := replacementForLocalPath(raw, kind, projectRoot)
			finding := PathLintFinding{
				File:   file,
				Line:   i + 1,
				Kind:   kind,
				Value:  raw,
				Status: "manual-review",
			}
			if ok {
				finding.Replacement = replacement
				if fix {
					finding.Status = "rewritten"
					if dryRun {
						finding.Status = "would-rewrite"
					}
					replacementsByOld[raw] = replacement
				} else {
					finding.Status = "fixable"
				}
			} else if projectRoot == "" {
				finding.Reason = "缺少 project-root，无法判断是否可改写为相对路径"
			} else {
				finding.Reason = "路径不在 project-root 下或属于外部工具链接"
			}
			findings = append(findings, finding)
		}
	}
	replacements := make([]pathReplacement, 0, len(replacementsByOld))
	for old, newValue := range replacementsByOld {
		replacements = append(replacements, pathReplacement{old: old, new: newValue})
	}
	sort.Slice(replacements, func(i, j int) bool { return len(replacements[i].old) > len(replacements[j].old) })
	return findings, replacements
}

func classifyPathFinding(value string) string {
	switch {
	case strings.HasPrefix(value, "file://"):
		return "file-url"
	case strings.HasPrefix(value, "vscode://"):
		return "vscode-url"
	case strings.HasPrefix(value, "/home/") || strings.HasPrefix(value, "/Users/"):
		return "absolute-path"
	default:
		return "unknown"
	}
}

func replacementForLocalPath(value, kind, projectRoot string) (string, bool) {
	if projectRoot == "" || kind == "vscode-url" {
		return "", false
	}
	candidate := value
	if kind == "file-url" {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Path == "" {
			return "", false
		}
		candidate = parsed.Path
	}
	rel, err := filepath.Rel(projectRoot, candidate)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func applyPathReplacements(content string, replacements []pathReplacement) string {
	if len(replacements) == 0 {
		return content
	}
	replacementsByOld := make(map[string]string, len(replacements))
	for _, replacement := range replacements {
		replacementsByOld[replacement.old] = replacement.new
	}
	return localPathPattern.ReplaceAllStringFunc(content, func(value string) string {
		if replacement, ok := replacementsByOld[value]; ok {
			return replacement
		}
		return value
	})
}
