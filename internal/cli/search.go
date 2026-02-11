package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "搜索远程技能",
	Long:  "通过GitHub API搜索带有 agent-skills 标签的远程技能仓库。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		limit, _ := cmd.Flags().GetInt("limit")
		return runSearch(args[0], target, limit)
	},
}

func init() {
	searchCmd.Flags().String("target", "", "按目标环境过滤搜索结果")
	searchCmd.Flags().Int("limit", 20, "限制返回结果数量，默认 20")
}

func runSearch(keyword, target string, limit int) error {
	// 检查init依赖（规范4.4：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Printf("搜索远程技能: %s\n", keyword)
	if target != "" {
		fmt.Printf("目标环境过滤: %s\n", target)
	}
	fmt.Printf("结果数量限制: %d\n", limit)

	// 搜索GitHub仓库
	fmt.Println("\n正在搜索GitHub...")
	results, err := searchGitHubRepositories(keyword, limit)
	if err != nil {
		// 如果GitHub API失败，显示备用信息
		fmt.Printf("⚠️  GitHub API搜索失败: %v\n", err)
		fmt.Println("\n备用搜索方法:")
		fmt.Println("1. 访问 https://github.com/topics/agent-skills")
		fmt.Println("2. 手动搜索相关技能仓库")
		fmt.Println("3. 使用 'skill-hub list' 查看本地已有技能")
		return nil
	}

	// 按目标环境过滤
	filteredResults := filterByTarget(results, target)

	// 显示结果
	displaySearchResults(filteredResults, keyword, target, limit)

	return nil
}
