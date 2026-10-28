package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	projectport "github.com/muidea/skill-hub/internal/pkg/projectlifecycleport"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/utils"
)

var registerCmd = &cobra.Command{
	Use:   "register <id>",
	Short: "登记已有本地技能",
	Long: `将当前项目中已有的 .agents/skills/<id>/SKILL.md 登记到 skill-hub 项目状态。

register 不会创建或覆盖任何技能内容。默认会先验证 SKILL.md，使用 --skip-validate 可跳过验证。`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeLocalSkillIDsForCwd,
	RunE: func(cmd *cobra.Command, args []string) error {
		skipValidate, _ := cmd.Flags().GetBool("skip-validate")
		return runRegister(args[0], skipValidate)
	},
}

func init() {
	registerCmd.Flags().Bool("skip-validate", false, "跳过SKILL.md验证，直接登记项目状态")
}

func runRegister(skillID string, skipValidate bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	var result *projectport.RegisterResult
	if client, ok := hubClientIfAvailable(); ok {
		data, err := client.RegisterSkill(context.Background(), httpapibiz.RegisterSkillRequest{
			ProjectPath:  cwd,
			SkillID:      skillID,
			SkipValidate: skipValidate,
		})
		if err != nil {
			return errors.Wrap(err, "通过服务登记技能失败")
		}
		result = data.Item
	} else {
		if err := CheckInitDependency(); err != nil {
			return err
		}
		ctx, err := RequireInitAndWorkspace(cwd)
		if err != nil {
			return err
		}
		result, err = localRuntime.Project().Register(ctx.Cwd, skillID, skipValidate)
		if err != nil {
			return err
		}
	}

	if jsonMode() {
		return writeJSON(result)
	}
	renderRegisterResult(result)
	return nil
}

func renderRegisterResult(result *projectport.RegisterResult) {
	if result == nil {
		printlnText("✅ 技能已登记到项目状态")
		return
	}
	if result.Registered {
		printfText("✅ 技能 '%s' 已登记到项目状态\n", result.SkillID)
	} else {
		printfText("✅ 技能 '%s' 已刷新项目状态\n", result.SkillID)
	}
	printlnText("说明: register 仅登记已有内容，不会创建或覆盖 SKILL.md")
	printlnText("\n下一步:")
	printfText("1. 使用 'skill-hub validate %s' 验证技能合规性\n", result.SkillID)
	printfText("2. 使用 'skill-hub feedback %s' 将技能反馈到仓库\n", result.SkillID)
}
