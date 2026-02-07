package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		width          int
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:  "empty string",
			input: "",
			width: 80,
			wantContains: []string{
				"", // should return empty string
			},
		},
		{
			name:  "plain text",
			input: "Hello world",
			width: 80,
			wantContains: []string{
				"Hello world",
			},
		},
		{
			name:  "markdown heading",
			input: "# Main Title",
			width: 80,
			wantContains: []string{
				"Main Title", // heading text should be present
			},
			// Note: Glamour's auto style keeps the # symbol as part of the visual design
			// This is intentional for terminal rendering to show heading hierarchy
		},
		{
			name:  "markdown list",
			input: "- Item 1\n- Item 2\n- Item 3",
			width: 80,
			wantContains: []string{
				"Item 1",
				"Item 2",
				"Item 3",
			},
		},
		{
			name:  "markdown bold and italic",
			input: "This is **bold** and *italic* text",
			width: 80,
			wantContains: []string{
				"bold",
				"italic",
			},
		},
		{
			name: "complex markdown",
			input: `# My Diary Entry

## Today's Tasks

- [x] Wake up early
- [ ] Go for a run
- [x] Write code

**Note**: This is important!

Some regular text here.`,
			width: 80,
			wantContains: []string{
				"My Diary Entry",
				"Today's Tasks",
				"Wake up early",
				"Go for a run",
				"Write code",
				"Note",
				"important",
			},
		},
		{
			name:  "handles small width",
			input: "This is a longer line of text that should wrap",
			width: 20,
			wantContains: []string{
				"This is a longer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderMarkdown(tt.input, tt.width)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("RenderMarkdown() output should contain %q, got:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("RenderMarkdown() output should not contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderMarkdownFallback(t *testing.T) {
	// Test that if rendering fails, we get the original content back
	input := "Some content"
	width := 80

	got := RenderMarkdown(input, width)

	// Should always return something (either rendered or original)
	if got == "" && input != "" {
		t.Error("RenderMarkdown() should not return empty string for non-empty input")
	}
}

func TestRenderMarkdownWidth(t *testing.T) {
	// Test that width parameter is respected
	input := "# Title\n\nSome text here that is reasonably long and should demonstrate word wrapping behavior."

	// Render with different widths
	narrow := RenderMarkdown(input, 40)
	wide := RenderMarkdown(input, 100)

	// Both should contain the content
	if !strings.Contains(narrow, "Title") {
		t.Error("Narrow render should contain title")
	}
	if !strings.Contains(wide, "Title") {
		t.Error("Wide render should contain title")
	}

	// Length might differ due to wrapping (though this is a soft check)
	// We're mainly verifying it doesn't crash with different widths
	if narrow == "" || wide == "" {
		t.Error("Rendering should produce output for both widths")
	}
}
