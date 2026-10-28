package cli

import (
	"context"
	"strings"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/configport"
	"github.com/muidea/skill-hub/pkg/errors"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "管理多Git仓库",
	Long: `管理多个Git仓库，支持添加、删除、启用、禁用仓库等操作。

多仓库功能允许您从多个来源获取技能，如：
- 个人仓库：您自己的技能集合
- 社区仓库：开源技能集合
- 官方仓库：skill-hub官方维护的技能

默认仓库即为归档仓库，所有修改都会归档到默认仓库。`,
}

var repoAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "添加新仓库",
	Long: `添加新的Git仓库到技能库。

示例:
  skill-hub repo add community https://github.com/skill-hub-community/awesome-skills.git
  skill-hub repo add team git@github.com:company/skills.git --branch develop
  skill-hub repo add local --type user  # 创建本地空仓库`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoAdd(cmd, args)
	},
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有仓库",
	Long:  `列出所有已配置的Git仓库，显示仓库状态和基本信息。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runRepoList(jsonOutput)
	},
}

var repoRemoveCmd = &cobra.Command{
	Use:               "remove <name>",
	Aliases:           []string{"rm"},
	Short:             "移除仓库",
	Long:              `从配置中移除指定的Git仓库。注意：这不会删除本地仓库文件。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoRemove(args[0])
	},
}

var repoSyncCmd = &cobra.Command{
	Use:   "sync [name]",
	Short: "同步仓库",
	Long: `同步指定仓库或所有仓库。

如果没有指定仓库名称，则同步所有启用的仓库。
使用 --all 参数强制同步所有仓库（包括禁用的）。`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		syncAll, _ := cmd.Flags().GetBool("all")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runRepoSync(args, syncAll, jsonOutput)
	},
}

var repoRebuildIndexCmd = &cobra.Command{
	Use:               "rebuild-index [name]",
	Short:             "重建仓库索引",
	Long:              `扫描指定仓库的 skills 目录并重建 registry.json。未指定仓库时重建默认仓库索引。`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runRepoRebuildIndex(args, jsonOutput)
	},
}

var repoEnableCmd = &cobra.Command{
	Use:               "enable <name>",
	Short:             "启用仓库",
	Long:              `启用之前被禁用的仓库。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoEnable(args[0])
	},
}

var repoDisableCmd = &cobra.Command{
	Use:               "disable <name>",
	Short:             "禁用仓库",
	Long:              `禁用指定仓库，禁用后该仓库的技能将不可用。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoDisable(args[0])
	},
}

var repoDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "设置默认仓库",
	Long: `设置默认仓库（归档仓库）。

所有通过 feedback 命令修改的技能都会归档到默认仓库。
如果技能在默认仓库中不存在则新增，存在则覆盖更新。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeRepoNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoDefault(args[0])
	},
}

func init() {
	// 添加命令标志
	repoRemoveCmd.Flags().Bool("yes", false, "确认移除仓库配置")
	repoDisableCmd.Flags().Bool("yes", false, "确认禁用仓库")
	repoAddCmd.Flags().String("branch", "main", "Git分支")
	repoAddCmd.Flags().String("type", "community", "仓库类型 (user/community/official)")
	repoAddCmd.Flags().String("description", "", "仓库描述")

	repoSyncCmd.Flags().Bool("all", false, "同步所有仓库（包括禁用的）")

	// 添加子命令
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	repoCmd.AddCommand(repoSyncCmd)
	repoCmd.AddCommand(repoRebuildIndexCmd)
	repoCmd.AddCommand(repoEnableCmd)
	repoCmd.AddCommand(repoDisableCmd)
	repoCmd.AddCommand(repoDefaultCmd)
}

// runRepoAdd 执行添加仓库操作
func runRepoAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	var url string
	if len(args) > 1 {
		url = args[1]
	}

	// 获取参数
	branch, _ := cmd.Flags().GetString("branch")
	repoType, _ := cmd.Flags().GetString("type")
	description, _ := cmd.Flags().GetString("description")

	// 验证名称
	if !isValidRepoName(name) {
		return errors.NewWithCode("runRepoAdd", errors.ErrInvalidInput, "仓库名称只能包含字母、数字、下划线和连字符")
	}

	// 创建仓库配置
	repoConfig := configport.RepositoryConfig{
		Name:        name,
		URL:         url,
		Branch:      branch,
		Type:        repoType,
		Description: description,
		Enabled:     true,
		IsArchive:   false, // 只有默认仓库才是归档仓库
	}

	if client, ok := hubClientIfAvailable(); ok {
		if err := client.AddRepo(context.Background(), repoConfig); err != nil {
			return errors.Wrap(err, "通过服务添加仓库失败")
		}
	} else if err := addRepository(repoConfig); err != nil {
		return errors.Wrap(err, "添加仓库失败")
	}

	if jsonMode() {
		return writeJSON(repoConfig)
	}
	printfText("✅ 仓库 '%s' 添加成功\n", name)
	if url != "" {
		printfText("   远程URL: %s\n", url)
		printfText("   分支: %s\n", branch)
	} else {
		printlnText("   类型: 本地空仓库")
	}
	printfText("   类型: %s\n", repoType)
	if description != "" {
		printfText("   描述: %s\n", description)
	}

	return nil
}

// runRepoList 执行列出仓库操作
func runRepoList(jsonOutput bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.ListRepos(context.Background())
		if err != nil {
			return errors.Wrap(err, "通过服务获取仓库列表失败")
		}
		if jsonOutput {
			return writeJSON(data)
		}
		renderRepositoryList(data.Items, data.DefaultRepo)
		return nil
	}

	repos, err := listRepositories(true)
	if err != nil {
		return errors.Wrap(err, "获取仓库列表失败")
	}

	defaultRepo := "main"
	if repo, err := defaultRepository(); err == nil && repo != nil {
		defaultRepo = repo.Name
	}

	if jsonOutput {
		return writeJSON(httpapibiz.RepoListData{
			DefaultRepo: defaultRepo,
			Items:       repos,
		})
	}
	renderRepositoryList(repos, defaultRepo)
	return nil
}

// runRepoRemove 执行移除仓库操作
func runRepoRemove(name string) error {
	if jsonMode() {
		if !mustGetBoolFlag(activeOutput.command, "yes") {
			return inputRequired("JSON 移除仓库配置需要 --yes")
		}
	} else {
		printfText("确定要移除仓库 '%s' 吗？(y/N): ", name)
		var confirm string
		_ = scanInputToken(&confirm)
		if strings.ToLower(confirm) != "y" {
			printlnText("操作已取消")
			return nil
		}
	}

	if client, ok := hubClientIfAvailable(); ok {
		if err := client.RemoveRepo(context.Background(), name); err != nil {
			return errors.Wrap(err, "通过服务移除仓库失败")
		}
	} else if err := removeRepository(name); err != nil {
		return errors.Wrap(err, "移除仓库失败")
	}

	printfText("✅ 仓库 '%s' 已从配置中移除\n", name)
	printlnText("注意：本地仓库文件仍然保留，如需完全删除请手动操作")
	if jsonMode() {
		return writeJSON(map[string]any{"name": name, "status": "removed"})
	}
	return nil
}

type repoSyncSummary struct {
	All     bool           `json:"all"`
	Total   int            `json:"total"`
	Synced  int            `json:"synced"`
	Skipped int            `json:"skipped"`
	Failed  int            `json:"failed"`
	Items   []repoSyncItem `json:"items"`
}

type repoSyncItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
	Message string `json:"message,omitempty"`
	Ahead   int    `json:"ahead"`
	Behind  int    `json:"behind"`
	Error   string `json:"error,omitempty"`
}

type repoRebuildIndexResult struct {
	RepoName string `json:"repo_name"`
	Status   string `json:"status"`
}

// runRepoSync 执行同步仓库操作
func runRepoSync(args []string, syncAll bool, jsonOutput bool) error {
	if jsonOutput {
		summary, err := runRepoSyncStructured(args, syncAll)
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
		if err != nil {
			return err
		}
		if summary.Failed > 0 {
			return errors.NewWithCodef("runRepoSync", errors.ErrGitOperation, "%d 个仓库同步失败", summary.Failed)
		}
		return nil
	}

	if len(args) > 0 {
		// 同步指定仓库
		name := args[0]
		printfText("正在同步仓库 '%s'...\n", name)

		result, err := syncConfiguredRepository(name)
		if err != nil {
			return errors.Wrapf(err, "同步仓库 '%s' 失败", name)
		}
		if isBlockedRepoSync(result) {
			return errors.NewWithCodef("runRepoSync", errors.ErrGitOperation, "仓库 '%s' 同步未执行: %s", name, result.Message)
		}

		printfText("✅ 仓库 '%s' 同步完成 (%s)\n", name, result.Status)
		if result.Message != "" {
			printfText("   %s\n", result.Message)
		}
	} else {
		// 同步所有仓库
		var repos []configport.RepositoryConfig
		if client, ok := hubClientIfAvailable(); ok {
			data, err := client.ListRepos(context.Background())
			if err != nil {
				return errors.Wrap(err, "通过服务获取仓库列表失败")
			}
			repos = data.Items
		} else {
			var err error
			repos, err = listRepositories(syncAll)
			if err != nil {
				return errors.Wrap(err, "获取仓库列表失败")
			}
		}

		if len(repos) == 0 {
			printlnText("暂无仓库需要同步")
			return nil
		}

		printfText("正在同步 %d 个仓库...\n", len(repos))

		successCount := 0
		failedRepos := []string{}

		for _, repo := range repos {
			if !repo.Enabled && !syncAll {
				printfText("跳过已禁用的仓库: %s\n", repo.Name)
				continue
			}

			printfText("\n同步仓库: %s\n", repo.Name)
			result, err := syncConfiguredRepository(repo.Name)
			if err != nil {
				printfText("❌ 同步失败: %v\n", err)
				failedRepos = append(failedRepos, repo.Name)
			} else if isBlockedRepoSync(result) {
				printfText("⚠️  同步未执行: %s\n", result.Message)
				failedRepos = append(failedRepos, repo.Name)
			} else {
				successCount++
				if result.Message != "" {
					printfText("   %s\n", result.Message)
				}
			}
		}

		printfText("\n✅ 同步完成: %d 成功", successCount)
		if len(failedRepos) > 0 {
			printfText(", %d 失败: %v\n", len(failedRepos), failedRepos)
		} else {
			printlnText()
		}
	}

	return nil
}

func runRepoRebuildIndex(args []string, jsonOutput bool) error {
	repoName := ""
	if len(args) > 0 {
		repoName = args[0]
	} else {
		defaultRepo, err := defaultRepository()
		if err != nil {
			return errors.Wrap(err, "获取默认仓库失败")
		}
		repoName = defaultRepo.Name
	}

	if err := rebuildRepositoryIndex(repoName); err != nil {
		return errors.Wrapf(err, "重建仓库 '%s' 索引失败", repoName)
	}

	result := repoRebuildIndexResult{RepoName: repoName, Status: "rebuilt"}
	if jsonOutput {
		return writeJSON(result)
	}
	printfText("✅ 仓库 '%s' 索引已重建\n", repoName)
	return nil
}

func runRepoSyncStructured(args []string, syncAll bool) (*repoSyncSummary, error) {
	summary := &repoSyncSummary{All: len(args) == 0}
	client, useService := hubClientIfAvailable()

	if len(args) > 0 {
		name := args[0]
		item := repoSyncItem{Name: name, Enabled: true}
		result, err := syncConfiguredRepositoryWithClient(client, useService, name, true)
		if err != nil {
			item.Status = "failed"
			item.Error = err.Error()
			summary.Failed = 1
			summary.Items = append(summary.Items, item)
			summary.Total = 1
			return summary, err
		}
		applyRepoSyncResult(&item, result)
		if isBlockedRepoSync(result) {
			item.Error = result.Message
			summary.Failed = 1
			summary.Total = 1
			summary.Items = append(summary.Items, item)
			return summary, errors.NewWithCodef("runRepoSyncStructured", errors.ErrGitOperation, "%s", result.Message)
		}
		summary.Synced = 1
		summary.Total = 1
		summary.Items = append(summary.Items, item)
		return summary, nil
	}

	repos, err := reposForSync(client, useService)
	if err != nil {
		return summary, err
	}
	summary.Total = len(repos)
	for _, repo := range repos {
		item := repoSyncItem{Name: repo.Name, Enabled: repo.Enabled}
		if !repo.Enabled && !syncAll {
			item.Status = "skipped"
			summary.Skipped++
			summary.Items = append(summary.Items, item)
			continue
		}
		result, syncErr := syncConfiguredRepositoryWithClient(client, useService, repo.Name, true)
		if syncErr != nil {
			item.Status = "failed"
			item.Error = syncErr.Error()
			summary.Failed++
		} else if isBlockedRepoSync(result) {
			applyRepoSyncResult(&item, result)
			item.Error = result.Message
			summary.Failed++
		} else {
			applyRepoSyncResult(&item, result)
			summary.Synced++
		}
		summary.Items = append(summary.Items, item)
	}
	return summary, nil
}

func syncConfiguredRepository(name string) (*httpapibiz.RepoSyncData, error) {
	client, useService := hubClientIfAvailable()
	return syncConfiguredRepositoryWithClient(client, useService, name, false)
}

func syncConfiguredRepositoryWithClient(client serviceBridgeClient, useService bool, name string, quiet bool) (*httpapibiz.RepoSyncData, error) {
	if useService {
		return client.SyncRepo(context.Background(), name)
	}
	if !quiet {
		return syncRepository(name)
	}
	var result *httpapibiz.RepoSyncData
	err := runSilencingStdout(func() error {
		var syncErr error
		result, syncErr = syncRepository(name)
		return syncErr
	})
	return result, err
}

func applyRepoSyncResult(item *repoSyncItem, result *httpapibiz.RepoSyncData) {
	if result == nil {
		item.Status = "synced"
		return
	}
	item.Status = result.Status
	item.Message = result.Message
	item.Ahead = result.Ahead
	item.Behind = result.Behind
}

func isBlockedRepoSync(result *httpapibiz.RepoSyncData) bool {
	return result != nil && (result.Status == "divergent" || result.Status == "blocked_dirty")
}

func reposForSync(client serviceBridgeClient, useService bool) ([]configport.RepositoryConfig, error) {
	if useService {
		data, err := client.ListRepos(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "通过服务获取仓库列表失败")
		}
		return data.Items, nil
	}
	repos, err := listRepositories(true)
	if err != nil {
		return nil, errors.Wrap(err, "获取仓库列表失败")
	}
	return repos, nil
}

// runRepoEnable 执行启用仓库操作
func runRepoEnable(name string) error {
	if client, ok := hubClientIfAvailable(); ok {
		if err := client.EnableRepo(context.Background(), name); err != nil {
			return errors.Wrap(err, "通过服务启用仓库失败")
		}
	} else if err := enableRepository(name); err != nil {
		return errors.Wrap(err, "启用仓库失败")
	}

	printfText("✅ 仓库 '%s' 已启用\n", name)
	if jsonMode() {
		return writeJSON(map[string]any{"name": name, "status": "enabled"})
	}
	return nil
}

// runRepoDisable 执行禁用仓库操作
func runRepoDisable(name string) error {
	if jsonMode() {
		if !mustGetBoolFlag(activeOutput.command, "yes") {
			return inputRequired("JSON 禁用仓库需要 --yes")
		}
	} else {
		printfText("确定要禁用仓库 '%s' 吗？禁用后该仓库的技能将不可用。(y/N): ", name)
		var confirm string
		_ = scanInputToken(&confirm)
		if strings.ToLower(confirm) != "y" {
			printlnText("操作已取消")
			return nil
		}
	}

	if client, ok := hubClientIfAvailable(); ok {
		if err := client.DisableRepo(context.Background(), name); err != nil {
			return errors.Wrap(err, "通过服务禁用仓库失败")
		}
	} else if err := disableRepository(name); err != nil {
		return errors.Wrap(err, "禁用仓库失败")
	}

	printfText("✅ 仓库 '%s' 已禁用\n", name)
	if jsonMode() {
		return writeJSON(map[string]any{"name": name, "status": "disabled"})
	}
	return nil
}

// runRepoDefault 执行设置默认仓库操作
func runRepoDefault(name string) error {
	if client, ok := hubClientIfAvailable(); ok {
		if err := client.SetDefaultRepo(context.Background(), name); err != nil {
			return errors.Wrap(err, "通过服务设置默认仓库失败")
		}
	} else {
		if _, err := getRepository(name); err != nil {
			return errors.Wrapf(err, "仓库 '%s' 不存在或未启用", name)
		}

		if err := setDefaultRepository(name); err != nil {
			return errors.Wrap(err, "保存配置失败")
		}
	}

	printfText("✅ 默认仓库已设置为 '%s'\n", name)
	printlnText("注意：所有通过 feedback 命令修改的技能都会归档到此仓库")
	if jsonMode() {
		return writeJSON(map[string]any{"name": name, "status": "default"})
	}
	return nil
}

func renderRepositoryList(repos []configport.RepositoryConfig, defaultRepo string) {
	if len(repos) == 0 {
		printlnText("暂无仓库配置")
		return
	}

	printlnText("已配置的仓库:")
	printlnText(strings.Repeat("=", 80))

	for _, repo := range repos {
		marker := " "
		if repo.Name == defaultRepo {
			marker = "★"
		}

		status := "✓"
		if !repo.Enabled {
			status = "✗"
		}

		archive := ""
		if repo.IsArchive {
			archive = " [归档]"
		}

		printfText("%s %s %s%s\n", marker, status, repo.Name, archive)
		printfText("   类型: %s\n", repo.Type)
		if repo.Description != "" {
			printfText("   描述: %s\n", repo.Description)
		}
		if repo.URL != "" {
			printfText("   远程: %s\n", repo.URL)
			printfText("   分支: %s\n", repo.Branch)
		} else {
			printfText("   类型: 本地仓库\n")
		}
		if repo.LastSync != "" {
			printfText("   最后同步: %s\n", repo.LastSync)
		}
		printlnText()
	}

	printfText("★ 表示默认仓库（归档仓库）\n")
	printfText("✓ 表示已启用，✗ 表示已禁用\n")
}

// isValidRepoName 验证仓库名称是否有效
func isValidRepoName(name string) bool {
	if name == "" || len(name) > 50 {
		return false
	}

	// 只允许字母、数字、下划线和连字符
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}

	return true
}
