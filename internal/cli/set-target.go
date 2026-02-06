package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var setTargetCmd = &cobra.Command{
	Use:   "set-target [cursor|claude]",
	Short: "设置当前项目的首选目标",
	Long: `设置当前项目的首选目标（Cursor 或 Claude）。

此命令会更新项目状态，使后续的 apply、feedback 等命令自动使用指定的目标适配器。

示例:
  skill-hub set-target cursor    # 设置为 Cursor
  skill-hub set-target claude    # 设置为 Claude
  skill-hub set-target ""        # 清除目标设置`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetTarget(args[0])
	},
}

func init() {
	rootCmd.AddCommand(setTargetCmd)
}

func runSetTarget(target string) error {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 验证目标值
	if target != spec.TargetCursor && target != spec.TargetClaudeCode && target != "" {
		return fmt.Errorf("无效的目标值: %s，可用选项: %s, %s", target, spec.TargetCursor, spec.TargetClaudeCode)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 设置首选目标
	if err := stateManager.SetPreferredTarget(cwd, target); err != nil {
		return fmt.Errorf("设置首选目标失败: %w", err)
	}

	// 显示结果
	if target == "" {
		fmt.Printf("✅ 已清除项目 '%s' 的首选目标\n", filepath.Base(cwd))
	} else {
		fmt.Printf("✅ 已将项目 '%s' 的首选目标设置为: %s\n", filepath.Base(cwd), target)
		fmt.Println("下次执行 'skill-hub apply' 时将自动使用此目标")
	}

	return nil
}
