package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/config"
	"skill-hub/internal/git"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git仓库操作",
	Long:  "管理技能Git仓库的克隆、同步、提交等操作。",
}

var gitSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "同步技能仓库",
	Long:  "从远程仓库拉取最新技能，更新本地副本。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSync()
	},
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看仓库状态",
	Long:  "显示技能Git仓库的当前状态，包括未提交的更改。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitStatus()
	},
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "提交更改",
	Long:  "提交本地更改到技能仓库，并推送到远程（如果已配置）。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitCommit()
	},
}

var gitPushCmd = &cobra.Command{
	Use:   "push",
	Short: "推送更改",
	Long:  "将本地提交推送到远程技能仓库。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitPush()
	},
}

var gitPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "拉取更新",
	Long:  "从远程技能仓库拉取最新更改。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitPull()
	},
}

var gitRemoteCmd = &cobra.Command{
	Use:   "remote [url]",
	Short: "查看或设置远程仓库",
	Long:  "查看当前默认技能仓库的远程配置，或设置/更新远程URL。",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		if len(args) == 0 {
			return runGitRemoteView(verbose)
		}
		return runGitRemoteSet(args[0])
	},
}

func init() {
	gitCmd.AddCommand(gitSyncCmd)
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitPullCmd)
	gitCmd.AddCommand(gitRemoteCmd)

	gitRemoteCmd.Flags().BoolP("verbose", "v", false, "显示详细远程仓库信息")
}

func runGitSync() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	return repo.Sync()
}

func runGitStatus() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillRepository()
	if err != nil {
		return fmt.Errorf("创建技能仓库失败: %w", err)
	}

	status, err := repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取状态失败: %w", err)
	}

	fmt.Println(status)
	return nil
}

func runGitCommit() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	// 获取提交信息
	fmt.Print("请输入提交信息: ")
	reader := bufio.NewReader(os.Stdin)
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	if message == "" {
		message = "更新技能"
	}

	return repo.PushChanges(message)
}

func runGitPush() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	// 检查是否有未提交的更改
	status, err := repo.GetStatus()
	if err != nil {
		return err
	}

	if strings.Contains(status, " M ") || strings.Contains(status, "?? ") {
		fmt.Println("⚠️  检测到工作区存在未提交的更改（包含已跟踪文件修改或未跟踪文件）。")
		fmt.Println("    - 选 Y: 先把当前更改一起提交并推送")
		fmt.Println("    - 选 N: 仅推送已经存在的提交，保留本地未提交更改")
		fmt.Println("    - 建议: 如需查看详细变更，请先运行 'skill-hub git status'")
		fmt.Print("是否在推送前先提交所有本地更改？ [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "y" || response == "Y" {
			return runGitCommit()
		}
	}

	// 直接推送已存在的提交（工作区可能仍有未提交更改）
	fmt.Println("推送到远程仓库...")
	repoImpl, err := git.NewSkillsRepository()
	if err != nil {
		return err
	}

	return repoImpl.Push()
}

func runGitPull() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	return repo.Sync()
}

func runGitRemoteSet(url string) error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repo, err := git.NewSkillsRepository()
	if err != nil {
		return err
	}

	if err := repo.SetRemote(url); err != nil {
		return fmt.Errorf("设置远程仓库失败: %w", err)
	}

	cfg, err := config.GetConfig()
	if err == nil && cfg.MultiRepo != nil {
		if repoCfg, ok := cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo]; ok {
			repoCfg.URL = url
			cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo] = repoCfg
			_ = config.SaveConfig(cfg)
		}
	}

	fmt.Printf("✅ 远程仓库已设置为: %s\n", url)
	fmt.Println("使用 'skill-hub git sync' 同步技能")
	return nil
}

func runGitRemoteView(verbose bool) error {
	if err := CheckInitDependency(); err != nil {
		return err
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}

	if cfg.MultiRepo == nil || !cfg.MultiRepo.Enabled {
		return fmt.Errorf("多仓库功能未启用，请先运行 'skill-hub init' 或使用 'skill-hub repo' 配置仓库")
	}

	defaultRepoName := cfg.MultiRepo.DefaultRepo
	if defaultRepoName == "" {
		defaultRepoName = "main"
	}

	repoCfg, ok := cfg.MultiRepo.Repositories[defaultRepoName]
	if !ok {
		return fmt.Errorf("默认仓库 '%s' 未在配置中找到，请检查 config.yaml", defaultRepoName)
	}

	repoPath, err := config.GetRepositoryPath(defaultRepoName)
	if err != nil {
		return fmt.Errorf("获取仓库路径失败: %w", err)
	}

	repo, err := git.NewRepository(repoPath)
	if err != nil {
		return fmt.Errorf("打开Git仓库失败: %w", err)
	}

	var remoteURLs []string
	if urls, err := repo.GetRemote(); err == nil {
		remoteURLs = urls
	}

	fmt.Printf("默认仓库: %s\n", defaultRepoName)
	fmt.Printf("状态: %s\n", func() string {
		if repoCfg.Enabled {
			return "已启用"
		}
		return "已禁用"
	}())

	fmt.Printf("类型: %s\n", repoCfg.Type)
	if repoCfg.IsArchive {
		fmt.Println("角色: 归档仓库")
	}

	fmt.Println()
	fmt.Println("远程配置:")
	if len(remoteURLs) == 0 {
		fmt.Println("  未检测到远程仓库(origin)，可以使用 'skill-hub git remote <url>' 进行设置")
	} else {
		for _, u := range remoteURLs {
			fmt.Printf("  origin  %s (fetch)\n", u)
			fmt.Printf("  origin  %s (push)\n", u)
			if !verbose {
				break
			}
		}
	}

	fmt.Println()
	fmt.Println("配置文件中的远程:")
	if repoCfg.URL == "" {
		fmt.Println("  URL:    未配置")
	} else {
		fmt.Printf("  URL:    %s\n", repoCfg.URL)
	}
	if repoCfg.Branch == "" {
		fmt.Println("  分支:   main")
	} else {
		fmt.Printf("  分支:   %s\n", repoCfg.Branch)
	}

	fmt.Println("仓库路径:")
	fmt.Printf("  %s\n", repoPath)

	if len(remoteURLs) > 0 && repoCfg.URL != "" && remoteURLs[0] != repoCfg.URL {
		fmt.Println()
		fmt.Println("提示: 实际Git远程URL与配置文件不同，如有需要请更新 config.yaml 或重新运行 'skill-hub git remote <url>'")
	}

	return nil
}
