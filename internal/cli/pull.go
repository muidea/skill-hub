package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"skill-hub/internal/git"
)

var (
	pullForce bool
	pullCheck bool
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "从远程仓库拉取最新技能",
	Long: `从远程技能仓库拉取最新更改到本地仓库，并更新技能注册表。

此命令仅同步仓库层（~/.skill-hub/repo/），不涉及项目工作目录的更新。
使用 --check 选项可以检查可用更新但不实际执行拉取操作。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull()
	},
}

func init() {
	pullCmd.Flags().BoolVar(&pullForce, "force", false, "强制拉取，忽略本地未提交的修改")
	pullCmd.Flags().BoolVar(&pullCheck, "check", false, "检查模式，仅显示可用的更新，不实际执行拉取操作")
}

func runPull() error {
	// 检查init依赖（规范4.12：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	if pullCheck {
		fmt.Println("检查远程仓库可用的更新...")
		// 这里可以实现检查逻辑，显示可用的更新数量
		// 暂时简单实现
		fmt.Println("检查功能待实现，使用 'skill-hub pull' 直接拉取更新")
		return nil
	}

	fmt.Println("正在从远程仓库拉取最新技能...")

	// 使用Git同步
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	// 同步仓库
	if err := repo.Sync(); err != nil {
		return fmt.Errorf("同步技能仓库失败: %w", err)
	}

	// 更新技能注册表
	fmt.Println("更新技能注册表...")
	if err := repo.UpdateRegistry(); err != nil {
		fmt.Printf("警告: 更新技能注册表失败: %v\n", err)
		fmt.Println("技能已拉取，但注册表未更新")
	}

	// 获取更新后的技能列表
	skills, err := repo.ListSkillsFromRemote()
	if err != nil {
		return fmt.Errorf("获取技能列表失败: %w", err)
	}

	fmt.Printf("\n✅ 技能仓库更新完成，共 %d 个技能\n", len(skills))
	fmt.Println("使用 'skill-hub status' 检查项目技能状态")
	fmt.Println("使用 'skill-hub apply' 将仓库更新应用到项目工作目录")

	return nil
}
