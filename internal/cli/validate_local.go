package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
)

var validateCmd = &cobra.Command{
	Use:               "validate <id>",
	Short:             "验证技能合规性",
	Long:              `验证指定技能的项目本地工作区的文件是否合规，包括检查 SKILL.md 的 YAML 语法、必需字段、文件结构等。验证范围包括项目本地文件和仓库源文件。如果该技能未在state.json里项目工作区登记，则提示该技能非法`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeEnabledSkillIDsForCwd,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0])
	},
}

func runValidate(skillID string) error {
	ctx, err := RequireInitAndWorkspace("", "")
	if err != nil {
		return err
	}

	fmt.Printf("验证技能合规性: %s\n", skillID)

	hasSkill, err := ctx.StateManager.ProjectHasSkill(ctx.Cwd, skillID)
	if err != nil {
		return errors.Wrap(err, "检查技能状态失败")
	}
	if !hasSkill {
		return errors.NewWithCodef("runValidate", errors.ErrSkillNotFound, "技能 %s 未在state.json里项目工作区登记，该技能非法", skillID)
	}

	// 检查项目本地工作区文件
	fmt.Println("检查项目本地工作区文件...")

	agentsSkillDir := filepath.Join(ctx.Cwd, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillDir); os.IsNotExist(err) {
		return errors.NewWithCodef("runValidate", errors.ErrFileNotFound, "项目本地工作区目录不存在: .agents/skills/%s", skillID)
	}
	fmt.Printf("✓ 项目本地工作区目录存在: .agents/skills/%s\n", skillID)

	// 2. 检查SKILL.md文件
	skillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return errors.NewWithCodef("runValidate", errors.ErrFileNotFound, "SKILL.md文件不存在: %s", skillMdPath)
	}
	fmt.Printf("✓ SKILL.md文件存在: %s\n", skillMdPath)

	// 3. 验证SKILL.md的YAML语法
	fmt.Println("验证SKILL.md的YAML语法...")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return errors.WrapWithCode(err, "runValidate", errors.ErrFileOperation, "读取SKILL.md失败")
	}

	skillData, err := skill.ParseFrontmatter(content)
	if err != nil {
		return errors.Wrap(err, "解析YAML frontmatter失败")
	}
	fmt.Println("✓ YAML语法正确")

	// 4. 检查必需字段
	fmt.Println("检查必需字段...")
	requiredFields := []string{"name", "description"}
	for _, field := range requiredFields {
		if _, ok := skillData[field]; !ok {
			return errors.NewWithCodef("runValidate", errors.ErrValidation, "缺少必需字段: %s", field)
		}
	}
	fmt.Println("✓ 必需字段完整")

	// 5. 检查文件结构
	fmt.Println("检查文件结构...")
	// 检查是否有其他支持的文件
	optionalFiles := []string{"README.md", "examples/", "config.yaml"}
	for _, file := range optionalFiles {
		filePath := filepath.Join(agentsSkillDir, file)
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("✓ 可选文件存在: %s\n", file)
		}
	}

	// 6. 检查仓库源文件（如果可用）
	fmt.Println("检查仓库源文件...")
	// TODO: 实现仓库源文件检查逻辑
	fmt.Println("⚠️  仓库源文件检查功能暂未实现")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ 验证通过！")
	fmt.Println("技能合规性验证完成")
	fmt.Println(strings.Repeat("=", 50))

	return nil
}
