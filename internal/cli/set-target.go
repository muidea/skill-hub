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
	Use:   "set-target <value>",
	Short: "设置项目目标环境",
	Long: `设置当前项目的首选目标环境，该设置会持久化到 state.json 中，影响后续 apply、status、feedback 等命令的行为。

支持的目标环境值:
  cursor      # 设置为 Cursor
  claude      # 设置为 Claude Code
  open_code   # 设置为 OpenCode

示例:
  skill-hub set-target open_code   # 设置项目为 OpenCode 环境
  skill-hub set-target cursor      # 设置项目为 Cursor 环境`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetTarget(args[0])
	},
}

func runSetTarget(target string) error {
	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 验证目标值（先规范化）
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode {
		return fmt.Errorf("无效的目标值: %s，可用选项: cursor, claude, open_code", target)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 设置首选目标（使用规范化后的值）
	if err := stateManager.SetPreferredTarget(cwd, normalizedTarget); err != nil {
		return fmt.Errorf("设置首选目标失败: %w", err)
	}

	// 显示结果
	fmt.Printf("✅ 已将项目 '%s' 的首选目标设置为: %s\n", filepath.Base(cwd), normalizedTarget)
	fmt.Println("下次执行 'skill-hub apply' 时将自动使用此目标")

	return nil
}
