# Themes

diaryctl's TUI and markdown rendering support multiple color themes. Choose a built-in preset or customize individual colors.

## Built-in Presets

| Preset | Style | Description |
|--------|-------|-------------|
| `default-dark` | Dark | Default theme, neutral colors |
| `default-light` | Light | Light background variant |
| `dracula` | Dark | [Dracula](https://draculatheme.com/) color scheme |
| `ayu-dark` | Dark | [Ayu](https://github.com/ayu-theme) dark variant |
| `ayu-light` | Light | Ayu light variant |
| `catppuccin-mocha` | Dark | [Catppuccin](https://catppuccin.com/) Mocha flavor |
| `catppuccin-latte` | Light | Catppuccin Latte flavor |
| `gruvbox-dark` | Dark | [Gruvbox](https://github.com/morhetz/gruvbox) dark variant |
| `gruvbox-light` | Light | Gruvbox light variant |

## Configuration

Set a preset in your config file:

```toml
[theme]
preset = "catppuccin-mocha"
```

## Custom Colors

Override individual colors on top of any preset. Values can be hex colors (`#RRGGBB`) or ANSI 256 color codes.

```toml
[theme]
preset = "default-dark"
primary = "#F8F8F2"
secondary = "#6272A4"
accent = "#BD93F9"
muted = "#6272A4"
danger = "#FF5555"
background = "#282A36"
markdown_style = "dark"     # "dark" or "light"
```

### Color Roles

| Color | Used For |
|-------|----------|
| `primary` | Main text, headers |
| `secondary` | Borders, list descriptions |
| `accent` | Focused/selected items, active elements |
| `muted` | Help text, pagination, dimmed items |
| `danger` | Delete prompts, warnings |
| `background` | Screen background fill |

### Markdown Style

The `markdown_style` setting controls how Glamour renders markdown content. Set it to `"dark"` or `"light"` to match your terminal background.

## Preview

The fastest way to preview a theme is to launch the TUI:

```bash
diaryctl
```

Or view a single entry:

```bash
diaryctl show <id>
```
