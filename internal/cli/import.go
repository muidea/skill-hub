package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectlifecycleservice "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var importCmd = &cobra.Command{
	Use:   "import <skills-dir>",
	Short: "批量导入并可选归档已有技能",
	Long: `扫描目录中的 .agents/skills/*/SKILL.md 风格技能，批量登记、验证，并可选归档到默认仓库。

默认会继续处理后续技能并在最后汇总失败项。使用 --fail-fast 可在首个失败项停止。
使用 --dry-run 可预览将要执行的登记、修复和归档动作。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := projectlifecycleservice.ImportOptions{
			Archive:        mustGetBoolFlag(cmd, "archive"),
			Force:          mustGetBoolFlag(cmd, "force"),
			DryRun:         mustGetBoolFlag(cmd, "dry-run"),
			FailFast:       mustGetBoolFlag(cmd, "fail-fast"),
			FixFrontmatter: mustGetBoolFlag(cmd, "fix-frontmatter"),
		}
		return runImport(args[0], opts)
	},
}

func init() {
	importCmd.Flags().Bool("archive", false, "验证通过后归档到默认仓库")
	importCmd.Flags().Bool("force", false, "批量流程中跳过交互确认（当前导入流程默认不覆盖源技能内容）")
	importCmd.Flags().Bool("dry-run", false, "演习模式，仅输出将要执行的操作")
	importCmd.Flags().Bool("fail-fast", false, "遇到首个失败技能时立即停止")
	importCmd.Flags().Bool("fix-frontmatter", false, "导入前修复缺失或不完整的SKILL.md frontmatter")
}

func mustGetBoolFlag(cmd *cobra.Command, name string) bool {
	value, _ := cmd.Flags().GetBool(name)
	return value
}

func runImport(skillsDir string, opts projectlifecycleservice.ImportOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	if opts.DryRun {
		fmt.Println("🔎 import 演习模式，不会修改项目状态、技能文件或仓库")
	}
	if opts.Force {
		fmt.Println("ℹ️  force 模式已启用：批量流程将不进行交互确认")
	}

	var summary *projectlifecycleservice.ImportSummary
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.ImportSkills(context.Background(), httpapibiz.ImportSkillsRequest{
			ProjectPath: cwd,
			SkillsDir:   skillsDir,
			Options:     opts,
		})
		if err != nil {
			return errors.Wrap(err, "通过服务导入技能失败")
		}
		summary = data.Item
	} else {
		if err := CheckInitDependency(); err != nil {
			return err
		}
		ctx, err := RequireInitAndWorkspace(cwd)
		if err != nil {
			return err
		}
		lifecycleSvc := projectlifecycleservice.New()
		summary, err = lifecycleSvc.Import(ctx.Cwd, skillsDir, opts)
		if err != nil && summary == nil {
			return err
		}
	}

	renderImportSummary(summary)
	if summary != nil && summary.Failed > 0 {
		return errors.NewWithCodef("runImport", errors.ErrValidation, "%d 个技能导入失败", summary.Failed)
	}
	return nil
}

func renderImportSummary(summary *projectlifecycleservice.ImportSummary) {
	if summary == nil {
		fmt.Println("\n=== import summary ===")
		fmt.Println("未返回导入摘要")
		return
	}

	fmt.Printf("发现 %d 个技能，目录: %s\n", summary.Discovered, summary.SkillsDir)
	fmt.Println("\n=== import summary ===")
	fmt.Printf("discovered: %d\n", summary.Discovered)
	fmt.Printf("registered: %d\n", summary.Registered)
	fmt.Printf("valid:      %d\n", summary.Valid)
	fmt.Printf("archived:   %d\n", summary.Archived)
	fmt.Printf("unchanged:  %d\n", summary.Unchanged)
	fmt.Printf("failed:     %d\n", summary.Failed)
	if len(summary.Failures) == 0 {
		return
	}
	fmt.Println("\n失败项:")
	for _, failure := range summary.Failures {
		fmt.Printf("- id=%s command=%s path=%s error=%s\n", failure.SkillID, failure.Command, failure.Path, failure.Error)
	}
}
