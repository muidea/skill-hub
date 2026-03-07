package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/muidea/skill-hub/pkg/errors"
)

// GitHubSearchResult 表示GitHub搜索结果的单个项目
type GitHubSearchResult struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	HTMLURL     string    `json:"html_url"`
	Stars       int       `json:"stargazers_count"`
	Forks       int       `json:"forks_count"`
	UpdatedAt   time.Time `json:"updated_at"`
	Language    string    `json:"language"`
	Topics      []string  `json:"topics"`
}

// GitHubSearchResponse 表示GitHub搜索API的响应
type GitHubSearchResponse struct {
	TotalCount int                  `json:"total_count"`
	Items      []GitHubSearchResult `json:"items"`
}

// searchGitHubRepositories 通过GitHub API搜索仓库
func searchGitHubRepositories(keyword string, limit int) ([]GitHubSearchResult, error) {
	// 构建搜索查询
	query := url.QueryEscape(keyword + " topic:agent-skills")
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=stars&order=desc&per_page=%d", query, limit)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "创建请求失败")
	}

	// 设置User-Agent（GitHub API要求）
	req.Header.Set("User-Agent", "skill-hub-cli")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "发送请求失败")
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.NewWithCodef("searchGitHubRepositories", errors.ErrAPIRequest, "GitHub API返回错误: %s - %s", resp.Status, string(body))
	}

	// 解析响应
	var searchResp GitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, errors.Wrap(err, "解析响应失败")
	}

	return searchResp.Items, nil
}

// filterByTarget 按目标环境过滤搜索结果
func filterByTarget(results []GitHubSearchResult, target string) []GitHubSearchResult {
	if target == "" {
		return results
	}

	var filtered []GitHubSearchResult
	targetLower := strings.ToLower(target)

	for _, result := range results {
		// 检查仓库描述、主题或README中是否包含目标关键词
		searchText := strings.ToLower(result.Description + " " + strings.Join(result.Topics, " ") + " " + result.FullName)

		// 根据目标环境匹配关键词
		isMatch := false
		switch targetLower {
		case "cursor":
			isMatch = strings.Contains(searchText, "cursor") ||
				strings.Contains(searchText, "cursorrules") ||
				strings.Contains(result.FullName, "cursor")
		case "claude", "claude_code":
			isMatch = strings.Contains(searchText, "claude") ||
				strings.Contains(searchText, "claude code") ||
				strings.Contains(result.FullName, "claude")
		case "open_code", "opencode":
			// open_code兼容性更广，很多技能可能没有明确标记
			// 我们放宽条件，只要不是明确标记为其他目标的都可以显示
			notCursor := !strings.Contains(searchText, "cursor") && !strings.Contains(result.FullName, "cursor")
			notClaude := !strings.Contains(searchText, "claude") && !strings.Contains(result.FullName, "claude")
			isMatch = notCursor && notClaude ||
				strings.Contains(searchText, "opencode") ||
				strings.Contains(searchText, "open code") ||
				strings.Contains(searchText, "skill-hub")
		}

		if isMatch {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// formatTimeAgo 格式化时间为相对时间
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

// displaySearchResults 显示搜索结果
func displaySearchResults(results []GitHubSearchResult, keyword, target string, limit int) {
	if len(results) == 0 {
		fmt.Println("\nℹ️  未找到相关技能")
		if target != "" {
			fmt.Printf("搜索关键词: %s (目标环境: %s)\n", keyword, target)
		} else {
			fmt.Printf("搜索关键词: %s\n", keyword)
		}
		return
	}

	// 按星标数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Stars > results[j].Stars
	})

	fmt.Println("\n搜索结果:")
	fmt.Println(strings.Repeat("=", 80))

	for i, result := range results {
		if i >= limit {
			break
		}

		fmt.Printf("%d. %s\n", i+1, result.FullName)
		if result.Description != "" {
			fmt.Printf("   描述: %s\n", result.Description)
		}
		fmt.Printf("   链接: %s\n", result.HTMLURL)
		fmt.Printf("   ⭐ 星标: %d | 🍴 Fork: %d | 📅 更新: %s\n",
			result.Stars, result.Forks, formatTimeAgo(result.UpdatedAt))

		if len(result.Topics) > 0 {
			fmt.Printf("   标签: %s\n", strings.Join(result.Topics, ", "))
		}

		if result.Language != "" {
			fmt.Printf("   语言: %s\n", result.Language)
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("找到 %d 个相关技能", len(results))
	if target != "" {
		fmt.Printf(" (已过滤目标环境: %s)", target)
	}
	fmt.Println()

	// 显示使用建议
	fmt.Println("\n使用建议:")
	fmt.Println("1. 查看技能详情: 访问上面的GitHub链接")
	fmt.Println("2. 添加技能到本地: 复制仓库URL，使用 'skill-hub init <git-url>'")
	fmt.Println("3. 创建自己的技能: 使用 'skill-hub create <skill-id>'")
}
