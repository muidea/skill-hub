package skill

import (
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		wantKey string
		wantVal string
	}{
		{
			name:    "valid frontmatter",
			content: "---\nname: test\nversion: 1.0.0\n---\n# Content",
			wantKey: "name",
			wantVal: "test",
		},
		{
			name:    "missing opening delimiter",
			content: "name: test\n---\n# Content",
			wantErr: true,
		},
		{
			name:    "missing closing delimiter",
			content: "---\nname: test\n# Content",
			wantErr: true,
		},
		{
			name:    "empty frontmatter",
			content: "---\n---\n# Content",
			wantErr: true,
		},
		{
			name:    "too short",
			content: "---",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ParseFrontmatter([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val, ok := data[tt.wantKey].(string); !ok || val != tt.wantVal {
				t.Fatalf("expected %s=%s, got %v", tt.wantKey, tt.wantVal, data[tt.wantKey])
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "top-level version",
			content: "---\nname: test\nversion: 2.3.4\n---\n# Content",
			want:    "2.3.4",
		},
		{
			name:    "metadata nested version",
			content: "---\nname: test\nmetadata:\n  version: 1.2.0\n---\n# Content",
			want:    "1.2.0",
		},
		{
			name:    "metadata version takes priority over top-level",
			content: "---\nname: test\nversion: 1.0.0\nmetadata:\n  version: 2.0.0\n---\n# Content",
			want:    "2.0.0",
		},
		{
			name:    "no version defaults to 1.0.0",
			content: "---\nname: test\n---\n# Content",
			want:    "1.0.0",
		},
		{
			name:    "invalid frontmatter defaults to 1.0.0",
			content: "no frontmatter here",
			want:    "1.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVersion([]byte(tt.content))
			if got != tt.want {
				t.Fatalf("ExtractVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	hash1 := ContentHash([]byte("hello"))
	hash2 := ContentHash([]byte("hello"))
	hash3 := ContentHash([]byte("world"))

	if hash1 != hash2 {
		t.Fatal("same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Fatal("different content should produce different hash")
	}
	if len(hash1) != 32 {
		t.Fatalf("expected 32-char hex string, got %d chars", len(hash1))
	}
}

func TestNormalizeCompatibility(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"nil", nil, ""},
		{"string", "Custom compat", "Custom compat"},
		{
			"map with all true",
			map[string]interface{}{
				"cursor":      true,
				"claude_code": true,
				"open_code":   true,
				"shell":       true,
			},
			"Designed for Cursor, Claude Code, OpenCode, Shell (or similar AI coding assistants)",
		},
		{
			"map with partial",
			map[string]interface{}{
				"cursor":      true,
				"claude_code": false,
				"open_code":   true,
			},
			"Designed for Cursor, OpenCode (or similar AI coding assistants)",
		},
		{"empty map", map[string]interface{}{}, ""},
		{"unsupported type", 42, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeCompatibility(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeCompatibility() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSkillMetadata(t *testing.T) {
	content := []byte("---\nname: my-skill\ndescription: A test skill\nversion: 2.0.0\nauthor: john\ntags: go, refactor\ncompatibility: Custom\n---\n# Content")
	meta, err := ParseSkillMetadata(content, "my-skill-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ID != "my-skill-id" {
		t.Fatalf("ID = %q, want %q", meta.ID, "my-skill-id")
	}
	if meta.Name != "my-skill" {
		t.Fatalf("Name = %q, want %q", meta.Name, "my-skill")
	}
	if meta.Description != "A test skill" {
		t.Fatalf("Description = %q", meta.Description)
	}
	if meta.Version != "2.0.0" {
		t.Fatalf("Version = %q, want %q", meta.Version, "2.0.0")
	}
	if meta.Author != "john" {
		t.Fatalf("Author = %q, want %q", meta.Author, "john")
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "go" || meta.Tags[1] != "refactor" {
		t.Fatalf("Tags = %v", meta.Tags)
	}
	if meta.Compatibility != "Custom" {
		t.Fatalf("Compatibility = %q", meta.Compatibility)
	}
}

func TestParseSkillMetadataDefaults(t *testing.T) {
	content := []byte("---\nname: x\ndescription: y\n---\n# Content")
	meta, err := ParseSkillMetadata(content, "fallback-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Version != "1.0.0" {
		t.Fatalf("Version = %q, want %q", meta.Version, "1.0.0")
	}
	if meta.Author != "unknown" {
		t.Fatalf("Author = %q, want %q", meta.Author, "unknown")
	}
}

func TestParseSkillMetadataSourceFallback(t *testing.T) {
	content := []byte("---\nname: x\ndescription: y\nsource: legacy-author\n---\n# Content")
	meta, err := ParseSkillMetadata(content, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Author != "legacy-author" {
		t.Fatalf("Author = %q, want %q", meta.Author, "legacy-author")
	}
}

func TestParseSkill(t *testing.T) {
	content := []byte("---\nname: my-skill\ndescription: desc\nmetadata:\n  version: 3.0.0\nauthor: alice\n---\n# Content")
	s, err := ParseSkill(content, "skill-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.ID != "skill-id" {
		t.Fatalf("ID = %q", s.ID)
	}
	if s.Version != "3.0.0" {
		t.Fatalf("Version = %q, want %q", s.Version, "3.0.0")
	}
	if s.Author != "alice" {
		t.Fatalf("Author = %q", s.Author)
	}
}

func TestValidateSkillFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid",
			content: "---\nname: test\ndescription: desc\n---\n# Content",
			wantErr: false,
		},
		{
			name:    "missing name",
			content: "---\ndescription: desc\n---\n# Content",
			wantErr: true,
		},
		{
			name:    "missing description",
			content: "---\nname: test\n---\n# Content",
			wantErr: true,
		},
		{
			name:    "invalid frontmatter",
			content: "no frontmatter",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSkillFile([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSkillFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
