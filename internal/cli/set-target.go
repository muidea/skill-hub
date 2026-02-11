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
	// 检查init依赖（规范4.2：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

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

	// 检查项目工作区状态（规范4.2：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, normalizedTarget)
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
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

	// 如果项目是新创建的，需要根据target初始化对应的文件和目录
	// 这个逻辑已经在EnsureProjectWorkspace中处理了

	// 显示结果
	fmt.Printf("✅ 已将项目 '%s' 的首选目标设置为: %s\n", filepath.Base(cwd), normalizedTarget)
	fmt.Println("下次执行 'skill-hub apply' 时将自动使用此目标")

	return nil
}
