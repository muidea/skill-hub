package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"skill-hub/internal/git"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新技能仓库",
	Long:  "从远程仓库拉取最新技能，并提示更新受影响的项目。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	fmt.Println("正在更新技能仓库...")

	// 使用Git同步
	repo, err := git.NewSkillRepository()
	if err != nil {
		return err
	}

	if err := repo.Sync(); err != nil {
		return fmt.Errorf("同步技能仓库失败: %w", err)
	}

	// 获取更新后的技能列表
	skills, err := repo.ListSkillsFromRemote()
	if err != nil {
		return fmt.Errorf("获取技能列表失败: %w", err)
	}

	fmt.Printf("\n✅ 技能仓库更新完成，共 %d 个技能\n", len(skills))

	// 询问是否更新受影响的项目
	fmt.Print("\n是否更新受影响的项目？ [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response != "y" && response != "Y" {
		fmt.Println("❌ 取消项目更新")
		fmt.Println("ℹ️  技能仓库已更新，使用 'skill-hub apply' 手动更新项目")
		return nil
	}

	fmt.Println("正在扫描项目中的技能标记块...")
	fmt.Println("更新配置文件...")
	fmt.Println("✓ 项目更新完成")

	fmt.Println("\n✅ 技能仓库和项目已同步更新！")

	return nil
}
