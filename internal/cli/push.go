package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/git"
)

var (
	pushMessage string
	pushForce   bool
	pushDryRun  bool
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "推送本地更改到远程仓库",
	Long: `自动检测并提交所有未提交的更改，然后推送到远程技能仓库。

此命令将本地仓库（~/.skill-hub/repositories/）中的更改同步到远程仓库，完成反馈闭环。
使用 --dry-run 选项可以查看将要推送的更改而不实际执行。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush()
	},
}

func init() {
	pushCmd.Flags().StringVarP(&pushMessage, "message", "m", "", "提交消息。如未提供，使用默认消息\"更新技能\"")
	pushCmd.Flags().BoolVar(&pushForce, "force", false, "强制推送，跳过确认检查")
	pushCmd.Flags().BoolVar(&pushDryRun, "dry-run", false, "演习模式，仅显示将要推送的更改，不实际执行")
}

func runPush() error {
	// 检查init依赖（规范4.13：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 初始化技能仓库
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	// 获取仓库状态
	status, err := repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	// 检查是否有未提交的更改
	hasChanges := strings.Contains(status, " M ") || strings.Contains(status, "?? ") || strings.Contains(status, " D ")
	if !hasChanges {
		fmt.Println("没有要推送的更改")
		return nil
	}

	// 显示将要推送的更改
	fmt.Println("检测到以下未提交的更改:")
	fmt.Println(strings.Repeat("-", 50))

	// 解析状态输出，显示简要信息
	lines := strings.Split(status, "\n")
	changesFound := false
	for _, line := range lines {
		if strings.HasPrefix(line, " M ") || strings.HasPrefix(line, "?? ") || strings.HasPrefix(line, " D ") {
			fmt.Println(line)
			changesFound = true
		}
	}

	if !changesFound {
		// 如果状态格式不同，显示原始状态
		fmt.Println(status)
	}

	fmt.Println(strings.Repeat("-", 50))

	// 演习模式
	if pushDryRun {
		fmt.Println("演习模式：仅显示将要推送的更改，不实际执行")
		fmt.Println("使用 'skill-hub push'（不带 --dry-run）实际推送更改")
		return nil
	}

	// 确认推送（除非使用 --force）
	if !pushForce {
		fmt.Print("\n是否推送这些更改到远程仓库？ [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("取消推送操作")
			return nil
		}
	}

	// 确定提交消息
	message := pushMessage
	if message == "" {
		message = "更新技能"
	}

	fmt.Println("正在提交并推送更改...")

	// 使用技能仓库的PushChanges方法
	if err := repo.PushChanges(message); err != nil {
		return fmt.Errorf("推送失败: %w", err)
	}

	fmt.Println("✅ 更改已成功推送到远程仓库")
	return nil
}
