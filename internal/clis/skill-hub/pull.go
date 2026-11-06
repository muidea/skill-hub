package cli

import (
	"context"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/gitport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
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
		cmd.SilenceUsage = true // Arguments have already been validated by Cobra.
		pullJSON = mustGetBoolFlag(cmd, "json")
		return runPull()
	},
}

func init() {
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "远端有更新且默认仓库存在未提交更改时，先自动 stash 再拉取")
	pullCmd.Flags().BoolVar(&pullCheck, "check", false, "检查模式，仅显示可用的更新，不实际执行拉取操作")
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
		printlnText("检查默认仓库远程的可用更新...")
		result, err := pullCheckDefaultRepository(client, useService)
		if err != nil {
			return errors.Wrap(err, "检查默认仓库远程更新失败")
		}
		renderPullCheckResult(result)
		return nil
	}

	printlnText("正在从默认仓库远程拉取最新技能...")

	result, err := pullSyncDefaultRepositoryResultQuiet(client, useService)
	if err != nil {
		if diagnostic := gitport.PullDiagnostic(err); diagnostic != nil {
			renderPullDiagnostic(diagnostic)
			return &reportedError{errors.Wrap(err, "同步技能仓库失败")}
		}
		if result != nil && result.StashRef != "" {
			printfText("本地改动保存在 stash %s，未自动恢复\n", result.StashRef)
		}
		return errors.Wrap(err, "同步技能仓库失败")
	}

	printfText("\n✅ 默认仓库更新完成，共 %d 个技能\n", result.SkillCount)
	if result.StashRef != "" {
		printfText("本地改动保存在 stash %s，未自动恢复\n", result.StashRef)
	}
	printlnText("使用 'skill-hub status' 检查项目技能状态")
	printlnText("使用 'skill-hub apply' 将仓库更新应用到项目工作目录")
	printlnText("如需同步所有启用仓库，请使用 'skill-hub repo sync'")

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
	StashRef     string `json:"stash_ref,omitempty"`
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

	result, err := pullSyncDefaultRepositoryResultQuiet(client, useService)
	if result != nil {
		summary.StashRef = result.StashRef
	}
	if err != nil {
		if d := gitport.PullDiagnostic(err); d != nil {
			summary.LocalCommit, summary.RemoteCommit = d.LocalCommit, d.RemoteCommit
			summary.Ahead, summary.Behind, summary.HasUpdates = d.Ahead, d.Behind, d.Behind > 0
		}
		summary.Status = "failed"
		summary.Error = err.Error()
		return summary, errors.Wrap(err, "同步技能仓库失败")
	}
	summary.Status = "synced"
	summary.SkillCount = result.SkillCount
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
		printfText("远程仓库: %s\n", result.RemoteURL)
	}
	printfText("状态: %s\n", result.Status)
	if result.Message != "" {
		printlnText(result.Message)
	}
	if result.LocalCommit != "" {
		printfText("本地提交: %s\n", shortCommit(result.LocalCommit))
	}
	if result.RemoteCommit != "" {
		printfText("远程提交: %s\n", shortCommit(result.RemoteCommit))
	}
	if result.Ahead > 0 || result.Behind > 0 {
		printfText("本地领先: %d，远程领先: %d\n", result.Ahead, result.Behind)
	}
	if result.HasUpdates {
		printlnText("可执行 'skill-hub pull' 拉取更新")
	}
}

func shortCommit(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

func pullSyncDefaultRepository(client serviceBridgeClient, useService bool) (int, error) {
	result, err := pullSyncDefaultRepositoryResult(client, useService)
	if result == nil {
		return 0, err
	}
	return result.SkillCount, err
}

func pullSyncDefaultRepositoryResult(client serviceBridgeClient, useService bool) (*spec.PullResult, error) {
	if useService {
		if pullForce {
			return nil, errors.NewWithCode("pullSyncDefaultRepository", errors.ErrInvalidInput, "服务模式暂不支持 pull --force；请停止服务后本地执行，或先提交/暂存默认仓库更改")
		}
		data, err := client.SyncSkillRepositoryAndRefresh(context.Background())
		if err != nil {
			return nil, err
		}
		return &spec.PullResult{SkillCount: data.SkillCount}, nil
	}

	return localRuntime.Git().SyncDefaultRepositoryWithResult(pullForce)
}

func pullSyncDefaultRepositoryResultQuiet(client serviceBridgeClient, useService bool) (*spec.PullResult, error) {
	if useService {
		return pullSyncDefaultRepositoryResult(client, true)
	}
	var result *spec.PullResult
	err := runSilencingStdout(func() error {
		var syncErr error
		result, syncErr = pullSyncDefaultRepositoryResult(nil, false)
		return syncErr
	})
	return result, err
}

func pullSyncDefaultRepositoryQuiet(client serviceBridgeClient, useService bool) (int, error) {
	result, err := pullSyncDefaultRepositoryResultQuiet(client, useService)
	if result == nil {
		return 0, err
	}
	return result.SkillCount, err
}
