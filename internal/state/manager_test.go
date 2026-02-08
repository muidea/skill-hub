package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"skill-hub/pkg/spec"
)

func TestStateManager(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建状态文件目录
	stateDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Failed to create state directory: %v", err)
	}

	statePath := filepath.Join(stateDir, "state.json")

	t.Run("Create state manager", func(t *testing.T) {
		manager := &StateManager{statePath: statePath}

		// 验证状态文件路径
		if manager.statePath != statePath {
			t.Errorf("State path = %v, want %v", manager.statePath, statePath)
		}
	})

	t.Run("Load and save project state", func(t *testing.T) {
		manager := &StateManager{statePath: statePath}

		projectPath := filepath.Join(tmpDir, "test-project")

		// 测试加载不存在的状态
		state, err := manager.LoadProjectState(projectPath)
		if err != nil {
			t.Errorf("LoadProjectState() error = %v", err)
		}

		if state == nil {
			t.Error("LoadProjectState() returned nil")
			return
		}

		if state.ProjectPath != projectPath {
			t.Errorf("Project path = %v, want %v", state.ProjectPath, projectPath)
		}

		if len(state.Skills) != 0 {
			t.Errorf("New state should have empty skills map")
		}

		// 添加技能到状态
		state.Skills["test-skill"] = spec.SkillVars{
			SkillID:   "test-skill",
			Version:   "1.0.0",
			Variables: map[string]string{"key": "value"},
		}

		state.PreferredTarget = "cursor"

		// 保存状态
		if err := manager.SaveProjectState(state); err != nil {
			t.Errorf("SaveProjectState() error = %v", err)
		}

		// 验证状态文件创建
		if _, err := os.Stat(manager.statePath); err != nil {
			t.Errorf("State file not created: %v", err)
		}

		// 重新加载状态
		reloadedState, err := manager.LoadProjectState(projectPath)
		if err != nil {
			t.Errorf("LoadProjectState(reload) error = %v", err)
		}

		if reloadedState.ProjectPath != projectPath {
			t.Errorf("Reloaded project path = %v, want %v", reloadedState.ProjectPath, projectPath)
		}

		if reloadedState.PreferredTarget != "cursor" {
			t.Errorf("Preferred target = %v, want cursor", reloadedState.PreferredTarget)
		}

		if len(reloadedState.Skills) != 1 {
			t.Errorf("Reloaded skills count = %d, want 1", len(reloadedState.Skills))
		}

		skillVars, exists := reloadedState.Skills["test-skill"]
		if !exists {
			t.Error("Skill 'test-skill' not found in reloaded state")
		}

		if skillVars.SkillID != "test-skill" {
			t.Errorf("Skill ID = %v, want test-skill", skillVars.SkillID)
		}

		if skillVars.Version != "1.0.0" {
			t.Errorf("Skill version = %v, want 1.0.0", skillVars.Version)
		}

		if value, exists := skillVars.Variables["key"]; !exists || value != "value" {
			t.Errorf("Skill variables = %v, want map[key:value]", skillVars.Variables)
		}
	})

	t.Run("Multiple projects state", func(t *testing.T) {
		manager := &StateManager{statePath: statePath}

		// 创建多个项目状态
		projects := []string{
			filepath.Join(tmpDir, "project-1"),
			filepath.Join(tmpDir, "project-2"),
			filepath.Join(tmpDir, "project-3"),
		}

		for i, projectPath := range projects {
			state := &spec.ProjectState{
				ProjectPath:     projectPath,
				PreferredTarget: "cursor",
				Skills: map[string]spec.SkillVars{
					"skill-1": {
						SkillID:   "skill-1",
						Version:   "1.0.0",
						Variables: map[string]string{"index": string(rune('0' + i))},
					},
				},
			}

			if err := manager.SaveProjectState(state); err != nil {
				t.Errorf("SaveProjectState(%s) error = %v", projectPath, err)
			}
		}

		// 验证所有项目状态
		for i, projectPath := range projects {
			state, err := manager.LoadProjectState(projectPath)
			if err != nil {
				t.Errorf("LoadProjectState(%s) error = %v", projectPath, err)
			}

			if state.ProjectPath != projectPath {
				t.Errorf("Project %d path = %v, want %v", i, state.ProjectPath, projectPath)
			}

			skillVars, exists := state.Skills["skill-1"]
			if !exists {
				t.Errorf("Project %d missing skill-1", i)
			}

			expectedValue := string(rune('0' + i))
			if value, exists := skillVars.Variables["index"]; !exists || value != expectedValue {
				t.Errorf("Project %d variables = %v, want map[index:%s]", i, skillVars.Variables, expectedValue)
			}
		}

		// 验证状态文件包含所有项目
		data, err := os.ReadFile(manager.statePath)
		if err != nil {
			t.Errorf("Failed to read state file: %v", err)
		}

		var allStates map[string]spec.ProjectState
		if err := json.Unmarshal(data, &allStates); err != nil {
			t.Errorf("Failed to parse state file: %v", err)
		}

		// 注意：由于之前的测试已经保存了状态，这里可能包含更多项目
		// 我们只验证我们的项目存在
		for _, projectPath := range projects {
			if _, exists := allStates[projectPath]; !exists {
				t.Errorf("Project %s not found in state file", projectPath)
			}
		}

		for _, projectPath := range projects {
			if _, exists := allStates[projectPath]; !exists {
				t.Errorf("Project %s not found in state file", projectPath)
			}
		}
	})

	t.Run("Update existing project", func(t *testing.T) {
		manager := &StateManager{statePath: statePath}

		projectPath := filepath.Join(tmpDir, "update-project")

		// 创建初始状态
		initialState := &spec.ProjectState{
			ProjectPath:     projectPath,
			PreferredTarget: "cursor",
			Skills: map[string]spec.SkillVars{
				"skill-1": {
					SkillID:   "skill-1",
					Version:   "1.0.0",
					Variables: map[string]string{"key": "initial"},
				},
			},
		}

		if err := manager.SaveProjectState(initialState); err != nil {
			t.Errorf("SaveProjectState(initial) error = %v", err)
		}

		// 更新状态
		updatedState := &spec.ProjectState{
			ProjectPath:     projectPath,
			PreferredTarget: "open_code",
			Skills: map[string]spec.SkillVars{
				"skill-1": {
					SkillID:   "skill-1",
					Version:   "2.0.0",
					Variables: map[string]string{"key": "updated"},
				},
				"skill-2": {
					SkillID:   "skill-2",
					Version:   "1.0.0",
					Variables: map[string]string{"new": "value"},
				},
			},
		}

		if err := manager.SaveProjectState(updatedState); err != nil {
			t.Errorf("SaveProjectState(updated) error = %v", err)
		}

		// 验证更新
		reloadedState, err := manager.LoadProjectState(projectPath)
		if err != nil {
			t.Errorf("LoadProjectState(after update) error = %v", err)
		}

		if reloadedState.PreferredTarget != "open_code" {
			t.Errorf("Preferred target after update = %v, want open_code", reloadedState.PreferredTarget)
		}

		if len(reloadedState.Skills) != 2 {
			t.Errorf("Skills count after update = %d, want 2", len(reloadedState.Skills))
		}

		// 验证skill-1更新
		skill1Vars, exists := reloadedState.Skills["skill-1"]
		if !exists {
			t.Error("skill-1 not found after update")
		}

		if skill1Vars.Version != "2.0.0" {
			t.Errorf("skill-1 version after update = %v, want 2.0.0", skill1Vars.Version)
		}

		if value, exists := skill1Vars.Variables["key"]; !exists || value != "updated" {
			t.Errorf("skill-1 variables after update = %v, want map[key:updated]", skill1Vars.Variables)
		}

		// 验证skill-2添加
		skill2Vars, exists := reloadedState.Skills["skill-2"]
		if !exists {
			t.Error("skill-2 not found after update")
		}

		if value, exists := skill2Vars.Variables["new"]; !exists || value != "value" {
			t.Errorf("skill-2 variables = %v, want map[new:value]", skill2Vars.Variables)
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		// 测试无效的JSON状态文件
		invalidJSONPath := filepath.Join(tmpDir, "invalid-state.json")
		if err := os.WriteFile(invalidJSONPath, []byte("{invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON file: %v", err)
		}

		manager := &StateManager{statePath: invalidJSONPath}
		_, err := manager.LoadProjectState("/some/path")
		if err == nil {
			t.Error("Expected error when loading invalid JSON")
		}

		// 测试在无效路径中保存状态（应该失败）
		// 创建一个文件路径，但确保父目录是文件而不是目录
		invalidPath := filepath.Join(tmpDir, "invalid", "state.json")

		// 创建父目录作为文件
		parentDir := filepath.Dir(invalidPath)
		if err := os.WriteFile(parentDir, []byte("this is a file, not a directory"), 0444); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		invalidManager := &StateManager{statePath: invalidPath}

		state := &spec.ProjectState{
			ProjectPath: "/test/path",
			Skills:      make(map[string]spec.SkillVars),
		}

		err = invalidManager.SaveProjectState(state)
		if err == nil {
			t.Error("Expected error when saving to invalid path")
		}

		// 测试相对路径转换
		manager2 := &StateManager{statePath: statePath}

		// 使用相对路径
		relativePath := "./test-project"
		state2, err := manager2.LoadProjectState(relativePath)
		if err != nil {
			t.Errorf("LoadProjectState(relative) error = %v", err)
		}

		// 验证路径被转换为绝对路径
		absPath, err := filepath.Abs(relativePath)
		if err != nil {
			t.Fatalf("Failed to get absolute path: %v", err)
		}

		if state2.ProjectPath != absPath {
			t.Errorf("Relative path not converted to absolute: %v, want %v", state2.ProjectPath, absPath)
		}
	})

	t.Run("State file structure", func(t *testing.T) {
		manager := &StateManager{statePath: statePath}

		projectPath := filepath.Join(tmpDir, "struct-test")

		// 创建复杂状态
		state := &spec.ProjectState{
			ProjectPath:     projectPath,
			PreferredTarget: "claude_code",
			Skills: map[string]spec.SkillVars{
				"complex-skill": {
					SkillID: "complex-skill",
					Version: "1.2.3",
					Variables: map[string]string{
						"name":    "Test Project",
						"version": "1.0.0",
						"author":  "Test Author",
						"url":     "https://example.com",
					},
				},
				"simple-skill": {
					SkillID:   "simple-skill",
					Version:   "0.1.0",
					Variables: map[string]string{},
				},
			},
		}

		if err := manager.SaveProjectState(state); err != nil {
			t.Errorf("SaveProjectState() error = %v", err)
		}

		// 验证JSON结构
		data, err := os.ReadFile(manager.statePath)
		if err != nil {
			t.Errorf("Failed to read state file: %v", err)
		}

		var parsedData map[string]interface{}
		if err := json.Unmarshal(data, &parsedData); err != nil {
			t.Errorf("Failed to parse state file JSON: %v", err)
		}

		// 验证顶层结构
		projectData, exists := parsedData[projectPath].(map[string]interface{})
		if !exists {
			t.Error("Project data not found in parsed JSON")
		}

		// 验证字段
		if preferredTarget, exists := projectData["preferred_target"].(string); !exists || preferredTarget != "claude_code" {
			t.Errorf("Preferred target in JSON = %v, want claude_code", preferredTarget)
		}

		skillsData, exists := projectData["skills"].(map[string]interface{})
		if !exists {
			t.Error("Skills data not found in parsed JSON")
		}

		if len(skillsData) != 2 {
			t.Errorf("Skills count in JSON = %d, want 2", len(skillsData))
		}

		// 验证复杂技能
		complexSkillData, exists := skillsData["complex-skill"].(map[string]interface{})
		if !exists {
			t.Error("complex-skill data not found in parsed JSON")
		}

		if version, exists := complexSkillData["version"].(string); !exists || version != "1.2.3" {
			t.Errorf("complex-skill version in JSON = %v, want 1.2.3", version)
		}

		variablesData, exists := complexSkillData["variables"].(map[string]interface{})
		if !exists {
			t.Error("complex-skill variables not found in parsed JSON")
		}

		if len(variablesData) != 4 {
			t.Errorf("complex-skill variables count = %d, want 4", len(variablesData))
		}

		if name, exists := variablesData["name"].(string); !exists || name != "Test Project" {
			t.Errorf("complex-skill name variable = %v, want Test Project", name)
		}
	})
}
