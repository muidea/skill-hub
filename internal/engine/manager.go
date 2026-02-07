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

	// 只支持SKILL.md格式
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return m.loadSkillFromMarkdown(skillMdPath, skillID)
	}

	return nil, fmt.Errorf("未找到SKILL.md文件")
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

	// 设置兼容性
	skill.Compatibility = spec.Compatibility{
		Cursor:     true,
		ClaudeCode: true,
		OpenCode:   false,
		Shell:      false,
	}

	// 从YAML读取兼容性设置
	if compatData, ok := skillData["compatibility"].(map[string]interface{}); ok {
		if cursorVal, ok := compatData["cursor"].(bool); ok {
			skill.Compatibility.Cursor = cursorVal
		}
		if claudeVal, ok := compatData["claude_code"].(bool); ok {
			skill.Compatibility.ClaudeCode = claudeVal
		}
		if openCodeVal, ok := compatData["open_code"].(bool); ok {
			skill.Compatibility.OpenCode = openCodeVal
		}
		if shellVal, ok := compatData["shell"].(bool); ok {
			skill.Compatibility.Shell = shellVal
		}
	}

	return skill, nil
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
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 检查SKILL.md文件是否存在
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		// 尝试在 skills/skills/ 子目录中查找
		skillsSubDir := filepath.Join(m.skillsDir, "skills", skillID)
		skillMdPath = filepath.Join(skillsSubDir, "SKILL.md")

		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			return "", fmt.Errorf("技能 '%s' 缺少SKILL.md文件", skillID)
		}
	}

	// 读取SKILL.md文件内容作为提示词
	promptData, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", fmt.Errorf("读取SKILL.md失败: %w", err)
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

	// 只检查SKILL.md格式
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return true
	}

	return false
}

// GetSkillsDir 获取技能目录路径（包级函数）
func GetSkillsDir() (string, error) {
	manager, err := NewSkillManager()
	if err != nil {
		return "", err
	}
	return manager.skillsDir, nil
}
