package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	gitpkg "github.com/muidea/skill-hub/internal/git"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
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
		return runPush()
	},
}

func init() {
	pushCmd.Flags().StringVarP(&pushMessage, "message", "m", "", "提交消息。如未提供，将根据待推送文件自动生成")
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "强制推送，跳过确认检查")
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "演习模式，仅显示将要推送的更改，不实际执行")
	pushCmd.Flags().BoolVar(&pushJSON, "json", false, "以JSON格式输出推送摘要")
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

	// 检查是否有未提交的更改
	changedLines := pushChangedLines(status)
	if len(changedLines) == 0 {
		fmt.Println("没有要推送的更改")
		return nil
	}

	// 显示将要推送的更改
	fmt.Println("检测到以下未提交的更改:")
	fmt.Println(strings.Repeat("-", 50))

	for _, line := range changedLines {
		fmt.Println(line)
	}

	fmt.Println(strings.Repeat("-", 50))

	// 演习模式
	if pushDryRun {
		fmt.Println("演习模式：仅显示将要推送的更改，不实际执行")
		fmt.Println("使用 'skill-hub push'（不带 --dry-run）实际推送更改")
		return nil
	}

	// 确认推送（除非使用 --force）
	if !pushForce {
		fmt.Print("\n是否推送这些更改到默认仓库远程？ [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("取消推送操作")
			return nil
		}
	}

	// 确定提交消息
	message := pushMessage
	if message == "" {
		message = gitpkg.SuggestedCommitMessage(pushChangedFiles(changedLines))
	}

	fmt.Println("正在提交并推送默认仓库更改...")

	if err := pushRepositoryChanges(client, useService, message); err != nil {
		return errors.Wrap(err, "推送失败")
	}

	fmt.Println("✅ 默认仓库更改已成功推送到远程仓库")
	return nil
}

type pushSummary struct {
	DryRun       bool     `json:"dry_run"`
	Force        bool     `json:"force"`
	Message      string   `json:"message,omitempty"`
	HasChanges   bool     `json:"has_changes"`
	ChangedFiles []string `json:"changed_files"`
	Status       string   `json:"status"`
	Error        string   `json:"error,omitempty"`
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

	changedLines := pushChangedLines(status)
	summary.HasChanges = len(changedLines) > 0
	summary.ChangedFiles = pushChangedFiles(changedLines)
	if message == "" {
		message = gitpkg.SuggestedCommitMessage(summary.ChangedFiles)
	}
	summary.Message = message
	if !summary.HasChanges {
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
