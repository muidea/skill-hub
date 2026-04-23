package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	globalservice "github.com/muidea/skill-hub/internal/modules/kernel/global/service"
	projectapplyservice "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var applyCmd = &cobra.Command{
	Use:   "apply [id]",
	Short: "应用技能到项目",
	Long:  `根据 state.json 中的启用记录，将技能物理分发到当前项目的标准 .agents/skills 目录。`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		global, _ := cmd.Flags().GetBool("global")
		agents, _ := cmd.Flags().GetStringArray("agent")
		if global {
			skillID := ""
			if len(args) > 0 {
				skillID = args[0]
			}
			return runApplyGlobal(skillID, agents, dryRun, force)
		}
		if len(args) > 0 {
			return errors.NewWithCode("apply", errors.ErrInvalidInput, "项目级 apply 不接受技能 ID；如需刷新全局单个技能，请使用 --global")
		}
		return runApply(dryRun, force)
	},
}

func init() {
	applyCmd.Flags().Bool("dry-run", false, "演习模式，仅显示将要执行的变更，不实际修改文件")
	applyCmd.Flags().Bool("force", false, "强制应用，即使检测到冲突也继续执行")
	applyCmd.Flags().Bool("global", false, "应用本机全局启用的技能")
	applyCmd.Flags().StringArray("agent", nil, "限制全局应用的 agent，可重复使用: codex, opencode, claude")
	_ = applyCmd.RegisterFlagCompletionFunc("agent", completeAgentNames)
}

func runApplyGlobal(skillID string, agents []string, dryRun bool, force bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		return runApplyGlobalViaService(client, skillID, agents, dryRun, force)
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}
	result, err := globalservice.New().Apply(skillID, agents, dryRun, force)
	if err != nil {
		return errors.Wrap(err, "应用全局技能失败")
	}
	renderGlobalApplyResult(result)
	return nil
}

type serviceGlobalApplyClient interface {
	ApplyGlobal(ctx context.Context, req httpapibiz.ApplyGlobalRequest) (*httpapibiz.ApplyGlobalData, error)
}

func runApplyGlobalViaService(client serviceGlobalApplyClient, skillID string, agents []string, dryRun bool, force bool) error {
	resp, err := client.ApplyGlobal(context.Background(), httpapibiz.ApplyGlobalRequest{
		SkillID: skillID,
		Agents:  agents,
		DryRun:  dryRun,
		Force:   force,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务应用全局技能失败")
	}
	renderGlobalApplyResult(resp.Item)
	return nil
}

func runApply(dryRun bool, force bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		return runApplyViaService(client, dryRun, force)
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	fmt.Println("正在应用技能到项目...")

	projectState := ctx.ProjectState
	if projectState == nil {
		return errors.NewWithCode("runApply", errors.ErrProjectInvalid, "项目状态无效")
	}

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

	// 应用所有技能
	for skillID, skillVars := range skills {
		fmt.Printf("应用技能: %s\n", skillID)

		sourceRepository := skillVars.SourceRepository
		if sourceRepository == "" {
			defaultRepo, err := defaultRepository()
			if err != nil {
				fmt.Printf("⚠️  获取默认仓库失败: %v\n", err)
				continue
			}
			if defaultRepo != nil {
				sourceRepository = defaultRepo.Name
			}
		}

		if dryRun {
			fmt.Println("  [演习] 将应用技能到标准项目技能目录")
			fmt.Printf("  变量: %v\n", skillVars.Variables)
		} else {
			if err := copyRepositorySkillToProject(ctx.Cwd, sourceRepository, skillID); err != nil {
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

func copyRepositorySkillToProject(projectPath, repoName, skillID string) error {
	repoPath, err := repositoryPath(repoName)
	if err != nil {
		return errors.Wrap(err, "copyRepositorySkillToProject: 获取仓库路径失败")
	}
	srcDir := filepath.Join(repoPath, "skills", skillID)
	if _, err := os.Stat(filepath.Join(srcDir, "SKILL.md")); err != nil {
		if os.IsNotExist(err) {
			return errors.NewWithCodef("copyRepositorySkillToProject", errors.ErrFileNotFound, "技能文件在仓库中不存在: %s", srcDir)
		}
		return errors.Wrap(err, "copyRepositorySkillToProject: 检查仓库技能失败")
	}
	dstDir := filepath.Join(projectPath, ".agents", "skills", skillID)
	return copySkillDirectory(srcDir, dstDir)
}

type serviceApplyClient interface {
	ApplyProject(ctx context.Context, req httpapibiz.ApplyProjectRequest) (*httpapibiz.ApplyProjectData, error)
}

func runApplyViaService(client serviceApplyClient, dryRun bool, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	resp, err := client.ApplyProject(context.Background(), httpapibiz.ApplyProjectRequest{
		ProjectPath: cwd,
		DryRun:      dryRun,
		Force:       force,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务应用技能失败")
	}

	renderApplyResult(resp.Item)
	return nil
}

func renderApplyResult(result *projectapplyservice.ApplyResult) {
	fmt.Println("正在应用技能到项目...")
	if result == nil {
		fmt.Println("ℹ️  未返回应用结果")
		return
	}

	fmt.Printf("项目路径: %s\n", result.ProjectPath)

	if len(result.Items) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		fmt.Println("使用 'skill-hub use <skill-id>' 启用技能")
		return
	}

	fmt.Printf("启用技能数: %d\n", len(result.Items))
	if result.DryRun {
		fmt.Println("\n=== 演习模式 (dry-run) ===")
		fmt.Println("将显示将要执行的变更，不实际修改文件")
	}

	for _, item := range result.Items {
		fmt.Printf("应用技能: %s\n", item.SkillID)
		switch item.Status {
		case "planned":
			fmt.Println("  [演习] 将应用技能到标准项目技能目录")
			fmt.Printf("  变量数量: %d\n", item.Variables)
		case "applied":
			fmt.Printf("✓ 成功应用技能: %s\n", item.SkillID)
		case "error":
			fmt.Printf("⚠️  应用技能失败: %s: %s\n", item.SkillID, item.Message)
		default:
			fmt.Printf("ℹ️  状态: %s\n", item.Status)
		}
	}

	if result.DryRun {
		fmt.Println("\n✅ 演习完成，未实际修改文件")
	} else {
		fmt.Println("\n✅ 所有技能应用完成")
	}
}

func renderGlobalApplyResult(result *globalservice.ApplyResult) {
	fmt.Println("正在刷新本机全局技能...")
	if result == nil {
		fmt.Println("ℹ️  未返回应用结果")
		return
	}
	fmt.Printf("全局镜像目录: %s\n", result.GlobalPath)
	if result.DryRun {
		fmt.Println("\n=== 演习模式 (dry-run) ===")
	}
	if len(result.Items) == 0 {
		fmt.Println("ℹ️  当前未启用任何全局技能")
		fmt.Println("使用 'skill-hub use <skill-id> --global --agent <agent>' 启用全局技能")
		return
	}
	for _, item := range result.Items {
		target := item.Agent
		if target == "" {
			target = "unknown"
		}
		switch item.Status {
		case globalservice.StatusPlanned:
			fmt.Printf("[计划] %s -> %s: %s\n", item.SkillID, target, item.TargetPath)
		case globalservice.StatusApplied:
			fmt.Printf("✓ 已刷新 %s -> %s\n", item.SkillID, target)
		case globalservice.StatusConflict:
			fmt.Printf("⚠️  冲突 %s -> %s: %s\n", item.SkillID, target, item.Message)
		case globalservice.StatusError:
			fmt.Printf("❌ 失败 %s -> %s: %s\n", item.SkillID, target, item.Message)
		default:
			fmt.Printf("ℹ️  %s -> %s: %s %s\n", item.SkillID, target, item.Status, item.Message)
		}
	}
	if result.DryRun {
		fmt.Println("\n✅ 演习完成，未实际修改文件")
	} else {
		fmt.Println("\n✅ 全局技能刷新完成")
	}
}
