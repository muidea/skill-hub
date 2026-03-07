package skill

import (
	"crypto/md5"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/muidea/skill-hub/pkg/spec"
)

func ParseFrontmatter(content []byte) (map[string]interface{}, error) {
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
	}

	var frontmatterLines []string
	foundEnd := false
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			foundEnd = true
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	if !foundEnd {
		return nil, fmt.Errorf("无效的SKILL.md格式: frontmatter未正确结束")
	}

	if len(frontmatterLines) == 0 {
		return nil, fmt.Errorf("无效的SKILL.md格式: frontmatter为空")
	}

	frontmatter := strings.Join(frontmatterLines, "\n")
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
	}

	return data, nil
}

func ExtractVersion(content []byte) string {
	data, err := ParseFrontmatter(content)
	if err != nil {
		return "1.0.0"
	}
	return extractVersionFromData(data)
}

func extractVersionFromData(data map[string]interface{}) string {
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if v, ok := metadata["version"].(string); ok {
			return v
		}
	}
	if v, ok := data["version"].(string); ok {
		return v
	}
	return "1.0.0"
}

func ContentHash(content []byte) string {
	hash := md5.Sum(content)
	return fmt.Sprintf("%x", hash)
}

func NormalizeCompatibility(compatData interface{}) string {
	if compatData == nil {
		return ""
	}
	switch v := compatData.(type) {
	case string:
		return v
	case map[string]interface{}:
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
			return "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
		}
	}
	return ""
}

func ParseSkillMetadata(content []byte, skillID string) (*spec.SkillMetadata, error) {
	data, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	meta := &spec.SkillMetadata{
		ID: skillID,
	}

	if name, ok := data["name"].(string); ok {
		meta.Name = name
	} else {
		meta.Name = skillID
	}

	if desc, ok := data["description"].(string); ok {
		meta.Description = desc
	}

	meta.Version = extractVersionFromData(data)

	if author, ok := data["author"].(string); ok {
		meta.Author = author
	} else if source, ok := data["source"].(string); ok {
		meta.Author = source
	} else {
		meta.Author = "unknown"
	}

	if tagsStr, ok := data["tags"].(string); ok {
		meta.Tags = strings.Split(tagsStr, ",")
		for i, tag := range meta.Tags {
			meta.Tags[i] = strings.TrimSpace(tag)
		}
	}

	meta.Compatibility = NormalizeCompatibility(data["compatibility"])

	return meta, nil
}

func ParseSkill(content []byte, skillID string) (*spec.Skill, error) {
	data, err := ParseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	s := &spec.Skill{
		ID: skillID,
	}

	if name, ok := data["name"].(string); ok {
		s.Name = name
	} else {
		s.Name = skillID
	}

	if desc, ok := data["description"].(string); ok {
		s.Description = desc
	}

	s.Version = extractVersionFromData(data)

	if author, ok := data["author"].(string); ok {
		s.Author = author
	} else if source, ok := data["source"].(string); ok {
		s.Author = source
	} else {
		s.Author = "unknown"
	}

	if tagsStr, ok := data["tags"].(string); ok {
		s.Tags = strings.Split(tagsStr, ",")
		for i, tag := range s.Tags {
			s.Tags[i] = strings.TrimSpace(tag)
		}
	}

	s.Compatibility = NormalizeCompatibility(data["compatibility"])

	return s, nil
}

func ValidateSkillFile(content []byte) error {
	data, err := ParseFrontmatter(content)
	if err != nil {
		return err
	}

	if _, ok := data["name"].(string); !ok {
		return fmt.Errorf("缺少必需字段: name")
	}
	if _, ok := data["description"].(string); !ok {
		return fmt.Errorf("缺少必需字段: description")
	}

	return nil
}
