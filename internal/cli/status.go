package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "检查技能状态",
	Long: `对比项目本地工作区文件与技能仓库源文件的差异，显示技能状态：
- Synced: 本地与仓库一致
- Modified: 本地有未反馈的修改
- Outdated: 仓库版本领先于本地
- Missing: 技能已启用但本地文件缺失`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skillID := ""
		if len(args) > 0 {
			skillID = args[0]
		}
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runStatus(skillID, verbose)
	},
}

func init() {
	statusCmd.Flags().Bool("verbose", false, "显示详细差异信息")
}

func runStatus(skillID string, verbose bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		cwd, err := os.Getwd()
		if err != nil {
			return utils.GetCwdErr(err)
		}

		data, err := client.GetProjectStatus(context.Background(), cwd, skillID)
		if err == nil && data.Item != nil {
			renderProjectStatusSummary(data.Item)
			if verbose {
				fmt.Println("\n服务模式当前仅返回状态摘要，详细 diff 仍需本地模式执行。")
			} else if skillID == "" {
				fmt.Println("\n使用 'skill-hub status <id>' 检查特定技能状态")
				fmt.Println("使用 'skill-hub status --verbose' 显示详细差异")
			}
			return nil
		}
	}

	ctx, err := RequireInitAndWorkspace("", "")
	if err != nil {
		return err
	}

	summary, err := projectstatusservice.New().Inspect(ctx.Cwd, skillID)
	if err != nil {
		return err
	}

	renderProjectStatusSummary(summary)

	itemMap := make(map[string]projectstatusservice.SkillStatusItem, len(summary.Items))
	for _, item := range summary.Items {
		itemMap[item.SkillID] = item
	}

	if verbose {
		fmt.Println("\n=== 详细差异信息 ===")
		for _, item := range summary.Items {
			showSkillDiff(item)
		}
	}

	if skillID != "" {
		if item, ok := itemMap[skillID]; ok {
			showSkillDetails(item)
		}
	} else if !verbose && summary.SkillCount > 0 {
		fmt.Println("\n使用 'skill-hub status <id>' 检查特定技能状态")
		fmt.Println("使用 'skill-hub status --verbose' 显示详细差异")
	}

	return nil
}

func renderProjectStatusSummary(summary *projectstatusservice.ProjectStatusSummary) {
	fmt.Println("检查技能状态...")

	if summary.SkillCount == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		return
	}

	fmt.Printf("项目路径: %s\n", summary.ProjectPath)
	fmt.Printf("启用技能数: %d\n\n", summary.SkillCount)
	fmt.Println("=== 技能状态 ===")

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

	fmt.Printf("%-*s %-*s 状态\n", maxIDLength, "ID", maxVersionLength, "本地版本")
	fmt.Println(strings.Repeat("-", maxIDLength+1+maxVersionLength+1+2))

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
		}

		localVersion := item.LocalVersion
		if localVersion == "" {
			localVersion = "—"
		}

		fmt.Printf("%-*s %-*s %s %s\n", maxIDLength, item.SkillID, maxVersionLength, localVersion, statusSymbol, item.Status)
	}

	fmt.Println("\n说明:")
	fmt.Println("✅ Synced: 本地与仓库一致")
	fmt.Println("⚠️  Modified: 本地有未反馈的修改")
	fmt.Println("🔄 Outdated: 仓库版本领先于本地")
	fmt.Println("❌ Missing: 技能已启用但本地文件缺失")
}

func showSkillDetails(item projectstatusservice.SkillStatusItem) {
	fmt.Println("\n=== 技能详情 ===")

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

	fmt.Printf("ID:         %s\n", item.SkillID)
	fmt.Printf("状态:       %s %s\n", statusSymbol, item.Status)
	if item.SourceRepository != "" {
		fmt.Printf("来源仓库:   %s\n", item.SourceRepository)
	}

	localVersion := item.LocalVersion
	if localVersion == "" {
		localVersion = "N/A"
	}
	fmt.Printf("本地版本:   %s\n", localVersion)

	repoVersion := item.RepoVersion
	if repoVersion == "" {
		repoVersion = "N/A"
	}
	fmt.Printf("仓库版本:   %s\n", repoVersion)

	fmt.Printf("本地路径:   %s\n", item.LocalPath)
	fmt.Printf("仓库路径:   %s\n", item.RepoPath)

	localInfo, localErr := os.Stat(item.LocalPath)
	repoInfo, repoErr := os.Stat(item.RepoPath)

	if localErr == nil || repoErr == nil {
		fmt.Println("更新时间对比:")
		if localErr == nil {
			fmt.Printf("  本地文件: %s\n", localInfo.ModTime().Format(time.RFC3339))
		} else {
			fmt.Println("  本地文件: 无法获取")
		}
		if repoErr == nil {
			fmt.Printf("  仓库文件: %s\n", repoInfo.ModTime().Format(time.RFC3339))
		} else {
			fmt.Println("  仓库文件: 无法获取")
		}
		if localErr == nil && repoErr == nil {
			fmt.Println("  注: 上述时间仅反映文件系统修改时间，可能与语义版本不完全一致，不代表版本新旧。")
		}
	}

	if item.Status != spec.SkillStatusMissing && localVersion != "N/A" && repoVersion != "N/A" {
		if desc := describeChangeDirection(item.Status, localVersion, repoVersion); desc != "" {
			fmt.Println(desc)
		}
	}
}

func showSkillDiff(item projectstatusservice.SkillStatusItem) {
	localContent, localErr := os.ReadFile(item.LocalPath)
	repoContent, repoErr := os.ReadFile(item.RepoPath)

	fmt.Printf("\n--- %s ---\n", item.SkillID)

	if localErr != nil && repoErr != nil {
		fmt.Println("⚠️  无法读取本地和仓库文件")
		return
	}

	if localErr != nil {
		fmt.Println("⚠️  无法读取本地文件")
		fmt.Printf("仓库文件: %s\n", item.RepoPath)
		return
	}

	if repoErr != nil {
		fmt.Println("⚠️  无法读取仓库文件（技能可能不在仓库中）")
		fmt.Printf("本地文件: %s\n", item.LocalPath)
		return
	}

	localLines := strings.Split(string(localContent), "\n")
	repoLines := strings.Split(string(repoContent), "\n")

	if string(localContent) == string(repoContent) {
		fmt.Println("✅ 本地与仓库内容完全一致")
		return
	}

	localVersion, localHash, lvErr := getLocalSkillInfo(item.LocalPath)
	repoVersion, repoHash, rvErr := getLocalSkillInfo(item.RepoPath)
	if lvErr == nil && rvErr == nil {
		status := determineSkillStatus(localVersion, localHash, repoVersion, repoHash)
		fmt.Printf("状态判定: %s\n", status)
		if desc := describeChangeDirection(status, localVersion, repoVersion); desc != "" {
			fmt.Println("变更方向:", desc)
		}
	}

	fmt.Printf("差异统计: 本地 %d 行, 仓库 %d 行\n", len(localLines), len(repoLines))
	fmt.Println("\n差异预览 (最多显示20行):")
	fmt.Println("符号说明: '-' 表示仓库侧内容，'+' 表示项目本地工作区内容。")

	diffLines := computeSimpleDiff(localLines, repoLines)
	displayCount := 0
	for _, line := range diffLines {
		if displayCount >= 20 {
			fmt.Printf("... 还有 %d 行差异未显示\n", len(diffLines)-20)
			break
		}
		fmt.Println(line)
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

func updateSkillStatus(projectPath, skillID, status, version string) error {
	stateManager, err := newStateManager()
	if err != nil {
		return errors.WrapWithCode(err, "updateSkillStatus", errors.ErrSystem, "创建状态管理器失败")
	}

	projectState, err := stateManager.LoadProjectState(projectPath)
	if err != nil {
		return errors.Wrap(err, "加载项目状态失败")
	}

	if skillVars, exists := projectState.Skills[skillID]; exists {
		skillVars.Status = status
		skillVars.Version = version
		projectState.Skills[skillID] = skillVars
	} else {
		projectState.Skills[skillID] = spec.SkillVars{
			SkillID: skillID,
			Version: version,
			Status:  status,
			Variables: map[string]string{
				"target": "open_code",
			},
		}
	}

	if err := stateManager.SaveProjectState(projectState); err != nil {
		return errors.Wrap(err, "保存项目状态失败")
	}

	return nil
}
