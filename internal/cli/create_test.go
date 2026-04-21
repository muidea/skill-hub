package cli

import (
	"strings"
	"testing"

	"github.com/muidea/skill-hub/pkg/skill"
)

func TestGenerateSkillContentUsesChineseTemplateAndFormatter(t *testing.T) {
	content, err := generateSkillContent("demo-skill", "用于演示的技能")
	if err != nil {
		t.Fatalf("generateSkillContent() error = %v", err)
	}

	required := []string{
		"description: 用于演示的技能",
		"## 适用场景",
		"## 工作流程",
		"## Formatter",
		"`skill-hub validate demo-skill --links`",
		"不要声明当前项目无法执行的 formatter",
	}
	for _, item := range required {
		if !strings.Contains(content, item) {
			t.Fatalf("generated content missing %q:\n%s", item, content)
		}
	}
	if err := skill.ValidateSkillFile([]byte(content)); err != nil {
		t.Fatalf("generated content should be a valid SKILL.md: %v\n%s", err, content)
	}
}
