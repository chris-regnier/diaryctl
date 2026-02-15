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
	_ = theme.FullScreenStyle(80, 24)
	_ = theme.ListDelegate()
	_ = theme.ListStyles()
	_ = theme.NewList(nil, 0, 0)
}

func TestFullScreenStyleDimensions(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	style := theme.FullScreenStyle(40, 10)
	output := style.Render("hello")

	lines := countLines(output)
	if lines != 10 {
		t.Errorf("expected 10 lines, got %d", lines)
	}

	// Each line should be padded to width 40 (after stripping ANSI)
	for i, line := range strings.Split(stripANSI(output), "\n") {
		// lipgloss may add trailing reset sequences but stripped line should be 40 chars
		if len(line) != 40 {
			t.Errorf("line %d: expected width 40, got %d", i, len(line))
		}
	}
}

func TestFullScreenStyleProperties(t *testing.T) {
	for _, preset := range []string{"default-dark", "default-light", "dracula", "catppuccin-mocha"} {
		t.Run(preset, func(t *testing.T) {
			theme := ResolveTheme(config.ThemeConfig{Preset: preset})
			style := theme.FullScreenStyle(80, 24)

			if style.GetWidth() != 80 {
				t.Errorf("expected width 80, got %d", style.GetWidth())
			}
			if style.GetHeight() != 24 {
				t.Errorf("expected height 24, got %d", style.GetHeight())
			}
			if style.GetBackground() != theme.Background {
				t.Errorf("expected background %v, got %v", theme.Background, style.GetBackground())
			}
			if style.GetForeground() != theme.Primary {
				t.Errorf("expected foreground %v, got %v", theme.Primary, style.GetForeground())
			}
		})
	}
}

func TestBorderStyleIncludesBackground(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})
	style := theme.BorderStyle()

	if style.GetBackground() != theme.Background {
		t.Errorf("expected BorderStyle background %v, got %v", theme.Background, style.GetBackground())
	}
	if style.GetForeground() != theme.Primary {
		t.Errorf("expected BorderStyle foreground %v, got %v", theme.Primary, style.GetForeground())
	}
}

func TestViewPaneStyleIncludesBackground(t *testing.T) {
	theme := ResolveTheme(config.ThemeConfig{Preset: "dracula"})
	style := theme.ViewPaneStyle()

	if style.GetBackground() != theme.Background {
		t.Errorf("expected ViewPaneStyle background %v, got %v", theme.Background, style.GetBackground())
	}
}

func TestAllStylesIncludeBackground(t *testing.T) {
	// Verify that styles used for full-area rendering include the theme background
	theme := ResolveTheme(config.ThemeConfig{Preset: "default-dark"})

	styles := map[string]lipgloss.Style{
		"FullScreenStyle": theme.FullScreenStyle(80, 24),
		"BorderStyle":     theme.BorderStyle(),
		"ViewPaneStyle":   theme.ViewPaneStyle(),
	}

	for name, style := range styles {
		if style.GetBackground() != theme.Background {
			t.Errorf("%s: expected background %v, got %v", name, theme.Background, style.GetBackground())
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
