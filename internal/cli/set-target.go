package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

var setTargetCmd = &cobra.Command{
	Use:   "set-target <value>",
	Short: "设置项目兼容目标",
	Long: `设置当前项目的首选兼容目标，该设置会持久化到 state.json 中，影响后续 apply、status、feedback 等命令的行为。

支持的取值:
  cursor
  claude
  open_code

示例:
  skill-hub set-target open_code
  skill-hub set-target claude`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTargetValues,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetTarget(args[0])
	},
}

func runSetTarget(target string) error {
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode {
		return errors.NewWithCodef("runSetTarget", errors.ErrInvalidInput, "无效的目标值: %s，可用选项: cursor, claude, open_code", target)
	}

	ctx, err := RequireInitAndWorkspace("", normalizedTarget)
	if err != nil {
		return err
	}

	if err := ctx.StateManager.SetPreferredTarget(ctx.Cwd, normalizedTarget); err != nil {
		return errors.Wrap(err, "设置首选目标失败")
	}

	fmt.Printf("✅ 已将项目 '%s' 的首选兼容目标设置为: %s\n", filepath.Base(ctx.Cwd), normalizedTarget)
	fmt.Println("下次执行 'skill-hub apply' 时将自动使用此目标")

	return nil
}
