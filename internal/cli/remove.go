package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	projectstatusservice "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "移除项目技能",
	Long: `从当前项目中移除指定的技能：
1. 从 state.json 中移除技能标记
2. 物理删除项目本地工作区对应的文件/配置
3. 保留仓库中的源文件不受影响

安全机制: 如果检测到本地有未反馈的修改，会弹出警告并要求确认。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEnabledSkillIDsForCwd,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args[0])
	},
}

func runRemove(skillID string) error {
	ctx, err := RequireInitAndWorkspace("", "")
	if err != nil {
		return err
	}

	fmt.Printf("正在从当前项目移除技能: %s\n", skillID)

	hasSkill, err := ctx.StateManager.ProjectHasSkill(ctx.Cwd, skillID)
	if err != nil {
		return errors.Wrap(err, "检查技能状态失败")
	}
	if !hasSkill {
		return errors.NewWithCodef("runRemove", errors.ErrSkillNotFound, "技能 %s 未在当前项目中启用", skillID)
	}

	statusItem, err := inspectRemovalStatus(ctx.Cwd, skillID)
	if err != nil {
		return errors.Wrap(err, "执行删除前安全检查失败")
	}

	renderRemovalWarning(statusItem)

	// 确认移除
	if !confirmRemoval(skillID, statusItem) {
		fmt.Println("❌ 操作已取消")
		return nil
	}

	// 物理删除项目本地工作区对应的文件/配置
	fmt.Println("\n=== 物理清理 ===")
	if err := removeProjectSkillArtifacts(ctx.Cwd, skillID); err != nil {
		return errors.Wrap(err, "清理项目技能文件失败")
	}

	fmt.Println("\n=== 更新状态 ===")
	if err := ctx.StateManager.RemoveSkillFromProject(ctx.Cwd, skillID); err != nil {
		return errors.Wrap(err, "从状态文件移除技能失败")
	}
	fmt.Printf("✓ 成功从 state.json 移除技能标记: %s\n", skillID)

	fmt.Println("\n✅ 技能移除完成")
	fmt.Println("注意: 仓库中的源文件不受影响")
	fmt.Println("使用 'skill-hub status' 检查当前状态")

	return nil
}

func inspectRemovalStatus(projectPath, skillID string) (*projectstatusservice.SkillStatusItem, error) {
	summary, err := projectstatusservice.New().Inspect(projectPath, skillID)
	if err != nil {
		return nil, err
	}
	if summary == nil || len(summary.Items) == 0 {
		return nil, errors.NewWithCode("inspectRemovalStatus", errors.ErrSkillNotFound, "未找到技能状态")
	}
	return &summary.Items[0], nil
}

func renderRemovalWarning(item *projectstatusservice.SkillStatusItem) {
	fmt.Println("⚠️  删除前安全检查:")
	if item == nil {
		fmt.Println("未获取到技能状态，将按常规删除处理")
		return
	}

	fmt.Printf("  当前状态: %s\n", item.Status)
	if item.SourceRepository != "" {
		fmt.Printf("  来源仓库: %s\n", item.SourceRepository)
	}

	switch item.Status {
	case spec.SkillStatusModified:
		fmt.Println("  警告: 本地存在未反馈修改，删除后这些修改将丢失。")
	case spec.SkillStatusOutdated:
		fmt.Println("  提示: 本地与来源仓库不一致，删除后将放弃当前项目中的本地副本。")
	case spec.SkillStatusMissing:
		fmt.Println("  提示: 项目本地工作区文件已经缺失，将仅清理项目状态。")
	default:
		fmt.Println("  本地工作区与来源仓库一致，可安全移除。")
	}
}

func removeProjectSkillArtifacts(projectPath, skillID string) error {
	adapter, err := getTargetAdapter(spec.TargetOpenCode)
	if err != nil {
		return errors.Wrap(err, "removeProjectSkillArtifacts: 获取适配器失败")
	}
	adapter.SetProjectMode()

	if err := applyInProjectDir(projectPath, func() error {
		return adapter.Remove(skillID)
	}); err != nil {
		return errors.Wrap(err, "removeProjectSkillArtifacts: 清理项目技能目录失败")
	}

	fmt.Println("✓ 已清理项目本地技能目录")

	agentsSkillDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillDir); err == nil {
		if err := os.RemoveAll(agentsSkillDir); err != nil {
			return errors.Wrap(err, "removeProjectSkillArtifacts: 删除项目工作区技能目录失败")
		}
		fmt.Printf("✓ 删除项目工作区目录: .agents/skills/%s\n", skillID)
	}

	return nil
}

func applyInProjectDir(projectPath string, fn func() error) error {
	originalCwd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, "applyInProjectDir: 获取当前目录失败")
	}
	if err := os.Chdir(projectPath); err != nil {
		return errors.Wrap(err, "applyInProjectDir: 切换项目目录失败")
	}
	defer func() {
		_ = os.Chdir(originalCwd)
	}()

	return fn()
}

// confirmRemoval 确认是否继续移除
func confirmRemoval(skillID string, item *projectstatusservice.SkillStatusItem) bool {
	fmt.Printf("\n⚠️  警告: 将移除技能 %s\n", skillID)
	if item != nil {
		switch item.Status {
		case spec.SkillStatusModified:
			fmt.Println("删除后将丢失未反馈的本地修改。")
		case spec.SkillStatusOutdated:
			fmt.Println("删除后将放弃当前项目中的旧版本本地副本。")
		}
	}
	fmt.Print("是否继续？(y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}
