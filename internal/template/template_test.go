package template

import (
	"testing"
)

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			name:     "no variables",
			template: "Hello World",
			want:     []string{},
		},
		{
			name:     "single variable",
			template: "Hello {{.name}}",
			want:     []string{"name"},
		},
		{
			name:     "multiple variables",
			template: "Hello {{.name}}, welcome to {{.project}}",
			want:     []string{"name", "project"},
		},
		{
			name:     "duplicate variables",
			template: "{{.x}} + {{.x}} = {{.y}}",
			want:     []string{"x", "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractVariables(tt.template)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractVariables() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractVariables()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		variables map[string]string
		want      string
	}{
		{
			name:      "simple replacement",
			template:  "Hello {{.name}}",
			variables: map[string]string{"name": "World"},
			want:      "Hello World",
		},
		{
			name:      "multiple replacements",
			template:  "{{.greeting}} {{.name}}",
			variables: map[string]string{"greeting": "Hello", "name": "World"},
			want:      "Hello World",
		},
		{
			name:      "no variables",
			template:  "Hello World",
			variables: map[string]string{"name": "World"},
			want:      "Hello World",
		},
		{
			name:      "variable not in template",
			template:  "Hello {{.name}}",
			variables: map[string]string{"greeting": "Hello"},
			want:      "Hello {{.name}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.template, tt.variables)
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSmartExtract_NoChanges(t *testing.T) {
	template := "Hello {{.name}}"
	variables := map[string]string{"name": "World"}
	modified := "Hello World"

	newTemplate, newVars, err := SmartExtract(template, modified, variables)
	if err != nil {
		t.Errorf("SmartExtract() error = %v", err)
		return
	}
	if newTemplate != template {
		t.Errorf("SmartExtract() template = %v, want %v", newTemplate, template)
	}
	if len(newVars) != len(variables) {
		t.Errorf("SmartExtract() vars length = %v, want %v", len(newVars), len(variables))
	}
	for k, v := range variables {
		if newVars[k] != v {
			t.Errorf("SmartExtract() vars[%s] = %v, want %v", k, newVars[k], v)
		}
	}
}
