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

--target 仅保留兼容旧脚本，不影响工作区结构、项目状态或技能兼容性声明。

创建的技能仅存在于项目本地，需要通过 feedback 命令同步到仓库。
create命令将会刷新state.json，标记当前项目工作区在使用该技能。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		return runCreate(args[0], target)
	},
}

func init() {
	createCmd.Flags().String("target", "open_code", targetFlagUsage)
}

func runCreate(skillID string, target string) error {
	_ = target

	if err := CheckInitDependency(); err != nil {
		return err
	}

	if !isValidSkillName(skillID) {
		return errors.NewWithCodef("runCreate", errors.ErrValidation, "技能ID '%s' 格式无效。应使用小写字母、数字和连字符，例如：my-logic-skill", skillID)
	}

	ctx, err := RequireInitAndWorkspace("", "")
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
			if err := refreshProjectState(ctx.Cwd, skillID, ""); err != nil {
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

	content, err := generateSkillContent(skillID, description, "")
	if err != nil {
		return errors.Wrap(err, "生成技能内容失败")
	}

	// 写入文件
	if err := os.WriteFile(skillFilePath, []byte(content), 0644); err != nil {
		return utils.WriteFileErr(err, skillFilePath)
	}

	fmt.Printf("✅ 技能模板创建成功: %s\n", skillFilePath)

	fmt.Println("正在刷新项目状态...")
	if err := refreshProjectState(ctx.Cwd, skillID, ""); err != nil {
		return errors.Wrap(err, "刷新项目状态失败")
	}

	fmt.Println("\n下一步:")
	fmt.Println("1. 编辑 SKILL.md 文件以完善技能内容")
	fmt.Printf("2. 使用 'skill-hub validate %s' 验证技能合规性\n", skillID)
	fmt.Printf("3. 使用 'skill-hub feedback %s' 将技能反馈到仓库\n", skillID)

	return nil
}

// generateSkillContent 生成技能内容
func generateSkillContent(name, description, target string) (string, error) {
	_ = target

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

## 使用说明

这是一个自定义技能模板，请根据您的项目需求进行修改。

## 变量

技能支持以下变量，可以在启用技能时配置：

- `+"`PROJECT_NAME`"+`: 项目名称 {{.PROJECT_NAME}}
- `+"`PROJECT_PATH`"+`: 项目路径 {{.PROJECT_PATH}}
- `+"`LANGUAGE`"+`: 编程语言 {{.LANGUAGE}}
- `+"`FRAMEWORK`"+`: 框架 {{.FRAMEWORK}}

## 最佳实践

### 1. 保持技能专注
- 每个技能应该专注于一个特定的任务或领域
- 避免创建过于通用的技能

### 2. 清晰的变量命名
- 使用有意义的变量名称
- 为每个变量提供清晰的描述

### 3. 结构化内容
- 使用清晰的章节结构
- 包含示例和代码片段
- 提供故障排除指南

### 4. 版本控制
- 每次重要修改时更新版本号
- 在metadata中记录修改历史

## 示例

### 添加新功能
当需要添加新功能时，可以参考以下结构：

`+"```markdown"+`
## 功能名称

### 用途
描述功能的用途和适用场景。

### 使用方法
`+"```"+`
具体的命令或代码示例
`+"```"+`

### 注意事项
- 注意事项1
- 注意事项2
`+"```"+`

### 配置说明
对于需要配置的功能：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `+"`CONFIG_1`"+` | 配置1说明 | 默认值1 |
| `+"`CONFIG_2`"+` | 配置2说明 | 默认值2 |

## 更新日志

### v1.0.0 (%s)
- 初始版本创建
`,
		name,
		description,
		author,
		timestamp,
		name,
		description,
		time.Now().Format("2006-01-02")))

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

func refreshProjectState(projectPath, skillID, target string) error {
	_ = target

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
