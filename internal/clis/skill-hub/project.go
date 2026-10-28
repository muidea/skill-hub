package cli

import (
	"os"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{Use: "project", Short: "管理项目工作区"}

var projectRegisterCmd = &cobra.Command{
	Use:   "register [path]",
	Short: "幂等登记项目工作区或查询原请求回执",
	Long: `登记项目工作区，无需预先存在技能，不扫描或导入已有技能。
path 默认为当前目录。必须提供由调用方保存的 --request-id。
相同 ID 和项目路径重复提交会返回原始回执，不重新执行；不同路径拒绝复用 ID。

使用 --query --request-id <ID> 查询原请求回执；查询不接受 path，
不读取当前项目状态、不创建工作区，也不重新执行登记。
回执保存在 SKILL_HUB_HOME 中；仅在同一 Hub home 内可查询和去重。`,
	Args: func(cmd *cobra.Command, args []string) error {
		if mustGetBoolFlag(cmd, "query") && len(args) != 0 {
			return errors.NewWithCode("project.register.query", errors.ErrInvalidInput, "回执查询仅使用 request-id，不接受项目路径")
		}
		return cobra.MaximumNArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		requestID, _ := cmd.Flags().GetString("request-id")
		query := mustGetBoolFlag(cmd, "query")
		var receipt *spec.ProjectRegistrationReceipt
		var err error
		if query {
			receipt, err = localRuntime.ProjectRegistration().QueryProjectRegistration(requestID)
		} else {
			if err := CheckInitDependency(); err != nil {
				return err
			}
			var projectPath string
			if len(args) == 1 {
				projectPath = args[0]
			} else {
				projectPath, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			receipt, err = localRuntime.ProjectRegistration().RegisterEmptyProject(projectPath, requestID)
		}
		if receipt != nil {
			if jsonMode() {
				if writeErr := writeJSON(receipt); writeErr != nil {
					return writeErr
				}
			} else {
				printfText("request-id: %s\n项目: %s\n请求状态: %s\n", receipt.RequestID, receipt.ProjectPath, receipt.Status)
				if receipt.Result != nil {
					printfText("新建项目: %t，登记技能数: %d\n", receipt.Result.ProjectCreated, receipt.Result.SkillCount)
				}
				if receipt.CommandError != nil {
					printlnText(receipt.CommandError.Message)
				}
			}
			if !query && err == nil && receipt.CommandError != nil {
				err = errors.NewWithCode("project.register", receipt.CommandError.Code, receipt.CommandError.Message)
			}
		}
		return err
	},
}

func init() {
	projectRegisterCmd.Flags().String("request-id", "", "调用方生成并保留的请求身份（同一 Hub home 内唯一）")
	projectRegisterCmd.Flags().Bool("query", false, "只查询该 request-id 的原始回执，不执行登记")
	_ = projectRegisterCmd.MarkFlagRequired("request-id")
	projectCmd.AddCommand(projectRegisterCmd)
}
