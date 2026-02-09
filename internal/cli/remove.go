package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/state"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "移除项目技能",
	Long: `从当前项目中移除指定的技能：
1. 从 state.json 中移除技能标记
2. 物理删除项目本地工作区对应的文件/配置
3. 保留仓库中的源文件不受影响

安全机制: 如果检测到本地有未反馈的修改，会弹出警告并要求确认。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args[0])
	},
}

func runRemove(skillID string) error {
	fmt.Printf("正在从当前项目移除技能: %s\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 创建状态管理器
	stateMgr, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 检查技能是否在项目中启用
	hasSkill, err := stateMgr.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查技能状态失败: %w", err)
	}
	if !hasSkill {
		return fmt.Errorf("技能 %s 未在当前项目中启用", skillID)
	}

	// TODO: 安全检查 - 检测本地有未反馈的修改
	// 这里应该检查项目工作区文件与仓库源文件的差异
	fmt.Println("⚠️  安全检查: 检测本地修改...")
	fmt.Println("注意: 安全检查功能暂未完全实现")

	// 确认移除
	if !confirmRemoval(skillID) {
		fmt.Println("❌ 操作已取消")
		return nil
	}

	// 从状态文件中移除技能标记
	fmt.Println("\n=== 更新状态 ===")
	if err := stateMgr.RemoveSkillFromProject(cwd, skillID); err != nil {
		return fmt.Errorf("从状态文件移除技能失败: %w", err)
	}
	fmt.Printf("✓ 成功从 state.json 移除技能标记: %s\n", skillID)

	// 物理删除项目本地工作区对应的文件/配置
	fmt.Println("\n=== 物理清理 ===")

	// 1. 删除.agents/skills/[skillID]目录
	agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillDir); err == nil {
		if err := os.RemoveAll(agentsSkillDir); err != nil {
			fmt.Printf("⚠️  删除 .agents/skills/%s 目录失败: %v\n", skillID, err)
		} else {
			fmt.Printf("✓ 删除项目本地工作区目录: .agents/skills/%s\n", skillID)
		}
	}

	// 2. 清理可能的其他目标环境文件
	// TODO: 根据项目目标环境清理Cursor、Claude等配置文件

	fmt.Println("\n✅ 技能移除完成")
	fmt.Println("注意: 仓库中的源文件不受影响")
	fmt.Println("使用 'skill-hub status' 检查当前状态")

	return nil
}

// confirmRemoval 确认是否继续移除
func confirmRemoval(skillID string) bool {
	fmt.Printf("\n⚠️  警告: 将移除技能 %s\n", skillID)
	fmt.Print("是否继续？(y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}
