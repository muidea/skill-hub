package service

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
)

type DedupeOptions struct {
	Canonical string `json:"canonical,omitempty"`
	Strategy  string `json:"strategy,omitempty"`
	Report    bool   `json:"report"`
}

type DuplicateReport struct {
	Scope      string           `json:"scope"`
	Canonical  string           `json:"canonical,omitempty"`
	Strategy   string           `json:"strategy"`
	SkillCount int              `json:"skill_count"`
	Groups     []DuplicateGroup `json:"groups"`
	Conflicts  int              `json:"conflicts"`
}

type DuplicateGroup struct {
	SkillID         string              `json:"skill_id"`
	Locations       []DuplicateLocation `json:"locations"`
	ContentDiffers  bool                `json:"content_differs"`
	CanonicalSource string              `json:"canonical_source,omitempty"`
}

type DuplicateLocation struct {
	Path         string `json:"path"`
	SkillDir     string `json:"skill_dir"`
	Hash         string `json:"hash"`
	ModifiedTime string `json:"modified_time"`
	IsCanonical  bool   `json:"is_canonical"`
}

type SyncCopiesOptions struct {
	Canonical string `json:"canonical"`
	Scope     string `json:"scope"`
	DryRun    bool   `json:"dry_run"`
	NoBackup  bool   `json:"no_backup"`
}

type SyncCopiesResult struct {
	Scope     string           `json:"scope"`
	Canonical string           `json:"canonical"`
	DryRun    bool             `json:"dry_run"`
	Synced    int              `json:"synced"`
	Unchanged int              `json:"unchanged"`
	Skipped   int              `json:"skipped"`
	Failures  []SyncFailure    `json:"failures,omitempty"`
	Items     []SyncCopiesItem `json:"items"`
}

type SyncCopiesItem struct {
	SkillID   string `json:"skill_id"`
	SourceDir string `json:"source_dir"`
	TargetDir string `json:"target_dir"`
	Status    string `json:"status"`
	BackupDir string `json:"backup_dir,omitempty"`
	Message   string `json:"message,omitempty"`
}

type SyncFailure struct {
	SkillID string `json:"skill_id"`
	Path    string `json:"path"`
	Error   string `json:"error"`
}

func (p *ProjectLifecycle) Dedupe(scope string, opts DedupeOptions) (*DuplicateReport, error) {
	if strings.TrimSpace(scope) == "" {
		scope = "."
	}
	absScope, err := filepath.Abs(scope)
	if err != nil {
		return nil, errors.Wrap(err, "Dedupe: 解析scope失败")
	}
	absCanonical := ""
	if strings.TrimSpace(opts.Canonical) != "" {
		absCanonical = opts.Canonical
		if !filepath.IsAbs(absCanonical) {
			absCanonical = filepath.Join(absScope, absCanonical)
		}
		absCanonical, err = filepath.Abs(absCanonical)
		if err != nil {
			return nil, errors.Wrap(err, "Dedupe: 解析canonical失败")
		}
	}
	strategy := normalizeDedupeStrategy(opts.Strategy)
	if !isValidDedupeStrategy(strategy) {
		return nil, errors.NewWithCodef("Dedupe", errors.ErrInvalidInput, "无效的dedupe策略: %s，可用选项: newest, canonical, fail-on-conflict", opts.Strategy)
	}

	locations, err := scanSkillLocations(absScope, absCanonical)
	if err != nil {
		return nil, err
	}
	groupsByID := map[string][]DuplicateLocation{}
	for _, loc := range locations {
		groupsByID[loc.SkillID] = append(groupsByID[loc.SkillID], loc.DuplicateLocation)
	}

	report := &DuplicateReport{
		Scope:     absScope,
		Canonical: absCanonical,
		Strategy:  strategy,
	}
	for skillID, locs := range groupsByID {
		if len(locs) < 2 {
			continue
		}
		sortLocations(locs)
		group := DuplicateGroup{
			SkillID:         skillID,
			Locations:       locs,
			ContentDiffers:  groupContentDiffers(locs),
			CanonicalSource: selectCanonicalSource(locs, strategy),
		}
		if group.ContentDiffers {
			report.Conflicts++
		}
		report.Groups = append(report.Groups, group)
	}
	sort.Slice(report.Groups, func(i, j int) bool { return report.Groups[i].SkillID < report.Groups[j].SkillID })
	report.SkillCount = len(report.Groups)

	if strategy == "fail-on-conflict" && report.Conflicts > 0 {
		return report, errors.NewWithCodef("Dedupe", errors.ErrValidation, "%d 个重复技能存在内容冲突", report.Conflicts)
	}
	return report, nil
}

func (p *ProjectLifecycle) SyncCopies(opts SyncCopiesOptions) (*SyncCopiesResult, error) {
	if strings.TrimSpace(opts.Scope) == "" {
		opts.Scope = "."
	}
	if strings.TrimSpace(opts.Canonical) == "" {
		return nil, errors.NewWithCode("SyncCopies", errors.ErrInvalidInput, "缺少 canonical 目录")
	}
	absScope, err := filepath.Abs(opts.Scope)
	if err != nil {
		return nil, errors.Wrap(err, "SyncCopies: 解析scope失败")
	}
	absCanonical := opts.Canonical
	if !filepath.IsAbs(absCanonical) {
		absCanonical = filepath.Join(absScope, absCanonical)
	}
	absCanonical, err = filepath.Abs(absCanonical)
	if err != nil {
		return nil, errors.Wrap(err, "SyncCopies: 解析canonical失败")
	}

	report, err := p.Dedupe(absScope, DedupeOptions{Canonical: absCanonical, Strategy: "canonical"})
	if err != nil {
		return nil, err
	}
	result := &SyncCopiesResult{
		Scope:     absScope,
		Canonical: absCanonical,
		DryRun:    opts.DryRun,
	}
	for _, group := range report.Groups {
		source := canonicalLocation(group.Locations)
		if source == nil {
			result.Skipped++
			result.Items = append(result.Items, SyncCopiesItem{
				SkillID: group.SkillID,
				Status:  "skipped",
				Message: "canonical目录中不存在该技能",
			})
			continue
		}
		for _, target := range group.Locations {
			if target.IsCanonical {
				continue
			}
			item := SyncCopiesItem{
				SkillID:   group.SkillID,
				SourceDir: source.SkillDir,
				TargetDir: target.SkillDir,
			}
			if source.Hash == target.Hash {
				item.Status = "unchanged"
				result.Unchanged++
				result.Items = append(result.Items, item)
				continue
			}
			if opts.DryRun {
				item.Status = "planned"
				result.Items = append(result.Items, item)
				continue
			}
			backupDir := ""
			if !opts.NoBackup {
				backupDir = fmt.Sprintf("%s.bak.%s", target.SkillDir, time.Now().Format("20060102-150405"))
				if err := copySkillDirectory(target.SkillDir, backupDir); err != nil {
					recordSyncFailure(result, group.SkillID, target.SkillDir, err)
					continue
				}
				item.BackupDir = backupDir
			}
			if err := syncSkillDirectory(source.SkillDir, target.SkillDir); err != nil {
				recordSyncFailure(result, group.SkillID, target.SkillDir, err)
				continue
			}
			item.Status = "synced"
			result.Synced++
			result.Items = append(result.Items, item)
		}
	}
	if len(result.Failures) > 0 {
		return result, errors.NewWithCodef("SyncCopies", errors.ErrFileOperation, "%d 个技能副本同步失败", len(result.Failures))
	}
	return result, nil
}

type skillLocation struct {
	SkillID string
	DuplicateLocation
}

func scanSkillLocations(scope, canonical string) ([]skillLocation, error) {
	var out []skillLocation
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
		if d.Name() != "SKILL.md" {
			return nil
		}
		skillDir := filepath.Dir(path)
		parent := filepath.Dir(skillDir)
		if filepath.Base(parent) != "skills" {
			return nil
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil
		}
		hash, hashErr := hashSkillDirectory(skillDir)
		if hashErr != nil {
			return nil
		}
		isCanonical := false
		if canonical != "" {
			rel, relErr := filepath.Rel(canonical, skillDir)
			isCanonical = relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
		}
		out = append(out, skillLocation{
			SkillID: filepath.Base(skillDir),
			DuplicateLocation: DuplicateLocation{
				Path:         path,
				SkillDir:     skillDir,
				Hash:         hash,
				ModifiedTime: info.ModTime().Format(time.RFC3339),
				IsCanonical:  isCanonical,
			},
		})
		return nil
	})
	return out, err
}

func hashSkillDirectory(skillDir string) (string, error) {
	manifest := map[string]string{}
	if err := filepath.WalkDir(skillDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.Contains(d.Name(), ".bak.") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest[rel] = skill.ContentHash(content)
		return nil
	}); err != nil {
		return "", err
	}
	keys := make([]string, 0, len(manifest))
	for key := range manifest {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hasher := sha256.New()
	for _, key := range keys {
		_, _ = hasher.Write([]byte(key))
		_, _ = hasher.Write([]byte("="))
		_, _ = hasher.Write([]byte(manifest[key]))
		_, _ = hasher.Write([]byte("\n"))
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func sortLocations(locs []DuplicateLocation) {
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].IsCanonical != locs[j].IsCanonical {
			return locs[i].IsCanonical
		}
		if locs[i].ModifiedTime != locs[j].ModifiedTime {
			return locs[i].ModifiedTime > locs[j].ModifiedTime
		}
		return locs[i].Path < locs[j].Path
	})
}

func groupContentDiffers(locs []DuplicateLocation) bool {
	if len(locs) == 0 {
		return false
	}
	first := locs[0].Hash
	for _, loc := range locs[1:] {
		if loc.Hash != first {
			return true
		}
	}
	return false
}

func selectCanonicalSource(locs []DuplicateLocation, strategy string) string {
	switch strategy {
	case "canonical":
		if loc := canonicalLocation(locs); loc != nil {
			return loc.SkillDir
		}
	case "newest", "fail-on-conflict":
		if len(locs) > 0 {
			return locs[0].SkillDir
		}
	}
	return ""
}

func canonicalLocation(locs []DuplicateLocation) *DuplicateLocation {
	for i := range locs {
		if locs[i].IsCanonical {
			return &locs[i]
		}
	}
	return nil
}

func normalizeDedupeStrategy(strategy string) string {
	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		return "newest"
	}
	return strategy
}

func isValidDedupeStrategy(strategy string) bool {
	return strategy == "newest" || strategy == "canonical" || strategy == "fail-on-conflict"
}

func recordSyncFailure(result *SyncCopiesResult, skillID, path string, err error) {
	result.Failures = append(result.Failures, SyncFailure{
		SkillID: skillID,
		Path:    path,
		Error:   err.Error(),
	})
}

func copySkillDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(destination, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, content, info.Mode())
	})
}

func syncSkillDirectory(source, destination string) error {
	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}
	sourceFiles := map[string]bool{}
	if err := filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(destination, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		sourceFiles[rel] = true
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, content, info.Mode())
	}); err != nil {
		return err
	}
	return filepath.WalkDir(destination, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(destination, path)
		if err != nil {
			return err
		}
		if sourceFiles[rel] {
			return nil
		}
		return os.Remove(path)
	})
}
