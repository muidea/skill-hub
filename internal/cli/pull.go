package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
)

var (
	pullForce bool
	pullCheck bool
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "从默认仓库拉取最新技能",
	Long: `从默认仓库（归档仓库）对应的远程拉取最新更改到本地仓库，并更新技能注册表。

此命令仅处理默认仓库，不负责多仓库同步；多仓库同步请使用 'skill-hub repo sync'。
此命令仅同步仓库层（~/.skill-hub/repositories/），不涉及项目工作目录的更新。
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
		fmt.Println("检查默认仓库远程的可用更新...")
		// 这里可以实现检查逻辑，显示可用的更新数量
		// 暂时简单实现
		fmt.Println("检查功能待实现，使用 'skill-hub pull' 直接拉取更新")
		return nil
	}

	fmt.Println("正在从默认仓库远程拉取最新技能...")

	if err := syncSkillRepositoryAndRefresh(); err != nil {
		return errors.Wrap(err, "同步技能仓库失败")
	}

	// 获取更新后的技能列表
	repo, err := newSkillRepository()
	if err != nil {
		return err
	}
	skills, err := repo.ListLocalSkills()
	if err != nil {
		return errors.Wrap(err, "获取技能列表失败")
	}

	fmt.Printf("\n✅ 默认仓库更新完成，共 %d 个技能\n", len(skills))
	fmt.Println("使用 'skill-hub status' 检查项目技能状态")
	fmt.Println("使用 'skill-hub apply' 将仓库更新应用到项目工作目录")
	fmt.Println("如需同步所有启用仓库，请使用 'skill-hub repo sync'")

	return nil
}
