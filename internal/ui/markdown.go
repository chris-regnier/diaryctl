package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownRenderer is a cached glamour renderer instance
var markdownRenderer *glamour.TermRenderer

// initMarkdownRenderer initializes the glamour renderer with the given width
func initMarkdownRenderer(width int) error {
	if width < 1 {
		width = 80 // sensible default
	}

	// Use "auto" style to adapt to terminal background (dark/light)
	// Alternative styles: "dark", "light", "pink", "notty"
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	markdownRenderer = renderer
	return nil
}

// updateMarkdownWidth updates the word wrap width for the renderer
func updateMarkdownWidth(width int) error {
	if width < 1 {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	markdownRenderer = renderer
	return nil
}

// RenderMarkdown renders markdown content to a rich text string suitable for terminal display.
// Returns the original content if rendering fails.
func RenderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}

	// Initialize or update renderer if needed
	if markdownRenderer == nil {
		if err := initMarkdownRenderer(width); err != nil {
			return content // fallback to raw content on error
		}
	} else {
		// Update width if it changed significantly (avoid re-creating for small changes)
		if err := updateMarkdownWidth(width); err != nil {
			return content
		}
	}

	rendered, err := markdownRenderer.Render(content)
	if err != nil {
		// If rendering fails, return original content
		return content
	}

	// Glamour adds trailing newlines, trim them for cleaner display
	return strings.TrimRight(rendered, "\n")
}
