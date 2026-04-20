package cli

import (
	"context"
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

var validateCmd = &cobra.Command{
	Use:   "validate [id]",
	Short: "验证本地新建技能的合规性",
	Long: `验证指定技能在项目本地工作区中的文件是否合规，主要用于 create 之后、feedback 之前的本地校验。

该命令检查项目工作区中的技能目录和 SKILL.md 内容，包括 YAML 语法、必需字段和基本文件结构。
使用 --links 可额外检查 SKILL.md 与技能目录内 Markdown 文件中的本地链接。
使用 --fix 可为缺失或不完整的 legacy SKILL.md 安全补齐 frontmatter，并在修改前创建备份。`,
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			return cobra.NoArgs(cmd, args)
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
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
		if opts.All {
			return runValidateAllWithOptions(opts)
		}
		return runValidateWithOptions(args[0], opts)
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
	validateCmd.Flags().Bool("json", false, "以JSON格式输出验证报告")
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
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}
	opts.ProjectPath = cwd
	if opts.ProjectRoot != "" {
		absProjectRoot, err := filepath.Abs(opts.ProjectRoot)
		if err != nil {
			return errors.Wrap(err, "解析project-root失败")
		}
		opts.ProjectRoot = absProjectRoot
	}

	var report *projectlifecycleservice.ValidateReport
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.ValidateProjectSkills(context.Background(), httpapibiz.ValidateProjectSkillsRequest{Options: opts})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务验证技能失败")
		}
		report = data.Item
	} else {
		if err := CheckInitDependency(); err != nil {
			return err
		}
		ctx, err := RequireInitAndWorkspace(cwd)
		if err != nil {
			return err
		}
		opts.ProjectPath = ctx.Cwd
		lifecycleSvc := projectlifecycleservice.New()
		report, err = lifecycleSvc.ValidateProjectSkills(opts)
		if err != nil && report == nil {
			return err
		}
	}

	if jsonOutput {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else {
		renderValidateReport(report)
	}
	if report != nil && report.Failed > 0 {
		return errors.NewWithCodef("runValidate", errors.ErrValidation, "%d 个技能验证失败", report.Failed)
	}
	return err
}

func renderValidateReport(report *projectlifecycleservice.ValidateReport) {
	if report == nil {
		fmt.Println("未返回验证报告")
		return
	}
	if report.Total == 0 {
		fmt.Println("ℹ️  当前项目未登记任何技能")
		return
	}
	if report.All {
		fmt.Printf("验证全部技能: %d\n", report.Total)
	} else {
		fmt.Printf("验证技能合规性: %s\n", report.SkillID)
	}
	fmt.Printf("项目路径: %s\n", report.ProjectPath)
	fmt.Printf("passed:   %d\n", report.Passed)
	fmt.Printf("failed:   %d\n", report.Failed)
	fmt.Printf("repaired: %d\n", report.Repaired)
	if report.Links {
		fmt.Printf("link issues: %d\n", report.LinkIssueCount)
	}

	for _, item := range report.Items {
		status := "PASS"
		if !item.Valid {
			status = "FAIL"
		}
		fmt.Printf("- [%s] %s\n", status, item.SkillID)
		if item.Repaired {
			fmt.Printf("  ✓ 已修复frontmatter，备份文件: %s\n", item.BackupPath)
		}
		for _, issue := range item.LinkIssues {
			fmt.Printf("  link [%s] %s:%d %s", issue.Status, issue.SourceFile, issue.Line, issue.Link)
			if issue.ResolvedPath != "" {
				fmt.Printf(" -> %s", issue.ResolvedPath)
			}
			if issue.Reason != "" {
				fmt.Printf(" (%s)", issue.Reason)
			}
			fmt.Println()
		}
		for _, itemErr := range item.Errors {
			fmt.Printf("  error: %s\n", itemErr)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	if report.Failed == 0 {
		fmt.Println("✅ 验证通过！")
		fmt.Println("本地技能合规性验证完成")
	} else {
		fmt.Printf("❌ %d 个技能验证失败\n", report.Failed)
	}
	fmt.Println(strings.Repeat("=", 50))
}
