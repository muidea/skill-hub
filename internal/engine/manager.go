package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/pkg/spec"
)

// SkillManager 管理技能加载和操作
type SkillManager struct {
	skillsDir string
}

// NewSkillManager 创建新的技能管理器
func NewSkillManager() (*SkillManager, error) {
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}
	return &SkillManager{skillsDir: skillsDir}, nil
}

// LoadSkill 加载指定ID的技能
func (m *SkillManager) LoadSkill(skillID string) (*spec.Skill, error) {
	// 首先尝试直接路径
	skillDir := filepath.Join(m.skillsDir, skillID)
	skill, err := m.loadSkillFromDirectory(skillDir, skillID)
	if err == nil {
		return skill, nil
	}

	// 如果失败，尝试在 skills/skills/ 子目录中查找
	skillsSubDir := filepath.Join(m.skillsDir, "skills", skillID)
	skill, err = m.loadSkillFromDirectory(skillsSubDir, skillID)
	if err == nil {
		return skill, nil
	}

	return nil, fmt.Errorf("技能 '%s' 不存在", skillID)
}

// loadSkillFromDirectory 从目录加载技能
func (m *SkillManager) loadSkillFromDirectory(skillDir, skillID string) (*spec.Skill, error) {
	// 检查技能目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("目录不存在")
	}

	// 首先尝试新格式：SKILL.md
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return m.loadSkillFromMarkdown(skillMdPath, skillID)
	}

	// 然后尝试旧格式：skill.yaml
	yamlPath := filepath.Join(skillDir, "skill.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return m.loadSkillFromYAML(yamlPath, skillID)
	}

	return nil, fmt.Errorf("未找到技能文件")
}

// loadSkillFromMarkdown 从SKILL.md文件加载技能
func (m *SkillManager) loadSkillFromMarkdown(mdPath, skillID string) (*spec.Skill, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML frontmatter
	var skillData map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err != nil {
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
	}

	// 转换为Skill对象
	skill := &spec.Skill{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skill.Name = name
	} else {
		skill.Name = skillID
	}

	// 设置描述
	if desc, ok := skillData["description"].(string); ok {
		skill.Description = desc
	}

	// 设置版本
	skill.Version = "1.0.0"
	if version, ok := skillData["version"].(string); ok {
		skill.Version = version
	}

	// 设置作者
	if source, ok := skillData["source"].(string); ok {
		skill.Author = source
	} else {
		skill.Author = "unknown"
	}

	// 设置标签
	if tagsStr, ok := skillData["tags"].(string); ok {
		skill.Tags = strings.Split(tagsStr, ",")
		for i, tag := range skill.Tags {
			skill.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 设置兼容性（默认为所有工具）
	skill.Compatibility = spec.Compatibility{
		Cursor:     true,
		ClaudeCode: true,
	}

	return skill, nil
}

// loadSkillFromYAML 从skill.yaml文件加载技能
func (m *SkillManager) loadSkillFromYAML(yamlPath, skillID string) (*spec.Skill, error) {
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("读取skill.yaml失败: %w", err)
	}

	var skill spec.Skill
	if err := yaml.Unmarshal(yamlData, &skill); err != nil {
		return nil, fmt.Errorf("解析skill.yaml失败: %w", err)
	}

	// 验证必需字段
	if skill.ID == "" {
		skill.ID = skillID
	}
	if skill.Name == "" {
		skill.Name = skillID
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}

	// 确保ID与目录名一致
	if skill.ID != skillID {
		return nil, fmt.Errorf("技能ID不匹配: 目录名为%s, skill.yaml中为%s", skillID, skill.ID)
	}

	return &skill, nil
}

// LoadAllSkills 加载所有技能
func (m *SkillManager) LoadAllSkills() ([]*spec.Skill, error) {
	var skills []*spec.Skill

	// 首先检查是否有 skills/skills/ 子目录（新格式）
	skillsSubDir := filepath.Join(m.skillsDir, "skills")
	if _, err := os.Stat(skillsSubDir); err == nil {
		// 加载 skills/skills/ 目录下的技能
		subSkills, err := m.loadSkillsFromDirectory(skillsSubDir)
		if err != nil {
			return nil, err
		}
		skills = append(skills, subSkills...)
	}

	// 然后加载根目录下的技能（旧格式）
	rootSkills, err := m.loadSkillsFromDirectory(m.skillsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	skills = append(skills, rootSkills...)

	return skills, nil
}

// loadSkillsFromDirectory 从目录加载所有技能
func (m *SkillManager) loadSkillsFromDirectory(dir string) ([]*spec.Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var skills []*spec.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(dir, skillID)

		// 尝试加载技能
		skill, err := m.loadSkillFromDirectory(skillDir, skillID)
		if err != nil {
			// 不输出警告，因为可能有很多非技能目录
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// GetSkillPrompt 获取技能的提示词内容
func (m *SkillManager) GetSkillPrompt(skillID string) (string, error) {
	// 首先尝试直接路径
	skillDir := filepath.Join(m.skillsDir, skillID)
	promptPath := filepath.Join(skillDir, "prompt.md")

	// 检查prompt.md文件是否存在
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		// 尝试在 skills/skills/ 子目录中查找
		skillsSubDir := filepath.Join(m.skillsDir, "skills", skillID)
		promptPath = filepath.Join(skillsSubDir, "prompt.md")

		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			// 对于SKILL.md格式，使用SKILL.md文件本身作为提示词
			skillMdPath := filepath.Join(skillsSubDir, "SKILL.md")
			if _, err := os.Stat(skillMdPath); err == nil {
				promptData, err := os.ReadFile(skillMdPath)
				if err != nil {
					return "", fmt.Errorf("读取SKILL.md失败: %w", err)
				}
				return string(promptData), nil
			}

			return "", fmt.Errorf("技能 '%s' 缺少提示词文件", skillID)
		}
	}

	promptData, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("读取提示词文件失败: %w", err)
	}

	return string(promptData), nil
}

// SkillExists 检查技能是否存在
func (m *SkillManager) SkillExists(skillID string) bool {
	// 首先尝试直接路径
	skillDir := filepath.Join(m.skillsDir, skillID)

	// 检查旧格式
	if m.checkSkillExistsInDirectory(skillDir) {
		return true
	}

	// 尝试在 skills/skills/ 子目录中查找
	skillsSubDir := filepath.Join(m.skillsDir, "skills", skillID)
	return m.checkSkillExistsInDirectory(skillsSubDir)
}

// checkSkillExistsInDirectory 检查目录中是否存在技能
func (m *SkillManager) checkSkillExistsInDirectory(skillDir string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return false
	}

	// 检查新格式：SKILL.md
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return true
	}

	// 检查旧格式：skill.yaml
	yamlPath := filepath.Join(skillDir, "skill.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return false
	}

	// 检查prompt.md是否存在
	promptPath := filepath.Join(skillDir, "prompt.md")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetSkillsDir 获取技能目录路径（包级函数）
func GetSkillsDir() (string, error) {
	manager, err := NewSkillManager()
	if err != nil {
		return "", err
	}
	return manager.skillsDir, nil
}
