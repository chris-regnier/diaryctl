package ui

import (
	"strings"
	"testing"
)

func TestRenderMarkdownWithStyleDark(t *testing.T) {
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
			got := RenderMarkdownWithStyle(tt.input, tt.width, "dark")
			// Strip ANSI codes for testing content, since Glamour adds color codes
			gotStripped := stripANSI(got)

			for _, want := range tt.wantContains {
				if !strings.Contains(gotStripped, want) {
					t.Errorf("RenderMarkdown() output should contain %q, got:\n%s", want, gotStripped)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(gotStripped, notWant) {
					t.Errorf("RenderMarkdown() output should not contain %q, got:\n%s", notWant, gotStripped)
				}
			}
		})
	}
}

func TestRenderMarkdownFallback(t *testing.T) {
	// Test that if rendering fails, we get the original content back
	input := "Some content"
	width := 80

	got := RenderMarkdownWithStyle(input, width, "dark")

	// Should always return something (either rendered or original)
	if got == "" && input != "" {
		t.Error("RenderMarkdownWithStyle() should not return empty string for non-empty input")
	}
}

func TestRenderMarkdownWidth(t *testing.T) {
	// Test that width parameter is respected
	input := "# Title\n\nSome text here that is reasonably long and should demonstrate word wrapping behavior."

	// Render with different widths
	narrow := RenderMarkdownWithStyle(input, 40, "dark")
	wide := RenderMarkdownWithStyle(input, 100, "dark")

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

func TestRenderMarkdownWithStyle(t *testing.T) {
	// Reset global state
	markdownRenderer = nil
	cachedWidth = 0
	cachedStyle = ""

	content := "# Hello"

	// Light style should produce output
	result := RenderMarkdownWithStyle(content, 80, "light")
	if result == "" {
		t.Error("expected non-empty rendered output for light style")
	}

	// Notty style should produce output
	markdownRenderer = nil
	cachedWidth = 0
	cachedStyle = ""
	result = RenderMarkdownWithStyle(content, 80, "notty")
	if result == "" {
		t.Error("expected non-empty rendered output for notty style")
	}
}

func TestRenderMarkdownStyleChange(t *testing.T) {
	markdownRenderer = nil
	cachedWidth = 0
	cachedStyle = ""

	content := "# Test"

	dark := RenderMarkdownWithStyle(content, 80, "dark")

	// Force re-creation by changing style
	notty := RenderMarkdownWithStyle(content, 80, "notty")

	// Different styles should produce different output
	if dark == notty {
		t.Error("expected different output for different styles")
	}
}

