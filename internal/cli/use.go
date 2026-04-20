package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <id>",
	Short: "使用技能",
	Long: `将技能标记为在当前项目中使用。此命令仅更新 state.json 中的状态记录，不直接修改项目文件。
需要通过 apply 命令进行物理分发。

如果项目工作区里首次使用技能，也会同步在state.json里完成项目工作区信息刷新`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		return runUse(args[0], target)
	},
}

func init() {
	useCmd.Flags().String("target", "open_code", targetFlagUsage)
	useCmd.ValidArgsFunction = completeSkillIDs
	_ = useCmd.RegisterFlagCompletionFunc("target", completeTargetValues)
}

func runUse(skillID string, target string) error {
	target = spec.NormalizeTarget(target)

	if client, ok := hubClientIfAvailable(); ok {
		return runUseViaService(client, skillID, target)
	}

	if err := CheckInitDependency(); err != nil {
		return err
	}

	repoManager, err := newRepositoryManager()
	if err != nil {
		return errors.Wrap(err, "创建多仓库管理器失败")
	}

	skills, err := repoManager.FindSkill(skillID)
	if err != nil {
		return errors.Wrap(err, "查找技能失败")
	}

	// 如果没有找到任何技能
	if len(skills) == 0 {
		return errors.SkillNotFound("runUse", skillID)
	}

	// 如果只有一个技能，直接使用
	selectedSkill, err := chooseSkillCandidate(skills)
	if err != nil {
		return err
	}

	// 加载完整技能信息
	fullSkill, err := repoManager.LoadSkill(skillID, selectedSkill.Repository)
	if err != nil {
		return errors.Wrap(err, "加载技能详情失败")
	}

	fmt.Printf("启用技能: %s (%s)\n", fullSkill.Name, skillID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)

	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	ctx, err := RequireInitAndWorkspace("", target)
	if err != nil {
		return err
	}

	hasSkill, err := ctx.StateManager.ProjectHasSkill(ctx.Cwd, skillID)
	if err != nil {
		return err
	}

	if hasSkill {
		if !confirmSkillReconfigure() {
			fmt.Println("❌ 取消操作")
			return nil
		}
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	if err := ctx.StateManager.AddSkillToProjectWithTarget(ctx.Cwd, skillID, fullSkill.Version, selectedSkill.Repository, variables, target); err != nil {
		return errors.Wrap(err, "保存项目状态失败")
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为使用！\n", skillID)

	fmt.Printf("项目目标: %s\n", target)
	fmt.Println("使用 'skill-hub apply' 将技能物理分发到当前项目")

	return nil
}

func runUseViaService(client serviceUseClient, skillID string, target string) error {
	target = spec.NormalizeTarget(target)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	candidates, err := client.FindSkillCandidates(context.Background(), skillID)
	if err != nil {
		return errors.Wrap(err, "通过服务查找技能失败")
	}
	if len(candidates) == 0 {
		return errors.SkillNotFound("runUseViaService", skillID)
	}

	selectedSkill, err := chooseSkillCandidate(candidates)
	if err != nil {
		return err
	}

	fullSkill, err := client.GetSkillDetail(context.Background(), skillID, selectedSkill.Repository)
	if err != nil {
		return errors.Wrap(err, "通过服务加载技能详情失败")
	}

	fmt.Printf("启用技能: %s (%s)\n", fullSkill.Name, skillID)
	fmt.Printf("来源仓库: %s\n", fullSkill.Repository)
	fmt.Printf("描述: %s\n", fullSkill.Description)
	if len(fullSkill.Tags) > 0 {
		fmt.Printf("标签: %s\n", strings.Join(fullSkill.Tags, ", "))
	}

	if projectStatus, err := client.GetProjectStatus(context.Background(), cwd, skillID); err == nil && projectStatus.Item != nil && len(projectStatus.Item.Items) > 0 {
		if !confirmSkillReconfigure() {
			fmt.Println("❌ 取消操作")
			return nil
		}
	}

	variables, err := promptSkillVariables(fullSkill)
	if err != nil {
		return err
	}

	resp, err := client.UseSkill(context.Background(), httpapibiz.UseSkillRequest{
		ProjectPath: cwd,
		SkillID:     skillID,
		Repository:  selectedSkill.Repository,
		Target:      target,
		Variables:   variables,
	})
	if err != nil {
		return errors.Wrap(err, "通过服务启用技能失败")
	}

	fmt.Printf("\n✅ 技能 '%s' 已成功标记为使用！\n", skillID)
	fmt.Printf("项目目标: %s\n", resp.Item.Target)
	fmt.Println("使用 'skill-hub apply' 将技能物理分发到当前项目")
	return nil
}

type serviceUseClient interface {
	FindSkillCandidates(ctx context.Context, skillID string) ([]spec.SkillMetadata, error)
	GetSkillDetail(ctx context.Context, skillID, repoName string) (*spec.Skill, error)
	GetProjectStatus(ctx context.Context, projectPath, skillID string) (*httpapibiz.ProjectStatusData, error)
	UseSkill(ctx context.Context, req httpapibiz.UseSkillRequest) (*httpapibiz.UseSkillData, error)
}

func chooseSkillCandidate(skills []spec.SkillMetadata) (spec.SkillMetadata, error) {
	if len(skills) == 1 {
		return skills[0], nil
	}

	fmt.Printf("发现 %d 个同名技能，请选择要使用的技能:\n", len(skills))
	for i, skill := range skills {
		fmt.Printf("  %d. [%s] %s - %s\n", i+1, skill.Repository, skill.Name, skill.Description)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请选择 (输入编号): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(skills) {
		return spec.SkillMetadata{}, errors.NewWithCode("chooseSkillCandidate", errors.ErrInvalidInput, "无效的选择")
	}

	return skills[choice-1], nil
}

func promptSkillVariables(fullSkill *spec.Skill) (map[string]string, error) {
	variables := make(map[string]string)
	if len(fullSkill.Variables) == 0 {
		fmt.Println("\n该技能没有可配置的变量")
		return variables, nil
	}

	fmt.Println("\n请设置技能变量 (按Enter使用默认值):")
	reader := bufio.NewReader(os.Stdin)
	for _, variable := range fullSkill.Variables {
		defaultValue := variable.Default
		fmt.Printf("%s [%s]: ", variable.Name, defaultValue)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			variables[variable.Name] = defaultValue
		} else {
			variables[variable.Name] = input
		}
	}
	return variables, nil
}

func confirmSkillReconfigure() bool {
	fmt.Println("⚠️  该技能已在当前项目启用")
	fmt.Print("是否重新配置变量？ [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)
	return response == "y" || response == "Y"
}
