package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
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
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runGitSyncWithOptions(jsonOutput)
	},
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看仓库状态",
	Long:  "显示技能Git仓库的当前状态，包括未提交的更改。",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runGitStatusWithOptions(jsonOutput)
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
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runGitPullWithOptions(jsonOutput)
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

	gitSyncCmd.Flags().Bool("json", false, "以JSON格式输出同步摘要")
	gitStatusCmd.Flags().Bool("json", false, "以JSON格式输出仓库状态")
	gitPullCmd.Flags().Bool("json", false, "以JSON格式输出拉取摘要")
	gitRemoteCmd.Flags().BoolP("verbose", "v", false, "显示详细远程仓库信息")
}

func runGitSyncWithOptions(jsonOutput bool) error {
	return runGitRepositorySync("sync", jsonOutput)
}

func runGitRepositorySync(command string, jsonOutput bool) error {
	if jsonOutput {
		summary, err := runGitRepositorySyncStructured(command)
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
		return err
	}

	client, useService := hubClientIfAvailable()
	if !useService {
		// 检查init依赖（规范4.14：该命令依赖init命令）
		if err := CheckInitDependency(); err != nil {
			return err
		}
	}
	_, err := pullSyncDefaultRepository(client, useService)
	return err
}

type gitSyncSummary struct {
	Command    string `json:"command"`
	Status     string `json:"status"`
	SkillCount int    `json:"skill_count,omitempty"`
	Error      string `json:"error,omitempty"`
}

func runGitRepositorySyncStructured(command string) (*gitSyncSummary, error) {
	summary := &gitSyncSummary{Command: command, Status: "unknown"}

	client, useService := hubClientIfAvailable()
	if !useService {
		if err := CheckInitDependency(); err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			return summary, err
		}
	}

	skillCount, err := pullSyncDefaultRepositoryQuiet(client, useService)
	if err != nil {
		summary.Status = "failed"
		summary.Error = err.Error()
		return summary, errors.Wrap(err, "同步技能仓库失败")
	}
	summary.Status = "synced"
	summary.SkillCount = skillCount
	return summary, nil
}

func runGitStatusWithOptions(jsonOutput bool) error {
	client, useService := hubClientIfAvailable()
	if !useService {
		// 检查init依赖（规范4.14：该命令依赖init命令）
		if err := CheckInitDependency(); err != nil {
			return err
		}
	}

	status, err := gitRepositoryStatus(client, useService)
	if err != nil {
		return errors.Wrap(err, "获取状态失败")
	}

	if jsonOutput {
		return writeJSON(buildGitStatusSummary(status))
	}

	fmt.Println(status)
	return nil
}

type gitStatusSummary struct {
	State        string   `json:"state"`
	HasChanges   bool     `json:"has_changes"`
	ChangedFiles []string `json:"changed_files"`
	RawStatus    string   `json:"raw_status"`
}

func gitRepositoryStatus(client serviceBridgeClient, useService bool) (string, error) {
	if useService {
		data, err := client.SkillRepositoryStatus(context.Background())
		if err != nil {
			return "", err
		}
		return data.Status, nil
	}
	return skillRepositoryStatus()
}

func buildGitStatusSummary(status string) gitStatusSummary {
	changedLines := pushChangedLines(status)
	state := "clean"
	if strings.Contains(status, "技能仓库未初始化") {
		state = "not_initialized"
	} else if len(changedLines) > 0 {
		state = "dirty"
	}
	return gitStatusSummary{
		State:        state,
		HasChanges:   len(changedLines) > 0,
		ChangedFiles: pushChangedFiles(changedLines),
		RawStatus:    status,
	}
}

func runGitCommit() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
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

	return pushSkillRepositoryChanges(message)
}

func runGitPush() error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 检查是否有未提交的更改
	status, err := skillRepositoryStatus()
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
	return pushSkillRepositoryCommits()
}

func runGitPullWithOptions(jsonOutput bool) error {
	return runGitRepositorySync("pull", jsonOutput)
}

func runGitRemoteSet(url string) error {
	// 检查init依赖（规范4.14：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	if err := setSkillRepositoryRemote(url); err != nil {
		return errors.Wrap(err, "设置远程仓库失败")
	}

	defaultRepo, err := defaultRepository()
	if err == nil {
		_ = updateRepositoryURL(defaultRepo.Name, url)
	}

	fmt.Printf("✅ 远程仓库已设置为: %s\n", url)
	fmt.Println("使用 'skill-hub git sync' 同步技能")
	return nil
}

func runGitRemoteView(verbose bool) error {
	if err := CheckInitDependency(); err != nil {
		return err
	}

	repoCfg, err := defaultRepository()
	if err != nil {
		return errors.Wrap(err, "获取默认仓库失败")
	}

	repoPath, err := repositoryPath(repoCfg.Name)
	if err != nil {
		return errors.Wrap(err, "获取仓库路径失败")
	}

	repo, err := newGitRepository(repoPath)
	if err != nil {
		return errors.Wrap(err, "打开Git仓库失败")
	}

	var remoteURLs []string
	if urls, err := repo.GetRemote(); err == nil {
		remoteURLs = urls
	}

	fmt.Printf("默认仓库: %s\n", repoCfg.Name)
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
