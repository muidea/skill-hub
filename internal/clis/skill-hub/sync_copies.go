package cli

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	projectport "github.com/muidea/skill-hub/internal/pkg/projectlifecycleport"
	"github.com/muidea/skill-hub/pkg/errors"
)

var syncCopiesCmd = &cobra.Command{
	Use:   "sync-copies",
	Short: "从canonical目录同步重复技能副本",
	Long: `将scope下重复的 .agents/skills/<id> 副本同步为canonical目录中的对应内容。

默认会在修改每个副本前创建 <skill-dir>.bak.<timestamp> 备份。不会删除任何技能副本目录。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		canonical, _ := cmd.Flags().GetString("canonical")
		scope, _ := cmd.Flags().GetString("scope")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		noBackup, _ := cmd.Flags().GetBool("no-backup")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runSyncCopies(projectport.SyncCopiesOptions{
			Canonical: canonical,
			Scope:     scope,
			DryRun:    dryRun,
			NoBackup:  noBackup,
		}, jsonOutput)
	},
}

func init() {
	syncCopiesCmd.Flags().String("canonical", "", "canonical技能目录，例如 .agents/skills")
	syncCopiesCmd.Flags().String("scope", ".", "扫描范围")
	syncCopiesCmd.Flags().Bool("dry-run", false, "演习模式，仅报告将同步的副本")
	syncCopiesCmd.Flags().Bool("no-backup", false, "同步前不创建备份")

	_ = syncCopiesCmd.MarkFlagRequired("canonical")
}

func runSyncCopies(opts projectport.SyncCopiesOptions, jsonOutput bool) error {
	absScope, err := filepath.Abs(opts.Scope)
	if err != nil {
		return errors.Wrap(err, "解析scope失败")
	}
	opts.Scope = absScope

	var result *projectport.SyncCopiesResult
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.SyncCopies(context.Background(), httpapibiz.SyncCopiesRequest{Options: opts})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务同步技能副本失败")
		}
		result = data.Item
	} else {
		result, err = localRuntime.Project().SyncCopies(opts)
		if err != nil && result == nil {
			return err
		}
	}

	if jsonOutput {
		if err := writeJSON(result); err != nil {
			return err
		}
	} else {
		renderSyncCopiesResult(result)
	}
	if result != nil && len(result.Failures) > 0 {
		return errors.NewWithCodef("runSyncCopies", errors.ErrFileOperation, "%d 个技能副本同步失败", len(result.Failures))
	}
	return err
}

func renderSyncCopiesResult(result *projectport.SyncCopiesResult) {
	if result == nil {
		printlnText("未返回同步结果")
		return
	}
	if result.DryRun {
		printlnText("🔎 sync-copies 演习模式，未修改文件")
	}
	printfText("扫描范围: %s\n", result.Scope)
	printfText("canonical: %s\n", result.Canonical)
	printfText("synced:    %d\n", result.Synced)
	printfText("unchanged: %d\n", result.Unchanged)
	printfText("skipped:   %d\n", result.Skipped)
	printfText("failed:    %d\n", len(result.Failures))
	for _, item := range result.Items {
		printfText("- [%s] %s -> %s", item.Status, item.SourceDir, item.TargetDir)
		if item.BackupDir != "" {
			printfText(" backup=%s", item.BackupDir)
		}
		if item.Message != "" {
			printfText(" %s", item.Message)
		}
		printlnText()
	}
	if len(result.Failures) > 0 {
		printlnText("\n失败项:")
		for _, failure := range result.Failures {
			printfText("- id=%s path=%s error=%s\n", failure.SkillID, failure.Path, failure.Error)
		}
	}
}
