package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/engine"
	"skill-hub/pkg/spec"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用技能",
	Long:  "显示本地技能仓库中的所有技能，支持按目标环境过滤。",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runList(target, verbose)
	},
}

func init() {
	listCmd.Flags().String("target", "", "按目标环境过滤技能列表")
	listCmd.Flags().Bool("verbose", false, "显示详细信息，包括技能描述、版本、兼容性等")
}

func runList(target string, verbose bool) error {
	manager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	skills, err := manager.LoadAllSkills()
	if err != nil {
		return err
	}

	// 按目标环境过滤技能
	var filteredSkills []*spec.Skill
	if target != "" {
		for _, skill := range skills {
			compatLower := strings.ToLower(skill.Compatibility)
			targetLower := strings.ToLower(target)

			// 检查技能是否兼容指定的目标环境
			isCompatible := false
			if targetLower == "cursor" && strings.Contains(compatLower, "cursor") {
				isCompatible = true
			} else if (targetLower == "claude" || targetLower == "claude_code") &&
				(strings.Contains(compatLower, "claude") || strings.Contains(compatLower, "claude_code")) {
				isCompatible = true
			} else if (targetLower == "open_code" || targetLower == "opencode") &&
				(strings.Contains(compatLower, "open_code") || strings.Contains(compatLower, "opencode")) {
				isCompatible = true
			}

			if isCompatible {
				filteredSkills = append(filteredSkills, skill)
			}
		}
		skills = filteredSkills
	}

	if len(skills) == 0 {
		if target != "" {
			fmt.Printf("ℹ️  未找到兼容 %s 目标的技能\n", target)
		} else {
			fmt.Println("ℹ️  未找到任何技能")
		}
		fmt.Println("使用 'skill-hub init' 初始化技能仓库")
		return nil
	}

	if verbose {
		// 详细模式显示
		fmt.Println("可用技能列表 (详细模式):")
		fmt.Println(strings.Repeat("=", 60))
		for i, skill := range skills {
			fmt.Printf("%d. ID: %s\n", i+1, skill.ID)
			fmt.Printf("   名称: %s\n", skill.Name)
			fmt.Printf("   版本: %s\n", skill.Version)
			if skill.Description != "" {
				fmt.Printf("   描述: %s\n", skill.Description)
			}
			if skill.Compatibility != "" {
				fmt.Printf("   兼容性: %s\n", skill.Compatibility)
			}
			if len(skill.Tags) > 0 {
				fmt.Printf("   标签: %s\n", strings.Join(skill.Tags, ", "))
			}
			if skill.Author != "" && skill.Author != "unknown" {
				fmt.Printf("   作者: %s\n", skill.Author)
			}
			fmt.Println()
		}
	} else {
		// 简要模式显示
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
	}

	if target != "" {
		fmt.Printf("\n已过滤显示兼容 %s 目标的技能\n", target)
	}
	fmt.Println("\n使用 'skill-hub use <skill-id>' 在当前项目启用技能")
	return nil
}
