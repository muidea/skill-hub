package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"skill-hub/internal/git"
)

var initCmd = &cobra.Command{
	Use:   "init [git-url]",
	Short: "初始化Skill Hub工作区",
	Long: `初始化Skill Hub工作区，创建必要的配置文件和目录结构。

如果提供了Git仓库URL，会克隆远程仓库到本地。
如果没有提供URL，会创建一个空的本地仓库。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(args)
	},
}

func runInit(args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	skillHubDir := filepath.Join(homeDir, ".skill-hub")

	fmt.Printf("正在初始化Skill Hub工作区: %s\n", skillHubDir)

	// 检查是否提供了Git URL
	var gitURL string
	if len(args) > 0 {
		gitURL = args[0]
		fmt.Printf("将克隆远程仓库: %s\n", gitURL)
	}

	// 创建目录结构
	dirs := []string{
		skillHubDir,
		filepath.Join(skillHubDir, "repo"),
		filepath.Join(skillHubDir, "repo", "skills"),
		filepath.Join(skillHubDir, "repo", "templates"),
		filepath.Join(homeDir, ".cursor"), // 全局Cursor配置目录
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		fmt.Printf("✓ 创建目录: %s\n", dir)
	}

	// 创建配置文件
	configPath := filepath.Join(skillHubDir, "config.yaml")
	configContent := fmt.Sprintf(`# Skill Hub 配置文件
repo_path: "~/.skill-hub/repo"
claude_config_path: "~/.claude/config.json"
cursor_config_path: "~/.cursor/rules"
default_tool: "cursor"
git_remote_url: "%s"
git_token: ""
git_branch: "main"
`, gitURL)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	fmt.Printf("✓ 创建配置文件: %s\n", configPath)

	// 创建初始技能示例
	exampleSkillDir := filepath.Join(skillHubDir, "repo", "skills", "git-expert")
	if err := os.MkdirAll(exampleSkillDir, 0755); err != nil {
		return fmt.Errorf("创建示例技能目录失败: %w", err)
	}

	// 创建git-expert SKILL.md
	gitExpertMD := `---
name: git-expert
description: 根据变更自动生成符合 Conventional Commits 规范的提交说明
compatibility:
  cursor: true
  claude_code: false
  open_code: false
metadata:
  version: 1.0.0
  author: skill-hub
  tags: git,workflow
---
# Git 提交专家

根据代码变更自动生成符合 Conventional Commits 规范的提交说明。

## 使用说明
1. 分析代码变更
2. 识别变更类型（feat, fix, docs, style, refactor, test, chore）
3. 生成简洁明了的提交说明

## 变量
- LANGUAGE: {{.LANGUAGE}} - 输出语言

## 示例
当检测到新功能时，生成：
feat: 添加用户登录功能

当修复bug时，生成：
fix: 修复登录页面样式错位问题
`

	if err := os.WriteFile(filepath.Join(exampleSkillDir, "SKILL.md"), []byte(gitExpertMD), 0644); err != nil {
		return fmt.Errorf("创建SKILL.md失败: %w", err)
	}
	fmt.Println("✓ 创建示例技能: git-expert (SKILL.md)")

	// 创建prompt.md
	promptMd := `# Git 提交专家

根据代码变更自动生成符合 Conventional Commits 规范的提交说明。

## 使用说明
1. 分析代码变更
2. 识别变更类型（feat, fix, docs, style, refactor, test, chore）
3. 生成简洁明了的提交说明

## 变量
- LANGUAGE: {{.LANGUAGE}} - 输出语言

## 示例
当检测到新功能时，生成：
feat: 添加用户登录功能

当修复bug时，生成：
fix: 修复登录页面样式错位问题
`

	if err := os.WriteFile(filepath.Join(exampleSkillDir, "prompt.md"), []byte(promptMd), 0644); err != nil {
		return fmt.Errorf("创建prompt.md失败: %w", err)
	}

	// 创建Claude示例技能
	claudeSkillDir := filepath.Join(skillHubDir, "repo", "skills", "claude-code-review")
	if err := os.MkdirAll(claudeSkillDir, 0755); err != nil {
		return fmt.Errorf("创建Claude技能目录失败: %w", err)
	}

	// 创建Claude SKILL.md
	claudeMD := `---
name: claude-code-review
description: 专业的代码审查助手，帮助发现代码中的问题和改进点
compatibility:
  cursor: false
  claude_code: true
  open_code: false
metadata:
  version: 1.0.0
  author: skill-hub
  tags: claude,code-review,quality
---
# Claude 代码审查助手

专业的代码审查助手，帮助发现代码中的问题和改进点。

## 审查流程
1. 代码结构分析
2. 潜在问题识别
3. 改进建议提供
4. 最佳实践指导

## 审查风格
- detailed: 详细审查，包含所有细节
- quick: 快速审查，只关注关键问题
- strict: 严格审查，遵循最佳实践

## 变量
- REVIEW_STYLE: {{.REVIEW_STYLE}} - 审查风格

## 示例输出
当使用detailed风格时：
1. 代码结构问题：函数过长，建议拆分
2. 性能问题：循环内重复计算，建议缓存结果
3. 可读性问题：变量命名不清晰，建议使用描述性名称
`

	if err := os.WriteFile(filepath.Join(claudeSkillDir, "SKILL.md"), []byte(claudeMD), 0644); err != nil {
		return fmt.Errorf("创建Claude SKILL.md失败: %w", err)
	}
	fmt.Println("✓ 创建Claude示例技能: claude-code-review (SKILL.md)")

	// 创建Claude prompt.md
	claudePrompt := `# Claude 代码审查助手

你是一个专业的代码审查助手，专注于帮助开发者发现代码中的问题和改进点。

## 审查重点
1. 代码逻辑错误和边界条件
2. 安全漏洞和潜在风险
3. 性能问题和优化机会
4. 代码可读性和维护性
5. 测试覆盖率和质量

## 审查风格: {{.REVIEW_STYLE}}

## 输出格式
请按照以下格式提供审查意见：
1. **问题类型**: [Bug/Security/Performance/Code Smell]
2. **严重程度**: [Critical/High/Medium/Low]
3. **位置**: 文件路径和行号
4. **问题描述**: 详细说明问题
5. **建议修复**: 具体的修复建议
6. **代码示例**: 修复前后的代码对比

## 特殊说明
- 对于安全相关问题，请特别标注并说明潜在风险
- 对于性能问题，请提供基准测试建议
- 对于代码风格问题，请参考项目规范
`

	if err := os.WriteFile(filepath.Join(claudeSkillDir, "prompt.md"), []byte(claudePrompt), 0644); err != nil {
		return fmt.Errorf("创建Claude prompt.md失败: %w", err)
	}

	// 创建registry.json
	registryPath := filepath.Join(skillHubDir, "repo", "registry.json")
	registryContent := `{
  "version": "1.0.0",
  "skills": [
    {
      "id": "git-expert",
      "name": "Git 提交专家",
      "version": "1.0.0",
      "author": "skill-hub",
      "description": "根据变更自动生成符合 Conventional Commits 规范的说明",
      "tags": ["git", "workflow"],
      "compatibility": {
        "cursor": true,
        "claude_code": false,
        "open_code": false
      }
    },
    {
      "id": "claude-code-review",
      "name": "Claude 代码审查助手",
      "version": "1.0.0",
      "author": "skill-hub",
      "description": "专业的代码审查助手，帮助发现代码中的问题和改进点",
      "tags": ["claude", "code-review", "quality"],
      "compatibility": {
        "cursor": false,
        "claude_code": true,
        "open_code": false
      }
    }
  ]
}
`

	if err := os.WriteFile(registryPath, []byte(registryContent), 0644); err != nil {
		return fmt.Errorf("创建registry.json失败: %w", err)
	}
	fmt.Printf("✓ 创建技能索引: %s\n", registryPath)

	// 如果提供了Git URL，克隆远程仓库
	if gitURL != "" {
		fmt.Println("\n正在克隆远程技能仓库...")
		repo, err := git.NewSkillRepository()
		if err != nil {
			return fmt.Errorf("创建技能仓库失败: %w", err)
		}

		if err := repo.CloneRemote(gitURL); err != nil {
			fmt.Printf("⚠️  克隆远程仓库失败: %v\n", err)
			fmt.Println("\n故障排除建议:")
			fmt.Println("1. 对于SSH URL (git@...):")
			fmt.Println("   - 确保SSH agent正在运行: eval $(ssh-agent) && ssh-add ~/.ssh/id_rsa")
			fmt.Println("   - 或使用HTTPS URL代替: https://github.com/user/repo.git")
			fmt.Println("2. 对于HTTPS URL:")
			fmt.Println("   - 确保网络连接正常")
			fmt.Println("   - 如果需要认证，设置Git token: skill-hub config set git_token YOUR_TOKEN")
			fmt.Println("3. 检查URL格式是否正确")
			fmt.Println("\n将继续使用本地示例技能")
		} else {
			fmt.Println("✅ 远程技能仓库克隆完成")
			// 覆盖本地示例技能
			os.RemoveAll(filepath.Join(skillHubDir, "repo", "skills", "git-expert"))
			os.RemoveAll(filepath.Join(skillHubDir, "repo", "skills", "claude-code-review"))
		}
	}

	fmt.Println("\n✅ Skill Hub 初始化完成！")
	fmt.Println("工作区位置:", skillHubDir)

	if gitURL != "" {
		fmt.Println("远程仓库:", gitURL)
		fmt.Println("使用 'skill-hub git sync' 同步最新技能")
	} else {
		fmt.Println("已创建示例技能:")
		fmt.Println("  - git-expert (Cursor专用)")
		fmt.Println("  - claude-code-review (Claude专用)")
	}

	fmt.Println("\n使用 'skill-hub list' 查看可用技能")

	return nil
}
