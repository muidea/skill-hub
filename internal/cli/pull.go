package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/errors"
)

var (
	pullForce bool
	pullCheck bool
	pullJSON  bool
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "从默认仓库拉取最新技能",
	Long: `从默认仓库（归档仓库）对应的远程拉取最新更改到本地仓库，并更新技能注册表。

此命令仅处理默认仓库，不负责多仓库同步；多仓库同步请使用 'skill-hub repo sync'。
此命令仅同步仓库层（~/.skill-hub/repositories/），不涉及项目工作目录的更新。
使用 --check 选项可以检查可用更新但不实际执行拉取操作。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull()
	},
}

func init() {
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "强制拉取，忽略本地未提交的修改")
	pullCmd.Flags().BoolVar(&pullCheck, "check", false, "检查模式，仅显示可用的更新，不实际执行拉取操作")
	pullCmd.Flags().BoolVar(&pullJSON, "json", false, "以JSON格式输出拉取摘要")
}

func runPull() error {
	if pullJSON {
		summary, err := runPullStructured()
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
		return err
	}

	client, useService := hubClientIfAvailable()
	if !useService {
		// 检查init依赖（规范4.12：该命令依赖init命令）
		if err := CheckInitDependency(); err != nil {
			return err
		}
	}

	if pullCheck {
		fmt.Println("检查默认仓库远程的可用更新...")
		result, err := pullCheckDefaultRepository(client, useService)
		if err != nil {
			return errors.Wrap(err, "检查默认仓库远程更新失败")
		}
		renderPullCheckResult(result)
		return nil
	}

	fmt.Println("正在从默认仓库远程拉取最新技能...")

	skillCount, err := pullSyncDefaultRepository(client, useService)
	if err != nil {
		return errors.Wrap(err, "同步技能仓库失败")
	}

	fmt.Printf("\n✅ 默认仓库更新完成，共 %d 个技能\n", skillCount)
	fmt.Println("使用 'skill-hub status' 检查项目技能状态")
	fmt.Println("使用 'skill-hub apply' 将仓库更新应用到项目工作目录")
	fmt.Println("如需同步所有启用仓库，请使用 'skill-hub repo sync'")

	return nil
}

type pullSummary struct {
	Check        bool   `json:"check"`
	Force        bool   `json:"force"`
	Status       string `json:"status"`
	SkillCount   int    `json:"skill_count,omitempty"`
	Message      string `json:"message,omitempty"`
	RemoteURL    string `json:"remote_url,omitempty"`
	LocalCommit  string `json:"local_commit,omitempty"`
	RemoteCommit string `json:"remote_commit,omitempty"`
	HasUpdates   bool   `json:"has_updates"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	Error        string `json:"error,omitempty"`
}

func runPullStructured() (*pullSummary, error) {
	summary := &pullSummary{
		Check:  pullCheck,
		Force:  pullForce,
		Status: "unknown",
	}

	client, useService := hubClientIfAvailable()
	if !useService {
		if err := CheckInitDependency(); err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			return summary, err
		}
	}

	if pullCheck {
		result, err := pullCheckDefaultRepository(client, useService)
		if err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			return summary, errors.Wrap(err, "检查默认仓库远程更新失败")
		}
		applyPullCheckToSummary(summary, result)
		return summary, nil
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

func pullCheckDefaultRepository(client serviceBridgeClient, useService bool) (*httpapibiz.SkillRepositoryCheckData, error) {
	if useService {
		return client.CheckSkillRepositoryUpdates(context.Background())
	}
	result, err := checkSkillRepositoryUpdates()
	if err != nil {
		return nil, err
	}
	return &httpapibiz.SkillRepositoryCheckData{
		Status:       result.Status,
		Message:      result.Message,
		RemoteURL:    result.RemoteURL,
		LocalCommit:  result.LocalCommit,
		RemoteCommit: result.RemoteCommit,
		HasUpdates:   result.HasUpdates,
		Ahead:        result.Ahead,
		Behind:       result.Behind,
	}, nil
}

func applyPullCheckToSummary(summary *pullSummary, result *httpapibiz.SkillRepositoryCheckData) {
	summary.Status = result.Status
	summary.Message = result.Message
	summary.RemoteURL = result.RemoteURL
	summary.LocalCommit = result.LocalCommit
	summary.RemoteCommit = result.RemoteCommit
	summary.HasUpdates = result.HasUpdates
	summary.Ahead = result.Ahead
	summary.Behind = result.Behind
}

func renderPullCheckResult(result *httpapibiz.SkillRepositoryCheckData) {
	if result.RemoteURL != "" {
		fmt.Printf("远程仓库: %s\n", result.RemoteURL)
	}
	fmt.Printf("状态: %s\n", result.Status)
	if result.Message != "" {
		fmt.Println(result.Message)
	}
	if result.LocalCommit != "" {
		fmt.Printf("本地提交: %s\n", shortCommit(result.LocalCommit))
	}
	if result.RemoteCommit != "" {
		fmt.Printf("远程提交: %s\n", shortCommit(result.RemoteCommit))
	}
	if result.Ahead > 0 || result.Behind > 0 {
		fmt.Printf("本地领先: %d，远程领先: %d\n", result.Ahead, result.Behind)
	}
	if result.HasUpdates {
		fmt.Println("可执行 'skill-hub pull' 拉取更新")
	}
}

func shortCommit(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func pullSyncDefaultRepository(client serviceBridgeClient, useService bool) (int, error) {
	if useService {
		data, err := client.SyncSkillRepositoryAndRefresh(context.Background())
		if err != nil {
			return 0, err
		}
		return data.SkillCount, nil
	}

	if err := syncSkillRepositoryAndRefresh(); err != nil {
		return 0, err
	}
	return pullLocalSkillCount()
}

func pullSyncDefaultRepositoryQuiet(client serviceBridgeClient, useService bool) (int, error) {
	if useService {
		return pullSyncDefaultRepository(client, true)
	}
	var skillCount int
	err := runSilencingStdout(func() error {
		var syncErr error
		skillCount, syncErr = pullSyncDefaultRepository(nil, false)
		return syncErr
	})
	return skillCount, err
}

func pullLocalSkillCount() (int, error) {
	repo, err := newSkillRepository()
	if err != nil {
		return 0, err
	}
	skills, err := repo.ListLocalSkills()
	if err != nil {
		return 0, errors.Wrap(err, "获取技能列表失败")
	}
	return len(skills), nil
}
