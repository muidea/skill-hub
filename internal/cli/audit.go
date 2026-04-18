package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var auditCmd = &cobra.Command{
	Use:   "audit [scope]",
	Short: "生成技能刷新审计报告",
	Long: `聚合项目技能数量、登记状态、validate --links、status、dedupe、lint --paths 和默认仓库推送状态，生成 Markdown 或 JSON 审计报告。

默认 scope 为 .agents/skills，默认输出 Markdown 到标准输出。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scope := ".agents/skills"
		if len(args) > 0 {
			scope = args[0]
		}
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		canonical, _ := cmd.Flags().GetString("canonical")
		projectRoot, _ := cmd.Flags().GetString("project-root")
		return runAudit(projectlifecycleservice.AuditOptions{
			Scope:       scope,
			Canonical:   canonical,
			ProjectRoot: projectRoot,
		}, format, output)
	},
}

func init() {
	auditCmd.Flags().String("output", "", "报告输出文件，未指定时输出到stdout")
	auditCmd.Flags().String("format", "markdown", "报告格式: markdown, json")
	auditCmd.Flags().String("canonical", "", "用于重复检测的canonical技能目录，默认不指定")
	auditCmd.Flags().String("project-root", "", "用于路径和链接审计的项目根目录，默认当前项目")
}

func runAudit(opts projectlifecycleservice.AuditOptions, format, output string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}
	opts.ProjectPath = cwd
	if opts.Scope != "" {
		absScope, err := filepath.Abs(opts.Scope)
		if err != nil {
			return errors.Wrap(err, "解析scope失败")
		}
		opts.Scope = absScope
	}
	if opts.Canonical != "" {
		absCanonical, err := filepath.Abs(opts.Canonical)
		if err != nil {
			return errors.Wrap(err, "解析canonical失败")
		}
		opts.Canonical = absCanonical
	}
	if opts.ProjectRoot != "" {
		absProjectRoot, err := filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return errors.Wrap(err, "解析project-root失败")
		}
		opts.ProjectRoot = absProjectRoot
	}
	if output != "" {
		absOutput, err := filepath.Abs(output)
		if err != nil {
			return errors.Wrap(err, "解析output失败")
		}
		output = absOutput
	}

	var report *projectlifecycleservice.AuditReport
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.AuditProjectSkills(context.Background(), httpapibiz.AuditProjectSkillsRequest{Options: opts})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务生成审计报告失败")
		}
		report = data.Item
	} else {
		if err := CheckInitDependency(); err != nil {
			return err
		}
		ctx, err := RequireInitAndWorkspace(cwd, "")
		if err != nil {
			return err
		}
		opts.ProjectPath = ctx.Cwd
		lifecycleSvc := projectlifecycleservice.New()
		report, err = lifecycleSvc.Audit(opts)
		if err != nil && report == nil {
			return err
		}
	}

	payload, err := renderAuditPayload(report, format)
	if err != nil {
		return err
	}
	if output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			return errors.WrapWithCode(err, "runAudit", errors.ErrFileOperation, "创建报告目录失败")
		}
		if err := os.WriteFile(output, payload, 0644); err != nil {
			return errors.WrapWithCode(err, "runAudit", errors.ErrFileOperation, "写入审计报告失败")
		}
		fmt.Printf("审计报告已写入: %s\n", output)
	} else {
		fmt.Print(string(payload))
	}
	if report != nil && len(report.Errors) > 0 {
		return errors.NewWithCodef("runAudit", errors.ErrValidation, "审计完成但发现 %d 个问题", len(report.Errors))
	}
	return nil
}

func renderAuditPayload(report *projectlifecycleservice.AuditReport, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return []byte(renderAuditMarkdown(report)), nil
	case "json":
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		return nil, errors.NewWithCodef("runAudit", errors.ErrInvalidInput, "无效的报告格式: %s", format)
	}
}

func renderAuditMarkdown(report *projectlifecycleservice.AuditReport) string {
	if report == nil {
		return "# Skill Hub Audit Report\n\n未返回审计报告。\n"
	}
	var out strings.Builder
	out.WriteString("# Skill Hub Audit Report\n\n")
	out.WriteString(fmt.Sprintf("- Generated At: `%s`\n", report.GeneratedAt))
	out.WriteString(fmt.Sprintf("- Project: `%s`\n", report.ProjectPath))
	out.WriteString(fmt.Sprintf("- Scope: `%s`\n", report.Scope))
	if report.Canonical != "" {
		out.WriteString(fmt.Sprintf("- Canonical: `%s`\n", report.Canonical))
	}
	if report.DefaultRepository != "" {
		out.WriteString(fmt.Sprintf("- Default Repository: `%s`\n", report.DefaultRepository))
	}
	out.WriteString("\n## Summary\n\n")
	out.WriteString("| Metric | Value |\n|---|---:|\n")
	out.WriteString(fmt.Sprintf("| Target Skills | %d |\n", report.TargetSkillCount))
	out.WriteString(fmt.Sprintf("| Registered | %d |\n", report.RegisteredCount))
	out.WriteString(fmt.Sprintf("| Unregistered | %d |\n", report.UnregisteredCount))
	if report.Validation != nil {
		out.WriteString(fmt.Sprintf("| Validation Passed | %d |\n", report.Validation.Passed))
		out.WriteString(fmt.Sprintf("| Validation Failed | %d |\n", report.Validation.Failed))
	}
	out.WriteString(fmt.Sprintf("| Duplicate Conflicts | %d |\n", report.DuplicateConflicts))
	out.WriteString(fmt.Sprintf("| Absolute Path Hits | %d |\n", report.AbsolutePathHits))
	out.WriteString(fmt.Sprintf("| Link Issues | %d |\n", report.LinkIssueCount))
	out.WriteString(fmt.Sprintf("| Status Synced | %d |\n", report.FeedbackSummary.Synced))
	out.WriteString(fmt.Sprintf("| Status Modified | %d |\n", report.FeedbackSummary.Modified))
	out.WriteString(fmt.Sprintf("| Status Outdated | %d |\n", report.FeedbackSummary.Outdated))
	out.WriteString(fmt.Sprintf("| Status Missing | %d |\n", report.FeedbackSummary.Missing))

	out.WriteString("\n## Remote Push\n\n")
	out.WriteString(fmt.Sprintf("- Status: `%s`\n", report.RemotePush.Status))
	out.WriteString(fmt.Sprintf("- Performed: `%t`\n", report.RemotePush.Performed))
	out.WriteString(fmt.Sprintf("- Dirty: `%t`\n", report.RemotePush.Dirty))
	out.WriteString(fmt.Sprintf("- Unpushed Commits: `%d`\n", report.RemotePush.UnpushedCommits))
	if report.RemotePush.Message != "" {
		out.WriteString(fmt.Sprintf("- Message: %s\n", report.RemotePush.Message))
	}

	if len(report.UnregisteredSkills) > 0 {
		out.WriteString("\n## Unregistered Skills\n\n")
		for _, skillID := range report.UnregisteredSkills {
			out.WriteString(fmt.Sprintf("- `%s`\n", skillID))
		}
	}
	if report.Validation != nil && len(report.Validation.Failures) > 0 {
		out.WriteString("\n## Validation Failures\n\n")
		for _, failure := range report.Validation.Failures {
			out.WriteString(fmt.Sprintf("- `%s`: %s\n", failure.SkillID, failure.Error))
		}
	}
	if report.Validation != nil && len(report.Validation.LinkIssues) > 0 {
		out.WriteString("\n## Link Issues\n\n")
		for _, issue := range report.Validation.LinkIssues {
			out.WriteString(fmt.Sprintf("- `%s` %s:%d `%s` -> `%s`\n", issue.SkillID, issue.SourceFile, issue.Line, issue.Link, issue.ResolvedPath))
		}
	}
	if report.PathLint != nil && len(report.PathLint.Findings) > 0 {
		out.WriteString("\n## Absolute Path Findings\n\n")
		for _, finding := range report.PathLint.Findings {
			out.WriteString(fmt.Sprintf("- `%s` %s:%d `%s`", finding.Status, finding.File, finding.Line, finding.Value))
			if finding.Replacement != "" {
				out.WriteString(fmt.Sprintf(" -> `%s`", finding.Replacement))
			}
			out.WriteString("\n")
		}
	}
	if report.DuplicateReport != nil && len(report.DuplicateReport.Groups) > 0 {
		out.WriteString("\n## Duplicate Groups\n\n")
		for _, group := range report.DuplicateReport.Groups {
			status := "identical"
			if group.ContentDiffers {
				status = "conflict"
			}
			out.WriteString(fmt.Sprintf("- `%s` %s locations=%d\n", group.SkillID, status, len(group.Locations)))
		}
	}
	if len(report.Errors) > 0 {
		out.WriteString("\n## Audit Errors\n\n")
		for _, item := range report.Errors {
			out.WriteString(fmt.Sprintf("- `%s`: %s\n", item.Step, item.Error))
		}
	}
	return out.String()
}
