package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/globalport"
	"github.com/muidea/skill-hub/internal/pkg/projectport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var applyCmd = &cobra.Command{
	Use:   "apply [id] | --pattern <id-or-glob>...",
	Short: "应用技能到项目",
	Long: `根据 state.json 中的启用记录，将技能物理分发到当前项目的标准 .agents/skills 目录。

不带 --pattern 时应用所有已启用技能；带 --pattern 时（类 glob，* 可跨 /，
匹配技能 ID 字段）逐个应用匹配 pattern 的技能，单个失败不影响其它技能的继续处理。
请引用带通配符的 pattern，避免 shell 在 skill-hub 启动前展开。`,
	Args: validatePatternOrExactID,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		global, _ := cmd.Flags().GetBool("global")
		patterns, err := readPatternOrExactID(cmd, args)
		if err != nil {
			return err
		}
		if jsonMode() && len(args) == 1 {
			if global {
				return runApplyGlobal(args[0], nil, dryRun, force)
			}
			return runApply(args[0], dryRun, force)
		}
		if len(patterns) > 0 {
			beginJSONBatch()
		}
		if global {
			return runApplyGlobalByPatterns(patterns, nil, dryRun, force)
		}
		return runApplyByPatterns(patterns, dryRun, force)
	},
}

func init() {
	applyCmd.Flags().Bool("dry-run", false, "演习模式，仅显示将要执行的变更，不实际修改文件")
	applyCmd.Flags().Bool("force", false, "强制应用，即使检测到冲突也继续执行")
	applyCmd.Flags().Bool("global", false, "应用本机全局启用的技能")
	applyCmd.Flags().StringArray("pattern", nil, "技能 ID 通配符（可重复）。请引用通配符避免 shell 展开。")
}

func runApplyGlobal(skillID string, agents []string, dryRun bool, force bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		return runApplyGlobalViaService(client, skillID, agents, dryRun, force)
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}
	result, err := localRuntime.Global().Apply(skillID, agents, dryRun, force)
	if err != nil {
		return errors.Wrap(err, "应用全局技能失败")
	}
	if jsonMode() {
		return writeGlobalApplyJSON(result)
	}
	renderGlobalApplyResult(result)
	return nil
}

type serviceGlobalApplyClient interface {
	ApplyGlobal(ctx context.Context, req httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error)
}

func runApplyGlobalViaService(client serviceGlobalApplyClient, skillID string, agents []string, dryRun bool, force bool) error {
	resp, err := client.ApplyGlobal(context.Background(), httpapibiz.ApplyGlobalRequest{
		SkillID: skillID,
		Agents:  agents,
		DryRun:  dryRun,
		Force:   force,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务应用全局技能失败")
	}
	if jsonMode() {
		return writeGlobalApplyJSON(resp.Item)
	}
	renderGlobalApplyResult(resp.Item)
	return nil
}

func runApply(skillID string, dryRun bool, force bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		return runApplyViaService(client, skillID, dryRun, force)
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	projectState := ctx.ProjectState
	if projectState == nil {
		return errors.NewWithCode("runApply", errors.ErrProjectInvalid, "项目状态无效")
	}

	result, err := localRuntime.Project().Apply(ctx.Cwd, skillID, dryRun, force)
	if err != nil {
		return errors.Wrap(err, "应用技能失败")
	}
	if jsonMode() {
		return writeApplyJSON(result)
	}
	renderApplyResult(result)

	return nil
}

type serviceApplyClient interface {
	ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
}

func runApplyViaService(client serviceApplyClient, skillID string, dryRun bool, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	resp, err := client.ApplyProject(context.Background(), httpapibiz.ApplyProjectRequest{
		ProjectPath: cwd,
		SkillID:     skillID,
		DryRun:      dryRun,
		Force:       force,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务应用技能失败")
	}

	if jsonMode() {
		return writeApplyJSON(resp.Item)
	}
	renderApplyResult(resp.Item)
	return nil
}

// runApplyByPatterns dispatches based on --pattern: not set reuses the
// existing "all enabled skills" path; set delegates to runApplyWithPatterns
// which resolves and applies each matched skill.
func runApplyByPatterns(patterns []string, dryRun bool, force bool) error {
	if len(patterns) == 0 {
		return runApply("", dryRun, force)
	}
	return runApplyWithPatterns(patterns, dryRun, force)
}

func runApplyWithPatterns(patterns []string, dryRun bool, force bool) error {
	ids, err := resolveProjectSkillIDsByPatterns(patterns)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	var failed []string
	for _, id := range ids {
		before := jsonResultCount()
		if err := runApply(id, dryRun, force); err != nil {
			if before == jsonResultCount() {
				recordBatchError(id, err)
			}
			fmt.Fprintf(os.Stderr, "❌ 应用技能 '%s' 失败: %v\n", id, err)
			failed = append(failed, id)
			continue
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== 批量应用摘要 ===\n成功: %d, 失败: %d\n失败列表: %s\n",
			len(ids)-len(failed), len(failed), strings.Join(failed, ", "))
		return errors.NewWithCodef("runApplyWithPatterns", errors.ErrSystem, "批量应用存在失败: %d", len(failed))
	}
	return nil
}

func renderApplyResult(result *projectport.ApplyResult) {
	printlnText("正在应用技能到项目...")
	if result == nil {
		printlnText("ℹ️  未返回应用结果")
		return
	}

	printfText("项目路径: %s\n", result.ProjectPath)

	if len(result.Items) == 0 {
		printlnText("ℹ️  当前项目未启用任何技能")
		printlnText("使用 'skill-hub use <skill-id>' 启用技能")
		return
	}

	printfText("启用技能数: %d\n", len(result.Items))
	if result.DryRun {
		printlnText("\n=== 演习模式 (dry-run) ===")
		printlnText("将显示将要执行的变更，不实际修改文件")
	}

	for _, item := range result.Items {
		printfText("应用技能: %s\n", item.SkillID)
		switch item.Status {
		case "planned":
			printlnText("  [演习] 将应用技能到标准项目技能目录")
			printfText("  变量数量: %d\n", item.Variables)
		case "applied":
			printfText("✓ 成功应用技能: %s\n", item.SkillID)
		case "error":
			printfText("⚠️  应用技能失败: %s: %s\n", item.SkillID, item.Message)
		default:
			printfText("ℹ️  状态: %s\n", item.Status)
		}
	}

	if result.DryRun {
		printlnText("\n✅ 演习完成，未实际修改文件")
	} else {
		printlnText("\n✅ 所有技能应用完成")
	}
}

// runApplyGlobalByPatterns mirrors runApplyByPatterns for the --global path.
func runApplyGlobalByPatterns(patterns []string, agents []string, dryRun bool, force bool) error {
	if len(patterns) == 0 {
		return runApplyGlobal("", agents, dryRun, force)
	}
	ids, err := resolveGlobalSkillIDsByPatterns(patterns, agents)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	var failed []string
	for _, id := range ids {
		before := jsonResultCount()
		if err := runApplyGlobal(id, agents, dryRun, force); err != nil {
			if before == jsonResultCount() {
				recordBatchError(id, err)
			}
			fmt.Fprintf(os.Stderr, "❌ 全局应用技能 '%s' 失败: %v\n", id, err)
			failed = append(failed, id)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== 全局批量应用摘要 ===\n失败列表: %s\n", strings.Join(failed, ", "))
		return errors.NewWithCodef("runApplyGlobalByPatterns", errors.ErrSystem, "批量全局应用存在失败: %d", len(failed))
	}
	return nil
}

func renderGlobalApplyResult(result *globalport.ApplyResult) {
	printlnText("正在刷新本机全局技能...")
	if result == nil {
		printlnText("ℹ️  未返回应用结果")
		return
	}
	printfText("全局镜像目录: %s\n", result.GlobalPath)
	if result.DryRun {
		printlnText("\n=== 演习模式 (dry-run) ===")
	}
	if len(result.Items) == 0 {
		printlnText("ℹ️  当前未启用任何全局技能")
		printlnText("使用 'skill-hub use <skill-id> --global' 启用全局技能")
		return
	}
	for _, item := range result.Items {
		target := item.Agent
		if target == "" {
			target = "unknown"
		}
		switch item.Status {
		case globalport.StatusPlanned:
			printfText("[计划] %s -> %s: %s\n", item.SkillID, target, item.TargetPath)
		case globalport.StatusApplied:
			printfText("✓ 已刷新 %s -> %s\n", item.SkillID, target)
		case globalport.StatusConflict:
			printfText("⚠️  冲突 %s -> %s: %s\n", item.SkillID, target, item.Message)
		case globalport.StatusError:
			printfText("❌ 失败 %s -> %s: %s\n", item.SkillID, target, item.Message)
		default:
			printfText("ℹ️  %s -> %s: %s %s\n", item.SkillID, target, item.Status, item.Message)
		}
	}
	if result.DryRun {
		printlnText("\n✅ 演习完成，未实际修改文件")
	} else {
		printlnText("\n✅ 全局技能刷新完成")
	}
}

func writeApplyJSON(result *projectport.ApplyResult) error {
	if result == nil {
		return errors.NewWithCode("apply", errors.ErrSystem, "未返回应用结果")
	}
	if err := writeJSON(result); err != nil {
		return err
	}
	for _, item := range result.Items {
		if item.Status == "error" {
			return errors.NewWithCode("apply", errors.ErrFileOperation, "部分技能应用失败，详见 items")
		}
	}
	return nil
}

func writeGlobalApplyJSON(result *globalport.ApplyResult) error {
	if result == nil {
		return errors.NewWithCode("apply", errors.ErrSystem, "未返回全局应用结果")
	}
	if err := writeJSON(result); err != nil {
		return err
	}
	for _, item := range result.Items {
		if item.Status == globalport.StatusError || item.Status == globalport.StatusConflict {
			return errors.NewWithCode("apply", errors.ErrFileOperation, "部分全局技能应用失败，详见 items")
		}
	}
	return nil
}
