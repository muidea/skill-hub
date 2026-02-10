package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/config"
	"skill-hub/pkg/utils"
)

// Adapter 本地接口定义（避免循环导入）
type Adapter interface {
	Apply(skillID string, content string, variables map[string]string) error
	Extract(skillID string) (string, error)
	Remove(skillID string) error
	List() ([]string, error)
	Supports() bool
	Cleanup() error
	GetBackupPath() string
	GetTarget() string
	GetSkillPath(skillID string) (string, error)
	WithProjectMode() Adapter
	WithGlobalMode() Adapter
	GetMode() string
}

// ClaudeAdapter 实现Claude配置文件的适配器
type ClaudeAdapter struct {
	configPath string
	mode       string // "global" 或 "project"
}

// NewClaudeAdapter 创建新的Claude适配器
func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{
		mode: "global",
	}
}

// WithProjectMode 设置为项目模式（向后兼容）
func (a *ClaudeAdapter) WithProjectMode() *ClaudeAdapter {
	a.mode = "project"
	return a
}

// WithGlobalMode 设置为全局模式（向后兼容）
func (a *ClaudeAdapter) WithGlobalMode() *ClaudeAdapter {
	a.mode = "global"
	return a
}

// SetProjectMode 设置为项目模式
func (a *ClaudeAdapter) SetProjectMode() {
	a.mode = "project"
}

// SetGlobalMode 设置为全局模式
func (a *ClaudeAdapter) SetGlobalMode() {
	a.mode = "global"
}

// GetTarget 获取适配器对应的target类型
func (a *ClaudeAdapter) GetTarget() string {
	return "claude_code"
}

// GetSkillPath 获取技能在目标系统中的路径
func (a *ClaudeAdapter) GetSkillPath(skillID string) (string, error) {
	return a.getConfigPath()
}

// GetMode 获取当前模式（project/global）
func (a *ClaudeAdapter) GetMode() string {
	return a.mode
}

// Apply 应用技能到Claude配置文件
func (a *ClaudeAdapter) Apply(skillID string, content string, variables map[string]string) error {
	// 获取配置文件路径
	configPath, err := a.getConfigPath()
	if err != nil {
		return err
	}
	a.configPath = configPath

	fmt.Printf("应用技能到Claude配置文件: %s\n", configPath)

	// 渲染模板内容
	renderedContent, err := a.renderTemplate(content, variables)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 读取现有配置
	configData, err := a.readConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建默认配置
			configData = a.createDefaultConfig()
		} else {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 注入技能内容
	if err := a.injectSkill(configData, skillID, renderedContent); err != nil {
		return fmt.Errorf("注入技能失败: %w", err)
	}

	// 写入配置文件
	return a.writeConfig(configData)
}

// Extract 从Claude配置文件提取技能内容
func (a *ClaudeAdapter) Extract(skillID string) (string, error) {
	configPath, err := a.getConfigPath()
	if err != nil {
		return "", err
	}
	a.configPath = configPath

	// 读取配置文件
	configData, err := a.readConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("配置文件不存在: %s", configPath)
		}
		return "", fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 提取技能内容
	return a.extractSkill(configData, skillID)
}

// Remove 从Claude配置文件移除技能
func (a *ClaudeAdapter) Remove(skillID string) error {
	configPath, err := a.getConfigPath()
	if err != nil {
		return err
	}
	a.configPath = configPath

	// 读取配置文件
	configData, err := a.readConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需移除
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 移除技能
	if err := a.removeSkill(configData, skillID); err != nil {
		return err
	}

	// 写入配置文件
	return a.writeConfig(configData)
}

// List 列出Claude配置文件中的所有技能
func (a *ClaudeAdapter) List() ([]string, error) {
	configPath, err := a.getConfigPath()
	if err != nil {
		return nil, err
	}
	a.configPath = configPath

	// 读取配置文件
	configData, err := a.readConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 列出所有技能
	return a.listSkills(configData), nil
}

// Supports 检查是否支持当前环境
func (a *ClaudeAdapter) Supports() bool {
	// 总是返回true，因为Claude适配器总是可用的
	return true
}

// GetConfigPath 获取配置文件路径（公开方法）
func (a *ClaudeAdapter) GetConfigPath() (string, error) {
	return a.getConfigPath()
}

// getConfigPath 获取配置文件路径
func (a *ClaudeAdapter) getConfigPath() (string, error) {
	if a.mode == "project" {
		// 项目级配置
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("获取当前目录失败: %w", err)
		}
		return filepath.Join(cwd, ".clauderc"), nil
	}

	// 全局配置
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}

	// 展开路径中的~
	return expandPath(cfg.ClaudeConfigPath), nil
}

// expandPath 展开路径中的~为用户主目录
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// readConfig 读取配置文件
func (a *ClaudeAdapter) readConfig() (map[string]interface{}, error) {
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		return nil, err
	}

	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return configData, nil
}

// writeConfig 写入配置文件（原子操作）
func (a *ClaudeAdapter) writeConfig(configData map[string]interface{}) error {
	return utils.SafeWriteJSONFile(a.configPath, configData)
}

// createDefaultConfig 创建默认配置
func (a *ClaudeAdapter) createDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"version": "1.0",
		"settings": map[string]interface{}{
			"editor": map[string]interface{}{
				"theme":    "dark",
				"fontSize": 14,
			},
		},
		"customInstructions": []interface{}{},
		"systemPrompts":      map[string]interface{}{},
	}
}

// renderTemplate 渲染模板内容
func (a *ClaudeAdapter) renderTemplate(content string, variables map[string]string) (string, error) {
	// 简单替换变量
	result := content
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// injectSkill 注入技能到配置
func (a *ClaudeAdapter) injectSkill(configData map[string]interface{}, skillID string, content string) error {
	// 创建带标记块的内容
	markedContent := fmt.Sprintf("/* SKILL-HUB BEGIN: %s */\n%s\n/* SKILL-HUB END: %s */",
		skillID, content, skillID)

	// 确保customInstructions数组存在
	if _, exists := configData["customInstructions"]; !exists {
		configData["customInstructions"] = []interface{}{}
	}

	instructions, ok := configData["customInstructions"].([]interface{})
	if !ok {
		// 如果不是数组，转换为数组
		instructions = []interface{}{}
	}

	// 查找是否已存在该技能的指令
	found := false
	for i, instr := range instructions {
		if instrMap, ok := instr.(map[string]interface{}); ok {
			if name, exists := instrMap["name"].(string); exists && name == skillID {
				// 更新现有指令
				instrMap["content"] = markedContent
				instructions[i] = instrMap
				found = true
				break
			}
		}
	}

	// 如果没找到，添加新指令
	if !found {
		newInstruction := map[string]interface{}{
			"name":    skillID,
			"content": markedContent,
		}
		instructions = append(instructions, newInstruction)
	}

	configData["customInstructions"] = instructions
	return nil
}

// extractSkill 从配置提取技能内容
func (a *ClaudeAdapter) extractSkill(configData map[string]interface{}, skillID string) (string, error) {
	instructions, exists := configData["customInstructions"]
	if !exists {
		return "", fmt.Errorf("未找到customInstructions字段")
	}

	instructionsList, ok := instructions.([]interface{})
	if !ok {
		return "", fmt.Errorf("customInstructions不是数组")
	}

	// 查找指定技能的指令
	for _, instr := range instructionsList {
		if instrMap, ok := instr.(map[string]interface{}); ok {
			if name, exists := instrMap["name"].(string); exists && name == skillID {
				if content, exists := instrMap["content"].(string); exists {
					// 提取标记块内的内容
					return extractMarkedContent(content, skillID)
				}
			}
		}
	}

	return "", fmt.Errorf("未找到技能 '%s'", skillID)
}

// removeSkill 从配置移除技能
func (a *ClaudeAdapter) removeSkill(configData map[string]interface{}, skillID string) error {
	instructions, exists := configData["customInstructions"]
	if !exists {
		return nil // 没有指令，无需移除
	}

	instructionsList, ok := instructions.([]interface{})
	if !ok {
		return fmt.Errorf("customInstructions不是数组")
	}

	// 过滤掉指定技能的指令
	var newInstructions []interface{}
	for _, instr := range instructionsList {
		if instrMap, ok := instr.(map[string]interface{}); ok {
			if name, exists := instrMap["name"].(string); exists && name == skillID {
				continue // 跳过要移除的技能
			}
		}
		newInstructions = append(newInstructions, instr)
	}

	configData["customInstructions"] = newInstructions
	return nil
}

// listSkills 列出所有技能
func (a *ClaudeAdapter) listSkills(configData map[string]interface{}) []string {
	var skillIDs []string

	instructions, exists := configData["customInstructions"]
	if !exists {
		return skillIDs
	}

	instructionsList, ok := instructions.([]interface{})
	if !ok {
		return skillIDs
	}

	// 收集所有技能ID
	for _, instr := range instructionsList {
		if instrMap, ok := instr.(map[string]interface{}); ok {
			if name, exists := instrMap["name"].(string); exists {
				// 检查是否包含Skill Hub标记
				if content, exists := instrMap["content"].(string); exists {
					if strings.Contains(content, "SKILL-HUB BEGIN:") {
						skillIDs = append(skillIDs, name)
					}
				}
			}
		}
	}

	return skillIDs
}

// extractMarkedContent 从标记块中提取内容
func extractMarkedContent(content, skillID string) (string, error) {
	beginMarker := fmt.Sprintf("/* SKILL-HUB BEGIN: %s */", skillID)
	endMarker := fmt.Sprintf("/* SKILL-HUB END: %s */", skillID)

	beginIdx := strings.Index(content, beginMarker)
	if beginIdx == -1 {
		return "", fmt.Errorf("未找到开始标记")
	}

	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("未找到结束标记")
	}

	// 提取标记块内的内容
	start := beginIdx + len(beginMarker)
	extracted := strings.TrimSpace(content[start:endIdx])

	return extracted, nil
}

// Cleanup 清理临时文件（备份文件、临时文件等）
func (a *ClaudeAdapter) Cleanup() error {
	if a.configPath == "" {
		// 如果没有设置配置文件路径，尝试获取
		configPath, err := a.getConfigPath()
		if err != nil {
			return err
		}
		a.configPath = configPath
	}

	// 清理临时文件
	backupPath := a.configPath + ".bak"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理备份文件失败 %s: %w", backupPath, err)
	}

	tmpPath := a.configPath + ".tmp"
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理临时文件失败 %s: %w", tmpPath, err)
	}

	return nil
}

// GetBackupPath 获取备份文件路径
func (a *ClaudeAdapter) GetBackupPath() string {
	if a.configPath == "" {
		// 如果没有设置配置文件路径，返回空
		return ""
	}
	return a.configPath + ".bak"
}
