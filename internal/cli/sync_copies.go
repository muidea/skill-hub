package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
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
		return runSyncCopies(projectlifecycleservice.SyncCopiesOptions{
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
	syncCopiesCmd.Flags().Bool("json", false, "以JSON格式输出同步结果")
	_ = syncCopiesCmd.MarkFlagRequired("canonical")
}

func runSyncCopies(opts projectlifecycleservice.SyncCopiesOptions, jsonOutput bool) error {
	absScope, err := filepath.Abs(opts.Scope)
	if err != nil {
		return errors.Wrap(err, "解析scope失败")
	}
	opts.Scope = absScope

	var result *projectlifecycleservice.SyncCopiesResult
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.SyncCopies(context.Background(), httpapibiz.SyncCopiesRequest{Options: opts})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务同步技能副本失败")
		}
		result = data.Item
	} else {
		lifecycleSvc := projectlifecycleservice.New()
		result, err = lifecycleSvc.SyncCopies(opts)
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

func renderSyncCopiesResult(result *projectlifecycleservice.SyncCopiesResult) {
	if result == nil {
		fmt.Println("未返回同步结果")
		return
	}
	if result.DryRun {
		fmt.Println("🔎 sync-copies 演习模式，未修改文件")
	}
	fmt.Printf("扫描范围: %s\n", result.Scope)
	fmt.Printf("canonical: %s\n", result.Canonical)
	fmt.Printf("synced:    %d\n", result.Synced)
	fmt.Printf("unchanged: %d\n", result.Unchanged)
	fmt.Printf("skipped:   %d\n", result.Skipped)
	fmt.Printf("failed:    %d\n", len(result.Failures))
	for _, item := range result.Items {
		fmt.Printf("- [%s] %s -> %s", item.Status, item.SourceDir, item.TargetDir)
		if item.BackupDir != "" {
			fmt.Printf(" backup=%s", item.BackupDir)
		}
		if item.Message != "" {
			fmt.Printf(" %s", item.Message)
		}
		fmt.Println()
	}
	if len(result.Failures) > 0 {
		fmt.Println("\n失败项:")
		for _, failure := range result.Failures {
			fmt.Printf("- id=%s path=%s error=%s\n", failure.SkillID, failure.Path, failure.Error)
		}
	}
}
