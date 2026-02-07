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

var (
	useTarget string
)

var useCmd = &cobra.Command{
	Use:   "use [skill-id]",
	Short: "在当前项目启用技能",
	Long: `在当前项目启用指定技能，并提示输入变量值。

使用 --target 参数指定首选目标工具 (cursor/claude_code/open_code)。
如果项目尚未绑定目标，此参数将设置项目的首选目标。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUse(args[0])
	},
}

func init() {
	useCmd.Flags().StringVar(&useTarget, "target", "", "首选目标工具: cursor, claude_code, open_code (为空时使用项目状态绑定的目标)")
}

func runUse(skillID string) error {
	// 检查技能是否存在
	manager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	if !manager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在，使用 'skill-hub list' 查看可用技能", skillID)
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

	// 检查项目是否已启用该技能
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
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
	if err := stateManager.AddSkillToProjectWithTarget(cwd, skillID, skill.Version, variables, useTarget); err != nil {
		return fmt.Errorf("保存项目状态失败: %w", err)
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功启用！\n", skillID)

	// 显示目标信息
	if useTarget != "" {
		fmt.Printf("项目首选目标已设置为: %s\n", useTarget)
	}
	fmt.Println("使用 'skill-hub apply' 将技能应用到当前项目")

	return nil
}
