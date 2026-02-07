package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownRenderer is a cached glamour renderer instance
var markdownRenderer *glamour.TermRenderer

// cachedWidth stores the width used for the current renderer
var cachedWidth int

// initMarkdownRenderer initializes the glamour renderer with the given width
func initMarkdownRenderer(width int) error {
	if width < 1 {
		width = 80 // sensible default
	}

	// Use "dark" style for rich terminal rendering with colors and styling
	// Alternative styles: "auto", "light", "pink", "notty"
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	markdownRenderer = renderer
	cachedWidth = width
	return nil
}

// updateMarkdownWidth updates the word wrap width for the renderer
func updateMarkdownWidth(width int) error {
	if width < 1 {
		width = 80
	}

	// Only recreate renderer if width actually changed
	if width == cachedWidth {
		return nil
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	markdownRenderer = renderer
	cachedWidth = width
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
