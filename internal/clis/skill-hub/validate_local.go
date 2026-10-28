package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/pkg/projectlifecycleport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var validateCmd = &cobra.Command{
	Use:   "validate <id> | --all | --pattern <id-or-glob>...",
	Short: "验证本地新建技能的合规性",
	Long: `验证指定技能在项目本地工作区中的文件是否合规，主要用于 create 之后、feedback 之前的本地校验。

该命令检查项目工作区中的技能目录和 SKILL.md 内容，包括 YAML 语法、必需字段和基本文件结构。
使用 --links 可额外检查 SKILL.md 与技能目录内 Markdown 文件中的本地链接。
使用 --fix 可为缺失或不完整的 legacy SKILL.md 安全补齐 frontmatter，并在修改前创建备份。

输入形式（必须二选一）：
  --all                 验证当前项目状态中登记的所有技能
  --pattern '<glob>'    验证匹配 pattern 的技能（可重复）

--pattern 语法（类 glob，匹配当前项目登记的技能 ID；* 可跨 /）：
  *        匹配零或多个任意字符
  ?        匹配恰好一个任意字符
  **       匹配全部`,
	Args:              validatePatternOrExactID,
	ValidArgsFunction: completeEnabledSkillIDsForCwd,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := validateCLIOptions{
			Fix:         mustGetBoolFlag(cmd, "fix"),
			All:         mustGetBoolFlag(cmd, "all"),
			Links:       mustGetBoolFlag(cmd, "links"),
			CheckRemote: mustGetBoolFlag(cmd, "check-remote"),
			JSON:        mustGetBoolFlag(cmd, "json"),
		}
		opts.ProjectRoot, _ = cmd.Flags().GetString("project-root")
		patterns, err := readPatternOrExactID(cmd, args)
		if err != nil {
			return err
		}
		if opts.All {
			if len(patterns) > 0 {
				return errors.NewWithCodef("validate", errors.ErrInvalidInput, "--all and --pattern are mutually exclusive")
			}
			return runValidateAllWithOptions(opts)
		}
		if len(patterns) == 0 {
			return errors.NewWithCodef("validate", errors.ErrInvalidInput, "specify --all or --pattern '<id-or-glob>'")
		}
		return runValidateByPatterns(patterns, opts)
	},
}

type validateCLIOptions struct {
	Fix         bool
	All         bool
	Links       bool
	CheckRemote bool
	ProjectRoot string
	JSON        bool
}

func init() {
	validateCmd.Flags().Bool("fix", false, "修复缺失或不完整的SKILL.md frontmatter，修改前自动备份")
	validateCmd.Flags().Bool("all", false, "验证当前项目状态中登记的所有技能")
	validateCmd.Flags().Bool("links", false, "检查SKILL.md和技能目录内Markdown文件的本地链接")
	validateCmd.Flags().Bool("check-remote", false, "同时检查HTTP/HTTPS远端链接")
	validateCmd.Flags().String("project-root", "", "解析项目相对链接的根目录，默认使用当前项目目录或metadata.project_root")

	validateCmd.Flags().StringArray("pattern", nil, "技能 ID 通配符（可重复）。请引用通配符避免 shell 展开。")
}

func runValidateWithOptions(skillID string, opts validateCLIOptions) error {
	opts.All = false
	return runValidateRequest(projectlifecycleservice.ValidateOptions{
		SkillID:     skillID,
		Fix:         opts.Fix,
		Links:       opts.Links,
		CheckRemote: opts.CheckRemote,
		ProjectRoot: opts.ProjectRoot,
	}, opts.JSON)
}

func runValidateAllWithOptions(opts validateCLIOptions) error {
	opts.All = true
	return runValidateRequest(projectlifecycleservice.ValidateOptions{
		All:         true,
		Fix:         opts.Fix,
		Links:       opts.Links,
		CheckRemote: opts.CheckRemote,
		ProjectRoot: opts.ProjectRoot,
	}, opts.JSON)
}

func runValidateRequest(opts projectlifecycleservice.ValidateOptions, jsonOutput bool) error {
	report, err := runValidateSilent(opts)
	if jsonOutput {
		if writeErr := writeJSON(report); writeErr != nil {
			return writeErr
		}
	} else {
		renderValidateReport(report)
	}
	if report != nil && report.Failed > 0 {
		return errors.NewWithCodef("runValidate", errors.ErrValidation, "%d 个技能验证失败", report.Failed)
	}
	return err
}

// runValidateSilent runs a single validation pass and returns the report
// without rendering it. Used both by runValidateRequest and the pattern
// batch path, which aggregates reports from many skill IDs and renders once.
func runValidateSilent(opts projectlifecycleservice.ValidateOptions) (*projectlifecycleservice.ValidateReport, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, utils.GetCwdErr(err)
	}
	opts.ProjectPath = cwd
	if opts.ProjectRoot != "" {
		absProjectRoot, err := filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return nil, errors.Wrap(err, "解析project-root失败")
		}
		opts.ProjectRoot = absProjectRoot
	}

	var report *projectlifecycleservice.ValidateReport
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.ValidateProjectSkills(context.Background(), httpapibiz.ValidateProjectSkillsRequest{Options: opts})
		if serviceErr != nil {
			return nil, errors.Wrap(serviceErr, "通过服务验证技能失败")
		}
		report = data.Item
	} else {
		if err := CheckInitDependency(); err != nil {
			return nil, err
		}
		ctx, err := RequireInitAndWorkspace(cwd)
		if err != nil {
			return nil, err
		}
		opts.ProjectPath = ctx.Cwd
		report, err = localRuntime.Project().Validate(opts)
		if err != nil && report == nil {
			return nil, err
		}
	}
	return report, err
}

// runValidateByPatterns resolves the patterns against the cross-repo skill
// set and runs the silent validation path for each matched ID, then renders
// a single aggregate report. Failures on individual skills are not fatal;
// the report surfaces them.
//
// When every pattern is a literal ID (no wildcards / **), preserve the
// pre-v0.8.13 single-ID semantics: each ID is validated directly against
// the project workspace, even if the skill is not yet in any repo. This
// is the only way the common "create + validate" workflow continues to
// work after the --pattern migration.
func runValidateByPatterns(patterns []string, opts validateCLIOptions) error {
	matchers, err := compilePatterns(patterns)
	if err != nil {
		return err
	}
	allLiteral := len(matchers) > 0
	for _, m := range matchers {
		if !m.IsLiteral() {
			allLiteral = false
			break
		}
	}
	if allLiteral {
		seen := make(map[string]struct{}, len(patterns))
		ids := make([]string, 0, len(patterns))
		for _, p := range patterns {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			ids = append(ids, p)
		}
		return runValidateByLiteralIDs(ids, opts)
	}
	ids, err := resolveProjectSkillIDsByPatterns(patterns)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		if opts.JSON {
			return writeJSON(map[string]any{"total": 0, "passed": 0, "failed": 0, "repaired": 0, "items": []any{}})
		}
		return nil
	}

	aggregate := &projectlifecycleservice.ValidateReport{
		SkillID:     strings.Join(patterns, ","),
		Fix:         opts.Fix,
		Links:       opts.Links,
		CheckRemote: opts.CheckRemote,
		ProjectRoot: opts.ProjectRoot,
	}
	var firstErr error
	for _, id := range ids {
		silentOpts := projectlifecycleservice.ValidateOptions{
			SkillID:     id,
			Fix:         opts.Fix,
			Links:       opts.Links,
			CheckRemote: opts.CheckRemote,
			ProjectRoot: opts.ProjectRoot,
		}
		report, rErr := runValidateSilent(silentOpts)
		if rErr != nil && firstErr == nil {
			firstErr = rErr
		}
		if report == nil {
			continue
		}
		if aggregate.ProjectPath == "" {
			aggregate.ProjectPath = report.ProjectPath
		}
		aggregate.Total += report.Total
		aggregate.Passed += report.Passed
		aggregate.Failed += report.Failed
		aggregate.Repaired += report.Repaired
		aggregate.LinkIssueCount += report.LinkIssueCount
		aggregate.Items = append(aggregate.Items, report.Items...)
		aggregate.LinkIssues = append(aggregate.LinkIssues, report.LinkIssues...)
		aggregate.Failures = append(aggregate.Failures, report.Failures...)
	}

	if opts.JSON {
		if writeErr := writeJSON(aggregate); writeErr != nil {
			return writeErr
		}
	} else {
		renderValidateReport(aggregate)
	}
	if aggregate.Failed > 0 {
		return errors.NewWithCodef("runValidateByPatterns", errors.ErrValidation, "%d 个技能验证失败", aggregate.Failed)
	}
	return firstErr
}

// runValidateByLiteralIDs is the same as runValidateByPatterns but takes
// already-resolved literal IDs (no pattern resolution against the repo).
// Used by the single-literal case to validate skills that exist only in
// the project workspace.
func runValidateByLiteralIDs(ids []string, opts validateCLIOptions) error {
	aggregate := &projectlifecycleservice.ValidateReport{
		SkillID:     strings.Join(ids, ","),
		Fix:         opts.Fix,
		Links:       opts.Links,
		CheckRemote: opts.CheckRemote,
		ProjectRoot: opts.ProjectRoot,
	}
	var firstErr error
	for _, id := range ids {
		silentOpts := projectlifecycleservice.ValidateOptions{
			SkillID:     id,
			Fix:         opts.Fix,
			Links:       opts.Links,
			CheckRemote: opts.CheckRemote,
			ProjectRoot: opts.ProjectRoot,
		}
		report, rErr := runValidateSilent(silentOpts)
		if rErr != nil && firstErr == nil {
			firstErr = rErr
		}
		if report == nil {
			continue
		}
		if aggregate.ProjectPath == "" {
			aggregate.ProjectPath = report.ProjectPath
		}
		aggregate.Total += report.Total
		aggregate.Passed += report.Passed
		aggregate.Failed += report.Failed
		aggregate.Repaired += report.Repaired
		aggregate.LinkIssueCount += report.LinkIssueCount
		aggregate.Items = append(aggregate.Items, report.Items...)
		aggregate.LinkIssues = append(aggregate.LinkIssues, report.LinkIssues...)
		aggregate.Failures = append(aggregate.Failures, report.Failures...)
	}
	if opts.JSON {
		if writeErr := writeJSON(aggregate); writeErr != nil {
			return writeErr
		}
	} else {
		renderValidateReport(aggregate)
	}
	if aggregate.Failed > 0 {
		return errors.NewWithCodef("runValidateByLiteralIDs", errors.ErrValidation, "%d 个技能验证失败", aggregate.Failed)
	}
	return firstErr
}

func renderValidateReport(report *projectlifecycleservice.ValidateReport) {
	if report == nil {
		printlnText("未返回验证报告")
		return
	}
	if report.Total == 0 {
		printlnText("ℹ️  当前项目未登记任何技能")
		return
	}
	if report.All {
		printfText("验证全部技能: %d\n", report.Total)
	} else {
		printfText("验证技能合规性: %s\n", report.SkillID)
	}
	printfText("项目路径: %s\n", report.ProjectPath)
	printfText("passed:   %d\n", report.Passed)
	printfText("failed:   %d\n", report.Failed)
	printfText("repaired: %d\n", report.Repaired)
	if report.Links {
		printfText("link issues: %d\n", report.LinkIssueCount)
	}

	for _, item := range report.Items {
		status := "PASS"
		if !item.Valid {
			status = "FAIL"
		}
		printfText("- [%s] %s\n", status, item.SkillID)
		if item.Repaired {
			printfText("  ✓ 已修复frontmatter，备份文件: %s\n", item.BackupPath)
		}
		for _, issue := range item.LinkIssues {
			printfText("  link [%s] %s:%d %s", issue.Status, issue.SourceFile, issue.Line, issue.Link)
			if issue.ResolvedPath != "" {
				printfText(" -> %s", issue.ResolvedPath)
			}
			if issue.Reason != "" {
				printfText(" (%s)", issue.Reason)
			}
			printlnText()
		}
		for _, itemErr := range item.Errors {
			printfText("  error: %s\n", itemErr)
		}
	}

	printlnText("\n" + strings.Repeat("=", 50))
	if report.Failed == 0 {
		printlnText("✅ 验证通过！")
		printlnText("本地技能合规性验证完成")
	} else {
		printfText("❌ %d 个技能验证失败\n", report.Failed)
	}
	printlnText(strings.Repeat("=", 50))
}
