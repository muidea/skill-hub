package opencode

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// validateSkillName 验证技能名称是否符合OpenCode规范
func validateSkillName(name string) error {
	// OpenCode命名规范：^[a-z0-9]+(-[a-z0-9]+)*$
	pattern := `^[a-z0-9]+(-[a-z0-9]+)*$`
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return fmt.Errorf("正则匹配失败: %w", err)
	}

	if !matched {
		return fmt.Errorf("技能名称 '%s' 不符合OpenCode规范。要求：小写字母数字，用连字符分隔，不能以连字符开头或结尾，不能有连续连字符", name)
	}

	// 检查长度：1-64字符
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("技能名称长度必须在1-64字符之间，当前长度：%d", len(name))
	}

	return nil
}

// validateDescription 验证描述长度
func validateDescription(desc string) error {
	// 描述长度：1-1024字符
	if len(desc) < 1 || len(desc) > 1024 {
		return fmt.Errorf("描述长度必须在1-1024字符之间，当前长度：%d", len(desc))
	}
	return nil
}

// convertToOpenCodeFormat 转换Skill Hub格式为OpenCode格式
func convertToOpenCodeFormat(content string, skillID string) (string, error) {
	// 解析原始内容中的frontmatter
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || lines[0] != "---" {
		// 如果没有frontmatter，创建基本的OpenCode格式
		return createBasicOpenCodeFormat(content, skillID)
	}

	// 提取frontmatter
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := true

	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			inFrontmatter = false
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, lines[i])
		} else {
			contentLines = append(contentLines, lines[i])
		}
	}

	// 解析原始frontmatter
	originalFrontmatter := strings.Join(frontmatterLines, "\n")
	var originalData map[string]interface{}
	if err := yaml.Unmarshal([]byte(originalFrontmatter), &originalData); err != nil {
		return "", fmt.Errorf("解析原始frontmatter失败: %w", err)
	}

	// 创建OpenCode兼容的frontmatter
	openCodeData := make(map[string]interface{})

	// 处理必需字段
	if name, ok := originalData["name"].(string); ok {
		if err := validateSkillName(name); err != nil {
			return "", err
		}
		openCodeData["name"] = name
	} else {
		// 使用技能ID作为名称
		if err := validateSkillName(skillID); err != nil {
			return "", err
		}
		openCodeData["name"] = skillID
	}

	if desc, ok := originalData["description"].(string); ok {
		if err := validateDescription(desc); err != nil {
			return "", err
		}
		openCodeData["description"] = desc
	} else {
		// 如果没有描述，使用默认描述
		defaultDesc := fmt.Sprintf("Skill: %s", skillID)
		openCodeData["description"] = defaultDesc
	}

	// 处理可选字段
	if license, ok := originalData["license"].(string); ok {
		openCodeData["license"] = license
	}

	if compatStr, ok := originalData["compatibility"].(string); ok {
		// 检查字符串中是否包含OpenCode
		if strings.Contains(strings.ToLower(compatStr), "opencode") {
			openCodeData["compatibility"] = compatStr
		}
	}

	// 添加metadata字段
	metadata := make(map[string]string)
	metadata["source"] = "skill-hub"
	if version, ok := originalData["version"].(string); ok {
		metadata["version"] = version
	}
	if author, ok := originalData["author"].(string); ok {
		metadata["author"] = author
	}
	openCodeData["metadata"] = metadata

	// 生成YAML frontmatter
	yamlData, err := yaml.Marshal(openCodeData)
	if err != nil {
		return "", fmt.Errorf("生成YAML失败: %w", err)
	}

	// 构建完整的SKILL.md内容
	frontmatter := string(yamlData)
	contentText := strings.Join(contentLines, "\n")

	return fmt.Sprintf("---\n%s---\n%s", frontmatter, contentText), nil
}

// createBasicOpenCodeFormat 创建基本的OpenCode格式
func createBasicOpenCodeFormat(content string, skillID string) (string, error) {
	// 验证技能ID
	if err := validateSkillName(skillID); err != nil {
		return "", err
	}

	// 创建基本的frontmatter
	frontmatter := map[string]interface{}{
		"name":        skillID,
		"description": fmt.Sprintf("Skill: %s", skillID),
		"metadata": map[string]string{
			"source": "skill-hub",
		},
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("生成YAML失败: %w", err)
	}

	return fmt.Sprintf("---\n%s---\n%s", string(yamlData), content), nil
}
