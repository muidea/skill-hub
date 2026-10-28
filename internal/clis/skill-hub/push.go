package cli

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/gitport"
	"github.com/muidea/skill-hub/pkg/errors"
)

var (
	pushMessage string
	pushForce   bool
	pushDryRun  bool
	pushJSON    bool
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "推送默认仓库的本地更改",
	Long: `自动检测并提交默认仓库（归档仓库）中的未提交更改，然后推送到对应远程仓库。

此命令默认只处理默认仓库，用于完成 feedback -> push 的归档闭环。
使用 --dry-run 选项可以查看将要推送的更改而不实际执行。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pushJSON = mustGetBoolFlag(cmd, "json")
		return runPush()
	},
}

func init() {
	pushCmd.Flags().StringVarP(&pushMessage, "message", "m", "", "提交消息。如未提供，将根据待推送文件自动生成")
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "强制推送，跳过确认检查")
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "演习模式，仅显示将要推送的更改，不实际执行")
}

func runPush() error {
	if pushJSON {
		if !pushDryRun && !pushForce {
			return errors.NewWithCode("runPush", errors.ErrInvalidInput, "JSON推送写入需要 --force，或使用 --dry-run 预览")
		}
		summary, err := runPushStructured()
		if writeErr := writeJSON(summary); writeErr != nil {
			return writeErr
		}
		return err
	}

	client, useService := hubClientIfAvailable()
	if !useService {
		// 检查init依赖（规范4.13：该命令依赖init命令）
		if err := CheckInitDependency(); err != nil {
			return err
		}
	}

	status, err := pushRepositoryStatus(client, useService)
	if err != nil {
		return errors.Wrap(err, "获取仓库状态失败")
	}

	state, err := pushRepositoryState(client, useService, status)
	if err != nil {
		return errors.Wrap(err, "检查待推送提交失败")
	}
	if !state.HasPendingPush {
		printlnText("没有要推送的更改")
		return nil
	}
	changedLines := state.ChangedLines

	// 显示将要推送的更改
	if len(changedLines) > 0 {
		printlnText("检测到以下未提交的更改:")
	} else {
		printfText("检测到 %d 个本地提交尚未推送:\n", state.Ahead)
	}
	printlnText(strings.Repeat("-", 50))

	for _, line := range changedLines {
		printlnText(line)
	}

	printlnText(strings.Repeat("-", 50))

	// 演习模式
	if pushDryRun {
		printlnText("演习模式：仅显示将要推送的更改，不实际执行")
		printlnText("使用 'skill-hub push'（不带 --dry-run）实际推送更改")
		return nil
	}

	// 确认推送（除非使用 --force）
	if !pushForce {
		printText("\n是否推送这些更改到默认仓库远程？ [y/N]: ")
		reader := bufio.NewReader(inputSource())
		response, _ := readInputLine(reader)
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			printlnText("取消推送操作")
			return nil
		}
	}

	// 确定提交消息
	message := pushMessage
	if message == "" {
		message = gitport.SuggestedCommitMessage(pushChangedFiles(changedLines))
	}

	printlnText("正在提交并推送默认仓库更改...")

	if err := pushRepositoryChanges(client, useService, message); err != nil {
		return errors.Wrap(err, "推送失败")
	}

	printlnText("✅ 默认仓库更改已成功推送到远程仓库")
	return nil
}

type pushSummary struct {
	DryRun         bool     `json:"dry_run"`
	Force          bool     `json:"force"`
	Message        string   `json:"message,omitempty"`
	HasChanges     bool     `json:"has_changes"`
	HasPendingPush bool     `json:"has_pending_push"`
	ChangedFiles   []string `json:"changed_files"`
	Ahead          int      `json:"ahead"`
	Behind         int      `json:"behind"`
	Status         string   `json:"status"`
	Error          string   `json:"error,omitempty"`
}

func runPushStructured() (*pushSummary, error) {
	summary := &pushSummary{
		DryRun: pushDryRun,
		Force:  pushForce,
		Status: "unknown",
	}

	message := pushMessage

	client, useService := hubClientIfAvailable()
	if !useService {
		if err := CheckInitDependency(); err != nil {
			summary.Status = "failed"
			summary.Error = err.Error()
			return summary, err
		}
	}

	status, err := pushRepositoryStatus(client, useService)
	if err != nil {
		summary.Status = "failed"
		summary.Error = err.Error()
		return summary, errors.Wrap(err, "获取仓库状态失败")
	}

	state, err := pushRepositoryState(client, useService, status)
	if err != nil {
		summary.Status = "failed"
		summary.Error = err.Error()
		return summary, errors.Wrap(err, "检查待推送提交失败")
	}
	summary.HasChanges = len(state.ChangedLines) > 0
	summary.HasPendingPush = state.HasPendingPush
	summary.ChangedFiles = pushChangedFiles(state.ChangedLines)
	summary.Ahead = state.Ahead
	summary.Behind = state.Behind
	if message == "" {
		message = gitport.SuggestedCommitMessage(summary.ChangedFiles)
	}
	summary.Message = message
	if !summary.HasPendingPush {
		summary.Status = "no_changes"
		return summary, nil
	}

	if pushDryRun {
		summary.Status = "planned"
		return summary, nil
	}

	var pushErr error
	if useService {
		_, pushErr = client.PushSkillRepositoryChanges(context.Background(), httpapibiz.PushSkillRepositoryRequest{
			Message:              message,
			Confirm:              true,
			ExpectedChangedFiles: summary.ChangedFiles,
		})
	} else {
		pushErr = runSilencingStdout(func() error {
			return pushSkillRepositoryChanges(message)
		})
	}
	if pushErr != nil {
		summary.Status = "failed"
		summary.Error = pushErr.Error()
		return summary, errors.Wrap(pushErr, "推送失败")
	}

	summary.Status = "pushed"
	return summary, nil
}

func pushRepositoryStatus(client serviceBridgeClient, useService bool) (string, error) {
	if useService {
		data, err := client.SkillRepositoryStatus(context.Background())
		if err != nil {
			return "", err
		}
		return data.Status, nil
	}
	return skillRepositoryStatus()
}

type repositoryPushState struct {
	ChangedLines   []string
	Ahead          int
	Behind         int
	HasPendingPush bool
}

func pushRepositoryState(client serviceBridgeClient, useService bool, status string) (*repositoryPushState, error) {
	state := &repositoryPushState{ChangedLines: pushChangedLines(status)}
	var (
		updates *httpapibiz.SkillRepositoryCheckData
		err     error
	)
	if useService {
		updates, err = client.CheckSkillRepositoryUpdates(context.Background())
	} else {
		localUpdates, checkErr := checkSkillRepositoryUpdates()
		if checkErr != nil {
			return nil, checkErr
		}
		if localUpdates != nil {
			updates = &httpapibiz.SkillRepositoryCheckData{
				Ahead:  localUpdates.Ahead,
				Behind: localUpdates.Behind,
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if updates != nil {
		state.Ahead = updates.Ahead
		state.Behind = updates.Behind
	}
	state.HasPendingPush = len(state.ChangedLines) > 0 || state.Ahead > 0
	return state, nil
}

func pushRepositoryChanges(client serviceBridgeClient, useService bool, message string) error {
	if useService {
		_, err := client.PushSkillRepositoryChanges(context.Background(), httpapibiz.PushSkillRepositoryRequest{Message: message, Confirm: true})
		return err
	}
	return pushSkillRepositoryChanges(message)
}

func pushChangedLines(status string) []string {
	var changed []string
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if isPushChangeLine(line) {
			changed = append(changed, line)
		}
	}
	return changed
}

func pushChangedFiles(changedLines []string) []string {
	files := make([]string, 0, len(changedLines))
	for _, line := range changedLines {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
			continue
		}
		files = append(files, strings.TrimSpace(line))
	}
	return files
}

func isPushChangeLine(line string) bool {
	for _, prefix := range []string{" M ", "?? ", " D ", "M  ", "A  ", "D  ", "R  ", "C  ", "AM ", "MM ", "AD ", "MD "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func runSilencingStdout(fn func() error) error {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return fn()
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, r)
		close(done)
	}()
	err = fn()
	_ = w.Close()
	os.Stdout = oldStdout
	<-done
	_ = r.Close()
	return err
}
