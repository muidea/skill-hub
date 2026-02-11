package cli

import (
	"fmt"
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
	// 检查init依赖（规范4.10：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Println("正在应用技能到项目...")

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目工作区状态（规范4.10：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, "")
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
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

// copySkillToProject 复制整个技能目录到项目
func copySkillToProject(skillID, projectPath string) error {
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

	// 源技能目录
	srcSkillDir := filepath.Join(repoPath, "skills", skillID)

	// 目标技能目录
	dstSkillDir := filepath.Join(projectPath, ".agents", "skills", skillID)

	// 确保目标目录存在
	if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 复制整个技能目录
	return copyDirectory(srcSkillDir, dstSkillDir)
}

// copyDirectory 复制整个目录
func copyDirectory(srcDir, dstDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 遍历源目录
	return filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 目标路径
		dstPath := filepath.Join(dstDir, relPath)

		// 如果是目录，创建目录
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 如果是文件，复制文件
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %w", srcPath, err)
		}

		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return fmt.Errorf("写入文件失败 %s: %w", dstPath, err)
		}

		return nil
	})
}
