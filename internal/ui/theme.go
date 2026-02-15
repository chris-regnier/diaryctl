package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
	Background    lipgloss.Color
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
		Background:    lipgloss.Color("235"),
		MarkdownStyle: "dark",
	},
	"default-light": {
		Primary:       lipgloss.Color("0"),
		Secondary:     lipgloss.Color("240"),
		Accent:        lipgloss.Color("27"),
		Muted:         lipgloss.Color("245"),
		Danger:        lipgloss.Color("1"),
		Background:    lipgloss.Color("254"),
		MarkdownStyle: "light",
	},
	"dracula": {
		Primary:       lipgloss.Color("#F8F8F2"),
		Secondary:     lipgloss.Color("#6272A4"),
		Accent:        lipgloss.Color("#BD93F9"),
		Muted:         lipgloss.Color("#6272A4"),
		Danger:        lipgloss.Color("#FF5555"),
		Background:    lipgloss.Color("#282A36"),
		MarkdownStyle: "dark",
	},
	"ayu-dark": {
		Primary:       lipgloss.Color("#BFBDB6"),
		Secondary:     lipgloss.Color("#565B66"),
		Accent:        lipgloss.Color("#E6B450"),
		Muted:         lipgloss.Color("#565B66"),
		Danger:        lipgloss.Color("#D95757"),
		Background:    lipgloss.Color("#0D1017"),
		MarkdownStyle: "dark",
	},
	"ayu-light": {
		Primary:       lipgloss.Color("#575F66"),
		Secondary:     lipgloss.Color("#8A9199"),
		Accent:        lipgloss.Color("#F2AE49"),
		Muted:         lipgloss.Color("#8A9199"),
		Danger:        lipgloss.Color("#E65050"),
		Background:    lipgloss.Color("#FAFAFA"),
		MarkdownStyle: "light",
	},
	"catppuccin-mocha": {
		Primary:       lipgloss.Color("#CDD6F4"),
		Secondary:     lipgloss.Color("#585B70"),
		Accent:        lipgloss.Color("#CBA6F7"),
		Muted:         lipgloss.Color("#6C7086"),
		Danger:        lipgloss.Color("#F38BA8"),
		Background:    lipgloss.Color("#1E1E2E"),
		MarkdownStyle: "dark",
	},
	"catppuccin-latte": {
		Primary:       lipgloss.Color("#4C4F69"),
		Secondary:     lipgloss.Color("#9CA0B0"),
		Accent:        lipgloss.Color("#8839EF"),
		Muted:         lipgloss.Color("#9CA0B0"),
		Danger:        lipgloss.Color("#D20F39"),
		Background:    lipgloss.Color("#EFF1F5"),
		MarkdownStyle: "light",
	},
	"gruvbox-dark": {
		Primary:       lipgloss.Color("#EBDBB2"),
		Secondary:     lipgloss.Color("#665C54"),
		Accent:        lipgloss.Color("#FABD2F"),
		Muted:         lipgloss.Color("#928374"),
		Danger:        lipgloss.Color("#FB4934"),
		Background:    lipgloss.Color("#282828"),
		MarkdownStyle: "dark",
	},
	"gruvbox-light": {
		Primary:       lipgloss.Color("#3C3836"),
		Secondary:     lipgloss.Color("#A89984"),
		Accent:        lipgloss.Color("#D79921"),
		Muted:         lipgloss.Color("#928374"),
		Danger:        lipgloss.Color("#CC241D"),
		Background:    lipgloss.Color("#FBF1C7"),
		MarkdownStyle: "light",
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
	if cfg.Background != "" {
		theme.Background = lipgloss.Color(cfg.Background)
	}
	if cfg.MarkdownStyle != "" {
		theme.MarkdownStyle = cfg.MarkdownStyle
	}

	return theme
}

// HelpStyle returns a lipgloss style for help/footer text.
func (t Theme) HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted).Background(t.Background)
}

// HeaderStyle returns a lipgloss style for headers.
func (t Theme) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Background(t.Background)
}

// AccentStyle returns a lipgloss style for accented/focused elements.
func (t Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent).Background(t.Background)
}

// DangerStyle returns a lipgloss style for warnings/delete prompts.
func (t Theme) DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Danger).Background(t.Background)
}

// BorderStyle returns a lipgloss style with a rounded border using secondary color.
func (t Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Secondary).
		BorderBackground(t.Background).
		Background(t.Background).
		Foreground(t.Primary)
}

// bgEscapeCode returns the raw ANSI escape sequence to set the theme's
// background color, for use with terminal control codes like \x1b[K.
func (t Theme) bgEscapeCode() string {
	s := string(t.Background)
	if strings.HasPrefix(s, "#") && len(s) == 7 {
		var r, g, b int
		fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	}
	return "\x1b[48;5;" + s + "m"
}

// PaintScreen fills every line to termWidth (with optional centering) and pads
// vertically to termHeight, using the theme background color. Each line is
// padded with background-colored spaces AND a terminal-level \x1b[K (erase to
// end of line) as a safety net, ensuring the background fills the full terminal
// width even if lipgloss.Width measurement is slightly off.
func (t Theme) PaintScreen(content string, termWidth, termHeight, contentWidth int) string {
	bgPad := lipgloss.NewStyle().Background(t.Background)
	clearEOL := t.bgEscapeCode() + "\x1b[K"

	leftPad := 0
	if contentWidth > 0 && contentWidth < termWidth {
		leftPad = (termWidth - contentWidth) / 2
	}

	leftStr := ""
	if leftPad > 0 {
		leftStr = bgPad.Render(strings.Repeat(" ", leftPad))
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		rightPad := max(termWidth-leftPad-w, 0)

		var b strings.Builder
		if leftPad > 0 {
			b.WriteString(leftStr)
		}
		b.WriteString(line)
		if rightPad > 0 {
			b.WriteString(bgPad.Render(strings.Repeat(" ", rightPad)))
		}
		b.WriteString(clearEOL)
		lines[i] = b.String()
	}

	emptyLine := bgPad.Render(strings.Repeat(" ", termWidth)) + clearEOL
	for len(lines) < termHeight {
		lines = append(lines, emptyLine)
	}

	return strings.Join(lines[:termHeight], "\n")
}

// ClearLineEnds appends a terminal-level erase-to-end-of-line (\x1b[K) to
// every line, ensuring the theme background fills to the right terminal edge.
// Use this for output produced by lipgloss.Place or similar that may not
// extend to the full terminal width.
func (t Theme) ClearLineEnds(content string) string {
	clearEOL := t.bgEscapeCode() + "\x1b[K"
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = line + clearEOL
	}
	return strings.Join(lines, "\n")
}

// ViewPaneStyle returns a lipgloss style for content view panes with themed background.
func (t Theme) ViewPaneStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Background).
		Foreground(t.Primary)
}

// NewList creates a list.Model with delegate and chrome styles derived from the theme.
func (t Theme) NewList(items []list.Item, width, height int) list.Model {
	l := list.New(items, t.ListDelegate(), width, height)
	l.Styles = t.ListStyles()
	return l
}

// ListDelegate returns a list.DefaultDelegate with item styles derived from the theme.
func (t Theme) ListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Background(t.Background).
		Padding(0, 0, 0, 2)
	d.Styles.NormalDesc = d.Styles.NormalTitle.
		Foreground(t.Muted)
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(t.Accent).
		Foreground(t.Accent).
		Background(t.Background).
		Padding(0, 0, 0, 1)
	d.Styles.SelectedDesc = d.Styles.SelectedTitle.
		Foreground(t.Secondary)
	d.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background).
		Padding(0, 0, 0, 2)
	d.Styles.DimmedDesc = d.Styles.DimmedTitle.
		Foreground(t.Muted)
	return d
}

// ListStyles returns list.Styles (chrome around the list) derived from the theme.
func (t Theme) ListStyles() list.Styles {
	s := list.DefaultStyles()
	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary).
		Background(t.Background)
	s.TitleBar = lipgloss.NewStyle().
		Background(t.Background)
	s.FilterPrompt = lipgloss.NewStyle().
		Foreground(t.Accent).
		Background(t.Background)
	s.FilterCursor = lipgloss.NewStyle().
		Foreground(t.Accent).
		Background(t.Background)
	s.PaginationStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)
	s.HelpStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)
	s.ActivePaginationDot = lipgloss.NewStyle().
		Foreground(t.Accent).
		Background(t.Background)
	s.InactivePaginationDot = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)
	s.NoItems = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)
	return s
}
