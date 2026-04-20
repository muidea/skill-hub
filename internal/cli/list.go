package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/logging"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用技能",
	Long:  "显示本地技能仓库中的所有技能，支持按仓库过滤。--target 参数保留用于兼容旧脚本，不再限制技能列表。",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		verbose, _ := cmd.Flags().GetBool("verbose")
		repoFilters, _ := cmd.Flags().GetStringSlice("repo")
		return runList(target, verbose, repoFilters)
	},
}

func init() {
	listCmd.Flags().String("target", "", targetFilterFlagUsage)
	listCmd.Flags().Bool("verbose", false, "显示详细信息，包括技能描述、版本、适用说明等")
	listCmd.Flags().StringSlice("repo", []string{}, "按仓库名称过滤技能列表（可多次使用指定多个仓库）")
}

func runList(target string, verbose bool, repoFilters []string) error {
	_ = target

	repoFilters = normalizeRepoFilters(repoFilters)

	var skillsMetadata []spec.SkillMetadata
	if client, ok := hubClientIfAvailable(); ok {
		var err error
		skillsMetadata, err = client.ListSkills(context.Background(), repoFilters, "")
		if err != nil {
			return errors.Wrap(err, "通过服务获取技能列表失败")
		}
	} else {
		// 检查init依赖（规范4.3：该命令依赖init命令）
		if err := CheckInitDependency(); err != nil {
			return err
		}

		repoManager, err := newRepositoryManager()
		if err != nil {
			return errors.Wrap(err, "创建多仓库管理器失败")
		}

		if len(repoFilters) == 0 {
			skillsMetadata, err = repoManager.ListSkills("")
			if err != nil {
				return errors.Wrap(err, "获取技能列表失败")
			}
		} else {
			availableRepos, err := repoManager.ListRepositories()
			if err != nil {
				return errors.Wrap(err, "获取仓库列表失败")
			}

			validRepos := make(map[string]bool, len(availableRepos))
			for _, repo := range availableRepos {
				validRepos[repo.Name] = true
			}

			for _, repoFilter := range repoFilters {
				if !validRepos[repoFilter] {
					return errors.NewWithCodef("runList", errors.ErrConfigInvalid, "仓库 '%s' 不存在或已禁用", repoFilter)
				}
			}

			skillsMetadata, err = repoManager.ListSkillsInRepositories(repoFilters)
			if err != nil {
				return errors.Wrap(err, "获取技能列表失败")
			}
		}

	}

	renderSkillList(skillsMetadata, "", repoFilters, verbose)
	return nil
}

func renderSkillList(skillsMetadata []spec.SkillMetadata, target string, repoFilters []string, verbose bool) {
	if len(skillsMetadata) == 0 {
		fmt.Println("ℹ️  未找到任何技能")
		return
	}

	if verbose {
		// 详细模式显示
		fmt.Println("可用技能列表 (详细模式):")
		fmt.Println(strings.Repeat("=", 60))
		for i, skill := range skillsMetadata {
			fmt.Printf("%d. ID: %s\n", i+1, skill.ID)
			fmt.Printf("   名称: %s\n", skill.Name)
			fmt.Printf("   版本: %s\n", skill.Version)
			fmt.Printf("   仓库: %s\n", skill.Repository)
			if skill.Description != "" {
				fmt.Printf("   描述: %s\n", skill.Description)
			}
			if skill.Compatibility != "" {
				fmt.Printf("   适用说明: %s\n", skill.Compatibility)
			}
			if len(skill.Tags) > 0 {
				fmt.Printf("   标签: %s\n", strings.Join(skill.Tags, ", "))
			}
			if skill.Author != "" && skill.Author != "unknown" {
				fmt.Printf("   作者: %s\n", skill.Author)
			}
			fmt.Println()
		}
	} else {
		// 简要模式显示 - 使用动态列宽
		fmt.Println("可用技能列表:")

		// 计算动态列宽
		widths := calculateColumnWidths(skillsMetadata)

		idTitleSpaces := widths.idMin - displayWidth("ID")
		nameTitleSpaces := widths.nameMin - displayWidth("名称")
		versionTitleSpaces := widths.versionMin - displayWidth("版本")
		repoTitleSpaces := widths.repoMin - displayWidth("仓库")
		toolsTitleSpaces := widths.toolsMin - displayWidth(targetColumnTitle)

		if idTitleSpaces < 0 {
			idTitleSpaces = 0
		}
		if nameTitleSpaces < 0 {
			nameTitleSpaces = 0
		}
		if versionTitleSpaces < 0 {
			versionTitleSpaces = 0
		}
		if repoTitleSpaces < 0 {
			repoTitleSpaces = 0
		}
		if toolsTitleSpaces < 0 {
			toolsTitleSpaces = 0
		}

		fmt.Printf("%s%s %s%s %s%s %s%s %s%s\n",
			"ID", strings.Repeat(" ", idTitleSpaces),
			"名称", strings.Repeat(" ", nameTitleSpaces),
			"版本", strings.Repeat(" ", versionTitleSpaces),
			"仓库", strings.Repeat(" ", repoTitleSpaces),
			targetColumnTitle, strings.Repeat(" ", toolsTitleSpaces))

		totalWidth := widths.idMin + widths.nameMin + widths.versionMin + widths.repoMin + widths.toolsMin + 4
		separator := strings.Repeat("-", totalWidth)
		fmt.Println(separator)

		for _, skill := range skillsMetadata {
			toolsStr := getCompatibilitySummary(skill.Compatibility)
			repoName := formatRepoName(skill.Repository, widths.repoMin)
			displaySkillID := formatSkillID(skill.ID, widths.idMin)

			idSpaces := widths.idMin - displayWidth(displaySkillID)
			nameSpaces := widths.nameMin - displayWidth(skill.Name)
			versionSpaces := widths.versionMin - displayWidth(skill.Version)
			repoSpaces := widths.repoMin - displayWidth(repoName)
			toolsSpaces := widths.toolsMin - displayWidth(toolsStr)

			if idSpaces < 0 {
				idSpaces = 0
			}
			if nameSpaces < 0 {
				nameSpaces = 0
			}
			if versionSpaces < 0 {
				versionSpaces = 0
			}
			if repoSpaces < 0 {
				repoSpaces = 0
			}
			if toolsSpaces < 0 {
				toolsSpaces = 0
			}

			fmt.Printf("%s%s %s%s %s%s %s%s %s%s\n",
				displaySkillID, strings.Repeat(" ", idSpaces),
				skill.Name, strings.Repeat(" ", nameSpaces),
				skill.Version, strings.Repeat(" ", versionSpaces),
				repoName, strings.Repeat(" ", repoSpaces),
				toolsStr, strings.Repeat(" ", toolsSpaces))
		}
	}

	if len(repoFilters) > 0 {
		fmt.Printf("\n已过滤显示仓库: %s\n", strings.Join(repoFilters, ", "))
	}
	fmt.Println("\n使用 'skill-hub use <skill-id>' 在当前项目启用技能")
}

func normalizeRepoFilters(repoFilters []string) []string {
	if len(repoFilters) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(repoFilters))
	var normalized []string
	for _, repo := range repoFilters {
		if repo == "" {
			continue
		}
		if _, ok := seen[repo]; ok {
			continue
		}
		seen[repo] = struct{}{}
		normalized = append(normalized, repo)
	}

	return normalized
}

// refreshRegistry 刷新技能索引，确保registry.json与skills目录同步
func refreshRegistry() error {
	logger := logging.GetGlobalLogger().WithOperation("refreshRegistry")
	startTime := time.Now()

	defaultRepo, err := defaultRepository()
	if err != nil {
		return errors.Wrap(err, "refreshRegistry: 获取默认仓库失败")
	}

	if err := rebuildRepositoryIndex(defaultRepo.Name); err != nil {
		return errors.Wrap(err, "refreshRegistry: 重建仓库索引失败")
	}

	logger.Info("registry.json刷新成功",
		"repo_name", defaultRepo.Name,
		"duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

// columnWidths 定义列宽配置
type columnWidths struct {
	idMin      int
	idMax      int
	nameMin    int
	nameMax    int
	versionMin int
	versionMax int
	repoMin    int
	repoMax    int
	toolsMin   int
	toolsMax   int
}

// calculateColumnWidths 计算每列的最佳宽度
func calculateColumnWidths(skills []spec.SkillMetadata) columnWidths {
	widths := columnWidths{
		idMin:      2,  // "ID" 最小宽度
		idMax:      50, // ID最大宽度（增加以支持路径格式的技能ID）
		nameMin:    4,  // "名称" 最小宽度
		nameMax:    30, // 名称最大宽度
		versionMin: 4,  // "版本" 最小宽度
		versionMax: 10, // 版本最大宽度
		repoMin:    4,  // "仓库" 最小宽度
		repoMax:    20, // 仓库最大宽度
		toolsMin:   6,  // "适用范围" 最小宽度
		toolsMax:   30, // 工具最大宽度
	}

	// 计算每列的实际最大数据长度
	for _, skill := range skills {
		toolsStr := getCompatibilitySummary(skill.Compatibility)

		// 更新ID列宽度（使用显示宽度）
		updateWidth(&widths.idMin, displayWidth(skill.ID), widths.idMax)

		// 更新名称列宽度（使用显示宽度）
		updateWidth(&widths.nameMin, displayWidth(skill.Name), widths.nameMax)

		// 更新版本列宽度（使用显示宽度）
		updateWidth(&widths.versionMin, displayWidth(skill.Version), widths.versionMax)

		// 更新仓库列宽度（使用显示宽度）
		// 使用formatRepoName计算格式化后的仓库名称显示宽度
		repoName := formatRepoName(skill.Repository, widths.repoMax)
		updateWidth(&widths.repoMin, displayWidth(repoName), widths.repoMax)

		// 更新工具列宽度（使用显示宽度）
		updateWidth(&widths.toolsMin, displayWidth(toolsStr), widths.toolsMax)
	}

	// 确保每列至少有标题的字节长度，并为中文字符标题添加额外空间
	// 中文字符在fmt.Printf中需要更多空间来正确对齐
	if widths.idMin < len("ID") {
		widths.idMin = len("ID")
	}
	// 基于显示宽度设置列宽
	// 计算每列的最大显示宽度
	for _, skill := range skills {
		toolsStr := getCompatibilitySummary(skill.Compatibility)

		// 更新ID列显示宽度
		updateWidth(&widths.idMin, displayWidth(skill.ID), widths.idMax)

		// 更新名称列显示宽度
		updateWidth(&widths.nameMin, displayWidth(skill.Name), widths.nameMax)

		// 更新版本列显示宽度
		updateWidth(&widths.versionMin, displayWidth(skill.Version), widths.versionMax)

		// 更新仓库列显示宽度
		repoName := formatRepoName(skill.Repository, widths.repoMax)
		updateWidth(&widths.repoMin, displayWidth(repoName), widths.repoMax)

		// 更新工具列显示宽度
		updateWidth(&widths.toolsMin, displayWidth(toolsStr), widths.toolsMax)
	}

	// 确保每列至少有标题的显示宽度
	titleDisplays := map[string]int{
		"ID":              displayWidth("ID"),
		"名称":              displayWidth("名称"),
		"版本":              displayWidth("版本"),
		"仓库":              displayWidth("仓库"),
		targetColumnTitle: displayWidth(targetColumnTitle),
	}

	if widths.idMin < titleDisplays["ID"] {
		widths.idMin = titleDisplays["ID"]
	}
	if widths.nameMin < titleDisplays["名称"] {
		widths.nameMin = titleDisplays["名称"]
	}
	if widths.versionMin < titleDisplays["版本"] {
		widths.versionMin = titleDisplays["版本"]
	}
	if widths.repoMin < titleDisplays["仓库"] {
		widths.repoMin = titleDisplays["仓库"]
	}
	if widths.toolsMin < titleDisplays[targetColumnTitle] {
		widths.toolsMin = titleDisplays[targetColumnTitle]
	}

	// 为后三列添加额外显示宽度补偿
	// 经验值：每个中文字符需要额外1显示宽度补偿
	widths.versionMin += 2 // "版本"有2个中文字符
	widths.repoMin += 2    // "仓库"有2个中文字符
	widths.toolsMin += 4   // "适用范围"有4个中文字符

	// 确保不超过最大宽度限制
	if widths.idMin > widths.idMax {
		widths.idMin = widths.idMax
	}
	if widths.nameMin > widths.nameMax {
		widths.nameMin = widths.nameMax
	}
	if widths.versionMin > widths.versionMax {
		widths.versionMin = widths.versionMax
	}
	if widths.repoMin > widths.repoMax {
		widths.repoMin = widths.repoMax
	}
	if widths.toolsMin > widths.toolsMax {
		widths.toolsMin = widths.toolsMax
	}
	// 为中文标题添加额外空间补偿
	if widths.nameMin < len("名称")+4 { // "名称"需要额外空间
		widths.nameMin = len("名称") + 4
	}
	if widths.versionMin < len("版本")+4 { // "版本"需要额外空间
		widths.versionMin = len("版本") + 4
	}
	if widths.repoMin < len("仓库")+4 { // "仓库"需要额外空间
		widths.repoMin = len("仓库") + 4
	}
	if widths.toolsMin < len(targetColumnTitle)+8 { // "适用范围"需要更多额外空间
		widths.toolsMin = len(targetColumnTitle) + 8
	}

	return widths
}

func getCompatibilitySummary(compatibility string) string {
	if strings.TrimSpace(compatibility) == "" {
		return "通用"
	}
	return "已声明"
}

// formatSkillID 格式化技能ID显示，考虑显示宽度
func formatSkillID(skillID string, maxDisplayWidth int) string {
	if skillID == "" {
		return ""
	}

	// 如果技能ID的显示宽度不超过最大宽度，直接返回
	if displayWidth(skillID) <= maxDisplayWidth {
		return skillID
	}

	// 尝试截断路径部分，保留最后一部分
	parts := strings.Split(skillID, "/")
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		if displayWidth(lastPart) <= maxDisplayWidth {
			return lastPart
		}
	}

	// 如果还是太长，截断并添加省略号
	// 我们需要找到合适的截断点，使得显示宽度不超过maxDisplayWidth-3（为"..."留空间）
	if maxDisplayWidth > 3 {
		targetWidth := maxDisplayWidth - 3
		truncated := ""
		currentWidth := 0

		// 逐个字符添加，直到达到目标宽度
		for _, r := range skillID {
			charWidth := 1
			if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
				charWidth = 2
			} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
				charWidth = 2
			} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
				charWidth = 2
			}

			if currentWidth+charWidth > targetWidth {
				break
			}
			truncated += string(r)
			currentWidth += charWidth
		}

		return truncated + "..."
	}

	// 如果最大宽度很小，直接截断
	truncated := ""
	currentWidth := 0
	for _, r := range skillID {
		charWidth := 1
		if r >= 0x4E00 && r <= 0x9FFF {
			charWidth = 2
		} else if r >= 0x3400 && r <= 0x4DBF {
			charWidth = 2
		} else if r >= 0x20000 && r <= 0x2A6DF {
			charWidth = 2
		}

		if currentWidth+charWidth > maxDisplayWidth {
			break
		}
		truncated += string(r)
		currentWidth += charWidth
	}
	return truncated
}

// formatRepoName 格式化仓库名称显示，考虑显示宽度
func formatRepoName(repo string, maxDisplayWidth int) string {
	if repo == "" {
		return "local"
	}

	// 如果仓库名称的显示宽度不超过最大宽度，直接返回
	if displayWidth(repo) <= maxDisplayWidth {
		return repo
	}

	// 尝试截断路径部分，保留仓库名
	parts := strings.Split(repo, "/")
	if len(parts) > 1 {
		repoName := parts[len(parts)-1]
		if displayWidth(repoName) <= maxDisplayWidth {
			return repoName
		}
	}

	// 如果还是太长，截断并添加省略号
	// 我们需要找到合适的截断点，使得显示宽度不超过maxDisplayWidth-3（为"..."留空间）
	if maxDisplayWidth > 3 {
		targetWidth := maxDisplayWidth - 3
		truncated := ""
		currentWidth := 0

		// 逐个字符添加，直到达到目标宽度
		for _, r := range repo {
			charWidth := 1
			if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
				charWidth = 2
			} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
				charWidth = 2
			} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
				charWidth = 2
			}

			if currentWidth+charWidth > targetWidth {
				break
			}
			truncated += string(r)
			currentWidth += charWidth
		}

		return truncated + "..."
	}

	// 如果最大宽度很小，直接截断
	truncated := ""
	currentWidth := 0
	for _, r := range repo {
		charWidth := 1
		if r >= 0x4E00 && r <= 0x9FFF {
			charWidth = 2
		} else if r >= 0x3400 && r <= 0x4DBF {
			charWidth = 2
		} else if r >= 0x20000 && r <= 0x2A6DF {
			charWidth = 2
		}

		if currentWidth+charWidth > maxDisplayWidth {
			break
		}
		truncated += string(r)
		currentWidth += charWidth
	}
	return truncated
}

// updateWidth 更新列宽，确保在最小和最大范围内
func updateWidth(currentWidth *int, newLength int, maxWidth int) {
	if newLength > *currentWidth && newLength <= maxWidth {
		*currentWidth = newLength
	}
}

// displayWidth 计算字符串在终端中的显示宽度（中文字符算2个宽度）
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
			width += 2
		} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
			width += 2
		} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
			width += 2
		} else {
			width += 1
		}
	}
	return width
}
