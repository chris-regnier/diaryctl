package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chris-regnier/diaryctl/internal/config"
)

func TestPagerViewFillsScreen(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	m := pagerModel{
		content:  "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
		maxWidth: 0,
		theme:    theme,
	}

	// Simulate window size to make it ready
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pagerModel)

	output := m.View()
	stripped := stripANSI(output)
	lines := strings.Split(stripped, "\n")

	if len(lines) != 24 {
		t.Errorf("expected 24 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 80 {
			t.Errorf("line %d: expected min width 80, got %d", i, len(line))
		}
	}
}

func TestPagerViewFillsScreen_WithMaxWidth(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "dracula"})
	m := pagerModel{
		content:  "Some content to display in the pager",
		maxWidth: 60,
		theme:    theme,
	}

	sized, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = sized.(pagerModel)

	output := m.View()
	stripped := stripANSI(output)
	lines := strings.Split(stripped, "\n")

	if len(lines) != 30 {
		t.Errorf("expected 30 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 100 {
			t.Errorf("line %d: expected min width 100, got %d", i, len(line))
		}
	}
}

func TestPagerViewPreservesContent(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	m := pagerModel{
		content:  "This is the pager content",
		maxWidth: 0,
		theme:    theme,
	}

	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pagerModel)

	output := m.View()
	stripped := stripANSI(output)

	if !strings.Contains(stripped, "pager content") {
		t.Error("expected pager content in output")
	}
	if !strings.Contains(stripped, "scroll") {
		t.Error("expected footer help text in output")
	}
}

func TestPagerPaintScreenDimensions(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "catppuccin-mocha"})
	output := theme.PaintScreen("test content", 80, 24, 80)

	lines := strings.Split(stripANSI(output), "\n")
	if len(lines) != 24 {
		t.Errorf("expected 24 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 80 {
			t.Errorf("line %d: expected min width 80, got %d", i, len(line))
		}
	}
}
