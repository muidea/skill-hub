package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muidea/skill-hub/internal/adapter"
	"github.com/muidea/skill-hub/internal/config"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
)

// SkillRepository 管理技能Git仓库
type SkillRepository struct {
	repo     *Repository
	stashRef string
}

// StashRef returns the stash created by the latest pull, without restoring it.
func (sr *SkillRepository) StashRef() string { return sr.stashRef }

type skillVersionConflict struct {
	SkillID       string
	LocalVersion  string
	RemoteVersion string
}

// NewSkillRepository 创建技能仓库管理器
func NewSkillRepository() (*SkillRepository, error) {
	repo, err := NewSkillsRepository()
	if err != nil {
		return nil, err
	}
	return &SkillRepository{repo: repo}, nil
}

// Sync 同步技能仓库（拉取最新更改）
func (sr *SkillRepository) Sync() error {
	return sr.SyncWithOptions(SyncOptions{})
}

type SyncOptions struct {
	Force bool
}

// SyncWithOptions 同步技能仓库（拉取最新更改）
func (sr *SkillRepository) SyncWithOptions(opts SyncOptions) error {
	sr.stashRef = ""
	fmt.Println("正在同步技能仓库...")

	if !sr.repo.IsInitialized() {
		return fmt.Errorf("技能仓库未初始化，请先设置远程仓库URL")
	}

	// 获取当前状态
	files, err := sr.repo.pullChangedFiles()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	// 检查是否有未提交的更改
	if hasPullConflicts(files) {
		return sr.pullFailure("merge_conflict", files, nil, "")
	}
	hasChanges := len(files) > 0
	if hasChanges {
		updates, err := sr.CheckUpdates()
		if err != nil {
			return fmt.Errorf("检查远程更新失败: %w", err)
		}
		if updates != nil && !updates.HasUpdates {
			fmt.Println("✅ 远程仓库已是最新，保留本地未提交更改")
			return nil
		}

		if !opts.Force {
			return sr.pullFailure("blocked_dirty", files, updates, "")
		}
		if err := sr.stashLocalChanges(); err != nil {
			return err
		}
	}

	// 拉取最新更改
	fmt.Println("从远程仓库拉取最新更改...")
	if err := sr.repo.Pull(); err != nil {
		if changed, inspectErr := sr.repo.pullChangedFiles(); inspectErr == nil {
			d := sr.repo.pullDiagnostic("", changed, nil)
			d.StashRef = sr.stashRef
			switch {
			case hasPullConflicts(changed):
				d.Kind = "merge_conflict"
			case apperrors.IsCode(err, apperrors.ErrGitDiverged):
				d.Kind = "divergent"
			case apperrors.IsCode(err, apperrors.ErrGitDirty):
				d.Kind = "blocked_dirty"
			}
			if d.Kind != "" {
				return pullDiagnosticError(d)
			}
		}
		if sr.stashRef != "" {
			return &apperrors.AppError{Op: "pull", Code: apperrors.ErrGitOperation, Err: err,
				Message: "拉取失败；本地改动仍保存在 stash " + sr.stashRef + "，未自动恢复",
				Details: map[string]any{"stash_ref": sr.stashRef}}
		}
		return fmt.Errorf("拉取失败: %w", err)
	}

	fmt.Println("✅ 技能仓库同步完成")
	return nil
}

func hasRepositoryChanges(status string) bool {
	for _, line := range strings.Split(status, "\n") {
		if isRepositoryChangeLine(line) {
			return true
		}
	}
	return false
}

func isRepositoryChangeLine(line string) bool {
	if len(line) < 2 {
		return false
	}
	if strings.HasPrefix(line, "?? ") {
		return true
	}
	if len(line) < 3 || line[2] != ' ' {
		return false
	}
	return isGitStatusCode(line[0]) || isGitStatusCode(line[1])
}

func isGitStatusCode(status byte) bool {
	switch status {
	case 'M', 'A', 'D', 'R', 'C', 'U', 'T':
		return true
	default:
		return false
	}
}

func (sr *SkillRepository) stashLocalChanges() error {
	message := fmt.Sprintf("skill-hub pull auto-stash %s", time.Now().Format("20060102-150405"))
	fmt.Printf("使用 git stash 暂存本地更改: %s\n", message)
	output, err := exec.Command("git", "-C", sr.repo.GetPath(), "stash", "push", "-u", "-m", message).CombinedOutput()
	if err != nil {
		return fmt.Errorf("自动暂存本地更改失败: %s: %w", strings.TrimSpace(string(output)), err)
	}
	ref, err := sr.repo.pullCommand("rev-parse", "--verify", "refs/stash").Output()
	if err != nil {
		return fmt.Errorf("更改已暂存，但读取 stash 引用失败: %w", err)
	}
	sr.stashRef = strings.TrimSpace(string(ref))
	fmt.Printf("✅ 本地更改已保存在 stash %s\n", sr.stashRef)
	return nil
}

func (sr *SkillRepository) CheckUpdates() (*RemoteUpdateStatus, error) {
	return sr.repo.CheckRemoteUpdates()
}

// PushChanges 推送本地更改到远程仓库
func (sr *SkillRepository) PushChanges(message string) error {
	if !sr.repo.IsInitialized() {
		return fmt.Errorf("技能仓库未初始化，请先设置远程仓库URL")
	}
	if sr.repo.remoteURL == "" {
		return fmt.Errorf("未设置远程仓库URL")
	}

	if err := sr.resolveRemoteSkillUpdatesByVersion(); err != nil {
		return err
	}

	// 检查是否有更改
	status, err := sr.repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	if hasRepositoryChanges(status) {
		if message == "" {
			message = SuggestedCommitMessageFromStatus(status)
		}

		fmt.Println("提交更改...")
		if err := sr.repo.Commit(message); err != nil {
			return fmt.Errorf("提交失败: %w", err)
		}
	}

	updates, err := sr.repo.CheckRemoteUpdates()
	if err != nil {
		return fmt.Errorf("检查本地待推送提交失败: %w", err)
	}
	if updates == nil || updates.Ahead == 0 {
		return fmt.Errorf("没有要推送的更改")
	}

	// 推送到远程
	fmt.Println("推送到远程仓库...")
	if err := sr.repo.Push(); err != nil {
		if _, ok := err.(*NonFastForwardError); ok {
			fmt.Println("远程仓库已有新提交，按技能版本自动合并后重试推送...")
			if resolveErr := sr.resolveRemoteSkillUpdatesByVersion(); resolveErr != nil {
				return fmt.Errorf("自动合并远程更新失败: %w", resolveErr)
			}
			if err := sr.repo.Push(); err != nil {
				return fmt.Errorf("推送失败: %w", err)
			}
			fmt.Println("✅ 更改已推送到远程仓库")
			return nil
		}
		return fmt.Errorf("推送失败: %w", err)
	}

	fmt.Println("✅ 更改已推送到远程仓库")
	return nil
}

func (sr *SkillRepository) resolveRemoteSkillUpdatesByVersion() error {
	updates, err := sr.repo.CheckRemoteUpdates()
	if err != nil {
		return fmt.Errorf("检查远程更新失败: %w", err)
	}
	if updates == nil || updates.Behind == 0 {
		return nil
	}

	status, err := sr.repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	hadLocalChanges := hasRepositoryChanges(status)
	stashed := false
	if hadLocalChanges {
		if err := sr.stashLocalChanges(); err != nil {
			return err
		}
		stashed = true
	}

	if updates.Ahead > 0 {
		fmt.Println("本地与远程存在分叉，按 skill 版本自动合并远程更新...")
		if err := sr.repo.MergeRemoteMainAllowConflicts(); err != nil {
			return err
		}
		if err := sr.resolveSkillConflictsByVersion(); err != nil {
			return err
		}
	} else {
		fmt.Println("从远程仓库拉取最新更改...")
		if err := sr.repo.Pull(); err != nil {
			if stashed {
				return fmt.Errorf("拉取远程更新失败，本地更改仍保留在 stash 中: %w", err)
			}
			return fmt.Errorf("拉取远程更新失败: %w", err)
		}
	}

	if stashed {
		fmt.Println("恢复本地更改并按 skill 版本解决冲突...")
		if err := sr.popStashedChangesAllowConflicts(); err != nil {
			return err
		}
		if err := sr.resolveSkillConflictsByVersion(); err != nil {
			return err
		}
	}

	if err := sr.UpdateRegistry(); err != nil {
		return fmt.Errorf("刷新技能索引失败: %w", err)
	}
	if err := sr.commitAutoMergeIfNeeded(); err != nil {
		return err
	}
	return nil
}

func (sr *SkillRepository) popStashedChangesAllowConflicts() error {
	cmd := exec.Command("git", "-C", sr.repo.GetPath(), "stash", "pop")
	// Force C locale so conflict markers ("CONFLICT") match regardless of
	// the host's LANG/LC_ALL.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Print(string(output))
	}
	if err == nil {
		return nil
	}
	if strings.Contains(string(output), "CONFLICT") {
		return nil
	}
	return fmt.Errorf("恢复本地更改失败: %s: %w", strings.TrimSpace(string(output)), err)
}

func (sr *SkillRepository) resolveSkillConflictsByVersion() error {
	if err := sr.resolveRegistryConflict(); err != nil {
		return err
	}

	unmerged, err := sr.unmergedSkillIDs()
	if err != nil {
		return err
	}
	if len(unmerged) == 0 {
		return nil
	}

	var unresolved []skillVersionConflict
	for _, skillID := range unmerged {
		choice, localVersion, remoteVersion, err := sr.chooseSkillConflictSide(skillID)
		if err != nil {
			return err
		}
		switch choice {
		case "ours":
			if err := sr.checkoutConflictSkillSide(skillID, "--ours"); err != nil {
				return err
			}
			fmt.Printf("保留本地版本: %s local=%s remote=%s\n", skillID, localVersion, remoteVersion)
		case "theirs":
			if err := sr.checkoutConflictSkillSide(skillID, "--theirs"); err != nil {
				return err
			}
			fmt.Printf("采用远端版本: %s local=%s remote=%s\n", skillID, localVersion, remoteVersion)
		default:
			unresolved = append(unresolved, skillVersionConflict{
				SkillID:       skillID,
				LocalVersion:  localVersion,
				RemoteVersion: remoteVersion,
			})
		}
	}

	if len(unresolved) > 0 {
		var lines []string
		for _, item := range unresolved {
			lines = append(lines, fmt.Sprintf("%s local=%s remote=%s", item.SkillID, item.LocalVersion, item.RemoteVersion))
		}
		return fmt.Errorf("存在版本相同但内容冲突的技能，无法自动选择:\n%s", strings.Join(lines, "\n"))
	}

	return sr.commitAutoMergeIfNeeded()
}

func (sr *SkillRepository) commitAutoMergeIfNeeded() error {
	status, err := sr.repo.GetStatus()
	if err != nil {
		return err
	}
	if hasRepositoryChanges(status) {
		return sr.repo.CommitWithSystemGit("resolve skill version conflicts")
	}
	return nil
}

func (sr *SkillRepository) resolveRegistryConflict() error {
	output, err := exec.Command("git", "-C", sr.repo.GetPath(), "diff", "--name-only", "--diff-filter=U", "--", "registry.json").CombinedOutput()
	if err != nil {
		return fmt.Errorf("检查 registry.json 冲突失败: %s: %w", strings.TrimSpace(string(output)), err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return nil
	}
	if err := sr.UpdateRegistry(); err != nil {
		return fmt.Errorf("重建 registry.json 失败: %w", err)
	}
	output, err = exec.Command("git", "-C", sr.repo.GetPath(), "add", "registry.json").CombinedOutput()
	if err != nil {
		return fmt.Errorf("暂存 registry.json 合并结果失败: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (sr *SkillRepository) unmergedSkillIDs() ([]string, error) {
	output, err := exec.Command("git", "-C", sr.repo.GetPath(), "diff", "--name-only", "--diff-filter=U").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("读取冲突文件失败: %s: %w", strings.TrimSpace(string(output)), err)
	}

	ids := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Split(filepath.ToSlash(strings.TrimSpace(line)), "/")
		if len(parts) >= 3 && parts[0] == "skills" {
			ids[parts[1]] = true
		}
	}

	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func (sr *SkillRepository) chooseSkillConflictSide(skillID string) (string, string, string, error) {
	localVersion, err := sr.conflictSkillVersion(skillID, ":2:")
	if err != nil {
		return "", "", "", err
	}
	remoteVersion, err := sr.conflictSkillVersion(skillID, ":3:")
	if err != nil {
		return "", "", "", err
	}

	cmp := compareSkillVersions(localVersion, remoteVersion)
	switch {
	case cmp > 0:
		return "ours", localVersion, remoteVersion, nil
	case cmp < 0:
		return "theirs", localVersion, remoteVersion, nil
	default:
		return "", localVersion, remoteVersion, nil
	}
}

func (sr *SkillRepository) conflictSkillVersion(skillID, stagePrefix string) (string, error) {
	ref := fmt.Sprintf("%sskills/%s/SKILL.md", stagePrefix, skillID)
	output, err := exec.Command("git", "-C", sr.repo.GetPath(), "show", ref).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("读取技能 %s 的冲突版本失败: %s: %w", skillID, strings.TrimSpace(string(output)), err)
	}
	return skill.ExtractVersion(output), nil
}

func (sr *SkillRepository) checkoutConflictSkillSide(skillID, side string) error {
	pattern := fmt.Sprintf("skills/%s", skillID)
	output, err := exec.Command("git", "-C", sr.repo.GetPath(), "checkout", side, "--", pattern).CombinedOutput()
	if err != nil {
		return fmt.Errorf("选择技能 %s 的%s版本失败: %s: %w", skillID, side, strings.TrimSpace(string(output)), err)
	}
	output, err = exec.Command("git", "-C", sr.repo.GetPath(), "add", pattern).CombinedOutput()
	if err != nil {
		return fmt.Errorf("暂存技能 %s 合并结果失败: %s: %w", skillID, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func compareSkillVersions(v1, v2 string) int {
	v1 = strings.Trim(v1, `" vV`)
	v2 = strings.Trim(v2, `" vV`)
	if v1 == v2 {
		return 0
	}

	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")
	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}
	for i := 0; i < maxLen; i++ {
		num1 := 0
		num2 := 0
		if i < len(v1Parts) {
			fmt.Sscanf(v1Parts[i], "%d", &num1)
		}
		if i < len(v2Parts) {
			fmt.Sscanf(v2Parts[i], "%d", &num2)
		}
		if num1 > num2 {
			return 1
		}
		if num1 < num2 {
			return -1
		}
	}

	if v1 > v2 {
		return 1
	}
	return -1
}

func SuggestedCommitMessageFromStatus(status string) string {
	return SuggestedCommitMessage(changedFilesFromStatus(status))
}

func SuggestedCommitMessage(changedFiles []string) string {
	skillIDs := skillIDsFromChangedFiles(changedFiles)
	if len(skillIDs) == 0 {
		for _, file := range changedFiles {
			if strings.TrimSpace(file) == "registry.json" {
				return "更新技能索引"
			}
		}
		return "更新技能仓库"
	}
	if len(skillIDs) == 1 {
		return fmt.Sprintf("更新技能: %s", skillIDs[0])
	}
	if len(skillIDs) <= 3 {
		return fmt.Sprintf("更新技能: %s", strings.Join(skillIDs, ", "))
	}
	return fmt.Sprintf("更新 %d 个技能: %s 等", len(skillIDs), strings.Join(skillIDs[:3], ", "))
}

func changedFilesFromStatus(status string) []string {
	var files []string
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if !isRepositoryChangeLine(line) {
			continue
		}
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		} else {
			files = append(files, strings.TrimSpace(line))
		}
	}
	return files
}

func skillIDsFromChangedFiles(changedFiles []string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, file := range changedFiles {
		parts := strings.Split(filepath.ToSlash(strings.TrimSpace(file)), "/")
		if len(parts) < 2 || parts[0] != "skills" || parts[1] == "" {
			continue
		}
		if seen[parts[1]] {
			continue
		}
		seen[parts[1]] = true
		ids = append(ids, parts[1])
	}
	sort.Strings(ids)
	return ids
}

// CloneRemote 克隆远程技能仓库
func (sr *SkillRepository) CloneRemote(url string) error {
	fmt.Printf("正在克隆远程技能仓库: %s\n", url)

	// 获取技能目录路径
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	// 如果目录已存在，备份
	if _, err := os.Stat(skillsDir); err == nil {
		backupDir := skillsDir + ".bak." + time.Now().Format("20060102-150405")
		fmt.Printf("备份现有技能目录到: %s\n", backupDir)
		if err := os.Rename(skillsDir, backupDir); err != nil {
			return fmt.Errorf("备份失败: %w", err)
		}
	}

	// 克隆仓库
	if err := sr.repo.Clone(url); err != nil {
		return fmt.Errorf("克隆失败: %w", err)
	}

	// 更新配置中的远程URL（多仓库模式）
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// 保存配置 - 更新默认仓库的URL
	if cfg.MultiRepo != nil {
		if defaultRepo, exists := cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo]; exists {
			defaultRepo.URL = url
			cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo] = defaultRepo

			// 保存更新后的配置
			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}
	}

	fmt.Println("✅ 远程技能仓库克隆完成")

	// 清理可能创建的备份目录
	if err := adapter.CleanupTimestampedBackupDirs(skillsDir); err != nil {
		fmt.Printf("⚠️  清理备份目录失败: %v\n", err)
	}

	return nil
}

// GetStatus 获取技能仓库状态
func (sr *SkillRepository) GetStatus() (string, error) {
	// 检查仓库目录是否存在且有.git目录
	repoPath := sr.repo.GetPath()
	gitDir := filepath.Join(repoPath, ".git")

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "技能仓库未初始化", nil
	}

	// 尝试获取git状态
	status, err := sr.repo.GetStatus()
	if err != nil {
		return "", err
	}

	result := "技能仓库状态:\n"

	// 显示远程URL（如果有）
	if sr.repo.remoteURL != "" {
		result += fmt.Sprintf("远程仓库: %s\n", sr.repo.remoteURL)
	} else {
		result += "远程仓库: 未设置\n"
	}

	// 显示最新提交（如果有）
	latestCommit, err := sr.repo.GetLatestCommit()
	if err == nil {
		result += fmt.Sprintf("最新提交: %s\n", latestCommit)
	}

	result += "文件状态:\n"
	result += status

	return result, nil
}

// ListSkillsFromRemote 从远程仓库列出技能
func (sr *SkillRepository) ListSkillsFromRemote() ([]*spec.Skill, error) {
	// 先同步到最新
	if err := sr.Sync(); err != nil {
		return nil, err
	}

	return sr.ListLocalSkills()
}

// ListLocalSkills 从本地仓库列出技能
func (sr *SkillRepository) ListLocalSkills() ([]*spec.Skill, error) {
	// 加载所有技能
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}

	// 只使用标准结构：直接从skills目录加载
	skills, err := sr.loadSkillsFromDirectory(skillsDir, false)
	if err != nil {
		return nil, err
	}

	return skills, nil
}

// loadSkillsFromDirectory 从目录加载技能
func (sr *SkillRepository) loadSkillsFromDirectory(dir string, recursive bool) ([]*spec.Skill, error) {
	var skills []*spec.Skill

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(dir, skillID)

		// 尝试加载技能
		skill, err := sr.loadSkill(skillDir, skillID)
		if err != nil {
			// 如果是递归模式，继续检查子目录
			if recursive {
				subSkills, _ := sr.loadSkillsFromDirectory(skillDir, true)
				skills = append(skills, subSkills...)
			}
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// loadSkill 加载单个技能
func (sr *SkillRepository) loadSkill(skillDir, skillID string) (*spec.Skill, error) {
	// 只支持SKILL.md格式
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return sr.loadSkillFromMarkdown(skillMdPath, skillID)
	}

	return nil, fmt.Errorf("未找到SKILL.md文件")
}

func (sr *SkillRepository) loadSkillFromMarkdown(mdPath, skillID string) (*spec.Skill, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	return skill.ParseSkill(content, skillID)
}

// ImportSkill 从远程仓库导入单个技能
func (sr *SkillRepository) ImportSkill(skillID string) error {
	// 先同步到最新
	if err := sr.Sync(); err != nil {
		return err
	}

	// 检查技能是否存在
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(skillsDir, skillID)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 在远程仓库中不存在", skillID)
	}

	// 检查技能文件
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 缺少SKILL.md文件", skillID)
	}

	fmt.Printf("✅ 技能 '%s' 已从远程仓库导入\n", skillID)
	return nil
}

// CreateSkill 创建新技能并推送到远程
func (sr *SkillRepository) CreateSkill(skill *spec.Skill, promptContent string) error {
	// 验证技能信息
	if skill.ID == "" {
		return fmt.Errorf("技能ID不能为空")
	}
	if skill.Name == "" {
		return fmt.Errorf("技能名称不能为空")
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}

	// 创建技能目录
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(skillsDir, skill.ID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 保存SKILL.md（包含frontmatter和内容）
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 构建frontmatter
	frontmatter := fmt.Sprintf(`---
name: %s
description: %s
metadata:
  version: %s
  author: %s
  tags: %s
---
`,
		skill.Name,
		skill.Description,
		skill.Version,
		skill.Author,
		strings.Join(skill.Tags, ","))

	// 组合frontmatter和内容
	skillContent := frontmatter + promptContent

	if err := os.WriteFile(skillMdPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("保存SKILL.md失败: %w", err)
	}

	fmt.Printf("✅ 技能 '%s' 创建成功\n", skill.ID)
	if err := sr.UpdateRegistry(); err != nil {
		fmt.Printf("⚠️  技能索引刷新失败: %v\n", err)
	}

	// 推送到远程仓库
	if sr.repo.IsInitialized() {
		message := fmt.Sprintf("添加新技能: %s", skill.ID)
		if err := sr.PushChanges(message); err != nil {
			fmt.Printf("⚠️  技能创建成功，但推送到远程失败: %v\n", err)
			fmt.Println("   使用 'skill-hub git push' 手动推送")
		}
	}

	return nil
}

// UpdateRegistry 更新技能注册表
func (sr *SkillRepository) UpdateRegistry() error {
	skills, err := sr.ListLocalSkills()
	if err != nil {
		return err
	}

	registry := buildRegistryFromSkills(skills)

	registryData, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化注册表失败: %w", err)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}
	if cfg.MultiRepo == nil {
		return fmt.Errorf("多仓库配置未初始化")
	}
	cacheRegistryPath, err := config.GetRepositoryCacheRegistryPath(cfg.MultiRepo.DefaultRepo)
	if err != nil {
		return fmt.Errorf("获取本地索引缓存路径失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cacheRegistryPath), 0755); err != nil {
		return fmt.Errorf("创建本地索引缓存目录失败: %w", err)
	}
	if err := os.WriteFile(cacheRegistryPath, registryData, 0644); err != nil {
		return fmt.Errorf("保存本地索引缓存失败: %w", err)
	}

	rootRegistryPath, err := config.GetRegistryPath()
	if err == nil {
		if err := os.MkdirAll(filepath.Dir(rootRegistryPath), 0755); err != nil {
			return fmt.Errorf("创建本地索引目录失败: %w", err)
		}
		if err := os.WriteFile(rootRegistryPath, registryData, 0644); err != nil {
			return fmt.Errorf("保存本地索引失败: %w", err)
		}
	}

	return nil
}

func buildRegistryFromSkills(skills []*spec.Skill) spec.Registry {
	registry := spec.Registry{
		Version: "1.0.0",
		Skills:  make([]spec.SkillMetadata, 0, len(skills)),
	}

	for _, skill := range skills {
		metadata := spec.SkillMetadata{
			ID:               skill.ID,
			Name:             skill.Name,
			Version:          skill.Version,
			Author:           skill.Author,
			Description:      skill.Description,
			Tags:             skill.Tags,
			Repository:       skill.Repository,
			RepositoryPath:   skill.RepositoryPath,
			RepositoryCommit: skill.RepositoryCommit,
		}
		registry.Skills = append(registry.Skills, metadata)
	}

	return registry
}
