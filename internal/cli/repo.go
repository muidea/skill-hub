package cli

import (
	"fmt"
	"strings"

	"skill-hub/internal/config"
	"skill-hub/internal/multirepo"
	"skill-hub/pkg/errors"

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
		return runRepoList()
	},
}

var repoRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "移除仓库",
	Long:    `从配置中移除指定的Git仓库。注意：这不会删除本地仓库文件。`,
	Args:    cobra.ExactArgs(1),
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
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		syncAll, _ := cmd.Flags().GetBool("all")
		return runRepoSync(args, syncAll)
	},
}

var repoEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "启用仓库",
	Long:  `启用之前被禁用的仓库。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoEnable(args[0])
	},
}

var repoDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "禁用仓库",
	Long:  `禁用指定仓库，禁用后该仓库的技能将不可用。`,
	Args:  cobra.ExactArgs(1),
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
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRepoDefault(args[0])
	},
}

func init() {
	// 添加命令标志
	repoAddCmd.Flags().String("branch", "main", "Git分支")
	repoAddCmd.Flags().String("type", "community", "仓库类型 (user/community/official)")
	repoAddCmd.Flags().String("description", "", "仓库描述")

	repoSyncCmd.Flags().Bool("all", false, "同步所有仓库（包括禁用的）")

	// 添加子命令
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	repoCmd.AddCommand(repoSyncCmd)
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
	repoConfig := config.RepositoryConfig{
		Name:        name,
		URL:         url,
		Branch:      branch,
		Type:        repoType,
		Description: description,
		Enabled:     true,
		IsArchive:   false, // 只有默认仓库才是归档仓库
	}

	// 添加仓库
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	if err := manager.AddRepository(repoConfig); err != nil {
		return errors.Wrap(err, "添加仓库失败")
	}

	fmt.Printf("✅ 仓库 '%s' 添加成功\n", name)
	if url != "" {
		fmt.Printf("   远程URL: %s\n", url)
		fmt.Printf("   分支: %s\n", branch)
	} else {
		fmt.Println("   类型: 本地空仓库")
	}
	fmt.Printf("   类型: %s\n", repoType)
	if description != "" {
		fmt.Printf("   描述: %s\n", description)
	}

	return nil
}

// runRepoList 执行列出仓库操作
func runRepoList() error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	repos, err := manager.ListRepositories()
	if err != nil {
		return errors.Wrap(err, "获取仓库列表失败")
	}

	if len(repos) == 0 {
		fmt.Println("暂无仓库配置")
		return nil
	}

	// 获取默认仓库
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "获取配置失败")
	}

	var defaultRepo string
	if cfg.MultiRepo != nil && cfg.MultiRepo.Enabled {
		defaultRepo = cfg.MultiRepo.DefaultRepo
	} else {
		defaultRepo = "main"
	}

	fmt.Println("已配置的仓库:")
	fmt.Println(strings.Repeat("=", 80))

	for _, repo := range repos {
		// 标记默认仓库
		marker := " "
		if repo.Name == defaultRepo {
			marker = "★"
		}

		// 状态标记
		status := "✓"
		if !repo.Enabled {
			status = "✗"
		}

		// 归档标记
		archive := ""
		if repo.IsArchive {
			archive = " [归档]"
		}

		fmt.Printf("%s %s %s%s\n", marker, status, repo.Name, archive)
		fmt.Printf("   类型: %s\n", repo.Type)
		if repo.Description != "" {
			fmt.Printf("   描述: %s\n", repo.Description)
		}
		if repo.URL != "" {
			fmt.Printf("   远程: %s\n", repo.URL)
			fmt.Printf("   分支: %s\n", repo.Branch)
		} else {
			fmt.Printf("   类型: 本地仓库\n")
		}
		if repo.LastSync != "" {
			fmt.Printf("   最后同步: %s\n", repo.LastSync)
		}
		fmt.Println()
	}

	fmt.Printf("★ 表示默认仓库（归档仓库）\n")
	fmt.Printf("✓ 表示已启用，✗ 表示已禁用\n")

	return nil
}

// runRepoRemove 执行移除仓库操作
func runRepoRemove(name string) error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	// 确认操作
	fmt.Printf("确定要移除仓库 '%s' 吗？(y/N): ", name)
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "y" {
		fmt.Println("操作已取消")
		return nil
	}

	if err := manager.RemoveRepository(name); err != nil {
		return errors.Wrap(err, "移除仓库失败")
	}

	fmt.Printf("✅ 仓库 '%s' 已从配置中移除\n", name)
	fmt.Println("注意：本地仓库文件仍然保留，如需完全删除请手动操作")

	return nil
}

// runRepoSync 执行同步仓库操作
func runRepoSync(args []string, syncAll bool) error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	if len(args) > 0 {
		// 同步指定仓库
		name := args[0]
		fmt.Printf("正在同步仓库 '%s'...\n", name)

		if err := manager.SyncRepository(name); err != nil {
			return errors.Wrapf(err, "同步仓库 '%s' 失败", name)
		}

		fmt.Printf("✅ 仓库 '%s' 同步完成\n", name)
	} else {
		// 同步所有仓库
		repos, err := manager.ListRepositories()
		if err != nil {
			return errors.Wrap(err, "获取仓库列表失败")
		}

		if len(repos) == 0 {
			fmt.Println("暂无仓库需要同步")
			return nil
		}

		fmt.Printf("正在同步 %d 个仓库...\n", len(repos))

		successCount := 0
		failedRepos := []string{}

		for _, repo := range repos {
			if !repo.Enabled && !syncAll {
				fmt.Printf("跳过已禁用的仓库: %s\n", repo.Name)
				continue
			}

			fmt.Printf("\n同步仓库: %s\n", repo.Name)
			if err := manager.SyncRepository(repo.Name); err != nil {
				fmt.Printf("❌ 同步失败: %v\n", err)
				failedRepos = append(failedRepos, repo.Name)
			} else {
				successCount++
			}
		}

		fmt.Printf("\n✅ 同步完成: %d 成功", successCount)
		if len(failedRepos) > 0 {
			fmt.Printf(", %d 失败: %v\n", len(failedRepos), failedRepos)
		} else {
			fmt.Println()
		}
	}

	return nil
}

// runRepoEnable 执行启用仓库操作
func runRepoEnable(name string) error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	if err := manager.EnableRepository(name); err != nil {
		return errors.Wrap(err, "启用仓库失败")
	}

	fmt.Printf("✅ 仓库 '%s' 已启用\n", name)
	return nil
}

// runRepoDisable 执行禁用仓库操作
func runRepoDisable(name string) error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	// 确认操作
	fmt.Printf("确定要禁用仓库 '%s' 吗？禁用后该仓库的技能将不可用。(y/N): ", name)
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "y" {
		fmt.Println("操作已取消")
		return nil
	}

	if err := manager.DisableRepository(name); err != nil {
		return errors.Wrap(err, "禁用仓库失败")
	}

	fmt.Printf("✅ 仓库 '%s' 已禁用\n", name)
	return nil
}

// runRepoDefault 执行设置默认仓库操作
func runRepoDefault(name string) error {
	manager, err := multirepo.NewManager()
	if err != nil {
		return errors.Wrap(err, "初始化多仓库管理器失败")
	}

	// 检查仓库是否存在
	if _, err := manager.GetRepository(name); err != nil {
		return errors.Wrapf(err, "仓库 '%s' 不存在或未启用", name)
	}

	// 获取配置
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "获取配置失败")
	}

	// 启用多仓库功能（如果尚未启用）
	if cfg.MultiRepo == nil {
		cfg.MultiRepo = &config.MultiRepoConfig{
			Enabled:      true,
			DefaultRepo:  name,
			Repositories: make(map[string]config.RepositoryConfig),
		}
	} else {
		cfg.MultiRepo.Enabled = true
		cfg.MultiRepo.DefaultRepo = name
	}

	// TODO: 保存配置到文件
	// 这里需要实现配置保存功能

	fmt.Printf("✅ 默认仓库已设置为 '%s'\n", name)
	fmt.Println("注意：所有通过 feedback 命令修改的技能都会归档到此仓库")

	return nil
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
