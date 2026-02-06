package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/internal/template"
	"skill-hub/pkg/spec"
)

var (
	adapterTarget string
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback [skill-id]",
	Short: "将项目内的手动修改反馈回技能仓库",
	Long: `将项目配置文件中手动修改的技能内容反向更新到本地技能仓库。

使用 --adapter 参数指定从哪个工具配置文件提取内容 (cursor/claude/auto)。
默认为 auto，会自动检测技能支持的工具。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(args[0])
	},
}

func init() {
	feedbackCmd.Flags().StringVar(&adapterTarget, "adapter", "auto", "目标适配器: cursor, claude, auto")
}

func runFeedback(skillID string) error {
	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目是否启用了该技能
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return err
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在当前项目启用", skillID)
	}

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 检查技能是否存在
	if !skillManager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在", skillID)
	}

	// 加载技能详情以确定支持的适配器
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 根据参数或自动检测选择适配器
	var fileContent string
	var adapterName string
	var extractErr error

	// 确定要尝试的适配器顺序
	tryCursor := false
	tryClaude := false

	if adapterTarget == "auto" {
		// 自动模式：首先尝试项目的首选目标
		projectState, err := stateManager.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("查找项目状态失败: %w", err)
		}

		if projectState != nil && projectState.PreferredTarget != "" {
			// 使用项目的首选目标
			normalizedTarget := spec.NormalizeTarget(projectState.PreferredTarget)
			if normalizedTarget == spec.TargetCursor && skill.Compatibility.Cursor {
				tryCursor = true
				fmt.Printf("🔍 使用项目首选目标: Cursor\n")
			} else if normalizedTarget == spec.TargetClaudeCode && skill.Compatibility.ClaudeCode {
				tryClaude = true
				fmt.Printf("🔍 使用项目首选目标: Claude Code\n")
			} else {
				// 首选目标不支持，回退到技能兼容性
				tryCursor = skill.Compatibility.Cursor
				tryClaude = skill.Compatibility.ClaudeCode
			}
		} else {
			// 没有首选目标，根据技能兼容性尝试
			tryCursor = skill.Compatibility.Cursor
			tryClaude = skill.Compatibility.ClaudeCode
		}
	} else if adapterTarget == spec.TargetCursor {
		tryCursor = true
		if !skill.Compatibility.Cursor {
			return fmt.Errorf("技能 '%s' 不支持 Cursor 适配器", skillID)
		}
	} else if adapterTarget == spec.TargetClaudeCode {
		tryClaude = true
		if !skill.Compatibility.ClaudeCode {
			return fmt.Errorf("技能 '%s' 不支持 Claude Code 适配器", skillID)
		}
	} else {
		return fmt.Errorf("无效的适配器目标: %s，可用选项: %s, %s, auto", adapterTarget, spec.TargetCursor, spec.TargetClaudeCode)
	}

	// 尝试Cursor适配器
	if tryCursor {
		cursorAdapter := cursor.NewCursorAdapter()
		fileContent, extractErr = cursorAdapter.Extract(skillID)
		if extractErr == nil {
			adapterName = "Cursor"
		}
	}

	// 如果Cursor适配器失败且需要尝试Claude适配器
	if fileContent == "" && tryClaude {
		claudeAdapter := claude.NewClaudeAdapter()
		fileContent, extractErr = claudeAdapter.Extract(skillID)
		if extractErr == nil {
			adapterName = "Claude"
		}
	}

	// 如果都没有提取到内容
	if fileContent == "" {
		if adapterTarget == "auto" {
			return fmt.Errorf("无法从任何配置文件中提取技能 '%s' 的内容。请确保技能已应用到目标工具。错误: %v", skillID, extractErr)
		} else {
			return fmt.Errorf("无法从 %s 配置文件中提取技能 '%s' 的内容。错误: %v", adapterTarget, skillID, extractErr)
		}
	}

	fmt.Printf("从 %s 配置文件提取到技能内容\n", adapterName)

	// 从仓库获取原始内容
	originalPrompt, err := skillManager.GetSkillPrompt(skillID)
	if err != nil {
		return fmt.Errorf("获取原始内容失败: %w", err)
	}

	// 获取项目变量
	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	skillVars, exists := skills[skillID]
	if !exists {
		return fmt.Errorf("未找到技能变量配置")
	}

	// 渲染原始内容（使用项目变量）
	renderedOriginal, err := renderTemplate(originalPrompt, skillVars.Variables)
	if err != nil {
		return fmt.Errorf("渲染原始内容失败: %w", err)
	}

	// 比较内容
	if strings.TrimSpace(fileContent) == strings.TrimSpace(renderedOriginal) {
		fmt.Println("✅ 技能内容未修改，无需反馈")
		return nil
	}

	// 显示差异
	fmt.Println("\n🔍 检测到手动修改:")
	fmt.Println("========================================")

	fileLines := strings.Split(strings.TrimSpace(fileContent), "\n")
	originalLines := strings.Split(strings.TrimSpace(renderedOriginal), "\n")

	// 简单差异显示
	maxLines := len(fileLines)
	if len(originalLines) > maxLines {
		maxLines = len(originalLines)
	}

	changesFound := false
	for i := 0; i < maxLines; i++ {
		var fileLine, originalLine string
		if i < len(fileLines) {
			fileLine = fileLines[i]
		}
		if i < len(originalLines) {
			originalLine = originalLines[i]
		}

		if fileLine != originalLine {
			if !changesFound {
				fmt.Println("行号 | 修改前                      | 修改后")
				fmt.Println("-----|---------------------------|---------------------------")
				changesFound = true
			}

			lineNum := i + 1
			fmt.Printf("%4d | %-25s | %-25s\n", lineNum,
				truncate(originalLine, 25),
				truncate(fileLine, 25))
		}
	}

	if !changesFound {
		fmt.Println("（仅空白字符差异）")
	}

	fmt.Println("========================================")

	// 确认反馈
	fmt.Print("\n是否将这些修改更新到技能仓库？ [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ 取消反馈操作")
		return nil
	}

	// 更新技能仓库
	fmt.Println("正在更新技能仓库...")

	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := fmt.Sprintf("%s/%s", skillsDir, skillID)
	promptPath := fmt.Sprintf("%s/prompt.md", skillDir)

	// 使用智能变量提取算法
	fmt.Println("正在分析变量变化...")

	// 提取原始模板中的变量
	templateVars := template.ExtractVariables(originalPrompt)

	if len(templateVars) > 0 {
		fmt.Printf("检测到 %d 个模板变量: %v\n", len(templateVars), templateVars)

		// 询问用户如何处理变量
		fmt.Println("\n检测到模板变量。请选择处理方式:")
		fmt.Println("1. 保存修改后的内容（包含具体值）")
		fmt.Println("2. 尝试智能提取变量值")
		fmt.Println("3. 手动编辑变量值")
		fmt.Print("请选择 (1/2/3, 默认 1): ")

		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		var newTemplate string
		var updatedVariables map[string]string

		switch choice {
		case "2":
			// 尝试智能提取
			newTemplate, updatedVariables, err = template.SmartExtract(originalPrompt, fileContent, skillVars.Variables)
			if err != nil {
				fmt.Printf("警告: 智能提取失败: %v\n", err)
				fmt.Println("将保存修改后的内容...")
				newTemplate = fileContent
				updatedVariables = skillVars.Variables
			} else {
				// 显示变量更新
				fmt.Println("变量更新:")
				changesFound := false
				for varName, oldValue := range skillVars.Variables {
					if newValue, exists := updatedVariables[varName]; exists && newValue != oldValue {
						fmt.Printf("  %s: %q -> %q\n", varName, oldValue, newValue)
						changesFound = true
					}
				}
				if !changesFound {
					fmt.Println("  (没有检测到变量值变化)")
				}

				// 询问是否更新项目变量
				fmt.Print("\n是否更新项目中的变量值？ [y/N]: ")
				updateVars, _ := reader.ReadString('\n')
				updateVars = strings.TrimSpace(updateVars)

				if updateVars == "y" || updateVars == "Y" {
					if err := stateManager.UpdateSkillVariables(cwd, skillID, updatedVariables); err != nil {
						fmt.Printf("警告: 更新项目变量失败: %v\n", err)
					} else {
						fmt.Println("✓ 更新项目变量")
					}
				}
			}

		case "3":
			// 手动编辑变量值
			fmt.Println("\n手动编辑变量值:")
			updatedVariables = make(map[string]string)
			for _, varName := range templateVars {
				currentValue := skillVars.Variables[varName]
				fmt.Printf("变量 %s (当前值: %q): ", varName, currentValue)
				newValue, _ := reader.ReadString('\n')
				newValue = strings.TrimSpace(newValue)
				if newValue != "" {
					updatedVariables[varName] = newValue
				} else {
					updatedVariables[varName] = currentValue
				}
			}

			// 使用更新后的变量渲染模板
			newTemplate = template.Render(originalPrompt, updatedVariables)

			// 更新项目变量
			if err := stateManager.UpdateSkillVariables(cwd, skillID, updatedVariables); err != nil {
				fmt.Printf("警告: 更新项目变量失败: %v\n", err)
			} else {
				fmt.Println("✓ 更新项目变量")
			}

		default:
			// 选项1或默认：保存修改后的内容
			fmt.Println("将保存修改后的内容（包含具体值）")
			newTemplate = fileContent
			updatedVariables = skillVars.Variables
		}

		// 写入更新后的模板
		if err := os.WriteFile(promptPath, []byte(newTemplate), 0644); err != nil {
			return fmt.Errorf("更新prompt.md失败: %w", err)
		}

		fmt.Println("✓ 更新 prompt.md")

	} else {
		// 没有变量，直接保存
		if err := os.WriteFile(promptPath, []byte(fileContent), 0644); err != nil {
			return fmt.Errorf("更新prompt.md失败: %w", err)
		}
		fmt.Println("✓ 更新 prompt.md (无变量)")
	}

	// 更新skill.yaml版本（重新加载技能以获取最新信息）
	updatedSkill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 增加版本号
	versionParts := strings.Split(updatedSkill.Version, ".")
	if len(versionParts) == 3 {
		// 简单增加修订版本号
		// 在实际实现中应该更智能地处理版本号
		updatedSkill.Version = fmt.Sprintf("%s.%s.%d",
			versionParts[0],
			versionParts[1],
			parseInt(versionParts[2])+1)
	}

	// 保存更新后的skill.yaml
	yamlPath := fmt.Sprintf("%s/skill.yaml", skillDir)
	yamlData, err := yaml.Marshal(updatedSkill)
	if err != nil {
		return fmt.Errorf("序列化skill.yaml失败: %w", err)
	}

	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("更新skill.yaml失败: %w", err)
	}

	fmt.Println("✓ 更新 skill.yaml")
	fmt.Printf("✓ 版本更新: %s\n", updatedSkill.Version)

	fmt.Println("\n✅ 反馈完成！")
	fmt.Println("使用 'skill-hub update' 同步到远程仓库")

	return nil
}

// truncate 截断字符串
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// parseInt 解析整数，失败返回0
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
