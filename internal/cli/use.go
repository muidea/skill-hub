package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
)

var useCmd = &cobra.Command{
	Use:   "use <id>",
	Short: "使用技能",
	Long: `将技能标记为在当前项目中使用。此命令仅更新 state.json 中的状态记录，不生成物理文件。
需要通过 apply 命令进行物理分发。

如果项目工作区里首次使用技能，也会同步在state.json里完成项目工作区信息刷新`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		return runUse(args[0], target)
	},
}

func init() {
	useCmd.Flags().String("target", "open_code", "技能目标环境，默认为 open_code")
}

func runUse(skillID string, target string) error {
	// 检查init依赖（规范4.8：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 检查技能是否存在
	if err := CheckSkillExists(skillID); err != nil {
		return err
	}

	manager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 加载技能详情
	skill, err := manager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	fmt.Printf("启用技能: %s (%s)\n", skill.Name, skillID)
	fmt.Printf("描述: %s\n", skill.Description)

	if len(skill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(skill.Tags, ", "))
	}

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目工作区状态（规范4.8：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, target)
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
	}

	// 检查项目是否已启用该技能
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return err
	}

	if hasSkill {
		fmt.Println("⚠️  该技能已在当前项目启用")
		fmt.Print("是否重新配置变量？ [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("❌ 取消操作")
			return nil
		}
	}

	// 收集变量值
	variables := make(map[string]string)

	if len(skill.Variables) > 0 {
		fmt.Println("\n请设置技能变量 (按Enter使用默认值):")

		reader := bufio.NewReader(os.Stdin)
		for _, variable := range skill.Variables {
			defaultValue := variable.Default
			if defaultValue == "" {
				defaultValue = ""
			}

			fmt.Printf("%s [%s]: ", variable.Name, defaultValue)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				variables[variable.Name] = defaultValue
			} else {
				variables[variable.Name] = input
			}
		}
	} else {
		fmt.Println("\n该技能没有可配置的变量")
	}

	// 保存到项目状态
	if err := stateManager.AddSkillToProjectWithTarget(cwd, skillID, skill.Version, variables, target); err != nil {
		return fmt.Errorf("保存项目状态失败: %w", err)
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为使用！\n", skillID)

	// 显示目标信息
	fmt.Printf("技能目标环境: %s\n", target)
	fmt.Println("使用 'skill-hub apply' 将技能物理分发到当前项目")

	return nil
}
