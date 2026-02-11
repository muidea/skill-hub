package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/state"
)

var validateCmd = &cobra.Command{
	Use:   "validate <id>",
	Short: "验证技能合规性",
	Long: `验证指定技能的项目本地工作区的文件是否合规，包括检查 SKILL.md 的 YAML 语法、必需字段、文件结构等。
验证范围包括项目本地文件和仓库源文件。

如果该技能未在state.json里项目工作区登记，则提示该技能非法`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args[0])
	},
}

func runValidate(skillID string) error {
	// 检查init依赖（规范4.7：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Printf("验证技能合规性: %s\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目工作区状态（规范4.7：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, "")
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
	}

	// 创建状态管理器
	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("创建状态管理器失败: %w", err)
	}

	// 检查技能是否在state.json里项目工作区登记
	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查技能状态失败: %w", err)
	}
	if !hasSkill {
		return fmt.Errorf("技能 %s 未在state.json里项目工作区登记，该技能非法", skillID)
	}

	// 检查项目本地工作区文件
	fmt.Println("检查项目本地工作区文件...")

	// 1. 检查.agents/skills/[skillID]目录
	agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	if _, err := os.Stat(agentsSkillDir); os.IsNotExist(err) {
		return fmt.Errorf("项目本地工作区目录不存在: .agents/skills/%s", skillID)
	}
	fmt.Printf("✓ 项目本地工作区目录存在: .agents/skills/%s\n", skillID)

	// 2. 检查SKILL.md文件
	skillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("SKILL.md文件不存在: %s", skillMdPath)
	}
	fmt.Printf("✓ SKILL.md文件存在: %s\n", skillMdPath)

	// 3. 验证SKILL.md的YAML语法
	fmt.Println("验证SKILL.md的YAML语法...")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter分隔符")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	if len(frontmatterLines) == 0 {
		return fmt.Errorf("无效的SKILL.md格式: frontmatter为空")
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML frontmatter
	var skillData map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err != nil {
		return fmt.Errorf("解析YAML frontmatter失败: %w", err)
	}
	fmt.Println("✓ YAML语法正确")

	// 4. 检查必需字段
	fmt.Println("检查必需字段...")
	requiredFields := []string{"name", "description"}
	for _, field := range requiredFields {
		if _, ok := skillData[field]; !ok {
			return fmt.Errorf("缺少必需字段: %s", field)
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
