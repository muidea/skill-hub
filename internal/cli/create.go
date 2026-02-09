package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "创建新技能模板",
	Long: `在项目当前工作区创建一个新技能。

如果指定了 --target 选项，则创建的技能将用于该目标环境。
否则将用于init初始化时设置的默认目标环境。

创建的技能仅存在于项目本地，需要通过 feedback 命令同步到仓库。
create命令将会刷新state.json，标记当前项目工作区在使用该技能。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		return runCreate(args[0], target)
	},
}

func init() {
	createCmd.Flags().String("target", "open_code", "技能目标环境，默认为 open_code")
}

func runCreate(skillID string, target string) error {
	// 验证技能ID格式
	if !isValidSkillName(skillID) {
		return fmt.Errorf("技能ID '%s' 格式无效。应使用小写字母、数字和连字符，例如：my-logic-skill", skillID)
	}

	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目是否已初始化（检查.agents目录）
	agentsDir := filepath.Join(cwd, ".agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return fmt.Errorf("项目未初始化，请先运行 'skill-hub init' 命令")
	}

	// 创建技能目录结构
	skillDir := filepath.Join(agentsDir, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 检查是否已存在同名技能文件
	skillFilePath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillFilePath); err == nil {
		fmt.Printf("⚠️  技能文件已存在: %s\n", skillFilePath)
		fmt.Print("是否覆盖？ [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("❌ 取消创建")
			return nil
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

	// 验证目标选项
	if !isValidTarget(target) {
		return fmt.Errorf("无效的目标选项: %s。可用选项: cursor, claude, open_code", target)
	}

	// 生成技能内容
	content, err := generateSkillContent(skillID, description, target)
	if err != nil {
		return fmt.Errorf("生成技能内容失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(skillFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	fmt.Printf("✅ 技能模板创建成功: %s\n", skillFilePath)

	// 刷新state.json，标记当前项目工作区在使用该技能
	fmt.Println("正在刷新项目状态...")
	// TODO: 实现state.json刷新逻辑

	fmt.Println("\n下一步:")
	fmt.Println("1. 编辑 SKILL.md 文件以完善技能内容")
	fmt.Printf("2. 使用 'skill-hub validate %s' 验证技能合规性\n", skillID)
	fmt.Printf("3. 使用 'skill-hub feedback %s' 将技能反馈到仓库\n", skillID)

	return nil
}

// generateSkillContent 生成技能内容
func generateSkillContent(name, description, target string) (string, error) {
	// 获取当前时间
	timestamp := time.Now().Format(time.RFC3339)

	// 获取当前用户（简化实现）
	author := "unknown"
	if user := os.Getenv("USER"); user != "" {
		author = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		author = user
	}

	// 生成兼容性描述
	compatDesc := generateCompatibilityDescription(target)

	// 使用字符串构建器创建模板
	var template strings.Builder

	template.WriteString(fmt.Sprintf(`---
name: %s
description: %s
compatibility: %s
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
		compatDesc,
		author,
		timestamp,
		name,
		description,
		time.Now().Format("2006-01-02")))

	return template.String(), nil
}

// generateCompatibilityDescription 生成兼容性描述
func generateCompatibilityDescription(target string) string {
	// 规范化目标值
	normalized := strings.ToLower(target)

	switch normalized {
	case "cursor":
		return "Designed for Cursor (or similar AI coding assistants)"
	case "claude":
		return "Designed for Claude Code (or similar AI coding assistants)"
	case "open_code":
		return "Designed for OpenCode (or similar AI coding assistants)"
	default:
		return "Designed for AI coding assistants"
	}
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

// isValidTarget 验证目标选项
func isValidTarget(target string) bool {
	validOptions := map[string]bool{
		"cursor":    true,
		"claude":    true, // 接受claude作为claude_code的简写
		"open_code": true,
	}

	return validOptions[target]
}
