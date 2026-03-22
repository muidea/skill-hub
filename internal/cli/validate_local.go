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
	Use:   "validate <id>",
	Short: "验证本地新建技能的合规性",
	Long: `验证指定技能在项目本地工作区中的文件是否合规，主要用于 create 之后、feedback 之前的本地校验。

该命令只检查项目工作区中的技能目录和 SKILL.md 内容，包括 YAML 语法、必需字段和基本文件结构。
如果该技能未在 state.json 里登记，或项目本地工作区目录不存在，则视为非法技能。`,
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

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ 验证通过！")
	fmt.Println("本地技能合规性验证完成")
	fmt.Println(strings.Repeat("=", 50))

	return nil
}
