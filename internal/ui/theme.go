package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/chris-regnier/diaryctl/internal/config"
)

// Theme holds resolved lipgloss colors for TUI rendering.
type Theme struct {
	Primary       lipgloss.Color
	Secondary     lipgloss.Color
	Accent        lipgloss.Color
	Muted         lipgloss.Color
	Danger        lipgloss.Color
	MarkdownStyle string
}

// Built-in presets.
var presets = map[string]Theme{
	"default-dark": {
		Primary:       lipgloss.Color("15"),
		Secondary:     lipgloss.Color("243"),
		Accent:        lipgloss.Color("33"),
		Muted:         lipgloss.Color("241"),
		Danger:        lipgloss.Color("9"),
		MarkdownStyle: "dark",
	},
	"default-light": {
		Primary:       lipgloss.Color("0"),
		Secondary:     lipgloss.Color("240"),
		Accent:        lipgloss.Color("27"),
		Muted:         lipgloss.Color("245"),
		Danger:        lipgloss.Color("1"),
		MarkdownStyle: "light",
	},
	"dracula": {
		Primary:       lipgloss.Color("#F8F8F2"),
		Secondary:     lipgloss.Color("#6272A4"),
		Accent:        lipgloss.Color("#BD93F9"),
		Muted:         lipgloss.Color("#6272A4"),
		Danger:        lipgloss.Color("#FF5555"),
		MarkdownStyle: "dark",
	},
}

// ResolveTheme builds a Theme from config, starting with a preset
// and applying any explicit overrides.
func ResolveTheme(cfg config.ThemeConfig) Theme {
	preset := cfg.Preset
	if preset == "" {
		preset = "default-dark"
	}

	theme, ok := presets[preset]
	if !ok {
		theme = presets["default-dark"]
	}

	if cfg.Primary != "" {
		theme.Primary = lipgloss.Color(cfg.Primary)
	}
	if cfg.Secondary != "" {
		theme.Secondary = lipgloss.Color(cfg.Secondary)
	}
	if cfg.Accent != "" {
		theme.Accent = lipgloss.Color(cfg.Accent)
	}
	if cfg.Muted != "" {
		theme.Muted = lipgloss.Color(cfg.Muted)
	}
	if cfg.Danger != "" {
		theme.Danger = lipgloss.Color(cfg.Danger)
	}
	if cfg.MarkdownStyle != "" {
		theme.MarkdownStyle = cfg.MarkdownStyle
	}

	return theme
}

// HelpStyle returns a lipgloss style for help/footer text.
func (t Theme) HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

// HeaderStyle returns a lipgloss style for headers.
func (t Theme) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
}

// AccentStyle returns a lipgloss style for accented/focused elements.
func (t Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent)
}

// DangerStyle returns a lipgloss style for warnings/delete prompts.
func (t Theme) DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Danger)
}

// BorderStyle returns a lipgloss style with a rounded border using secondary color.
func (t Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Secondary)
}
