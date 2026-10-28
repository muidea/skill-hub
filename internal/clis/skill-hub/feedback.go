package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/projectport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/utils"
)

var (
	feedbackDryRun bool
	feedbackForce  bool
	feedbackAll    bool
	feedbackJSON   bool
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback <id> | --all | --pattern <id-or-glob>...",
	Short: "将项目工作区技能修改内容更新至到本地仓库",
	Long: `将项目工作区本地的技能修改同步回本地技能仓库。

此命令会：
1. 提取项目工作区本地文件内容
2. 与本地仓库源文件对比，显示差异
3. 检查版本号，必要时自动升级 patch 版本
4. 经用户确认后更新本地仓库文件

使用 --dry-run 参数演习模式，仅显示将要同步的差异。
使用 --force 参数强制更新，即使有冲突也继续执行。

--pattern 语法（类 glob，匹配当前项目登记的技能 ID；* 可跨 /）：
  *        匹配零或多个任意字符
  ?        匹配恰好一个任意字符
  **       匹配全部

输入形式（必须二选一）：
  --all                 反馈当前项目状态中登记的全部技能
  --pattern '<glob>'    反馈匹配 pattern 的技能（可重复）

未命中任何技能时静默通过。`,
	Args:              validatePatternOrExactID,
	ValidArgsFunction: completeEnabledSkillIDsForCwd,
	RunE: func(cmd *cobra.Command, args []string) error {
		feedbackJSON = mustGetBoolFlag(cmd, "json")
		patterns, err := readPatternOrExactID(cmd, args)
		if err != nil {
			return err
		}
		if feedbackAll {
			if len(patterns) > 0 {
				return errors.NewWithCodef("feedback", errors.ErrInvalidInput, "--all and --pattern are mutually exclusive")
			}
			return runFeedbackAll()
		}
		if len(patterns) == 0 {
			return errors.NewWithCodef("feedback", errors.ErrInvalidInput, "specify --all or --pattern '<id-or-glob>'")
		}
		return runFeedbackByPatterns(patterns)
	},
}

func init() {
	feedbackCmd.Flags().BoolVar(&feedbackDryRun, "dry-run", false, "演习模式，仅显示将要同步的差异")
	feedbackCmd.Flags().BoolVar(&feedbackForce, "force", false, "强制更新，即使有冲突也继续执行")
	feedbackCmd.Flags().BoolVar(&feedbackAll, "all", false, "反馈当前项目状态中登记的全部技能")

	feedbackCmd.Flags().StringArray("pattern", nil, "技能 ID 通配符（可重复）。请引用通配符避免 shell 展开。")
}

func runFeedback(skillID string) error {
	if feedbackJSON {
		if !feedbackForce && !feedbackDryRun {
			return errors.NewWithCode("runFeedback", errors.ErrInvalidInput, "JSON反馈写入需要 --force，或使用 --dry-run 预览")
		}
		summary, err := runFeedbackStructured([]string{skillID}, false)
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
		if err != nil {
			return err
		}
		if summary.Failed > 0 {
			return errors.NewWithCodef("runFeedback", errors.ErrValidation, "%d 个技能反馈失败", summary.Failed)
		}
		return nil
	}

	if client, ok := hubClientIfAvailable(); ok {
		return runFeedbackViaService(client, skillID)
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	printfText("收集技能 '%s' 的反馈...\n", skillID)

	hasSkill, err := localRuntime.ProjectState().HasSkill(ctx.Cwd, skillID)
	if err != nil {
		return errors.Wrap(err, "检查项目技能状态失败")
	}

	if !hasSkill {
		return errors.NewWithCodef("runFeedback", errors.ErrSkillNotFound, "技能 '%s' 未在项目工作区中启用", skillID)
	}

	projectSkillDir := filepath.Join(ctx.Cwd, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return errors.NewWithCodef("runFeedback", errors.ErrFileNotFound, "项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return errors.WrapWithCode(err, "runFeedback", errors.ErrFileOperation, "读取项目工作区文件失败")
	}

	skillExists, err := localRuntime.Repository().CheckDefaultSkill(skillID)
	if err != nil {
		return errors.Wrap(err, "检查技能存在状态失败")
	}

	defaultRepo, err := defaultRepository()
	if err != nil {
		return errors.Wrap(err, "获取默认仓库失败")
	}

	repoDir, err := localRuntime.Repository().Path(defaultRepo.Name)
	if err != nil {
		return errors.Wrap(err, "获取仓库路径失败")
	}

	repoSkillDir := filepath.Join(repoDir, "skills", skillID)
	repoSkillPath := filepath.Join(repoSkillDir, "SKILL.md")

	var repoContent []byte
	if skillExists {
		repoContent, err = os.ReadFile(repoSkillPath)
		if err != nil {
			return errors.WrapWithCode(err, "runFeedback", errors.ErrFileOperation, "读取本地仓库文件失败")
		}
	} else {
		printfText("ℹ️  技能 '%s' 在本地仓库中不存在，将作为新技能创建\n", skillID)
		repoContent = []byte{}
	}

	projectStr := strings.TrimSpace(string(projectContent))
	repoStr := strings.TrimSpace(string(repoContent))

	changes, err := compareSkillDirectories(projectSkillDir, repoSkillDir, skillExists)
	if err != nil {
		return errors.Wrap(err, "比较技能目录失败")
	}

	hasContentChanges := projectStr != repoStr

	if !skillExists {
		printlnText("\n📝 新建技能内容:")
		printlnText("========================================")
		printfText("技能目录: %s\n", skillID)
		printfText("文件数量: %d\n", len(changes))
		for _, change := range changes {
			printfText("  - %s\n", change)
		}
		printlnText("========================================")
	} else if len(changes) == 0 && !hasContentChanges {
		printlnText("✅ 技能内容未修改")
		return nil
	} else {
		printlnText("\n🔍 检测到修改:")
		printlnText("========================================")
		printfText("技能目录: %s\n", skillID)
		printfText("修改文件数: %d\n", len(changes))

		if len(changes) > 0 {
			printlnText("\n修改的文件:")
			for _, change := range changes {
				printfText("  - %s\n", change)
			}
		}

		if hasContentChanges {
			printlnText("\nSKILL.md 内容差异:")
			printlnText("行号 | 修改前                      | 修改后")
			printlnText("-----|---------------------------|---------------------------")

			projectLines := strings.Split(projectStr, "\n")
			repoLines := strings.Split(repoStr, "\n")
			maxLines := len(projectLines)
			if len(repoLines) > maxLines {
				maxLines = len(repoLines)
			}

			for i := 0; i < maxLines; i++ {
				var projectLine, repoLine string
				if i < len(projectLines) {
					projectLine = projectLines[i]
				}
				if i < len(repoLines) {
					repoLine = repoLines[i]
				}

				if projectLine != repoLine {
					lineNum := i + 1
					printfText("%4d | %-25s | %-25s\n", lineNum, repoLine, projectLine)
				}
			}
		}
	}

	versionUpdated := false
	if hasContentChanges || len(changes) > 0 {
		projectVersion := normalizeVersionToXYZ(getSkillVersionFromContent(projectContent))
		repoVersion := "0.0.0"
		if skillExists && len(repoContent) > 0 {
			repoVersion = normalizeVersionToXYZ(getSkillVersionFromContent(repoContent))
		}
		if skillExists && compareVersions(projectVersion, repoVersion) < 0 {
			return errors.NewWithCodef(
				"runFeedback",
				errors.ErrValidation,
				"项目工作区技能版本 %s 低于默认仓库版本 %s，请先执行 skill-hub apply --pattern %s 后再合并反馈",
				projectVersion,
				repoVersion,
				skillID,
			)
		}

		if feedbackDryRun {
			printlnText("========================================")
			printlnText("\n✅ 演习模式完成，未进行实际修改")
			return nil
		}

		if compareVersions(projectVersion, repoVersion) <= 0 {
			newVersion := bumpPatchVersion(repoVersion)
			printfText("\n🔧 自动升级版本号: %s -> %s\n", projectVersion, newVersion)

			if err := updateSkillMdVersion(projectSkillPath, newVersion); err != nil {
				return errors.Wrap(err, "更新版本号失败")
			}
			printfText("✓ 已更新项目工作区 SKILL.md 版本号\n")

			_, err = os.ReadFile(projectSkillPath)
			if err != nil {
				return errors.Wrap(err, "重新读取项目文件失败")
			}
			versionUpdated = true
		} else {
			printfText("\n✓ 使用用户指定的版本号: %s\n", projectVersion)
		}
	}

	printlnText("========================================")

	if feedbackDryRun {
		printlnText("\n✅ 演习模式完成，未进行实际修改")
		return nil
	}

	if feedbackForce {
		printlnText("\n🔧 强制模式，直接更新本地仓库...")
	} else {
		printText("\n是否将这些修改更新到本地仓库？ [y/N]: ")
		var response string
		_ = scanInputToken(&response)
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			printlnText("❌ 取消反馈操作")
			return nil
		}
	}

	if err := archiveToDefaultRepository(skillID, projectSkillDir); err != nil {
		return errors.Wrap(err, "归档技能到默认仓库失败")
	}

	printlnText("✓ 更新本地仓库文件")
	printlnText("✓ 技能索引已刷新")

	printlnText("✓ 技能已归档到默认仓库")

	printlnText("\n✅ 反馈完成！")
	printfText("技能 '%s' 已保存到默认仓库: %s\n", skillID, defaultRepo.Name)
	if versionUpdated {
		printlnText("提示: 版本号已自动升级，可使用 'skill-hub status' 查看更新后的状态")
	}
	printlnText("使用 'skill-hub push' 同步到远程仓库")

	return nil
}

type feedbackSummary struct {
	ProjectPath string         `json:"project_path"`
	DryRun      bool           `json:"dry_run"`
	Force       bool           `json:"force"`
	Total       int            `json:"total"`
	Applied     int            `json:"applied"`
	Skipped     int            `json:"skipped"`
	Planned     int            `json:"planned"`
	Failed      int            `json:"failed"`
	Items       []feedbackItem `json:"items"`
}

type feedbackItem struct {
	SkillID string                     `json:"skill_id"`
	Status  string                     `json:"status"`
	Preview *projectport.PreviewResult `json:"preview,omitempty"`
	Result  *projectport.PreviewResult `json:"result,omitempty"`
	Error   string                     `json:"error,omitempty"`
}

func runFeedbackAll() error {
	if !feedbackForce && !feedbackDryRun {
		return errors.NewWithCode("runFeedbackAll", errors.ErrInvalidInput, "批量反馈需要 --force，或使用 --dry-run 预览")
	}
	summary, err := runFeedbackStructured(nil, true)
	if feedbackJSON {
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
	} else {
		renderFeedbackSummary(summary)
	}
	if err != nil {
		return err
	}
	if summary.Failed > 0 {
		return errors.NewWithCodef("runFeedbackAll", errors.ErrValidation, "%d 个技能反馈失败", summary.Failed)
	}
	return nil
}

// runFeedbackByPatterns resolves patterns against the current project's
// registered skills, then runs the structured feedback pipeline over the
// matched skill IDs. Literal IDs keep the direct single-ID path so newly
// registered local skills are not dependent on the repository index.
func runFeedbackByPatterns(patterns []string) error {
	matchers, err := compilePatterns(patterns)
	if err != nil {
		return err
	}
	// When every pattern is a literal ID (no wildcards / **), preserve
	// the v0.8.12 single-ID semantics: each ID is feedback'd directly
	// from the project workspace, even if the skill is not yet in any
	// repo. This is the only way the common "create + feedback"
	// workflow continues to work after the --pattern migration.
	allLiteral := len(matchers) > 0
	for _, m := range matchers {
		if !m.IsLiteral() {
			allLiteral = false
			break
		}
	}
	if allLiteral {
		ids := make([]string, 0, len(patterns))
		seen := make(map[string]struct{}, len(patterns))
		for _, p := range patterns {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			ids = append(ids, p)
		}
		if len(ids) == 1 && !feedbackForce && !feedbackDryRun {
			return runFeedback(ids[0])
		}
		if !feedbackForce && !feedbackDryRun {
			return errors.NewWithCode("runFeedbackByPatterns", errors.ErrInvalidInput, "批量反馈需要 --force，或使用 --dry-run 预览")
		}
		summary, err := runFeedbackStructured(ids, false)
		if feedbackJSON {
			if writeErr := writeJSON(summary); writeErr != nil {
				return writeErr
			}
		} else {
			renderFeedbackSummary(summary)
		}
		if err != nil {
			return err
		}
		if summary.Failed > 0 {
			return errors.NewWithCodef("runFeedbackByPatterns", errors.ErrValidation, "%d 个技能反馈失败", summary.Failed)
		}
		return nil
	}
	if !feedbackForce && !feedbackDryRun {
		return errors.NewWithCode("runFeedbackByPatterns", errors.ErrInvalidInput, "批量反馈需要 --force，或使用 --dry-run 预览")
	}
	ids, err := resolveProjectSkillIDsByPatterns(patterns)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		if feedbackJSON {
			return writeJSON(&feedbackSummary{DryRun: feedbackDryRun, Force: feedbackForce, Items: []feedbackItem{}})
		}
		return nil
	}

	summary, err := runFeedbackStructured(ids, false)
	if feedbackJSON {
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
	} else {
		renderFeedbackSummary(summary)
	}
	if err != nil {
		return err
	}
	if summary.Failed > 0 {
		return errors.NewWithCodef("runFeedbackByPatterns", errors.ErrValidation, "%d 个技能反馈失败", summary.Failed)
	}
	return nil
}

func runFeedbackStructured(skillIDs []string, all bool) (*feedbackSummary, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, utils.GetCwdErr(err)
	}
	summary := &feedbackSummary{
		ProjectPath: cwd,
		DryRun:      feedbackDryRun,
		Force:       feedbackForce,
	}

	if client, ok := hubClientIfAvailable(); ok {
		if all {
			statusData, statusErr := client.GetProjectStatus(context.Background(), cwd, "")
			if statusErr != nil {
				return summary, errors.Wrap(statusErr, "通过服务读取项目技能状态失败")
			}
			for _, item := range statusData.Item.Items {
				skillIDs = append(skillIDs, item.SkillID)
			}
		}
		runFeedbackItems(summary, skillIDs, func(skillID string) feedbackItem {
			return feedbackOneViaService(client, cwd, skillID, false)
		})
		return summary, nil
	}

	ctx, err := RequireInitAndWorkspace(cwd)
	if err != nil {
		return summary, err
	}
	summary.ProjectPath = ctx.Cwd
	if all {
		for skillID := range ctx.ProjectState.Skills {
			skillIDs = append(skillIDs, skillID)
		}
	}
	runFeedbackItems(summary, skillIDs, func(skillID string) feedbackItem {
		return feedbackOneLocal(ctx.Cwd, skillID, false)
	})
	return summary, nil
}

func runFeedbackItems(summary *feedbackSummary, skillIDs []string, fn func(string) feedbackItem) {
	sort.Strings(skillIDs)
	summary.Total = len(skillIDs)
	for _, skillID := range skillIDs {
		item := fn(skillID)
		summary.Items = append(summary.Items, item)
		switch item.Status {
		case "applied":
			summary.Applied++
		case "skipped":
			summary.Skipped++
		case "planned":
			summary.Planned++
		case "failed":
			summary.Failed++
		}
	}
}

type serviceFeedbackClient interface {
	PreviewFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
	ApplyFeedback(ctx context.Context, req httpapibiz.FeedbackRequest) (*httpapibiz.FeedbackPreviewData, error)
}

func runFeedbackViaService(client serviceFeedbackClient, skillID string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	item := feedbackOneViaService(client, cwd, skillID, true)
	if item.Error != "" {
		return errors.NewWithCode("runFeedbackViaService", errors.ErrValidation, item.Error)
	}
	return nil
}

func feedbackOneViaService(client serviceFeedbackClient, projectPath, skillID string, render bool) feedbackItem {
	req := httpapibiz.FeedbackRequest{
		ProjectPath: projectPath,
		SkillID:     skillID,
	}

	preview, err := client.PreviewFeedback(context.Background(), req)
	if err != nil {
		return feedbackItem{SkillID: skillID, Status: "failed", Error: errors.Wrap(err, "通过服务预览反馈失败").Error()}
	}
	if render {
		renderFeedbackPreview(preview.Item)
	}

	if preview.Item != nil && preview.Item.NoChanges {
		if render {
			printlnText("✅ 技能内容未修改")
		}
		return feedbackItem{SkillID: skillID, Status: "skipped", Preview: preview.Item}
	}
	if preview.Item != nil && preview.Item.Blocked {
		return feedbackItem{SkillID: skillID, Status: "failed", Preview: preview.Item, Error: preview.Item.BlockReason}
	}

	if feedbackDryRun {
		if render {
			printlnText("\n✅ 演习模式完成，未进行实际修改")
		}
		return feedbackItem{SkillID: skillID, Status: "planned", Preview: preview.Item}
	}

	if render {
		if feedbackForce {
			printlnText("\n🔧 强制模式，直接更新本地仓库...")
		} else if !confirmFeedback() {
			return feedbackItem{SkillID: skillID, Status: "skipped", Preview: preview.Item}
		}
	}

	result, err := client.ApplyFeedback(context.Background(), req)
	if err != nil {
		return feedbackItem{SkillID: skillID, Status: "failed", Preview: preview.Item, Error: errors.Wrap(err, "通过服务执行反馈失败").Error()}
	}
	if render {
		renderFeedbackCompletion(result.Item)
	}
	return feedbackItem{SkillID: skillID, Status: "applied", Preview: preview.Item, Result: result.Item}
}

func feedbackOneLocal(projectPath, skillID string, render bool) feedbackItem {
	preview, err := localRuntime.Project().PreviewFeedback(projectPath, skillID)
	if err != nil {
		return feedbackItem{SkillID: skillID, Status: "failed", Error: err.Error()}
	}
	if render {
		renderFeedbackPreview(preview)
	}
	if preview.NoChanges {
		if render {
			printlnText("✅ 技能内容未修改")
		}
		return feedbackItem{SkillID: skillID, Status: "skipped", Preview: preview}
	}
	if preview.Blocked {
		return feedbackItem{SkillID: skillID, Status: "failed", Preview: preview, Error: preview.BlockReason}
	}
	if feedbackDryRun {
		if render {
			printlnText("\n✅ 演习模式完成，未进行实际修改")
		}
		return feedbackItem{SkillID: skillID, Status: "planned", Preview: preview}
	}
	result, err := localRuntime.Project().ApplyFeedback(projectPath, skillID)
	if err != nil {
		return feedbackItem{SkillID: skillID, Status: "failed", Preview: preview, Error: err.Error()}
	}
	if render {
		renderFeedbackCompletion(result)
	}
	return feedbackItem{SkillID: skillID, Status: "applied", Preview: preview, Result: result}
}

func confirmFeedback() bool {
	printText("\n是否将这些修改更新到本地仓库？ [y/N]: ")
	var response string
	_ = scanInputToken(&response)
	response = strings.TrimSpace(response)
	if response != "y" && response != "Y" {
		printlnText("❌ 取消反馈操作")
		return false
	}
	return true
}

func renderFeedbackSummary(summary *feedbackSummary) {
	if summary == nil {
		printlnText("未返回反馈摘要")
		return
	}
	if summary.DryRun {
		printlnText("🔎 feedback 演习模式，未修改仓库")
	}
	printfText("项目路径: %s\n", summary.ProjectPath)
	printfText("total:   %d\n", summary.Total)
	printfText("applied: %d\n", summary.Applied)
	printfText("planned: %d\n", summary.Planned)
	printfText("skipped: %d\n", summary.Skipped)
	printfText("failed:  %d\n", summary.Failed)
	for _, item := range summary.Items {
		printfText("- [%s] %s", item.Status, item.SkillID)
		if item.Preview != nil && item.Preview.ResolvedVersion != "" && item.Preview.NeedsVersionBump {
			printfText(" version=%s", item.Preview.ResolvedVersion)
		}
		if item.Error != "" {
			printfText(" error=%s", item.Error)
		}
		printlnText()
	}
}

func renderFeedbackPreview(preview *projectport.PreviewResult) {
	if preview == nil {
		printlnText("ℹ️  未返回反馈预览")
		return
	}

	printfText("收集技能 '%s' 的反馈...\n", preview.SkillID)
	if !preview.SkillExists {
		printfText("ℹ️  技能 '%s' 在本地仓库中不存在，将作为新技能创建\n", preview.SkillID)
	}

	if len(preview.Changes) == 0 && !preview.HasContentChanges && preview.SkillExists {
		return
	}

	printlnText("\n🔍 检测到修改:")
	printlnText("========================================")
	printfText("技能目录: %s\n", preview.SkillID)
	printfText("修改文件数: %d\n", len(preview.Changes))
	if len(preview.Changes) > 0 {
		printlnText("\n修改的文件:")
		for _, change := range preview.Changes {
			printfText("  - %s\n", change)
		}
	}
	if preview.NeedsVersionBump {
		printfText("\n🔧 自动升级版本号: %s -> %s\n", preview.ProjectVersion, preview.ResolvedVersion)
	} else if preview.ProjectVersion != "" {
		printfText("\n✓ 使用用户指定的版本号: %s\n", preview.ProjectVersion)
	}
	if preview.Blocked {
		printfText("\n⛔ 已阻止反馈: %s\n", preview.BlockReason)
	}
	printlnText("========================================")
}

func renderFeedbackCompletion(result *projectport.PreviewResult) {
	if result == nil {
		printlnText("\n✅ 反馈完成！")
		return
	}

	printlnText("✓ 更新本地仓库文件")
	printlnText("✓ 技能索引已刷新")
	printlnText("✓ 技能已归档到默认仓库")
	printlnText("\n✅ 反馈完成！")
	printfText("技能 '%s' 已保存到默认仓库: %s\n", result.SkillID, result.DefaultRepo)
	if result.NeedsVersionBump {
		printlnText("提示: 版本号已自动升级，可使用 'skill-hub status' 查看更新后的状态")
	}
	printlnText("使用 'skill-hub push' 同步到远程仓库")
}

func getSkillVersionFromContent(content []byte) string {
	return skill.ExtractVersion(content)
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

func copySkillDirectory(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return errors.WrapWithCode(err, "copySkillDirectory", errors.ErrFileOperation, "创建目标目录失败")
	}

	srcFiles := make(map[string]bool)
	err := filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
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
	})
	if err != nil {
		return errors.Wrap(err, "遍历源目录失败")
	}

	dstFiles := make(map[string]bool)
	err = filepath.Walk(dstDir, func(dstPath string, info os.FileInfo, err error) error {
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
	})
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "遍历目标目录失败")
	}

	for relPath := range srcFiles {
		srcPath := filepath.Join(srcDir, relPath)
		dstPath := filepath.Join(dstDir, relPath)

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return errors.Wrapf(err, "创建目录失败 %s", filepath.Dir(dstPath))
		}

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return utils.ReadFileErr(err, srcPath)
		}

		info, err := os.Stat(srcPath)
		if err != nil {
			return errors.Wrapf(err, "获取文件权限失败 %s", srcPath)
		}

		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return utils.WriteFileErr(err, dstPath)
		}

		delete(dstFiles, relPath)
	}

	for relPath := range dstFiles {
		dstPath := filepath.Join(dstDir, relPath)
		if err := os.Remove(dstPath); err != nil {
			return utils.DeleteFileErr(err, dstPath)
		}
	}

	return nil
}
