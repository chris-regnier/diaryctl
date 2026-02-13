# Custom Themes

**Status:** Designed

## Overview

Config-driven theming for the TUI interface and markdown rendering. Users can customize colors, styles, and markdown rendering through the existing `config.toml` or select from built-in presets.

## Current State

The TUI uses hard-coded lipgloss colors and a hard-coded glamour "dark" preset. There is no user-facing theme configuration.

## Proposed Interface

### Configuration

Add to `~/.diaryctl/config.toml`:

```toml
[theme]
preset = "default-dark"    # "default-dark", "default-light", "dracula"

# Override individual colors (hex or ANSI 256)
primary = "#7C3AED"        # Headers, selected items
secondary = "#06B6D4"      # Borders, accents
accent = "#F59E0B"         # Highlights, active states
muted = "241"              # Help text, footers
danger = "#EF4444"         # Delete prompts, warnings

# Markdown rendering
markdown_style = "dark"    # "dark", "light", "notty", or path to glamour JSON
```

### Built-in Presets

| Preset | Description |
|--------|-------------|
| `default-dark` | Current colors, dark glamour (default) |
| `default-light` | Light-friendly palette, light glamour |
| `dracula` | Dracula color scheme |

### CLI Preview

```bash
# Preview current theme
diaryctl config theme

# List available presets
diaryctl config theme --list
```

## Implementation

### Theme Struct

```go
type ThemeConfig struct {
    Preset        string `mapstructure:"preset"`
    Primary       string `mapstructure:"primary"`
    Secondary     string `mapstructure:"secondary"`
    Accent        string `mapstructure:"accent"`
    Muted         string `mapstructure:"muted"`
    Danger        string `mapstructure:"danger"`
    MarkdownStyle string `mapstructure:"markdown_style"`
}
```

### Theme Resolution

```go
func ResolveTheme(cfg ThemeConfig) Theme {
    // 1. Start from preset defaults
    theme := presets[cfg.Preset]

    // 2. Override with any explicit config values
    if cfg.Primary != "" {
        theme.Primary = lipgloss.Color(cfg.Primary)
    }
    // ... etc for each color slot

    // 3. Resolve markdown style
    theme.MarkdownStyle = resolveMarkdownStyle(cfg.MarkdownStyle)

    return theme
}
```

### Integration Points

- `internal/ui/theme.go` — Theme struct, presets, resolution logic
- `internal/ui/picker.go` — Apply theme colors to date list, borders, selection
- `internal/ui/pager.go` — Apply theme to footer, viewport chrome
- `internal/ui/markdown.go` — Pass markdown style to glamour renderer
- `internal/config/config.go` — Add `ThemeConfig` to `Config` struct

### Color Slot Usage

| Slot | Used For |
|------|----------|
| `primary` | Headers, selected item highlight, title text |
| `secondary` | Borders, template picker accents |
| `accent` | Active states, cursor, focused elements |
| `muted` | Help text, footers, disabled items |
| `danger` | Delete confirmation, warning prompts |

## Design Decisions

### Why config-driven over theme files?

- Fits the existing Viper/TOML configuration pattern
- Single file to manage, no directory discovery needed
- Presets provide good defaults without file management
- Can add file-based themes later if demand grows

### Why named color slots over per-component styling?

- Keeps the theme surface small and approachable
- Ensures visual consistency across all TUI components
- Easy to create presets — just 5 colors + markdown style
- Per-component overrides can be added later without breaking changes

### Why include markdown rendering?

- Glamour's "dark" preset doesn't suit all terminals
- Users on light terminals get poor contrast today
- Custom glamour JSON allows power users to fully control rendering

## Related Features

- [TUI Search and Filter](tui-search.md) — Search UI uses theme colors
- [TUI Configuration](tui-configuration.md) — Broader runtime customization (proposed)

---

*Feature designed February 2026.*
