package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/git"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git仓库操作",
	Long:  "管理技能Git仓库的克隆、同步、提交等操作。",
}

var gitCloneCmd = &cobra.Command{
	Use:   "clone [url]",
	Short: "克隆远程技能仓库",
	Long:  "克隆指定的远程Git仓库到本地技能目录。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitClone(args[0])
	},
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
	Short: "设置远程仓库",
	Long:  "设置或更新技能仓库的远程URL。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitRemote(args[0])
	},
}

func init() {
	gitCmd.AddCommand(gitCloneCmd)
	gitCmd.AddCommand(gitSyncCmd)
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitPullCmd)
	gitCmd.AddCommand(gitRemoteCmd)
}

func runGitClone(url string) error {
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	return repo.CloneRemote(url)
}

func runGitSync() error {
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	return repo.Sync()
}

func runGitStatus() error {
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	status, err := repo.GetStatus()
	if err != nil {
		return err
	}

	fmt.Println(status)
	return nil
}

func runGitCommit() error {
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
		fmt.Println("⚠️  检测到未提交的更改")
		fmt.Print("是否先提交更改？ [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "y" || response == "Y" {
			return runGitCommit()
		}
	}

	// 直接推送
	fmt.Println("正在推送到远程仓库...")
	repoImpl, err := git.NewSkillsRepository()
	if err != nil {
		return err
	}

	return repoImpl.Push()
}

func runGitPull() error {
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	return repo.Sync()
}

func runGitRemote(url string) error {
	repo, err := git.NewSkillsRepository()
	if err != nil {
		return err
	}

	if err := repo.SetRemote(url); err != nil {
		return fmt.Errorf("设置远程仓库失败: %w", err)
	}

	fmt.Printf("✅ 远程仓库已设置为: %s\n", url)
	fmt.Println("使用 'skill-hub git sync' 同步技能")
	return nil
}
