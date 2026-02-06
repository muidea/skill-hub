package cli

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"skill-hub/internal/adapter"
	"skill-hub/internal/adapter/claude"
	"skill-hub/internal/adapter/cursor"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "检查项目内技能状态",
	Long:  "对比项目内配置文件与技能仓库的差异，检测是否有手动修改。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func runStatus() error {
	fmt.Println("检查项目技能状态...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 加载项目状态
	stateManager, err := state.NewStateManager()
	if err != nil {
		return err
	}

	// 获取项目状态以显示目标信息
	projectState, err := stateManager.FindProjectByPath(cwd)
	if err != nil {
		return fmt.Errorf("查找项目状态失败: %w", err)
	}

	// 显示项目信息
	fmt.Printf("项目路径: %s\n", cwd)
	if projectState != nil && projectState.PreferredTarget != "" {
		normalizedTarget := spec.NormalizeTarget(projectState.PreferredTarget)
		targetName := "Cursor"
		if normalizedTarget == spec.TargetClaudeCode {
			targetName = "Claude Code"
		}
		fmt.Printf("Context Detected: %s | Project: %s\n", targetName, cwd)
	} else {
		fmt.Println("Context Detected: Unknown | Project: (未绑定)")
	}
	fmt.Println()

	skills, err := stateManager.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		return nil
	}

	// 创建适配器
	adapters := []struct {
		name     string
		adapter  adapter.Adapter
		filePath string
	}{
		{"Cursor", cursor.NewCursorAdapter(), ""},
		{"Claude", claude.NewClaudeAdapter(), ""},
	}

	// 检查文件是否存在并获取路径
	for i := range adapters {
		// 对于Cursor适配器，需要特殊处理获取路径
		if cursorAdapter, ok := adapters[i].adapter.(*cursor.CursorAdapter); ok {
			// 设置全局模式以获取路径
			cursorAdapter.WithGlobalMode()
			path, err := cursorAdapter.GetFilePath()
			if err == nil {
				adapters[i].filePath = path
			}
		} else if claudeAdapter, ok := adapters[i].adapter.(*claude.ClaudeAdapter); ok {
			// Claude适配器需要设置模式
			claudeAdapter.WithGlobalMode()
			// 获取配置路径
			path, err := claudeAdapter.GetConfigPath()
			if err == nil {
				adapters[i].filePath = path
			}
		}
	}

	// 加载技能管理器
	skillManager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	allModifiedSkills := make(map[string][]string) // adapter -> skillIDs
	allSyncedSkills := make(map[string][]string)   // adapter -> skillIDs

	// 检查每个适配器
	for _, adapterInfo := range adapters {
		adapterName := adapterInfo.name
		adpt := adapterInfo.adapter
		filePath := adapterInfo.filePath

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("\nℹ️  未找到 %s 配置文件: %s\n", adapterName, filePath)
			fmt.Printf("   使用 'skill-hub apply --target %s' 应用技能\n", strings.ToLower(adapterName))
			continue
		}

		fmt.Printf("\n扫描 %s 配置文件: %s\n", adapterName, filePath)

		modifiedSkills := []string{}
		syncedSkills := []string{}

		for skillID, skillVars := range skills {
			// 检查技能是否支持当前适配器
			skill, err := skillManager.LoadSkill(skillID)
			if err != nil {
				continue
			}

			// 检查适配器支持
			if !checkAdapterSupport(adpt, skill) {
				continue
			}

			// 从文件提取内容
			fileContent, err := adpt.Extract(skillID)
			if err != nil {
				// 技能未在该适配器中应用
				continue
			}

			// 从仓库获取原始内容
			originalPrompt, err := skillManager.GetSkillPrompt(skillID)
			if err != nil {
				continue
			}

			// 渲染原始内容（使用项目变量）
			renderedOriginal, err := renderTemplate(originalPrompt, skillVars.Variables)
			if err != nil {
				continue
			}

			// 计算哈希值进行比较
			fileHash := sha256.Sum256([]byte(strings.TrimSpace(fileContent)))
			originalHash := sha256.Sum256([]byte(strings.TrimSpace(renderedOriginal)))

			if fileHash == originalHash {
				syncedSkills = append(syncedSkills, skillID)
			} else {
				modifiedSkills = append(modifiedSkills, skillID)
			}
		}

		if len(syncedSkills) > 0 || len(modifiedSkills) > 0 {
			allSyncedSkills[adapterName] = syncedSkills
			allModifiedSkills[adapterName] = modifiedSkills
		}
	}

	// 显示结果
	fmt.Println("\n=== 技能状态汇总 ===")

	currentTime := time.Now().Format("15:04")
	hasAnySkills := false

	for adapterName, syncedSkills := range allSyncedSkills {
		modifiedSkills := allModifiedSkills[adapterName]

		if len(syncedSkills) == 0 && len(modifiedSkills) == 0 {
			continue
		}

		hasAnySkills = true
		fmt.Printf("\n%s:\n", adapterName)
		fmt.Println("ID          状态      最后检查")
		fmt.Println("----------------------------------")

		for _, skillID := range syncedSkills {
			fmt.Printf("%-12s ✅ 同步   %s\n", skillID, currentTime)
		}

		for _, skillID := range modifiedSkills {
			fmt.Printf("%-12s ⚠️ 已修改  %s\n", skillID, currentTime)
		}

		if len(modifiedSkills) > 0 {
			fmt.Printf("\n⚠️  检测到手动修改的技能:\n")
			for _, skillID := range modifiedSkills {
				fmt.Printf("  - %s\n", skillID)
			}
			fmt.Printf("使用 'skill-hub feedback %s' 将修改反馈回仓库\n", modifiedSkills[0])
		}
	}

	if !hasAnySkills {
		fmt.Println("\nℹ️  未在任何配置文件中找到已应用的技能")
		fmt.Println("使用 'skill-hub apply' 应用技能到目标工具")
	} else {
		// 检查是否有任何修改
		totalModified := 0
		for _, modifiedSkills := range allModifiedSkills {
			totalModified += len(modifiedSkills)
		}

		if totalModified == 0 {
			fmt.Println("\n✅ 所有技能状态正常，未检测到手动修改")
		}
	}

	fmt.Println("\n如需更新技能，使用 'skill-hub update'")

	return nil
}

// checkAdapterSupport 检查适配器是否支持该技能
func checkAdapterSupport(adpt adapter.Adapter, skill *spec.Skill) bool {
	// 使用类型断言
	if _, ok := adpt.(*cursor.CursorAdapter); ok {
		return skill.Compatibility.Cursor
	}
	if _, ok := adpt.(*claude.ClaudeAdapter); ok {
		return skill.Compatibility.ClaudeCode
	}
	return false
}

// renderTemplate 渲染模板内容
func renderTemplate(content string, variables map[string]string) (string, error) {
	// 简单替换变量
	result := content
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}
