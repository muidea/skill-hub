package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"skill-hub/internal/config"
	"skill-hub/pkg/spec"
)

// StateManager 管理项目状态
type StateManager struct {
	statePath string
}

// GetStatePath 获取状态文件路径
func (m *StateManager) GetStatePath() string {
	return m.statePath
}

// StateFile 表示状态文件的完整结构
type StateFile struct {
	Projects map[string]spec.ProjectConfig `json:"projects"`
}

// NewStateManager 创建新的状态管理器
func NewStateManager() (*StateManager, error) {
	repoPath, err := config.GetRepoPath()
	if err != nil {
		return nil, err
	}

	statePath := filepath.Join(repoPath, "state.json")
	return &StateManager{statePath: statePath}, nil
}

// LoadProjectState 加载指定项目的状态
func (m *StateManager) LoadProjectState(projectPath string) (*spec.ProjectState, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 读取状态文件
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空状态，默认目标为 open_code
			return &spec.ProjectState{
				ProjectPath:     absPath,
				PreferredTarget: spec.TargetOpenCode,
				Skills:          make(map[string]spec.SkillVars),
			}, nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	// 解析所有项目状态
	var allStates map[string]spec.ProjectState
	if err := json.Unmarshal(data, &allStates); err != nil {
		return nil, fmt.Errorf("解析状态文件失败: %w", err)
	}

	// 查找当前项目状态
	if state, exists := allStates[absPath]; exists {
		return &state, nil
	}

	// 项目状态不存在，创建新状态，默认目标为 open_code
	return &spec.ProjectState{
		ProjectPath:     absPath,
		PreferredTarget: spec.TargetOpenCode,
		Skills:          make(map[string]spec.SkillVars),
	}, nil
}

// SaveProjectState 保存项目状态
func (m *StateManager) SaveProjectState(state *spec.ProjectState) error {
	// 读取现有所有状态
	allStates := make(map[string]spec.ProjectState)

	if data, err := os.ReadFile(m.statePath); err == nil {
		if err := json.Unmarshal(data, &allStates); err != nil {
			// 如果解析失败，使用空map
			allStates = make(map[string]spec.ProjectState)
		}
	}

	// 更新当前项目状态
	allStates[state.ProjectPath] = *state

	// 写入文件
	data, err := json.MarshalIndent(allStates, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(m.statePath, data, 0644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}

	return nil
}

// AddSkillToProject 添加技能到项目
func (m *StateManager) AddSkillToProject(projectPath, skillID, version string, variables map[string]string) error {
	return m.AddSkillToProjectWithTarget(projectPath, skillID, version, variables, "")
}

// AddSkillToProjectWithTarget 添加技能到项目并指定目标
func (m *StateManager) AddSkillToProjectWithTarget(projectPath, skillID, version string, variables map[string]string, target string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	// 如果指定了target且当前没有preferred_target，则设置它
	if target != "" && state.PreferredTarget == "" {
		state.PreferredTarget = target
	}

	state.Skills[skillID] = spec.SkillVars{
		SkillID:   skillID,
		Version:   version,
		Variables: variables,
	}

	return m.SaveProjectState(state)
}

// SetPreferredTarget 设置项目的首选目标
func (m *StateManager) SetPreferredTarget(projectPath, target string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	// 验证目标值
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode && normalizedTarget != "" {
		return fmt.Errorf("无效的目标值: %s，可用选项: %s, %s, %s", target, spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode)
	}

	state.PreferredTarget = normalizedTarget
	return m.SaveProjectState(state)
}

// GetPreferredTarget 获取项目的首选目标
func (m *StateManager) GetPreferredTarget(projectPath string) (string, error) {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return "", err
	}
	return spec.NormalizeTarget(state.PreferredTarget), nil
}

// FindProjectByPath 通过路径查找项目（支持递归向上查找）
func (m *StateManager) FindProjectByPath(path string) (*spec.ProjectState, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 读取所有项目状态
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在，返回nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}

	var allStates map[string]spec.ProjectState
	if err := json.Unmarshal(data, &allStates); err != nil {
		return nil, fmt.Errorf("解析状态文件失败: %w", err)
	}

	// 递归向上查找
	currentPath := absPath
	for {
		// 检查当前路径是否有绑定
		if state, exists := allStates[currentPath]; exists {
			// 规范化目标类型
			state.PreferredTarget = spec.NormalizeTarget(state.PreferredTarget)
			return &state, nil
		}

		// 到达根目录，停止查找
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			break
		}
		currentPath = parentPath
	}

	return nil, nil // 未找到
}

// RemoveSkillFromProject 从项目移除技能
func (m *StateManager) RemoveSkillFromProject(projectPath, skillID string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	delete(state.Skills, skillID)
	return m.SaveProjectState(state)
}

// GetProjectSkills 获取项目的所有技能
func (m *StateManager) GetProjectSkills(projectPath string) (map[string]spec.SkillVars, error) {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return nil, err
	}
	return state.Skills, nil
}

// ProjectHasSkill 检查项目是否启用了指定技能
func (m *StateManager) ProjectHasSkill(projectPath, skillID string) (bool, error) {
	skills, err := m.GetProjectSkills(projectPath)
	if err != nil {
		return false, err
	}

	_, exists := skills[skillID]
	return exists, nil
}

// UpdateSkillVariables 更新项目中技能的变量值
func (m *StateManager) UpdateSkillVariables(projectPath, skillID string, variables map[string]string) error {
	state, err := m.LoadProjectState(projectPath)
	if err != nil {
		return err
	}

	// 检查技能是否存在
	skillVars, exists := state.Skills[skillID]
	if !exists {
		return fmt.Errorf("技能 '%s' 未在项目中启用", skillID)
	}

	// 更新变量值
	skillVars.Variables = variables
	state.Skills[skillID] = skillVars

	return m.SaveProjectState(state)
}
