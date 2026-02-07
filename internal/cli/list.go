package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/engine"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有可用技能",
	Long:  "列出本地技能仓库中的所有可用技能，显示状态、版本和适用工具。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func runList() error {
	manager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	skills, err := manager.LoadAllSkills()
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  未找到任何技能")
		fmt.Println("使用 'skill-hub init' 初始化技能仓库")
		return nil
	}

	fmt.Println("可用技能列表:")
	fmt.Println("ID          名称                版本      适用工具")
	fmt.Println("--------------------------------------------------")

	for _, skill := range skills {
		tools := []string{}
		compatLower := strings.ToLower(skill.Compatibility)
		if strings.Contains(compatLower, "cursor") {
			tools = append(tools, "cursor")
		}
		if strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") {
			tools = append(tools, "claude_code")
		}
		if strings.Contains(compatLower, "shell") {
			tools = append(tools, "shell")
		}
		if strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code") {
			tools = append(tools, "open_code")
		}

		toolsStr := ""
		if len(tools) > 0 {
			toolsStr = tools[0]
			for i := 1; i < len(tools); i++ {
				toolsStr += "," + tools[i]
			}
		}

		fmt.Printf("%-12s %-20s %-10s %s\n",
			skill.ID,
			skill.Name,
			skill.Version,
			toolsStr)
	}

	fmt.Println("\n使用 'skill-hub use <skill-id>' 在当前项目启用技能")
	return nil
}
