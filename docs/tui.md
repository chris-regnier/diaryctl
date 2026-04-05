# TUI Guide

diaryctl includes an interactive terminal user interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Launch it by running `diaryctl` with no arguments in an interactive terminal.

## Screens

The TUI has five screens you navigate between:

### 1. Today View

The default screen. Shows today's entry content in a scrollable viewport.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll content |
| `n` | Create a new entry |
| `e` | Edit today's entry |
| `d` | Browse by date (go to date list) |
| `q` | Quit |

### 2. Date List

A paginated list of days that have entries, newest first.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate list |
| `Enter` | View entries for selected day |
| `n` | Create a new entry |
| `t` | Jump to today |
| `q` | Quit |

### 3. Day Detail

Shows all entries for a selected day.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate entries |
| `Enter` | View full entry |
| `n` | Create a new entry |
| `Esc` | Back to date list |
| `q` | Quit |

### 4. Entry Detail

Full entry content rendered as markdown in a scrollable viewport.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll content |
| `e` | Edit this entry |
| `d` | Delete this entry |
| `c` | Manage contexts |
| `Esc` | Back |
| `q` | Quit |

### 5. Context Panel

Add or remove contexts from an entry.

**Key bindings:**

| Key | Action |
|-----|--------|
| `j` / `k` or arrows | Navigate contexts |
| `Enter` | Toggle context |
| `Esc` | Back to entry |

## Features

- **Markdown rendering**: Entry content is rendered with syntax highlighting and formatting via [Glamour](https://github.com/charmbracelet/glamour)
- **Theming**: The TUI respects your configured theme (see [Themes](themes.md))
- **Editor integration**: Pressing `e` or `n` opens your configured editor, then returns to the TUI
- **Responsive layout**: Adapts to terminal width and height

## Non-Interactive Fallback

When stdout is not a terminal (e.g. piped to another command), diaryctl skips the TUI and outputs a text summary instead:

```bash
diaryctl | head -20           # non-interactive output
diaryctl daily --no-interactive  # force non-interactive mode
```
