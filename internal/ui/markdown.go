package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownRenderer is a cached glamour renderer instance
var markdownRenderer *glamour.TermRenderer

// cachedWidth stores the width used for the current renderer
var cachedWidth int

// cachedStyle stores the style used for the current renderer
var cachedStyle string

// initMarkdownRenderer initializes the glamour renderer with the given width and style
func initMarkdownRenderer(width int, style string) error {
	if width < 1 {
		width = 80
	}
	if style == "" {
		style = "dark"
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	markdownRenderer = renderer
	cachedWidth = width
	cachedStyle = style
	return nil
}

// updateMarkdownRenderer recreates the renderer if width or style changed
func updateMarkdownRenderer(width int, style string) error {
	if width < 1 {
		width = 80
	}
	if style == "" {
		style = "dark"
	}

	if width == cachedWidth && style == cachedStyle {
		return nil
	}

	return initMarkdownRenderer(width, style)
}

// RenderMarkdownWithStyle renders markdown content using the specified glamour style.
func RenderMarkdownWithStyle(content string, width int, style string) string {
	if content == "" {
		return ""
	}

	if markdownRenderer == nil || style != cachedStyle {
		if err := initMarkdownRenderer(width, style); err != nil {
			return content
		}
	} else {
		if err := updateMarkdownRenderer(width, style); err != nil {
			return content
		}
	}

	rendered, err := markdownRenderer.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimRight(rendered, "\n")
}

// RenderMarkdown renders markdown content to a rich text string suitable for terminal display.
// Uses "dark" style by default. Returns the original content if rendering fails.
func RenderMarkdown(content string, width int) string {
	return RenderMarkdownWithStyle(content, width, "dark")
}
