package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"skill-hub/internal/adapter"
	"skill-hub/internal/config"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "应用技能到项目",
	Long: `根据 state.json 中的启用记录和目标环境设置，将技能物理分发到项目。具体行为取决于项目工作区设置的目标环境：
- cursor: 注入到 .cursorrules 文件
- claude: 更新 Claude 配置文件
- open_code: 复制技能到项目工作区 .agents/skills/ 目录`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		return runApply(dryRun, force)
	},
}

func init() {
	applyCmd.Flags().Bool("dry-run", false, "演习模式，仅显示将要执行的变更，不实际修改文件")
	applyCmd.Flags().Bool("force", false, "强制应用，即使检测到冲突也继续执行")
}

func runApply(dryRun bool, force bool) error {
	fmt.Println("正在应用技能到项目...")

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

	// 获取项目状态
	projectState, err := stateMgr.FindProjectByPath(cwd)
	if err != nil {
		return fmt.Errorf("查找项目状态失败: %w", err)
	}

	if projectState == nil || projectState.PreferredTarget == "" {
		return fmt.Errorf("项目未设置目标环境，请先使用 'skill-hub set-target <value>' 设置目标环境")
	}

	target := spec.NormalizeTarget(projectState.PreferredTarget)
	fmt.Printf("项目目标环境: %s\n", target)
	fmt.Printf("项目路径: %s\n", cwd)

	// 获取项目启用的技能
	skills, err := stateMgr.GetProjectSkills(cwd)
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		fmt.Println("使用 'skill-hub use <skill-id>' 启用技能")
		return nil
	}

	fmt.Printf("启用技能数: %d\n", len(skills))

	if dryRun {
		fmt.Println("\n=== 演习模式 (dry-run) ===")
		fmt.Println("将显示将要执行的变更，不实际修改文件")
	}

	// 根据目标环境应用技能
	adapter, err := adapter.GetAdapterForTarget(target)
	if err != nil {
		return fmt.Errorf("获取适配器失败: %w", err)
	}

	// 设置为项目模式
	adapter.SetProjectMode()

	// 应用所有技能
	for skillID, skillVars := range skills {
		fmt.Printf("应用技能: %s\n", skillID)

		// 从仓库获取技能内容
		content, err := getSkillContent(skillID)
		if err != nil {
			fmt.Printf("⚠️  获取技能内容失败: %s: %v\n", skillID, err)
			continue
		}

		if dryRun {
			fmt.Printf("  [演习] 将应用技能到: %s\n", target)
			fmt.Printf("  变量: %v\n", skillVars.Variables)
		} else {
			// 实际应用技能
			if err := adapter.Apply(skillID, content, skillVars.Variables); err != nil {
				fmt.Printf("⚠️  应用技能失败: %s: %v\n", skillID, err)
			} else {
				fmt.Printf("✓ 成功应用技能: %s\n", skillID)
			}
		}
	}

	if dryRun {
		fmt.Println("\n✅ 演习完成，未实际修改文件")
	} else {
		fmt.Println("\n✅ 所有技能应用完成")
	}

	return nil
}

// applyToCursor 应用技能到Cursor
func applyToCursor(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到 Cursor ===")

	// 检查.cursorrules文件
	cursorRulesPath := filepath.Join(projectPath, ".cursorrules")

	if dryRun {
		fmt.Printf("将更新文件: %s\n", cursorRulesPath)
		fmt.Println("将注入以下技能:")
		for skillID := range skills {
			fmt.Printf("  - %s\n", skillID)
		}
		fmt.Println("\n注意: 实际实现需要将技能内容注入到.cursorrules文件中")
		return nil
	}

	// TODO: 实际实现 - 将技能内容注入到.cursorrules文件中
	fmt.Println("⚠️  Cursor适配器功能暂未完全实现")
	fmt.Println("将创建/更新.cursorrules文件并注入技能内容")

	return nil
}

// applyToClaude 应用技能到Claude
func applyToClaude(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到 Claude ===")

	// 检查Claude配置文件路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}
	claudeConfigPath := filepath.Join(homeDir, ".claude", "config.json")

	if dryRun {
		fmt.Printf("将更新配置文件: %s\n", claudeConfigPath)
		fmt.Println("将注入以下技能:")
		for skillID := range skills {
			fmt.Printf("  - %s\n", skillID)
		}
		fmt.Println("\n注意: 实际实现需要更新Claude配置文件")
		return nil
	}

	// TODO: 实际实现 - 更新Claude配置文件
	fmt.Println("⚠️  Claude适配器功能暂未完全实现")
	fmt.Println("将更新Claude配置文件以包含技能内容")

	return nil
}

// applyToProjectWorkspace 应用技能到项目工作区
func applyToProjectWorkspace(projectPath string, skills map[string]spec.SkillVars, dryRun bool, force bool) error {
	fmt.Println("\n=== 应用技能到项目工作区 ===")

	// 创建.agents/skills目录
	agentsSkillsDir := filepath.Join(projectPath, ".agents", "skills")

	if dryRun {
		fmt.Printf("将创建/更新目录: %s\n", agentsSkillsDir)
		fmt.Println("将创建以下技能目录:")
		for skillID := range skills {
			skillDir := filepath.Join(agentsSkillsDir, skillID)
			fmt.Printf("  - %s\n", skillDir)
		}
		fmt.Println("\n注意: 实际实现需要创建.agents/skills/[id]/目录结构")
		return nil
	}

	// 创建.agents/skills目录
	if err := os.MkdirAll(agentsSkillsDir, 0755); err != nil {
		return fmt.Errorf("创建.agents/skills目录失败: %w", err)
	}
	fmt.Printf("✓ 创建目录: %s\n", agentsSkillsDir)

	// 为每个技能创建目录和文件
	createdCount := 0
	for skillID := range skills {
		skillDir := filepath.Join(agentsSkillsDir, skillID)
		skillMdPath := filepath.Join(skillDir, "SKILL.md")

		// 检查目录是否已存在
		dirExists := false
		if _, err := os.Stat(skillDir); err == nil {
			dirExists = true
			if !force {
				fmt.Printf("⚠️  技能目录已存在: %s (使用 --force 覆盖)\n", skillDir)
			} else {
				// 强制模式，删除现有目录
				if err := os.RemoveAll(skillDir); err != nil {
					fmt.Printf("⚠️  删除现有目录失败: %s: %v\n", skillDir, err)
				} else {
					dirExists = false // 标记为已删除
				}
			}
		}

		// 如果目录不存在或强制覆盖，创建目录和文件
		if !dirExists {
			// 创建技能目录
			if err := os.MkdirAll(skillDir, 0755); err != nil {
				fmt.Printf("⚠️  创建技能目录失败: %s: %v\n", skillDir, err)
			} else {
				// 从仓库复制技能文件
				if err := copySkillFromRepo(skillID, skillMdPath); err != nil {
					fmt.Printf("⚠️  创建SKILL.md文件失败: %s: %v\n", skillMdPath, err)
				} else {
					fmt.Printf("✓ 创建技能目录: %s\n", skillDir)
					createdCount++
				}
			}
		} else {
			// 目录已存在且未强制覆盖，但检查文件是否存在
			if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
				// 目录存在但文件不存在，复制文件
				if err := copySkillFromRepo(skillID, skillMdPath); err != nil {
					fmt.Printf("⚠️  补充SKILL.md文件失败: %s: %v\n", skillMdPath, err)
				} else {
					fmt.Printf("✓ 补充技能文件: %s\n", skillMdPath)
					createdCount++
				}
			} else {
				// 目录和文件都已存在，检查是否需要更新
				// 这里可以添加更复杂的逻辑，比如比较版本等
				fmt.Printf("ℹ️  技能文件已存在: %s\n", skillMdPath)
			}
		}
	}

	fmt.Printf("\n✅ 成功创建/更新 %d 个技能\n", createdCount)
	fmt.Println("技能已应用到项目工作区")
	fmt.Println("使用 'skill-hub status' 检查技能状态")

	return nil
}

// getSkillContent 从仓库获取技能内容
func getSkillContent(skillID string) (string, error) {
	// 获取配置
	cfg, err := config.GetConfig()
	if err != nil {
		return "", fmt.Errorf("获取配置失败: %w", err)
	}

	// 展开repo路径中的~符号
	repoPath := cfg.RepoPath
	if repoPath == "" {
		return "", fmt.Errorf("仓库路径未配置")
	}

	// 处理~符号
	if repoPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取用户主目录失败: %w", err)
		}
		repoPath = filepath.Join(homeDir, repoPath[1:])
	}

	// 构建源文件路径
	srcPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", fmt.Errorf("技能文件在仓库中不存在: %s", srcPath)
	}

	// 读取文件内容
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("读取技能文件失败: %w", err)
	}

	return string(content), nil
}

// copySkillFromRepo 从仓库复制技能文件
func copySkillFromRepo(skillID, destPath string) error {
	// 获取配置
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}

	// 展开repo路径中的~符号
	repoPath := cfg.RepoPath
	if repoPath == "" {
		return fmt.Errorf("仓库路径未配置")
	}

	// 处理~符号
	if repoPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}
		repoPath = filepath.Join(homeDir, repoPath[1:])
	}

	// 构建源文件路径
	srcPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")

	// 检查源文件是否存在
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("技能文件在仓库中不存在: %s", srcPath)
	}

	// 复制文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	return nil
}
