package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/chris-regnier/diaryctl/internal/config"
)

func TestResolveThemeDefaultDark(t *testing.T) {
	cfg := config.ThemeConfig{Preset: "default-dark"}
	theme := ResolveTheme(cfg)

	if string(theme.Primary) == "" {
		t.Error("expected primary color to be set")
	}
	if theme.MarkdownStyle != "dark" {
		t.Errorf("expected markdown_style 'dark', got %q", theme.MarkdownStyle)
	}
}

func TestResolveThemeDefaultLight(t *testing.T) {
	cfg := config.ThemeConfig{Preset: "default-light"}
	theme := ResolveTheme(cfg)

	if theme.MarkdownStyle != "light" {
		t.Errorf("expected markdown_style 'light', got %q", theme.MarkdownStyle)
	}
}

func TestResolveThemeOverrides(t *testing.T) {
	cfg := config.ThemeConfig{
		Preset:  "default-dark",
		Primary: "#FF0000",
	}
	theme := ResolveTheme(cfg)

	if string(theme.Primary) != "#FF0000" {
		t.Errorf("expected primary '#FF0000', got %q", string(theme.Primary))
	}
}

func TestResolveThemeMarkdownStyleOverride(t *testing.T) {
	cfg := config.ThemeConfig{
		Preset:        "default-dark",
		MarkdownStyle: "notty",
	}
	theme := ResolveTheme(cfg)

	if theme.MarkdownStyle != "notty" {
		t.Errorf("expected markdown_style 'notty', got %q", theme.MarkdownStyle)
	}
}

func TestResolveThemeUnknownPresetFallsBack(t *testing.T) {
	cfg := config.ThemeConfig{Preset: "nonexistent"}
	theme := ResolveTheme(cfg)

	if theme.MarkdownStyle != "dark" {
		t.Errorf("expected fallback to dark, got %q", theme.MarkdownStyle)
	}
}

func TestTUIConfigIncludesTheme(t *testing.T) {
	cfg := TUIConfig{
		Editor:          "vim",
		DefaultTemplate: "daily",
		MaxWidth:        100,
		Theme:           ResolveTheme(config.ThemeConfig{Preset: "dracula"}),
	}
	if cfg.Theme.MarkdownStyle != "dark" {
		t.Errorf("expected dracula markdown_style 'dark', got %q", cfg.Theme.MarkdownStyle)
	}
}

func TestResolveThemeAllPresets(t *testing.T) {
	cases := []struct {
		preset        string
		markdownStyle string
	}{
		{"default-dark", "dark"},
		{"default-light", "light"},
		{"dracula", "dark"},
		{"ayu-dark", "dark"},
		{"ayu-light", "light"},
		{"catppuccin-mocha", "dark"},
		{"catppuccin-latte", "light"},
		{"gruvbox-dark", "dark"},
		{"gruvbox-light", "light"},
	}

	for _, tc := range cases {
		t.Run(tc.preset, func(t *testing.T) {
			theme := ResolveTheme(config.ThemeConfig{Preset: tc.preset})

			if string(theme.Primary) == "" {
				t.Error("expected primary color to be set")
			}
			if string(theme.Accent) == "" {
				t.Error("expected accent color to be set")
			}
			if string(theme.Danger) == "" {
				t.Error("expected danger color to be set")
			}
			if string(theme.Background) == "" {
				t.Error("expected background color to be set")
			}
			if theme.MarkdownStyle != tc.markdownStyle {
				t.Errorf("expected markdown_style %q, got %q", tc.markdownStyle, theme.MarkdownStyle)
			}
		})
	}
}

func TestThemeStyleMethods(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})

	// Verify style methods don't panic and return non-zero styles
	_ = theme.HelpStyle()
	_ = theme.HeaderStyle()
	_ = theme.AccentStyle()
	_ = theme.DangerStyle()
	_ = theme.BorderStyle()
	_ = theme.ViewPaneStyle()
	_ = theme.PaintScreen("test", 80, 24, 80)
	_ = theme.ListDelegate()
	_ = theme.ListStyles()
	_ = theme.NewList(nil, 0, 0)
}

func TestPaintScreenDimensions(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	output := theme.PaintScreen("hello", 40, 10, 40)

	lines := strings.Split(stripANSI(output), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 40 {
			t.Errorf("line %d: expected min width 40, got %d", i, len(line))
		}
	}
}

func TestPaintScreenCentering(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	// termWidth=100, contentWidth=60 => leftPad=20
	output := theme.PaintScreen("hello", 100, 5, 60)

	lines := strings.Split(stripANSI(output), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if len(line) < 100 {
			t.Errorf("line %d: expected min width 100, got %d", i, len(line))
		}
	}
	// First line should have left padding (spaces before "hello")
	first := stripANSI(lines[0])
	if !strings.HasPrefix(first, "                    ") { // 20 spaces
		t.Errorf("expected 20 chars of left padding, got: %q", first[:20])
	}
}

func TestPaintScreenFillsHeight(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	// Content is 2 lines, screen is 10 lines
	output := theme.PaintScreen("line1\nline2", 40, 10, 40)

	lines := strings.Split(stripANSI(output), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}
	// All 10 lines should be at least 40 chars (filled with background)
	for i, line := range lines {
		if len(line) < 40 {
			t.Errorf("line %d: expected min width 40, got %d", i, len(line))
		}
	}
}

func TestAllStylesIncludeBackground(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})

	styles := map[string]lipgloss.Style{
		"HelpStyle":     theme.HelpStyle(),
		"HeaderStyle":   theme.HeaderStyle(),
		"AccentStyle":   theme.AccentStyle(),
		"DangerStyle":   theme.DangerStyle(),
		"BorderStyle":   theme.BorderStyle(),
		"ViewPaneStyle": theme.ViewPaneStyle(),
	}

	for name, style := range styles {
		if style.GetBackground() != theme.Background {
			t.Errorf("%s: expected background %v, got %v", name, theme.Background, style.GetBackground())
		}
	}
}

func TestBorderStyleIncludesBorderBackground(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	style := theme.BorderStyle()

	if style.GetBorderBottomBackground() != theme.Background {
		t.Errorf("expected border background %v, got %v", theme.Background, style.GetBorderBottomBackground())
	}
}

func TestListStylesIncludeBackground(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	s := theme.ListStyles()

	styles := map[string]lipgloss.Style{
		"FilterPrompt":        s.FilterPrompt,
		"FilterCursor":        s.FilterCursor,
		"PaginationStyle":     s.PaginationStyle,
		"HelpStyle":           s.HelpStyle,
		"ActivePaginationDot": s.ActivePaginationDot,
		"NoItems":             s.NoItems,
	}

	for name, style := range styles {
		if style.GetBackground() != theme.Background {
			t.Errorf("ListStyles.%s: expected background %v, got %v", name, theme.Background, style.GetBackground())
		}
	}
}

func TestBgEscapeCode(t *testing.T) {
	// 256-color theme (default-dark uses "235")
	theme256 := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	code256 := theme256.bgEscapeCode()
	if code256 != "\x1b[48;5;235m" {
		t.Errorf("expected 256-color escape, got %q", code256)
	}

	// True-color theme (dracula uses "#282A36")
	themeHex := ResolveTheme(config.ThemeConfig{Preset: "dracula"})
	codeHex := themeHex.bgEscapeCode()
	if codeHex != "\x1b[48;2;40;42;54m" {
		t.Errorf("expected true-color escape, got %q", codeHex)
	}
}

func TestPaintScreenIncludesClearEOL(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	output := theme.PaintScreen("hello", 40, 3, 40)

	// Every line should end with \x1b[K (erase to end of line)
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if !strings.HasSuffix(line, "\x1b[K") {
			t.Errorf("line %d: expected to end with \\x1b[K erase sequence", i)
		}
	}
}

func TestResolveThemeBackgroundOverride(t *testing.T) {
	cfg := config.ThemeConfig{
		Preset:     "default-dark",
		Background: "#112233",
	}
	theme := ResolveTheme(cfg)

	if string(theme.Background) != "#112233" {
		t.Errorf("expected background '#112233', got %q", string(theme.Background))
	}
}
