package ui

import (
	"testing"

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
}
