package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
	"github.com/muidea/skill-hub/pkg/errors"
)

var dedupeCmd = &cobra.Command{
	Use:   "dedupe <scope>",
	Short: "检测嵌套项目中的重复技能",
	Long: `扫描scope下所有 .agents/skills/<id>/SKILL.md，按技能ID分组并报告重复、内容hash、更新时间、canonical来源和冲突状态。

该命令默认只报告，不修改文件。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		canonical, _ := cmd.Flags().GetString("canonical")
		strategy, _ := cmd.Flags().GetString("strategy")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		return runDedupe(args[0], projectlifecycleservice.DedupeOptions{
			Canonical: canonical,
			Strategy:  strategy,
			Report:    true,
		}, jsonOutput)
	},
}

func init() {
	dedupeCmd.Flags().String("canonical", "", "canonical技能目录，例如 .agents/skills")
	dedupeCmd.Flags().String("strategy", "newest", "canonical选择策略: newest, canonical, fail-on-conflict")
	dedupeCmd.Flags().Bool("json", false, "以JSON格式输出重复检测报告")
}

func runDedupe(scope string, opts projectlifecycleservice.DedupeOptions, jsonOutput bool) error {
	absScope, err := filepath.Abs(scope)
	if err != nil {
		return errors.Wrap(err, "解析scope失败")
	}

	var report *projectlifecycleservice.DuplicateReport
	if client, ok := hubClientIfAvailable(); ok {
		data, serviceErr := client.DedupeSkills(context.Background(), httpapibiz.DedupeRequest{
			Scope:   absScope,
			Options: opts,
		})
		if serviceErr != nil {
			return errors.Wrap(serviceErr, "通过服务检测重复技能失败")
		}
		report = data.Item
	} else {
		lifecycleSvc := projectlifecycleservice.New()
		report, err = lifecycleSvc.Dedupe(absScope, opts)
		if err != nil && report == nil {
			return err
		}
	}

	if jsonOutput {
		if err := writeJSON(report); err != nil {
			return err
		}
	} else {
		renderDedupeReport(report)
	}
	if err != nil || (report != nil && opts.Strategy == "fail-on-conflict" && report.Conflicts > 0) {
		return errors.NewWithCodef("runDedupe", errors.ErrValidation, "%d 个重复技能存在内容冲突", report.Conflicts)
	}
	return nil
}

func renderDedupeReport(report *projectlifecycleservice.DuplicateReport) {
	if report == nil {
		fmt.Println("未返回重复检测报告")
		return
	}
	fmt.Printf("扫描范围: %s\n", report.Scope)
	if report.Canonical != "" {
		fmt.Printf("canonical: %s\n", report.Canonical)
	}
	fmt.Printf("策略: %s\n", report.Strategy)
	fmt.Printf("重复技能组: %d\n", report.SkillCount)
	fmt.Printf("冲突数: %d\n", report.Conflicts)
	if len(report.Groups) == 0 {
		fmt.Println("未发现重复技能")
		return
	}
	for _, group := range report.Groups {
		status := "identical"
		if group.ContentDiffers {
			status = "conflict"
		}
		fmt.Printf("\n== %s [%s] ==\n", group.SkillID, status)
		if group.CanonicalSource != "" {
			fmt.Printf("canonical_source: %s\n", group.CanonicalSource)
		}
		for _, loc := range group.Locations {
			marker := ""
			if loc.IsCanonical {
				marker = " canonical"
			}
			fmt.Fprintf(os.Stdout, "- %s%s\n  hash: %s\n  modified: %s\n", loc.SkillDir, marker, loc.Hash, loc.ModifiedTime)
		}
	}
}
