package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "搜索远程技能",
	Long:  "通过 GitHub API 搜索带有 agent-skills 标签的远程技能仓库。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		return runSearch(args[0], limit)
	},
}

func init() {
	searchCmd.Flags().Int("limit", 20, "限制返回结果数量，默认 20")
}

func runSearch(keyword string, limit int) error {
	printfText("搜索远程技能: %s\n", keyword)
	printfText("结果数量限制: %d\n", limit)

	if client, ok := hubClientIfAvailable(); ok {
		printlnText("\n正在通过本地服务搜索远端技能...")
		results, err := client.SearchRemoteSkills(context.Background(), keyword, limit)
		if err != nil {
			if jsonMode() {
				return err
			}
			printfText("⚠️  本地服务搜索失败: %v\n", err)
			printlnText("\n备用搜索方法:")
			printlnText("1. 稍后重试，或检查本地服务网络访问能力")
			printlnText("2. 访问 https://github.com/topics/agent-skills")
			printlnText("3. 使用 'skill-hub list' 查看本地已有技能")
			return nil
		}
		if jsonMode() {
			return writeSearchJSON(results)
		}
		displaySearchResults(results, keyword, limit)
		return nil
	}

	// 兼容路径：服务不可用时退回本地实现
	if err := CheckInitDependency(); err != nil {
		return err
	}

	results, err := localRuntime.Skill().SearchRemote(keyword, limit)
	if err != nil {
		if jsonMode() {
			return err
		}
		printfText("⚠️  远端搜索失败: %v\n", err)
		printlnText("\n备用搜索方法:")
		printlnText("1. 启动 'skill-hubd' 后重试")
		printlnText("2. 访问 https://github.com/topics/agent-skills")
		printlnText("3. 使用 'skill-hub list' 查看本地已有技能")
		return nil
	}

	if jsonMode() {
		return writeSearchJSON(results)
	}
	displaySearchResults(results, keyword, limit)
	return nil
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "刚刚"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d分钟前", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%d小时前", hours)
	case duration < 30*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d天前", days)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / (24 * 30))
		return fmt.Sprintf("%d个月前", months)
	default:
		years := int(duration.Hours() / (24 * 365))
		return fmt.Sprintf("%d年前", years)
	}
}

func displaySearchResults(results []spec.RemoteSearchResult, keyword string, limit int) {
	if len(results) == 0 {
		printlnText("\nℹ️  未找到相关技能")
		printfText("搜索关键词: %s\n", keyword)
		return
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Stars > results[j].Stars
	})

	printlnText("\n搜索结果:")
	printlnText(strings.Repeat("=", 80))

	for i, result := range results {
		if i >= limit {
			break
		}

		printfText("%d. %s\n", i+1, result.FullName)
		if result.Description != "" {
			printfText("   描述: %s\n", result.Description)
		}
		printfText("   链接: %s\n", result.HTMLURL)
		printfText("   ⭐ 星标: %d | 🍴 Fork: %d | 📅 更新: %s\n",
			result.Stars, result.Forks, formatTimeAgo(result.UpdatedAt))

		if len(result.Topics) > 0 {
			printfText("   标签: %s\n", strings.Join(result.Topics, ", "))
		}

		if result.Language != "" {
			printfText("   语言: %s\n", result.Language)
		}

		printlnText()
	}

	printlnText(strings.Repeat("=", 80))
	printfText("找到 %d 个相关技能\n", len(results))

	printlnText("\n使用建议:")
	printlnText("1. 查看技能详情: 访问上面的GitHub链接")
	printlnText("2. 添加技能到本地: 复制仓库URL，使用 'skill-hub init <git-url>'")
	printlnText("3. 创建自己的技能: 使用 'skill-hub create <skill-id>'")
}

func writeSearchJSON(results []spec.RemoteSearchResult) error {
	if results == nil {
		results = []spec.RemoteSearchResult{}
	}
	return writeJSON(map[string]any{"items": results, "total": len(results)})
}
