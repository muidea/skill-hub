package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	globalservice "github.com/muidea/skill-hub/internal/pkg/globalport"
	projectstatusservice "github.com/muidea/skill-hub/internal/pkg/projectport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status [id] | --pattern <id-or-glob>...",
	Short: "检查技能状态",
	Long: `对比项目本地工作区文件与技能仓库源文件的差异，显示技能状态：
- synced: 目标副本与来源一致
- modified: 目标副本被本地修改
- outdated: 来源已更新，目标需要刷新
- missing: 已启用但目标副本不存在

不带 --pattern 时检查所有已启用技能；带 --pattern 时只显示 ID 匹配
pattern 的已启用技能。请引用带通配符的 pattern，避免 shell 在 skill-hub 启动前展开。`,
	Args: validatePatternOrExactID,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		global, _ := cmd.Flags().GetBool("global")
		patterns, err := readPatternOrExactID(cmd, args)
		if err != nil {
			return err
		}
		if global {
			return runStatusGlobalDispatch(patterns, nil, jsonOutput)
		}
		if len(patterns) == 0 {
			return runStatus("", verbose, jsonOutput)
		}
		return runStatusByPatterns(patterns, verbose, jsonOutput)
	},
}

func init() {
	statusCmd.Flags().Bool("verbose", false, "显示详细差异信息")

	statusCmd.Flags().Bool("global", false, "检查本机全局技能状态")
	statusCmd.Flags().StringArray("pattern", nil, "技能 ID 通配符（可重复）。请引用通配符避免 shell 展开。")
}

func runStatusGlobalDispatch(patterns []string, agents []string, jsonOutput bool) error {
	// No --pattern: existing behavior — show all globally-enabled skills.
	if len(patterns) == 0 {
		return runStatusGlobal("", agents, jsonOutput)
	}
	// With --pattern: resolve by ID, then filter the global Inspect summary.
	return runStatusGlobalByPatterns(patterns, agents, jsonOutput)
}

func runStatusGlobal(skillID string, agents []string, jsonOutput bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.GetGlobalStatus(context.Background(), skillID, agents)
		if err == nil && data.Item != nil {
			normalizeGlobalStatusSummary(data.Item)
			if jsonOutput {
				return writeJSON(data.Item)
			}
			renderGlobalStatusSummary(data.Item)
			return nil
		}
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}
	summary, err := localRuntime.Global().Inspect(skillID, agents)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(summary)
	}
	renderGlobalStatusSummary(summary)
	return nil
}

func runStatus(skillID string, verbose bool, jsonOutputOpt ...bool) error {
	jsonOutput := false
	if len(jsonOutputOpt) > 0 {
		jsonOutput = jsonOutputOpt[0]
	}

	if client, ok := hubClientIfAvailable(); ok {
		cwd, err := os.Getwd()
		if err != nil {
			return utils.GetCwdErr(err)
		}

		data, err := client.GetProjectStatus(context.Background(), cwd, skillID)
		if err == nil && data.Item != nil {
			normalizeProjectStatusSummary(data.Item)
			if jsonOutput {
				return writeJSON(data.Item)
			}
			renderProjectStatusSummary(data.Item)
			if verbose {
				printlnText("\n服务模式当前仅返回状态摘要，详细 diff 仍需本地模式执行。")
			} else if skillID == "" {
				printlnText("\n使用 'skill-hub status --pattern <id>' 检查特定技能状态")
				printlnText("使用 'skill-hub status --verbose' 显示详细差异")
			}
			return nil
		}
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	summary, err := localRuntime.Project().Inspect(ctx.Cwd, skillID)
	if err != nil {
		return err
	}

	if jsonOutput {
		return writeJSON(summary)
	}

	renderProjectStatusSummary(summary)

	itemMap := make(map[string]projectstatusservice.SkillStatusItem, len(summary.Items))
	for _, item := range summary.Items {
		itemMap[item.SkillID] = item
	}

	if verbose {
		printlnText("\n=== 详细差异信息 ===")
		for _, item := range summary.Items {
			showSkillDiff(item)
		}
	}

	if skillID != "" {
		if item, ok := itemMap[skillID]; ok {
			showSkillDetails(item)
		}
	} else if !verbose && summary.SkillCount > 0 {
		printlnText("\n使用 'skill-hub status --pattern <id>' 检查特定技能状态")
		printlnText("使用 'skill-hub status --verbose' 显示详细差异")
	}

	return nil
}

// runStatusByPatterns renders the status of the project's enabled skills
// whose IDs match the supplied patterns. An empty match set renders an empty
// summary silently, mirroring batch pattern semantics.
func runStatusByPatterns(patterns []string, verbose bool, jsonOutput bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		cwd, err := os.Getwd()
		if err != nil {
			return utils.GetCwdErr(err)
		}
		data, err := client.GetProjectStatus(context.Background(), cwd, "")
		if err != nil {
			return errors.Wrap(err, "通过服务获取项目状态失败")
		}
		if data.Item == nil {
			return nil
		}
		normalizeProjectStatusSummary(data.Item)
		filtered, err := filterStatusSummaryByPatterns(data.Item, patterns)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(filtered)
		}
		renderProjectStatusSummary(filtered)
		return nil
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}
	summary, err := localRuntime.Project().Inspect(ctx.Cwd, "")
	if err != nil {
		return err
	}
	filtered, err := filterStatusSummaryByPatterns(summary, patterns)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(filtered)
	}
	renderProjectStatusSummary(filtered)
	if verbose {
		printlnText("\n=== 详细差异信息 ===")
		for _, item := range filtered.Items {
			showSkillDiff(item)
		}
	}
	return nil
}

func filterStatusSummaryByPatterns(summary *projectstatusservice.ProjectStatusSummary, patterns []string) (*projectstatusservice.ProjectStatusSummary, error) {
	if summary == nil {
		return nil, nil
	}
	ids := make([]string, 0, len(summary.Items))
	for _, item := range summary.Items {
		ids = append(ids, item.SkillID)
	}
	matched, err := filterSkillIDsByPatterns(patterns, ids)
	if err != nil {
		return nil, err
	}
	return filterStatusSummaryByIDs(summary, stringSetFromSlice(matched)), nil
}

func normalizeProjectStatusSummary(summary *projectstatusservice.ProjectStatusSummary) {
	if summary == nil {
		return
	}
	if summary.Scope == "" {
		summary.Scope = "project"
	}
	for i := range summary.Items {
		item := &summary.Items[i]
		item.Status = spec.NormalizeSkillStatus(item.Status)
		if item.LegacyStatus == "" {
			item.LegacyStatus = spec.LegacyProjectSkillStatus(item.Status)
		}
	}
}

func filterStatusSummaryByIDs(summary *projectstatusservice.ProjectStatusSummary, ids map[string]struct{}) *projectstatusservice.ProjectStatusSummary {
	if summary == nil {
		return nil
	}
	out := &projectstatusservice.ProjectStatusSummary{
		ProjectPath: summary.ProjectPath,
		Scope:       summary.Scope,
		SkillCount:  0,
	}
	for _, item := range summary.Items {
		if _, ok := ids[item.SkillID]; !ok {
			continue
		}
		out.Items = append(out.Items, item)
	}
	out.SkillCount = len(out.Items)
	return out
}

func uniqueGlobalSkillIDs(items []globalservice.StatusItem) map[string]struct{} {
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		ids[item.SkillID] = struct{}{}
	}
	return ids
}

func normalizeGlobalStatusSummary(summary *globalservice.StatusSummary) {
	if summary == nil {
		return
	}
	if summary.Scope == "" {
		summary.Scope = "global"
	}
	for i := range summary.Items {
		normalizeGlobalStatusItem(&summary.Items[i])
	}
	for i := range summary.Orphaned {
		normalizeGlobalStatusItem(&summary.Orphaned[i])
	}
}

func normalizeGlobalStatusItem(item *globalservice.StatusItem) {
	rawStatus := item.Status
	item.Status = spec.NormalizeSkillStatus(item.Status)
	if item.LegacyStatus != "" {
		return
	}
	switch rawStatus {
	case "ok", "not_applied", "modified", "stale", "conflict", "orphaned", "missing_agent_dir", "error":
		item.LegacyStatus = rawStatus
		return
	}
	switch item.Status {
	case spec.StatusSynced:
		item.LegacyStatus = "ok"
	case spec.StatusMissing:
		item.LegacyStatus = "not_applied"
	case spec.StatusModified:
		item.LegacyStatus = "modified"
	case spec.StatusOutdated:
		item.LegacyStatus = "stale"
	case spec.StatusConflict:
		item.LegacyStatus = "conflict"
	case spec.StatusOrphaned:
		item.LegacyStatus = "orphaned"
	case spec.StatusUnavailable:
		item.LegacyStatus = "error"
	}
}

// runStatusGlobalByPatterns renders globally enabled skills whose IDs match
// the supplied patterns. An empty match set renders an empty summary silently.
func runStatusGlobalByPatterns(patterns []string, agents []string, jsonOutput bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.GetGlobalStatus(context.Background(), "", agents)
		if err != nil {
			return errors.Wrap(err, "通过服务获取全局状态失败")
		}
		if data.Item == nil {
			return nil
		}
		normalizeGlobalStatusSummary(data.Item)
		filtered, err := filterGlobalStatusSummaryByPatterns(data.Item, patterns)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSON(filtered)
		}
		renderGlobalStatusSummary(filtered)
		return nil
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}
	summary, err := localRuntime.Global().Inspect("", agents)
	if err != nil {
		return err
	}
	filtered, err := filterGlobalStatusSummaryByPatterns(summary, patterns)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(filtered)
	}
	renderGlobalStatusSummary(filtered)
	return nil
}

func filterGlobalStatusSummaryByPatterns(summary *globalservice.StatusSummary, patterns []string) (*globalservice.StatusSummary, error) {
	if summary == nil {
		return nil, nil
	}
	ids := make([]string, 0, len(summary.Items))
	for _, item := range summary.Items {
		ids = append(ids, item.SkillID)
	}
	matched, err := filterSkillIDsByPatterns(patterns, ids)
	if err != nil {
		return nil, err
	}
	return filterGlobalStatusSummaryByIDs(summary, stringSetFromSlice(matched)), nil
}

func filterGlobalStatusSummaryByIDs(summary *globalservice.StatusSummary, ids map[string]struct{}) *globalservice.StatusSummary {
	if summary == nil {
		return nil
	}
	out := &globalservice.StatusSummary{
		Scope:       summary.Scope,
		GlobalPath:  summary.GlobalPath,
		Agents:      summary.Agents,
		Items:       []globalservice.StatusItem{},
		Orphaned:    summary.Orphaned,
		GeneratedAt: summary.GeneratedAt,
	}
	for _, item := range summary.Items {
		if _, ok := ids[item.SkillID]; !ok {
			continue
		}
		out.Items = append(out.Items, item)
	}
	out.SkillCount = len(uniqueGlobalSkillIDs(out.Items))
	return out
}

func renderProjectStatusSummary(summary *projectstatusservice.ProjectStatusSummary) {
	printlnText("检查技能状态...")

	if summary.SkillCount == 0 {
		printlnText("ℹ️  当前项目未启用任何技能")
		return
	}

	printfText("项目路径: %s\n", summary.ProjectPath)
	printfText("启用技能数: %d\n\n", summary.SkillCount)
	printlnText("=== 技能状态 ===")

	maxIDLength := 2
	maxVersionLength := 7
	for _, item := range summary.Items {
		if len(item.SkillID) > maxIDLength {
			maxIDLength = len(item.SkillID)
		}
		if len(item.LocalVersion) > maxVersionLength {
			maxVersionLength = len(item.LocalVersion)
		}
	}

	printfText("%-*s %-*s 状态\n", maxIDLength, "ID", maxVersionLength, "本地版本")
	printlnText(strings.Repeat("-", maxIDLength+1+maxVersionLength+1+2))

	for _, item := range summary.Items {
		statusSymbol := "❓"
		switch item.Status {
		case spec.SkillStatusSynced:
			statusSymbol = "✅"
		case spec.SkillStatusModified:
			statusSymbol = "⚠️"
		case spec.SkillStatusOutdated:
			statusSymbol = "🔄"
		case spec.SkillStatusMissing:
			statusSymbol = "❌"
		case spec.SkillStatusModifiedAgainstOutdatedRepo:
			statusSymbol = "⛔"
		}

		localVersion := item.LocalVersion
		if localVersion == "" {
			localVersion = "—"
		}

		printfText("%-*s %-*s %s %s\n", maxIDLength, item.SkillID, maxVersionLength, localVersion, statusSymbol, item.Status)
	}

	printlnText("\n说明:")
	printlnText("✅ synced: 目标副本与来源一致")
	printlnText("⚠️  modified: 目标副本有本地修改")
	printlnText("🔄 outdated: 来源已更新，目标需要刷新")
	printlnText("⛔ diverged: 来源已更新且目标也有修改，需先手工合并")
	printlnText("❌ missing: 已启用但目标副本不存在")
}

func renderGlobalStatusSummary(summary *globalservice.StatusSummary) {
	printlnText("检查本机全局技能状态...")
	if summary == nil {
		printlnText("ℹ️  未返回状态")
		return
	}
	printfText("全局镜像目录: %s\n", summary.GlobalPath)
	if len(summary.Agents) == 0 {
		printlnText("ℹ️  未检测到可用 agent")
	} else {
		printlnText("\n=== Agent 检测 ===")
		for _, agent := range summary.Agents {
			state := "detected"
			if agent.Configured {
				state = "configured"
			} else if !agent.Detected {
				state = "not_detected"
			}
			printfText("%-8s %-14s %s\n", agent.Name, state, agent.SkillsDir)
		}
	}

	if len(summary.Items) == 0 {
		printlnText("\nℹ️  当前未启用任何全局技能")
		printlnText("使用 'skill-hub use <skill-id> --global' 启用全局技能")
		return
	}

	rows := summarizeGlobalStatusItems(summary.Items)
	printfText("\n全局启用技能数: %d\n", len(rows))
	printlnText("\n=== 全局技能状态 ===")
	maxIDLength := 2
	maxVersionLength := len("版本")
	maxAgentLength := len("Agent")
	for _, row := range rows {
		if len(row.SkillID) > maxIDLength {
			maxIDLength = len(row.SkillID)
		}
		if len(row.Version) > maxVersionLength {
			maxVersionLength = len(row.Version)
		}
		if len(row.Agents) > maxAgentLength {
			maxAgentLength = len(row.Agents)
		}
	}
	printfText("%-*s %-*s %-*s 状态\n", maxIDLength, "ID", maxVersionLength, "版本", maxAgentLength, "Agent")
	printlnText(strings.Repeat("-", maxIDLength+1+maxVersionLength+1+maxAgentLength+1+12))
	for _, row := range rows {
		printfText("%-*s %-*s %-*s %s %s\n", maxIDLength, row.SkillID, maxVersionLength, row.Version, maxAgentLength, row.Agents, globalStatusSymbol(row.Status), row.Status)
		if row.ShowAgentDetails {
			for _, item := range row.Items {
				detail := fmt.Sprintf("%s: %s %s", item.Agent, globalStatusSymbol(item.Status), item.Status)
				if item.Message != "" {
					detail += ": " + item.Message
				}
				printfText("%-*s %-*s %-*s   %s\n", maxIDLength, "", maxVersionLength, "", maxAgentLength, "", detail)
			}
		}
	}
	if len(summary.Orphaned) > 0 {
		printlnText("\n=== 孤儿托管目录 ===")
		for _, item := range summary.Orphaned {
			printfText("⚠️  %s -> %s: %s\n", item.SkillID, item.Agent, item.TargetPath)
		}
	}

	printlnText("\n说明:")
	printlnText("✅ synced: 目标副本与来源一致")
	printlnText("➕ missing: 已启用但目标副本不存在")
	printlnText("⚠️  modified/conflict/orphaned: 需要人工确认后处理")
	printlnText("🔄 outdated: 来源已变化，可运行 'skill-hub apply --global' 刷新")
	printlnText("❌ unavailable: agent skills 目录或来源不可用")
	printlnText("⚠️  mixed: 不同 agent 的状态不一致，详见该技能下方的 agent 明细")
}

type globalStatusRow struct {
	SkillID          string
	Version          string
	Agents           string
	Status           string
	Items            []globalservice.StatusItem
	ShowAgentDetails bool
}

func summarizeGlobalStatusItems(items []globalservice.StatusItem) []globalStatusRow {
	groups := make(map[string][]globalservice.StatusItem)
	ids := make([]string, 0)
	for _, item := range items {
		if _, ok := groups[item.SkillID]; !ok {
			ids = append(ids, item.SkillID)
		}
		groups[item.SkillID] = append(groups[item.SkillID], item)
	}
	sort.Strings(ids)

	rows := make([]globalStatusRow, 0, len(ids))
	for _, id := range ids {
		group := groups[id]
		sort.SliceStable(group, func(i, j int) bool {
			return group[i].Agent < group[j].Agent
		})

		agents := make([]string, 0, len(group))
		versions := make(map[string]struct{})
		statuses := make(map[string]struct{})
		showDetails := false
		for _, item := range group {
			agents = append(agents, item.Agent)
			versions[item.Version] = struct{}{}
			statuses[item.Status] = struct{}{}
			showDetails = showDetails || item.Message != ""
		}

		version := group[0].Version
		if version == "" {
			version = "—"
		}
		status := group[0].Status
		if len(versions) > 1 {
			version = spec.StatusMixed
			showDetails = true
		}
		if len(statuses) > 1 {
			status = spec.StatusMixed
			showDetails = true
		}

		rows = append(rows, globalStatusRow{
			SkillID:          id,
			Version:          version,
			Agents:           strings.Join(agents, ","),
			Status:           status,
			Items:            group,
			ShowAgentDetails: showDetails,
		})
	}
	return rows
}

func globalStatusSymbol(status string) string {
	switch status {
	case globalservice.StatusOK:
		return "✅"
	case globalservice.StatusNotApplied:
		return "➕"
	case globalservice.StatusStale:
		return "🔄"
	case globalservice.StatusMissingAgentDir:
		return "❌"
	case globalservice.StatusModified, globalservice.StatusConflict, globalservice.StatusOrphaned:
		return "⚠️"
	case spec.StatusMixed:
		return "⚠️"
	default:
		return "❓"
	}
}

func showSkillDetails(item projectstatusservice.SkillStatusItem) {
	printlnText("\n=== 技能详情 ===")

	statusSymbol := "❓"
	switch item.Status {
	case spec.SkillStatusSynced:
		statusSymbol = "✅"
	case spec.SkillStatusModified:
		statusSymbol = "⚠️"
	case spec.SkillStatusOutdated:
		statusSymbol = "🔄"
	case spec.SkillStatusMissing:
		statusSymbol = "❌"
	}

	printfText("ID:         %s\n", item.SkillID)
	printfText("状态:       %s %s\n", statusSymbol, item.Status)
	if item.SourceRepository != "" {
		printfText("来源仓库:   %s\n", item.SourceRepository)
	}

	localVersion := item.LocalVersion
	if localVersion == "" {
		localVersion = "N/A"
	}
	printfText("本地版本:   %s\n", localVersion)

	repoVersion := item.RepoVersion
	if repoVersion == "" {
		repoVersion = "N/A"
	}
	printfText("仓库版本:   %s\n", repoVersion)

	printfText("本地路径:   %s\n", item.LocalPath)
	printfText("仓库路径:   %s\n", item.RepoPath)

	localInfo, localErr := os.Stat(item.LocalPath)
	repoInfo, repoErr := os.Stat(item.RepoPath)

	if localErr == nil || repoErr == nil {
		printlnText("更新时间对比:")
		if localErr == nil {
			printfText("  本地文件: %s\n", localInfo.ModTime().Format(time.RFC3339))
		} else {
			printlnText("  本地文件: 无法获取")
		}
		if repoErr == nil {
			printfText("  仓库文件: %s\n", repoInfo.ModTime().Format(time.RFC3339))
		} else {
			printlnText("  仓库文件: 无法获取")
		}
		if localErr == nil && repoErr == nil {
			printlnText("  注: 上述时间仅反映文件系统修改时间，可能与语义版本不完全一致，不代表版本新旧。")
		}
	}

	if item.Status != spec.SkillStatusMissing && localVersion != "N/A" && repoVersion != "N/A" {
		if desc := describeChangeDirection(item.Status, localVersion, repoVersion); desc != "" {
			printlnText(desc)
		}
	}
}

func showSkillDiff(item projectstatusservice.SkillStatusItem) {
	localContent, localErr := os.ReadFile(item.LocalPath)
	repoContent, repoErr := os.ReadFile(item.RepoPath)

	printfText("\n--- %s ---\n", item.SkillID)

	if localErr != nil && repoErr != nil {
		printlnText("⚠️  无法读取本地和仓库文件")
		return
	}

	if localErr != nil {
		printlnText("⚠️  无法读取本地文件")
		printfText("仓库文件: %s\n", item.RepoPath)
		return
	}

	if repoErr != nil {
		printlnText("⚠️  无法读取仓库文件（技能可能不在仓库中）")
		printfText("本地文件: %s\n", item.LocalPath)
		return
	}

	localLines := strings.Split(string(localContent), "\n")
	repoLines := strings.Split(string(repoContent), "\n")

	if string(localContent) == string(repoContent) {
		printlnText("✅ 本地与仓库内容完全一致")
		return
	}

	localVersion, localHash, lvErr := getLocalSkillInfo(item.LocalPath)
	repoVersion, repoHash, rvErr := getLocalSkillInfo(item.RepoPath)
	if lvErr == nil && rvErr == nil {
		status := determineSkillStatus(localVersion, localHash, repoVersion, repoHash)
		printfText("状态判定: %s\n", status)
		if desc := describeChangeDirection(status, localVersion, repoVersion); desc != "" {
			printlnText("变更方向:", desc)
		}
	}

	printfText("差异统计: 本地 %d 行, 仓库 %d 行\n", len(localLines), len(repoLines))
	printlnText("\n差异预览 (最多显示20行):")
	printlnText("符号说明: '-' 表示仓库侧内容，'+' 表示项目本地工作区内容。")

	diffLines := computeSimpleDiff(localLines, repoLines)
	displayCount := 0
	for _, line := range diffLines {
		if displayCount >= 20 {
			printfText("... 还有 %d 行差异未显示\n", len(diffLines)-20)
			break
		}
		printlnText(line)
		displayCount++
	}
}

func computeSimpleDiff(local, repo []string) []string {
	var result []string

	localSet := make(map[string]bool)
	for _, line := range local {
		localSet[line] = true
	}

	repoSet := make(map[string]bool)
	for _, line := range repo {
		repoSet[line] = true
	}

	for _, line := range repo {
		if !localSet[line] && strings.TrimSpace(line) != "" {
			result = append(result, fmt.Sprintf("-%s", line))
		}
	}

	for _, line := range local {
		if !repoSet[line] && strings.TrimSpace(line) != "" {
			result = append(result, fmt.Sprintf("+%s", line))
		}
	}

	if len(result) == 0 {
		for i := 0; i < len(local) && i < len(repo); i++ {
			if local[i] != repo[i] {
				result = append(result, fmt.Sprintf("-%s", repo[i]))
				result = append(result, fmt.Sprintf("+%s", local[i]))
			}
		}
	}

	return result
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

func describeChangeDirection(status, localVersion, repoVersion string) string {
	switch status {
	case spec.SkillStatusModified:
		return "当前以仓库为基线，本地在其基础上发生了修改（本地内容偏离仓库，需评估是否反馈）。"
	case spec.SkillStatusOutdated:
		return "当前以本地为基线，仓库中的技能内容比本地版本更新（仓库较新，建议同步更新）。"
	case spec.SkillStatusSynced:
		return "本地与仓库版本一致，若有差异仅为格式或元数据层面的轻微变动。"
	default:
		return ""
	}
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
		} else {
			return spec.SkillStatusModified
		}
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
		} else if num1 < num2 {
			return -1
		}
	}

	if len(v1Parts) > len(v2Parts) {
		return 1
	} else if len(v1Parts) < len(v2Parts) {
		return -1
	}

	if v1 > v2 {
		return 1
	}
	return -1
}
