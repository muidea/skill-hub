package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/pkg/errors"
	"skill-hub/pkg/fs"
	"skill-hub/pkg/spec"
)

// SkillManager 管理技能加载和操作
type SkillManager struct {
	skillsDir string
	fs        fs.FileSystem
	path      fs.Path
}

// NewSkillManager 创建新的技能管理器
func NewSkillManager() (*SkillManager, error) {
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}
	return NewSkillManagerWithFS(skillsDir, fs.NewRealFileSystem(), fs.NewRealPath()), nil
}

// NewSkillManagerWithFS 创建技能管理器并指定文件系统（用于测试）
func NewSkillManagerWithFS(skillsDir string, fileSystem fs.FileSystem, path fs.Path) *SkillManager {
	return &SkillManager{
		skillsDir: skillsDir,
		fs:        fileSystem,
		path:      path,
	}
}

// LoadSkill 加载指定ID的技能
func (m *SkillManager) LoadSkill(skillID string) (*spec.Skill, error) {
	// 只使用标准结构：skills/skillID
	skillDir := filepath.Join(m.skillsDir, skillID)
	skill, err := m.loadSkillFromDirectory(skillDir, skillID)
	if err == nil {
		return skill, nil
	}

	return nil, errors.NewSkillError("not_found", fmt.Sprintf("技能 '%s' 不存在", skillID))
}

// loadSkillFromDirectory 从目录加载技能
func (m *SkillManager) loadSkillFromDirectory(skillDir, skillID string) (*spec.Skill, error) {
	// 检查技能目录是否存在
	if _, err := m.fs.Stat(skillDir); m.fs.IsNotExist(err) {
		return nil, errors.NewSkillError("not_found", "目录不存在")
	}

	// 只支持SKILL.md格式
	skillMdPath := m.path.Join(skillDir, "SKILL.md")
	if _, err := m.fs.Stat(skillMdPath); err == nil {
		return m.loadSkillFromMarkdown(skillMdPath, skillID)
	}

	return nil, errors.NewSkillError("not_found", "未找到SKILL.md文件")
}

// loadSkillFromMarkdown 从SKILL.md文件加载技能
func (m *SkillManager) loadSkillFromMarkdown(mdPath, skillID string) (*spec.Skill, error) {
	content, err := m.fs.ReadFile(mdPath)
	if err != nil {
		return nil, errors.WrapSkillError("io", "读取SKILL.md失败", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, errors.NewSkillError("invalid", "无效的SKILL.md格式: 缺少frontmatter")
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
		return nil, errors.WrapSkillError("invalid", "解析frontmatter失败", err)
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
	// 从YAML读取兼容性设置（字符串格式）
	if compatData, ok := skillData["compatibility"]; ok {
		switch v := compatData.(type) {
		case string:
			skill.Compatibility = v
		case map[string]interface{}:
			// 向后兼容：将对象格式转换为字符串
			var compatList []string
			if cursorVal, ok := v["cursor"].(bool); ok && cursorVal {
				compatList = append(compatList, "Cursor")
			}
			if claudeVal, ok := v["claude_code"].(bool); ok && claudeVal {
				compatList = append(compatList, "Claude Code")
			}
			if openCodeVal, ok := v["open_code"].(bool); ok && openCodeVal {
				compatList = append(compatList, "OpenCode")
			}
			if shellVal, ok := v["shell"].(bool); ok && shellVal {
				compatList = append(compatList, "Shell")
			}
			if len(compatList) > 0 {
				skill.Compatibility = "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
			}
		}
	}

	return skill, nil
}

// LoadAllSkills 加载所有技能
func (m *SkillManager) LoadAllSkills() ([]*spec.Skill, error) {
	// 只使用标准结构：直接从skills目录加载
	skills, err := m.loadSkillsFromDirectory(m.skillsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// 目录不存在，返回空列表
		return []*spec.Skill{}, nil
	}

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
	// 只使用标准结构：skills/skillID
	skillDir := filepath.Join(m.skillsDir, skillID)
	return m.checkSkillExistsInDirectory(skillDir)
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
