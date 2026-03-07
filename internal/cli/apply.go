package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/internal/adapter"
	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
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
	ctx, err := RequireInitAndWorkspace("", "")
	if err != nil {
		return err
	}

	fmt.Println("正在应用技能到项目...")

	projectState := ctx.ProjectState
	if projectState == nil || projectState.PreferredTarget == "" {
		return errors.NewWithCode("runApply", errors.ErrProjectInvalid,
			"项目未设置目标环境，请先使用 'skill-hub set-target <value>' 设置目标环境")
	}

	target := spec.NormalizeTarget(projectState.PreferredTarget)
	fmt.Printf("项目目标环境: %s\n", target)
	fmt.Printf("项目路径: %s\n", ctx.Cwd)

	skills, err := ctx.StateManager.GetProjectSkills(ctx.Cwd)
	if err != nil {
		return errors.Wrap(err, "runApply: 获取项目技能失败")
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
		return errors.WrapWithCode(err, "runApply", errors.ErrSystem, "获取适配器失败")
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

func getSkillContent(skillID string) (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "获取配置失败")
	}

	var repoPath string
	if cfg.MultiRepo != nil {
		rootDir, err := config.GetRootDir()
		if err != nil {
			return "", errors.Wrap(err, "获取根目录失败")
		}
		repoPath = filepath.Join(rootDir, "repositories", cfg.MultiRepo.DefaultRepo)
	} else {
		return "", errors.NewWithCode("getSkillContent", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	srcPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", errors.NewWithCodef("getSkillContent", errors.ErrFileNotFound, "技能文件在仓库中不存在: %s", srcPath)
	}

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", errors.WrapWithCode(err, "getSkillContent", errors.ErrFileOperation, "读取技能文件失败")
	}

	return string(content), nil
}
