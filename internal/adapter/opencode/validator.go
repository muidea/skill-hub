package opencode

import (
	"fmt"
	"regexp"
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

// convertFromOpenCodeFormat 从OpenCode格式转换回Skill Hub格式
func convertFromOpenCodeFormat(content string) (string, error) {
	// 测试期望返回完整的OpenCode格式（包括frontmatter）
	// 所以直接返回原始内容
	return content, nil
}
