package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setTargetCmd = &cobra.Command{
	Use:   "set-target <value>",
	Short: "保留兼容的目标命令",
	Long: `set-target 仅保留用于兼容旧脚本。skill-hub 当前统一使用标准 .agents/skills 工作区，target 不再写入项目状态，也不再影响 apply、status、feedback 等命令。

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
	_ = target

	fmt.Println("ℹ️  set-target 已保留为兼容命令；target 不再写入项目状态，也不影响后续业务逻辑")
	fmt.Println("skill-hub 当前统一使用标准 .agents/skills 工作区")
	return nil
}
