package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/config"
	"skill-hub/internal/engine"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
)

// CheckInitDependency 检查init依赖，如果本地仓库不存在则返回错误
// 符合规范要求：所有命令（除init外）都需要检查本地仓库是否存在
func CheckInitDependency() error {
	// 尝试加载配置，如果失败说明未初始化
	_, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("本地仓库未初始化，请先运行 'skill-hub init'")
	}
	return nil
}

// CheckProjectWorkspace 检查项目工作区状态
// 符合规范要求：检查当前目录是否存在于state.json中
func CheckProjectWorkspace(cwd string) (*spec.ProjectState, error) {
	stateManager, err := state.NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("创建状态管理器失败: %w", err)
	}

	projectState, err := stateManager.LoadProjectState(cwd)
	if err != nil {
		return nil, fmt.Errorf("加载项目状态失败: %w", err)
	}

	return projectState, nil
}

// EnsureProjectWorkspace 确保项目工作区存在
// 符合规范要求：如果当前目录不存在于state.json中，则提示是否需要新建项目工作区
func EnsureProjectWorkspace(cwd, target string) (*spec.ProjectState, error) {
	stateManager, err := state.NewStateManager()
	if err != nil {
		return nil, fmt.Errorf("创建状态管理器失败: %w", err)
	}

	// 检查项目是否真正存在于状态文件中
	projectState, err := stateManager.FindProjectByPath(cwd)
	if err != nil {
		return nil, fmt.Errorf("查找项目失败: %w", err)
	}

	// 如果项目不存在于状态文件中，需要初始化
	if projectState == nil {
		fmt.Printf("当前目录 '%s' 未在skill-hub中注册\n", filepath.Base(cwd))
		fmt.Print("是否创建新的项目工作区？ [Y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" || strings.ToLower(response) == "y" {
			// 创建项目工作区
			return createNewProjectWorkspace(cwd, target, stateManager)
		} else {
			return nil, fmt.Errorf("操作取消")
		}
	}

	return projectState, nil
}

// createNewProjectWorkspace 创建新的项目工作区
func createNewProjectWorkspace(cwd, target string, stateManager *state.StateManager) (*spec.ProjectState, error) {
	fmt.Println("正在创建新的项目工作区...")

	// 如果target为空，使用默认值open_code
	if target == "" {
		target = spec.TargetOpenCode
	}

	// 验证目标值
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode {
		return nil, fmt.Errorf("无效的目标值: %s，可用选项: cursor, claude, open_code", target)
	}

	// 根据target初始化对应的文件和目录
	if err := initializeTargetFiles(cwd, normalizedTarget); err != nil {
		return nil, fmt.Errorf("初始化目标文件失败: %w", err)
	}

	// 创建项目状态
	projectState := &spec.ProjectState{
		ProjectPath:     cwd,
		PreferredTarget: normalizedTarget,
		Skills:          make(map[string]spec.SkillVars),
	}

	// 保存项目状态
	if err := stateManager.SaveProjectState(projectState); err != nil {
		return nil, fmt.Errorf("保存项目状态失败: %w", err)
	}

	fmt.Printf("✅ 已创建项目工作区，目标环境: %s\n", normalizedTarget)
	return projectState, nil
}

// initializeTargetFiles 根据目标环境初始化对应的文件和目录
func initializeTargetFiles(cwd, target string) error {
	switch target {
	case spec.TargetOpenCode:
		// 创建.agents目录结构
		agentsDir := filepath.Join(cwd, ".agents")
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			return fmt.Errorf("创建.agents目录失败: %w", err)
		}
		fmt.Printf("✓ 创建目录: %s\n", agentsDir)

		skillsDir := filepath.Join(agentsDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("创建skills目录失败: %w", err)
		}
		fmt.Printf("✓ 创建目录: %s\n", skillsDir)

	case spec.TargetClaudeCode:
		// 创建.claude目录
		claudeDir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			return fmt.Errorf("创建.claude目录失败: %w", err)
		}
		fmt.Printf("✓ 创建目录: %s\n", claudeDir)

		// 创建空的config.json
		configPath := filepath.Join(claudeDir, "config.json")
		configContent := `{
  "skills": {}
}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("创建config.json失败: %w", err)
		}
		fmt.Printf("✓ 创建文件: %s\n", configPath)

	case spec.TargetCursor:
		// 创建.cursorrules文件
		cursorRulesPath := filepath.Join(cwd, ".cursorrules")
		cursorRulesContent := `# Cursor Rules
# This file is managed by skill-hub

# Available skills will be injected here`
		if err := os.WriteFile(cursorRulesPath, []byte(cursorRulesContent), 0644); err != nil {
			return fmt.Errorf("创建.cursorrules文件失败: %w", err)
		}
		fmt.Printf("✓ 创建文件: %s\n", cursorRulesPath)

	default:
		return fmt.Errorf("不支持的目标环境: %s", target)
	}

	return nil
}

// CheckSkillExists 检查技能是否存在
func CheckSkillExists(skillID string) error {
	// 检查init依赖
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 创建技能管理器
	manager, err := engine.NewSkillManager()
	if err != nil {
		return err
	}

	// 检查技能是否存在
	if !manager.SkillExists(skillID) {
		return fmt.Errorf("技能 '%s' 不存在，使用 'skill-hub list' 查看可用技能", skillID)
	}

	return nil
}

// CheckSkillInProject 检查技能是否在项目中
func CheckSkillInProject(cwd, skillID string) (bool, error) {
	stateManager, err := state.NewStateManager()
	if err != nil {
		return false, err
	}

	return stateManager.ProjectHasSkill(cwd, skillID)
}
