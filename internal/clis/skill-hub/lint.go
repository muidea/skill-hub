package cli

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	projectport "github.com/muidea/skill-hub/internal/pkg/projectlifecycleport"
	"github.com/muidea/skill-hub/pkg/errors"
)

var lintCmd = &cobra.Command{
	Use:   "lint [scope]",
	Short: "审计项目技能内容",
	Long: `审计项目技能内容中的可移植性问题。

当前支持 --paths，用于扫描 .agents/skills 下 SKILL.md 及资源文件中的 file://、vscode://、/home/...、/Users/... 等本机绝对路径。
默认只报告；使用 --fix 可将 project-root 内的本机路径改写为相对路径。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := cmd.Flags().GetBool("paths")
		if !paths {
			return errors.NewWithCode("runLint", errors.ErrValidation, "当前仅支持 --paths")
		}
		scope := "."
		if len(args) > 0 {
			scope = args[0]
		}
		projectRoot, _ := cmd.Flags().GetString("project-root")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runLintPaths(projectport.PathLintOptions{
			Scope:       scope,
			ProjectRoot: projectRoot,
			Fix:         mustGetBoolFlag(cmd, "fix"),
			DryRun:      mustGetBoolFlag(cmd, "dry-run"),
			NoBackup:    mustGetBoolFlag(cmd, "no-backup"),
		}, jsonOutput)
	},
}

func init() {
	lintCmd.Flags().Bool("paths", false, "扫描技能内容中的本机绝对路径")
	lintCmd.Flags().String("project-root", "", "用于改写相对路径的项目根目录，默认从 .agents 位置推断")
	lintCmd.Flags().Bool("fix", false, "将可安全改写的本机路径替换为相对路径")
	lintCmd.Flags().Bool("dry-run", false, "演习模式，仅报告将改写的路径")
	lintCmd.Flags().Bool("no-backup", false, "修复前不创建备份文件")
}

func runLintPaths(opts projectport.PathLintOptions, jsonOutput bool) error {
	absScope, err := filepath.Abs(opts.Scope)
	if err != nil {
		return errors.Wrap(err, "解析scope失败")
	}
	opts.Scope = absScope

	if opts.ProjectRoot != "" {
		absProjectRoot, err := filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return errors.Wrap(err, "解析project-root失败")
		}
		opts.ProjectRoot = absProjectRoot
	}

	var report *projectport.PathLintReport
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.LintPaths(context.Background(), httpapibiz.PathLintRequest{Options: opts})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务审计技能路径失败")
		}
		report = data.Item
	} else {
		report, err = localRuntime.Project().LintPaths(opts)
		if err != nil && report == nil {
			return err
		}
	}

	if jsonOutput {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else {
		renderPathLintReport(report)
	}
	return err
}

func renderPathLintReport(report *projectport.PathLintReport) {
	if report == nil {
		printlnText("未返回路径审计报告")
		return
	}
	if report.DryRun {
		printlnText("🔎 lint --paths 演习模式，未修改文件")
	}
	printfText("扫描范围: %s\n", report.Scope)
	if report.ProjectRoot != "" {
		printfText("project-root: %s\n", report.ProjectRoot)
	}
	printfText("files:         %d\n", report.FilesScanned)
	printfText("findings:      %d\n", report.FindingCount)
	printfText("rewritten:     %d\n", report.Rewritten)
	printfText("manual-review: %d\n", report.ManualReview)
	for _, backup := range report.Backups {
		printfText("backup:        %s\n", backup)
	}
	for _, finding := range report.Findings {
		printfText("- [%s] %s:%d %s %s", finding.Status, finding.File, finding.Line, finding.Kind, finding.Value)
		if finding.Replacement != "" {
			printfText(" -> %s", finding.Replacement)
		}
		if finding.Reason != "" {
			printfText(" (%s)", finding.Reason)
		}
		printlnText()
	}
}
