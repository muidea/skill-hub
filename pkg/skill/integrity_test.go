package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCoreInformationPreservedAllowsRetiredCompatibilityRemoval(t *testing.T) {
	existingDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\ncompatibility: legacy\nmetadata:\n  version: 1.0.0\n---\n# Test\n\n## Workflow\n")
	candidateDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\nmetadata:\n  version: 1.0.1\n---\n# Test\n\n## Workflow\n")

	if err := EnsureCoreInformationPreserved(existingDir, candidateDir); err != nil {
		t.Fatalf("EnsureCoreInformationPreserved() error = %v, want nil", err)
	}
}

func TestEnsureCoreInformationPreservedRejectsRequiredInformationRemoval(t *testing.T) {
	existingDir := writeIntegritySkill(t, "---\nname: test\ndescription: test skill\nmetadata:\n  version: 1.0.0\n---\n# Test\n\n## Workflow\n")
	candidateDir := writeIntegritySkill(t, "---\nname: test\nmetadata:\n  version: 1.0.1\n---\n# Test\n")

	err := EnsureCoreInformationPreserved(existingDir, candidateDir)
	if err == nil {
		t.Fatal("EnsureCoreInformationPreserved() error = nil, want required information removal error")
	}
	for _, omission := range []string{"frontmatter.description", "section: workflow"} {
		if !strings.Contains(err.Error(), omission) {
			t.Errorf("error = %q, want omission %q", err, omission)
		}
	}
}

func TestEnsureCoreInformationPreservedFencedCodeBlocksAndFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		existing    string
		candidate   string
		wantErrPart string
	}{
		{
			name:      "allows editing shell comment inside backtick fence",
			existing:  "---\nname: remote-verify\ndescription: remote check\nversion: 1.0.0\n---\n# Remote Verify\n\n## Workflow\n```sh\n# 2. 执行远端标准（或 --suite abnormal）功能回归：\npython3 -B run.py\n```\n",
			candidate: "---\nname: remote-verify\ndescription: remote check\nversion: 1.0.1\n---\n# Remote Verify\n\n## Workflow\n```sh\n# 2. 执行远端标准（以及独立的 --suite abnormal）功能回归：\npython3 -B run.py\n```\n",
		},
		{
			name:      "allows removing hash comments inside tilde and indented fences",
			existing:  "---\nname: tilde-skill\ndescription: tilde test\nversion: 1.0.0\n---\n# Tilde Skill\n\n## Steps\n- Step 1\n  ~~~bash\n  ## Sub-comment in fence\n  echo hi\n  ~~~\n",
			candidate: "---\nname: tilde-skill\ndescription: tilde test\nversion: 1.0.1\n---\n# Tilde Skill\n\n## Steps\n- Step 1\n  ~~~bash\n  echo updated\n  ~~~\n",
		},
		{
			name:      "allows editing hash comments inside yaml frontmatter",
			existing:  "---\n# old frontmatter comment\nname: fm-skill\ndescription: fm test\nversion: 1.0.0\n---\n# FM Skill\n\n## Workflow\nbody\n",
			candidate: "---\n# updated frontmatter comment\nname: fm-skill\ndescription: fm test\nversion: 1.0.1\n---\n# FM Skill\n\n## Workflow\nbody\n",
		},
		{
			name:        "nested backtick fences keep outer fence intact and still enforce later headings",
			existing:    "---\nname: nested-fence\ndescription: nested test\nversion: 1.0.0\n---\n# Nested\n\n````markdown\n```sh\n# inside nested fence\n```\n````\n\n## Real Section\nkeep\n",
			candidate:   "---\nname: nested-fence\ndescription: nested test\nversion: 1.0.1\n---\n# Nested\n\n````markdown\n```sh\n# changed inside nested fence\n```\n````\n",
			wantErrPart: "section: real section",
		},
		{
			name:        "inline triple backticks do not suppress subsequent real headings",
			existing:    "---\nname: inline-ticks\ndescription: inline test\nversion: 1.0.0\n---\n# Inline\n\nUse ```inline``` here.\n\n## Required Section\nkeep\n",
			candidate:   "---\nname: inline-ticks\ndescription: inline test\nversion: 1.0.1\n---\n# Inline\n\nUse ```inline``` here.\n",
			wantErrPart: "section: required section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingDir := writeIntegritySkill(t, tt.existing)
			candidateDir := writeIntegritySkill(t, tt.candidate)

			err := EnsureCoreInformationPreserved(existingDir, candidateDir)
			if tt.wantErrPart == "" {
				if err != nil {
					t.Fatalf("EnsureCoreInformationPreserved() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("EnsureCoreInformationPreserved() error = nil, want %q", tt.wantErrPart)
			}
			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErrPart)
			}
		})
	}
}

func TestSkillHeadings_CRLFAndFences(t *testing.T) {
	content := "---\r\n# frontmatter comment\r\nname: crlf-skill\r\ndescription: crlf test\r\nversion: 1.0.0\r\n---\r\n# CRLF Title\r\n\r\n```sh\r\n# code comment\r\n## code subcomment\r\n```\r\n\r\n## 1. Real Guide\r\ntext\r\n"
	got := skillHeadings([]byte(content))
	for _, want := range []string{"crlf title", "real guide"} {
		if _, ok := got[want]; !ok {
			t.Errorf("skillHeadings() missing %q, got %v", want, got)
		}
	}
	for _, unwanted := range []string{"frontmatter comment", "code comment", "code subcomment"} {
		if _, ok := got[unwanted]; ok {
			t.Errorf("skillHeadings() unexpectedly included %q, got %v", unwanted, got)
		}
	}
}

func writeIntegritySkill(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}
