package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/adapter/opencode"
	"skill-hub/internal/config"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/internal/template"
	"skill-hub/pkg/spec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	feedbackTarget string
	archiveFlag    bool
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback [skill-id]",
	Short: "将项目内的手动修改反馈回技能仓库",
	Long: `将项目配置文件中手动修改的技能内容反向更新到本地技能仓库。

使用 --target 参数指定从哪个工具配置文件提取内容 (cursor/claude_code/open_code/all/auto)。
默认为空，会使用状态绑定的目标或自动检测。

使用 --archive 参数在反馈完成后将技能归档到正式技能仓库。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(args[0])
	},
}

func init() {
	feedbackCmd.Flags().StringVar(&feedbackTarget, "target", "", "目标工具: cursor, claude_code, open_code, all, auto (为空时使用状态绑定的目标)")
	feedbackCmd.Flags().BoolVar(&archiveFlag, "archive", false, "反馈完成后归档到技能仓库")
}

func runFeedback(skillID string) error {
	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查.agents/skills/目录下是否存在该技能
	agentsSkillsDir := filepath.Join(cwd, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillsDir); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 在当前项目的 .agents/skills/ 目录中不存在", skillID)
	}

	// 检查SKILL.md文件是否存在
	skillMdPath := filepath.Join(agentsSkillsDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("技能文件 SKILL.md 在当前项目的 .agents/skills/%s/ 目录中不存在", skillID)
	}

	// 从本地项目加载技能信息
	skill, err := loadSkillFromLocalProject(cwd, skillID)
	skillNotFound := false
	if err != nil {
		// 检查是否是"文件不存在"错误
		if strings.Contains(err.Error(), "读取SKILL.md失败") && strings.Contains(err.Error(), "no such file or directory") {
			fmt.Println("🔍 本地项目中未找到技能文件，将创建新技能")
			skillNotFound = true
			// 创建临时技能对象
			skill = &spec.Skill{
				ID:          skillID,
				Name:        skillID,
				Description: fmt.Sprintf("从项目反馈的新技能: %s", skillID),
				Version:     "1.0.0",
			}
		} else {
			return fmt.Errorf("加载本地技能失败: %w", err)
		}
	}

	// 初始化状态管理器（用于目标解析）
	stateManager, err := state.NewStateManager()
	if err != nil {
		// 状态管理器初始化失败不影响反馈操作
		fmt.Println("⚠️  状态管理器初始化失败，将使用默认设置")
	}

	// 确定目标工具
	resolvedTarget := feedbackTarget
	if resolvedTarget == "" {
		// 如果没有指定target，尝试从状态获取
		projectState, err := stateManager.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("查找项目状态失败: %w", err)
		}

		if projectState == nil || projectState.PreferredTarget == "" {
			// 未绑定项目，使用auto
			resolvedTarget = "auto"
			fmt.Println("🔍 项目未绑定目标，使用自动检测模式")
		} else {
			resolvedTarget = spec.NormalizeTarget(projectState.PreferredTarget)
			fmt.Printf("🔍 使用状态绑定的目标: %s\n", resolvedTarget)
		}
	} else {
		resolvedTarget = spec.NormalizeTarget(resolvedTarget)
		fmt.Printf("🔍 使用指定的目标: %s\n", resolvedTarget)
	}

	// 根据参数或自动检测选择适配器
	var fileContent string
	var adapterName string
	var extractErr error

	// 确定要尝试的适配器顺序
	tryCursor := false
	tryClaude := false
	tryOpenCode := false

	switch resolvedTarget {
	case "", "auto":
		// 自动模式：首先尝试项目的首选目标
		projectState, err := stateManager.FindProjectByPath(cwd)
		if err != nil {
			return fmt.Errorf("查找项目状态失败: %w", err)
		}

		if projectState != nil && projectState.PreferredTarget != "" {
			// 使用项目的首选目标
			normalizedTarget := spec.NormalizeTarget(projectState.PreferredTarget)
			compatLower := strings.ToLower(skill.Compatibility)

			if normalizedTarget == spec.TargetCursor && strings.Contains(compatLower, "cursor") {
				tryCursor = true
				fmt.Printf("🔍 使用项目首选目标: Cursor\n")
			} else if normalizedTarget == spec.TargetClaudeCode && (strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") || strings.Contains(compatLower, "claude")) {
				tryClaude = true
				fmt.Printf("🔍 使用项目首选目标: Claude Code\n")
			} else if normalizedTarget == spec.TargetOpenCode && (strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code")) {
				tryOpenCode = true
				fmt.Printf("🔍 使用项目首选目标: OpenCode\n")
			} else {
				// 首选目标不支持，回退到技能兼容性
				tryCursor = strings.Contains(compatLower, "cursor")
				tryClaude = strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") || strings.Contains(compatLower, "claude")
				tryOpenCode = strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code")
			}
		} else {
			// 没有首选目标，根据技能兼容性尝试
			compatLower := strings.ToLower(skill.Compatibility)
			tryCursor = strings.Contains(compatLower, "cursor")
			tryClaude = strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") || strings.Contains(compatLower, "claude")
			tryOpenCode = strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code")
		}
	case spec.TargetAll:
		// 尝试所有适配器
		compatLower := strings.ToLower(skill.Compatibility)
		tryCursor = strings.Contains(compatLower, "cursor")
		tryClaude = strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") || strings.Contains(compatLower, "claude")
		tryOpenCode = strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code")

		if !tryCursor && !tryClaude && !tryOpenCode {
			return fmt.Errorf("技能 '%s' 不支持任何适配器", skillID)
		}
	case spec.TargetCursor:
		tryCursor = true
		if !strings.Contains(strings.ToLower(skill.Compatibility), "cursor") {
			return fmt.Errorf("技能 '%s' 不支持 Cursor 适配器", skillID)
		}
	case spec.TargetClaudeCode:
		tryClaude = true
		if !(strings.Contains(strings.ToLower(skill.Compatibility), "claude code") || strings.Contains(strings.ToLower(skill.Compatibility), "claude_code") || strings.Contains(strings.ToLower(skill.Compatibility), "claude")) {
			return fmt.Errorf("技能 '%s' 不支持 Claude Code 适配器", skillID)
		}
	case spec.TargetOpenCode:
		tryOpenCode = true
		if !(strings.Contains(strings.ToLower(skill.Compatibility), "opencode") || strings.Contains(strings.ToLower(skill.Compatibility), "open_code")) {
			return fmt.Errorf("技能 '%s' 不支持 OpenCode 适配器", skillID)
		}
	default:
		return fmt.Errorf("无效的目标: %s，可用选项: %s, %s, %s, %s, auto", resolvedTarget, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll)
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

	// 如果前两个适配器失败且需要尝试OpenCode适配器
	if fileContent == "" && tryOpenCode {
		opencodeAdapter := opencode.NewOpenCodeAdapter()
		fileContent, extractErr = opencodeAdapter.Extract(skillID)
		if extractErr == nil {
			adapterName = "OpenCode"
		}
	}

	// 如果都没有提取到内容
	if fileContent == "" {
		if resolvedTarget == "auto" || resolvedTarget == "" {
			return fmt.Errorf("无法从任何配置文件中提取技能 '%s' 的内容。请确保技能已应用到目标工具。错误: %v", skillID, extractErr)
		} else {
			return fmt.Errorf("无法从 %s 配置文件中提取技能 '%s' 的内容。错误: %v", resolvedTarget, skillID, extractErr)
		}
	}

	fmt.Printf("从 %s 配置文件提取到技能内容\n", adapterName)

	// 从本地项目获取原始技能内容
	var originalContent []byte
	if !skillNotFound {
		localSkillPath := filepath.Join(cwd, ".agents", "skills", skillID, "SKILL.md")
		originalContent, err = os.ReadFile(localSkillPath)
		if err != nil {
			return fmt.Errorf("读取本地技能文件失败: %w", err)
		}
	} else {
		// 技能不存在，创建空的原始内容
		originalContent = []byte(fmt.Sprintf(`---
name: %s
description: 从项目反馈的新技能
compatibility: Designed for AI coding assistants
metadata:
  version: "1.0.0"
  author: "skill-hub"
  created_at: "%s"
---
# %s

从项目反馈的新技能。

## 使用说明

这是一个通过feedback命令从项目中创建的新技能。`, skillID, time.Now().Format(time.RFC3339), skillID))
	}

	// 尝试获取项目变量（如果技能已启用）
	var skillVariables map[string]string
	if stateManager != nil {
		skills, err := stateManager.GetProjectSkills(cwd)
		if err == nil {
			if skillVars, exists := skills[skillID]; exists {
				skillVariables = skillVars.Variables
				fmt.Println("🔍 使用项目变量配置")
			} else {
				skillVariables = make(map[string]string)
				fmt.Println("🔍 技能未在项目中启用，使用空变量配置")
			}
		} else {
			skillVariables = make(map[string]string)
			fmt.Println("🔍 无法获取项目状态，使用空变量配置")
		}
	} else {
		skillVariables = make(map[string]string)
		fmt.Println("🔍 状态管理器不可用，使用空变量配置")
	}

	// 渲染原始内容（使用项目变量）
	renderedOriginal := template.Render(string(originalContent), skillVariables)

	// 比较内容
	hasChanges := skillNotFound || strings.TrimSpace(fileContent) != strings.TrimSpace(renderedOriginal)

	if !hasChanges {
		fmt.Println("✅ 技能内容未修改")

		// 如果没有变化但使用了--archive参数，仍然执行归档
		if archiveFlag {
			fmt.Println("📦 检测到--archive参数，执行归档操作...")

			// 先检查技能是否在仓库中存在，如果不存在则先创建
			skillManager, err := engine.NewSkillManager()
			if err == nil && !skillManager.SkillExists(skillID) {
				fmt.Println("🔍 技能在仓库中不存在，先创建技能...")
				// 创建技能目录和文件，并获取解析后的技能对象
				createdSkill, err := createSkillInRepository(skillID, fileContent)
				if err != nil {
					return fmt.Errorf("创建技能失败: %w", err)
				}
				// 更新技能对象，使用解析后的版本
				skill = createdSkill
				fmt.Println("✅ 技能已创建")
			}

			// 执行归档
			if err := archiveSkill(skillID, skill.Version, cwd); err != nil {
				return fmt.Errorf("归档失败: %w", err)
			}
			fmt.Println("✅ 技能归档完成！")

			// 刷新技能索引
			fmt.Println("🔄 刷新技能索引...")
			if err := refreshSkillRegistryAfterArchive(); err != nil {
				fmt.Printf("⚠️  刷新技能索引失败: %v\n", err)
				fmt.Println("技能已归档，但索引未更新，请手动运行 'skill-hub init' 刷新索引")
			} else {
				fmt.Println("✅ 技能索引已刷新")
			}
			return nil
		}

		fmt.Println("无需反馈")
		return nil
	}

	// 如果是新技能，显示不同的消息
	var response string
	if skillNotFound {
		fmt.Println("🔍 检测到新技能，将添加到技能仓库")
		// 对于新技能，直接执行添加，不需要用户确认
		response = "y"
	} else {
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
		response, _ = reader.ReadString('\n')
		response = strings.TrimSpace(response)
	}

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
	templateVars := template.ExtractVariables(string(originalContent))

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
			newTemplate, updatedVariables, err = template.SmartExtract(string(originalContent), fileContent, skillVariables)
			if err != nil {
				fmt.Printf("警告: 智能提取失败: %v\n", err)
				fmt.Println("将保存修改后的内容...")
				newTemplate = fileContent
				updatedVariables = skillVariables
			} else {
				// 显示变量更新
				fmt.Println("变量更新:")
				changesFound := false
				for varName, oldValue := range skillVariables {
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
					if stateManager != nil {
						if err := stateManager.UpdateSkillVariables(cwd, skillID, updatedVariables); err != nil {
							fmt.Printf("警告: 更新项目变量失败: %v\n", err)
						} else {
							fmt.Println("✓ 更新项目变量")
						}
					} else {
						fmt.Println("⚠️  状态管理器不可用，无法更新项目变量")
					}
				}
			}

		case "3":
			// 手动编辑变量值
			fmt.Println("\n手动编辑变量值:")
			updatedVariables = make(map[string]string)
			for _, varName := range templateVars {
				currentValue := skillVariables[varName]
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
			newTemplate = template.Render(string(originalContent), updatedVariables)

			// 更新项目变量
			if stateManager != nil {
				if err := stateManager.UpdateSkillVariables(cwd, skillID, updatedVariables); err != nil {
					fmt.Printf("警告: 更新项目变量失败: %v\n", err)
				} else {
					fmt.Println("✓ 更新项目变量")
				}
			} else {
				fmt.Println("⚠️  状态管理器不可用，无法更新项目变量")
			}

		default:
			// 选项1或默认：保存修改后的内容
			fmt.Println("将保存修改后的内容（包含具体值）")
			newTemplate = fileContent
			updatedVariables = skillVariables
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

	// 重新初始化技能管理器以更新SKILL.md版本
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return fmt.Errorf("初始化技能管理器失败: %w", err)
	}

	// 更新SKILL.md版本（重新加载技能以获取最新信息）
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

	// 读取当前的SKILL.md文件
	skillMdPath = fmt.Sprintf("%s/SKILL.md", skillDir)
	skillMdContent, err := os.ReadFile(skillMdPath)
	if err != nil {
		return fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析并更新frontmatter中的版本号
	updatedContent, err := updateVersionInFrontmatter(string(skillMdContent), updatedSkill.Version)
	if err != nil {
		return fmt.Errorf("更新frontmatter版本号失败: %w", err)
	}

	// 保存更新后的SKILL.md
	if err := os.WriteFile(skillMdPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("更新SKILL.md失败: %w", err)
	}

	fmt.Println("✓ 更新 SKILL.md")
	fmt.Printf("✓ 版本更新: %s\n", updatedSkill.Version)

	// 如果启用了归档标志，执行归档操作
	if archiveFlag {
		fmt.Println("\n📦 开始归档技能...")
		if err := archiveSkill(skillID, updatedSkill.Version, cwd); err != nil {
			fmt.Printf("⚠️  归档失败: %v\n", err)
			fmt.Println("技能已更新但未归档，请手动处理")
		} else {
			fmt.Println("✅ 技能归档完成！")

			// 刷新技能索引
			fmt.Println("🔄 刷新技能索引...")
			if err := refreshSkillRegistryAfterArchive(); err != nil {
				fmt.Printf("⚠️  刷新技能索引失败: %v\n", err)
				fmt.Println("技能已归档，但索引未更新，请手动运行 'skill-hub init' 刷新索引")
			} else {
				fmt.Println("✅ 技能索引已刷新")
			}
		}
	}

	fmt.Println("\n✅ 反馈完成！")
	if !archiveFlag {
		fmt.Println("使用 'skill-hub update' 同步到远程仓库")
		fmt.Println("使用 'skill-hub feedback --archive' 归档技能到正式仓库")
	}

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

// updateVersionInFrontmatter 更新SKILL.md frontmatter中的版本号
func updateVersionInFrontmatter(content string, newVersion string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return "", fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
	}

	var result []string
	result = append(result, lines[0]) // 添加开头的 ---

	inMetadata := false
	versionUpdated := false

	for i := 1; i < len(lines); i++ {
		line := lines[i]

		if line == "---" {
			// frontmatter结束
			result = append(result, line)
			// 添加剩余的内容
			result = append(result, lines[i+1:]...)
			break
		}

		// 检查是否进入metadata部分
		if strings.HasPrefix(strings.TrimSpace(line), "metadata:") {
			inMetadata = true
			result = append(result, line)
			continue
		}

		// 在metadata中查找version字段
		if inMetadata && strings.HasPrefix(strings.TrimSpace(line), "version:") {
			// 更新版本号
			result = append(result, fmt.Sprintf("  version: %s", newVersion))
			versionUpdated = true
			continue
		}

		// 如果不在metadata中，但找到version字段（直接位于根级别）
		if !inMetadata && strings.HasPrefix(strings.TrimSpace(line), "version:") {
			result = append(result, fmt.Sprintf("version: %s", newVersion))
			versionUpdated = true
			continue
		}

		result = append(result, line)
	}

	// 如果没有找到version字段，在metadata中添加
	if !versionUpdated {
		// 重新构建内容，在metadata中添加version
		return addVersionToFrontmatter(content, newVersion)
	}

	return strings.Join(result, "\n"), nil
}

// addVersionToFrontmatter 在frontmatter中添加版本号
func addVersionToFrontmatter(content string, version string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return "", fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
	}

	var result []string
	result = append(result, lines[0]) // 添加开头的 ---

	frontmatterEnd := -1
	metadataFound := false

	// 查找frontmatter结束位置和metadata
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatterEnd = i
			break
		}

		if strings.TrimSpace(lines[i]) == "metadata:" {
			metadataFound = true
		}

		result = append(result, lines[i])
	}

	if frontmatterEnd == -1 {
		return "", fmt.Errorf("无效的SKILL.md格式: frontmatter没有正确结束")
	}

	// 如果找到metadata，在metadata中添加version
	if metadataFound {
		// 在metadata:后添加version
		var updatedResult []string
		for _, line := range result {
			updatedResult = append(updatedResult, line)
			if strings.TrimSpace(line) == "metadata:" {
				updatedResult = append(updatedResult, fmt.Sprintf("  version: %s", version))
			}
		}
		result = updatedResult
	} else {
		// 在frontmatter末尾添加metadata
		result = append(result, "metadata:")
		result = append(result, fmt.Sprintf("  version: %s", version))
	}

	// 添加结束的 --- 和剩余内容
	result = append(result, "---")
	for i := frontmatterEnd + 1; i < len(lines); i++ {
		result = append(result, lines[i])
	}

	return strings.Join(result, "\n"), nil
}

// createSkillInRepository 在技能仓库中创建新技能
// 返回解析后的技能对象和错误
func createSkillInRepository(skillID, content string) (*spec.Skill, error) {
	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return nil, fmt.Errorf("获取技能目录失败: %w", err)
	}

	// 创建技能目录
	skillDir := filepath.Join(skillsDir, skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return nil, fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 写入SKILL.md文件
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("写入SKILL.md失败: %w", err)
	}

	// 解析技能内容以获取版本等信息
	// 简单解析frontmatter
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || lines[0] != "---" {
		// 没有frontmatter，使用默认值
		return &spec.Skill{
			ID:      skillID,
			Name:    skillID,
			Version: "1.0.0",
		}, nil
	}

	// 提取frontmatter
	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		// 解析失败，使用默认值
		return &spec.Skill{
			ID:      skillID,
			Name:    skillID,
			Version: "1.0.0",
		}, nil
	}

	// 创建技能对象
	result := &spec.Skill{
		ID: skillID,
	}

	if name, ok := data["name"].(string); ok {
		result.Name = name
	} else {
		result.Name = skillID
	}

	if version, ok := data["version"].(string); ok {
		result.Version = version
	} else {
		result.Version = "1.0.0"
	}

	if desc, ok := data["description"].(string); ok {
		result.Description = desc
	}

	if compat, ok := data["compatibility"].(string); ok {
		result.Compatibility = compat
	}

	return result, nil
}

// archiveSkill 归档技能到正式技能仓库
func archiveSkill(skillID, version, projectPath string) error {
	fmt.Printf("归档技能 '%s' (版本: %s)...\n", skillID, version)

	// 获取技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return fmt.Errorf("创建技能管理器失败: %w", err)
	}

	// 获取技能目录
	skillsDir, err := engine.GetSkillsDir()
	if err != nil {
		return fmt.Errorf("获取技能目录失败: %w", err)
	}

	// 检查技能是否存在
	if !skillManager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在", skillID)
	}

	// 加载技能详情
	skill, err := skillManager.LoadSkill(skillID)
	if err != nil {
		return fmt.Errorf("加载技能失败: %w", err)
	}

	// 确定目标目录（正式技能目录）
	targetDir := filepath.Join(skillsDir, skillID)

	// 获取当前技能文件路径
	// 只使用标准结构：skills/skillID
	sourceDir := filepath.Join(skillsDir, skillID)

	// 如果源目录和目标目录相同，且目录已存在，说明技能已在正确位置
	if sourceDir == targetDir {
		if _, err := os.Stat(targetDir); err == nil {
			fmt.Println("✓ 技能已在正确位置，无需归档操作")
			// 仍然返回nil表示成功，但调用者需要知道需要刷新registry
			return nil
		}
	}

	// 检查目标目录是否已存在
	if _, err := os.Stat(targetDir); err == nil {
		// 目录已存在，询问是否覆盖
		fmt.Printf("⚠️  技能 '%s' 已存在于正式仓库\n", skillID)
		fmt.Print("是否覆盖？ [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			return fmt.Errorf("取消归档操作")
		}

		// 备份原有目录
		backupDir := targetDir + ".backup." + time.Now().Format("20060102-150405")
		if err := os.Rename(targetDir, backupDir); err != nil {
			return fmt.Errorf("备份原有目录失败: %w", err)
		}
		fmt.Printf("✓ 原有目录已备份到: %s\n", backupDir)
	}

	// 获取当前技能文件路径
	// 只使用标准结构：skills/skillID
	sourceSkillPath := filepath.Join(sourceDir, "SKILL.md")

	// 检查技能文件是否存在
	if _, err := os.Stat(sourceSkillPath); os.IsNotExist(err) {
		return fmt.Errorf("找不到技能文件: %s", skillID)
	}

	// 如果源目录和目标目录相同，跳过复制（技能已在正确位置）
	if sourceDir == targetDir {
		fmt.Println("✓ 技能已在正确位置，无需复制")
		// 直接返回成功
		return nil
	}

	// 创建目标目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 复制技能文件
	sourceFiles := []string{"SKILL.md", "prompt.md"}
	for _, filename := range sourceFiles {
		sourceFile := filepath.Join(sourceDir, filename)
		targetFile := filepath.Join(targetDir, filename)

		if _, err := os.Stat(sourceFile); err == nil {
			// 读取源文件内容
			content, err := os.ReadFile(sourceFile)
			if err != nil {
				return fmt.Errorf("读取文件失败 %s: %w", filename, err)
			}

			// 写入目标文件
			if err := os.WriteFile(targetFile, content, 0644); err != nil {
				return fmt.Errorf("写入文件失败 %s: %w", filename, err)
			}

			fmt.Printf("✓ 复制 %s\n", filename)
		}
	}

	// 更新归档记录
	archiveRecord := spec.ArchiveInfo{
		SkillID:    skillID,
		Version:    version,
		ArchivedAt: time.Now().Format(time.RFC3339),
	}

	// 保存归档记录（简化实现，实际应该保存到数据库或文件）
	fmt.Printf("✓ 归档记录: 技能=%s, 版本=%s, 时间=%s\n",
		archiveRecord.SkillID,
		archiveRecord.Version,
		archiveRecord.ArchivedAt)

	// 显示归档信息
	fmt.Printf("\n📋 归档完成信息:\n")
	fmt.Printf("   技能ID: %s\n", skillID)
	fmt.Printf("   技能名称: %s\n", skill.Name)
	fmt.Printf("   版本: %s\n", version)
	fmt.Printf("   描述: %s\n", skill.Description)
	if len(skill.Tags) > 0 {
		fmt.Printf("   标签: %s\n", strings.Join(skill.Tags, ", "))
	}
	fmt.Printf("   来源项目: %s\n", projectPath)
	fmt.Printf("   目标目录: %s\n", targetDir)

	return nil
}

// refreshSkillRegistryAfterArchive 归档后刷新技能索引
func refreshSkillRegistryAfterArchive() error {
	// 获取repo目录
	repoPath, err := config.GetRepoPath()
	if err != nil {
		return fmt.Errorf("获取repo目录失败: %w", err)
	}

	// 调用init.go中的refreshSkillRegistry函数
	// 由于我们不能直接调用另一个文件中的未导出函数，
	// 我们需要复制其逻辑或重构代码
	// 这里我们简单实现刷新逻辑

	registryPath := filepath.Join(repoPath, "registry.json")
	skillsDir := filepath.Join(repoPath, "skills")

	// 检查skills目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		// 如果skills目录不存在，创建空的registry.json
		registryContent := `{
  "version": "1.0.0",
  "skills": []
}`
		return os.WriteFile(registryPath, []byte(registryContent), 0644)
	}

	// 扫描skills目录下的所有子目录
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("读取skills目录失败: %w", err)
	}

	var skills []spec.SkillMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(skillsDir, skillID)
		skillMdPath := filepath.Join(skillDir, "SKILL.md")

		// 检查是否存在SKILL.md文件
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			continue
		}

		// 解析SKILL.md文件
		skillMeta, err := parseSkillMetadataForArchive(skillMdPath, skillID)
		if err != nil {
			fmt.Printf("⚠️  解析技能 %s 失败: %v\n", skillID, err)
			continue
		}

		skills = append(skills, *skillMeta)
	}

	// 创建registry对象
	registry := spec.Registry{
		Version: "1.0.0",
		Skills:  skills,
	}

	// 转换为JSON
	registryJSON, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化registry失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(registryPath, registryJSON, 0644); err != nil {
		return fmt.Errorf("写入registry.json失败: %w", err)
	}

	fmt.Printf("✓ 已索引 %d 个技能\n", len(skills))
	return nil
}

// parseSkillMetadataForArchive 从SKILL.md文件解析技能元数据（用于归档后刷新）
func parseSkillMetadataForArchive(mdPath, skillID string) (*spec.SkillMetadata, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML frontmatter
	var skillData map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err != nil {
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
	}

	// 创建技能元数据对象
	skillMeta := &spec.SkillMetadata{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skillMeta.Name = name
	} else {
		skillMeta.Name = skillID
	}

	// 设置描述
	if desc, ok := skillData["description"].(string); ok {
		skillMeta.Description = desc
	}

	// 设置版本
	skillMeta.Version = "1.0.0"
	if version, ok := skillData["version"].(string); ok {
		skillMeta.Version = version
	}

	// 设置作者
	if author, ok := skillData["author"].(string); ok {
		skillMeta.Author = author
	} else if source, ok := skillData["source"].(string); ok {
		skillMeta.Author = source
	} else {
		skillMeta.Author = "unknown"
	}

	// 设置标签
	if tagsStr, ok := skillData["tags"].(string); ok {
		skillMeta.Tags = strings.Split(tagsStr, ",")
		for i, tag := range skillMeta.Tags {
			skillMeta.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 设置兼容性
	if compatData, ok := skillData["compatibility"]; ok {
		switch v := compatData.(type) {
		case string:
			skillMeta.Compatibility = v
		case map[string]interface{}:
			// 向后兼容：将对象格式转换为字符串
			var compatList []string
			if cursorVal, ok := v["cursor"].(bool); ok && cursorVal {
				compatList = append(compatList, "Cursor")
			}
			if claudeVal, ok := v["claude_code"].(bool); ok && claudeVal {
				compatList = append(compatList, "Claude Code")
			}
			if openCodeVal, ok := v["open_code"].(bool); ok && openCodeVal {
				compatList = append(compatList, "OpenCode")
			}
			if shellVal, ok := v["shell"].(bool); ok && shellVal {
				compatList = append(compatList, "Shell")
			}
			if len(compatList) > 0 {
				skillMeta.Compatibility = "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
			}
		}
	}

	return skillMeta, nil
}
