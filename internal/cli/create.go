package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/skill"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

var createCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "创建新技能模板",
	Long: `在当前项目的本地技能工作区创建一个新技能，默认写入 .agents/skills/<id>/。

创建的技能仅存在于项目本地，需要通过 feedback 命令同步到仓库。
create命令将会刷新state.json，标记当前项目工作区在使用该技能。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCreate(args[0])
	},
}

func runCreate(skillID string) error {
	if err := CheckInitDependency(); err != nil {
		return err
	}

	if !isValidSkillName(skillID) {
		return errors.NewWithCodef("runCreate", errors.ErrValidation, "技能ID '%s' 格式无效。应使用小写字母、数字和连字符，例如：my-logic-skill", skillID)
	}

	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}

	agentsDir := filepath.Join(ctx.Cwd, ".agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			return errors.WrapWithCode(err, "runCreate", errors.ErrFileOperation, "创建.agents目录失败")
		}
		fmt.Printf("✓ 创建.agents目录: %s\n", agentsDir)
	}

	skillDir := filepath.Join(agentsDir, "skills", skillID)
	skillFilePath := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillFilePath); err == nil {
		fmt.Printf("ℹ️  技能文件已存在: %s\n", skillFilePath)
		if err := validateSkillFile(skillFilePath); err != nil {
			fmt.Printf("⚠️  技能文件验证失败: %v\n", err)
			fmt.Print("是否重新创建？ [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(response)
			if response != "y" && response != "Y" {
				fmt.Println("❌ 取消操作")
				return nil
			}
			fmt.Println("✅ 将重新创建技能文件")
		} else {
			fmt.Println("✅ 技能文件验证通过")
			if alreadyRegisteredAndSynced(ctx.Cwd, skillID, skillDir) {
				fmt.Printf("✅ 技能 '%s' 已在本地仓库登记且与仓库一致，无需操作\n", skillID)
				return nil
			}
			fmt.Println("正在刷新项目状态...")
			if err := refreshProjectState(ctx.Cwd, skillID); err != nil {
				return errors.Wrap(err, "刷新项目状态失败")
			}
			fmt.Printf("✅ 技能 '%s' 已成功登记到项目状态\n", skillID)
			fmt.Println("\n下一步:")
			fmt.Printf("1. 使用 'skill-hub feedback %s' 将技能反馈到仓库\n", skillID)
			return nil
		}
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return errors.WrapWithCode(err, "runCreate", errors.ErrFileOperation, "创建技能目录失败")
	}

	for _, sub := range []string{"scripts", "references", "assets"} {
		subDir := filepath.Join(skillDir, sub)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return errors.Wrapf(err, "创建子目录 %s 失败", sub)
		}
	}

	// 收集技能描述
	fmt.Printf("请输入技能描述 (按Enter使用默认描述): ")
	reader := bufio.NewReader(os.Stdin)
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// 使用默认描述如果用户未输入
	if description == "" {
		description = fmt.Sprintf("为项目定制的 %s 技能", skillID)
	}

	content, err := generateSkillContent(skillID, description)
	if err != nil {
		return errors.Wrap(err, "生成技能内容失败")
	}

	// 写入文件
	if err := os.WriteFile(skillFilePath, []byte(content), 0644); err != nil {
		return utils.WriteFileErr(err, skillFilePath)
	}

	fmt.Printf("✅ 技能模板创建成功: %s\n", skillFilePath)

	fmt.Println("正在刷新项目状态...")
	if err := refreshProjectState(ctx.Cwd, skillID); err != nil {
		return errors.Wrap(err, "刷新项目状态失败")
	}

	fmt.Println("\n下一步:")
	fmt.Println("1. 编辑 SKILL.md 文件以完善技能内容")
	fmt.Printf("2. 使用 'skill-hub validate %s' 验证技能合规性\n", skillID)
	fmt.Printf("3. 使用 'skill-hub feedback %s' 将技能反馈到仓库\n", skillID)

	return nil
}

// generateSkillContent 生成技能内容
func generateSkillContent(name, description string) (string, error) {
	// 获取当前时间
	timestamp := time.Now().Format(time.RFC3339)

	// 获取当前用户（简化实现）
	author := "unknown"
	if user := os.Getenv("USER"); user != "" {
		author = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		author = user
	}

	// 使用字符串构建器创建模板
	var template strings.Builder

	template.WriteString(fmt.Sprintf(`---
name: %s
description: %s
metadata:
  version: "1.0.0"
  author: "%s"
  created_at: "%s"
---
# %s

%s

## 适用场景

- TODO: 用中文说明这个技能应在什么任务、项目或业务场景下触发。
- TODO: 保留命令、文件路径、API 名称等技术标识的原始写法。

## 工作流程

1. TODO: 用中文描述执行该任务前需要检查的上下文。
2. TODO: 用中文描述核心处理步骤。
3. TODO: 用中文描述验证方式和完成标准。

## Formatter

- `+"`SKILL.md`"+` / Markdown / YAML: 保持标题、列表和代码块格式稳定；归档前运行 `+"`skill-hub validate %s --links`"+`。
- `+"`scripts/`"+`: 当前模板未包含脚本；新增 Go/Python/JavaScript/TypeScript/Shell 等脚本时，必须在本段补充项目可运行的具体 formatter 命令。
- 常见 formatter 示例：Go 使用 `+"`gofmt -w <files>`"+`，Python 优先使用仓库已有的 `+"`ruff format <files>`"+` 或 `+"`black <files>`"+`，JavaScript/TypeScript 优先使用仓库已有的 `+"`npm run format`"+` 或 `+"`prettier`"+`，Shell 优先使用仓库已有 formatter 或语法检查。
- 不要声明当前项目无法执行的 formatter；如果对应文件类型没有 formatter，明确写出人工格式要求。

## 输出要求

- TODO: 用中文说明执行完成后应向用户交付什么结果。
- TODO: 说明是否需要更新文档、测试或将修改反馈到 skill-hub 默认仓库。

## 注意事项

- TODO: 补充边界条件、禁止事项或需要用户确认的操作。
`,
		name,
		description,
		author,
		timestamp,
		name,
		description,
		name))

	return template.String(), nil
}

// isValidSkillName 验证技能名称格式
func isValidSkillName(name string) bool {
	if name == "" {
		return false
	}

	// 检查是否只包含小写字母、数字和连字符
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}

	// 不能以连字符开头或结尾
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return false
	}

	// 不能包含连续连字符
	if strings.Contains(name, "--") {
		return false
	}

	return true
}

func validateSkillFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return errors.WrapWithCode(err, "validateSkillFile", errors.ErrFileOperation, "读取技能文件失败")
	}
	return skill.ValidateSkillFile(content)
}

func alreadyRegisteredAndSynced(projectPath, skillID, skillDir string) bool {
	stateManager, err := newStateManager()
	if err != nil {
		return false
	}
	projectState, err := stateManager.LoadProjectState(projectPath)
	if err != nil || projectState.Skills == nil {
		return false
	}
	if _, inState := projectState.Skills[skillID]; !inState {
		return false
	}
	repoSkillDir, err := getRepoSkillDirPath(skillID)
	if err != nil {
		return false
	}
	equal, err := skillDirsEqual(skillDir, repoSkillDir)
	return err == nil && equal
}

func refreshProjectState(projectPath, skillID string) error {
	stateManager, err := newStateManager()
	if err != nil {
		return errors.WrapWithCode(err, "refreshProjectState", errors.ErrSystem, "创建状态管理器失败")
	}

	projectState, err := stateManager.LoadProjectState(projectPath)
	if err != nil {
		return errors.Wrap(err, "加载项目状态失败")
	}

	// 更新技能状态
	if projectState.Skills == nil {
		projectState.Skills = make(map[string]spec.SkillVars)
	}

	// 设置技能变量（如果有的话）
	projectState.Skills[skillID] = spec.SkillVars{
		SkillID:   skillID,
		Variables: map[string]string{},
	}

	if err := stateManager.SaveProjectState(projectState); err != nil {
		return errors.Wrap(err, "保存项目状态失败")
	}

	return nil
}
