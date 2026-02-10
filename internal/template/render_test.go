package template

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name        string
		tmplContent string
		vars        map[string]string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple text with no variables",
			tmplContent: "Hello World",
			vars:        nil,
			want:        "Hello World",
			wantErr:     false,
		},
		{
			name:        "single variable",
			tmplContent: "Hello {{.name}}",
			vars:        map[string]string{"name": "Alice"},
			want:        "Hello Alice",
			wantErr:     false,
		},
		{
			name:        "multiple variables",
			tmplContent: "{{.greeting}} {{.name}}! Today is {{.day}}.",
			vars: map[string]string{
				"greeting": "Hello",
				"name":     "Bob",
				"day":      "Monday",
			},
			want:    "Hello Bob! Today is Monday.",
			wantErr: false,
		},
		{
			name:        "missing variable uses empty string",
			tmplContent: "Hello {{.name}}",
			vars:        map[string]string{},
			want:        "Hello <no value>",
			wantErr:     false,
		},
		{
			name:        "invalid template syntax",
			tmplContent: "Hello {{.name",
			vars:        map[string]string{"name": "Alice"},
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.tmplContent, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderEmptyVars(t *testing.T) {
	tmplContent := "Just plain text"
	got, err := Render(tmplContent, nil)
	if err != nil {
		t.Fatalf("Render() unexpected error: %v", err)
	}
	if got != tmplContent {
		t.Errorf("Render() = %q, want %q", got, tmplContent)
	}
}

func TestRenderComplexTemplate(t *testing.T) {
	tmplContent := `# Meeting Notes

Date: {{.date}}
Attendees: {{.attendees}}

## Summary
{{.summary}}

## Action Items
{{.actions}}`

	vars := map[string]string{
		"date":      "2024-01-15",
		"attendees": "Alice, Bob, Charlie",
		"summary":   "Discussed project roadmap",
		"actions":   "- Review design docs\n- Schedule follow-up",
	}

	got, err := Render(tmplContent, vars)
	if err != nil {
		t.Fatalf("Render() unexpected error: %v", err)
	}

	// Check that all variables were substituted
	if !strings.Contains(got, "2024-01-15") {
		t.Errorf("Render() missing date substitution")
	}
	if !strings.Contains(got, "Alice, Bob, Charlie") {
		t.Errorf("Render() missing attendees substitution")
	}
	if !strings.Contains(got, "Discussed project roadmap") {
		t.Errorf("Render() missing summary substitution")
	}
}
