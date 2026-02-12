# Starship Preset

**Status:** Proposed

## Overview

An official Starship preset configuration for diaryctl, providing a polished, ready-to-use prompt integration.

## Quick Start

Add to `~/.config/starship.toml`:

```toml
# Option A: Custom command (standalone, no init required)
[custom.diaryctl]
command = "diaryctl status"
when = "command -v diaryctl"
shell = ["bash", "--nologin"]
format = "[$output]($style) "
style = "bold yellow"
```

```toml
# Option B: Environment variables (requires init eval, faster)
# Add to ~/.bashrc or ~/.zshrc:
# eval "$(diaryctl init bash)"

[env_var.DIARYCTL_TODAY]
format = "[$env_value](bold green)"

[env_var.DIARYCTL_STREAK]
format = "[$env_value](bold yellow)"

[env_var.DIARYCTL_STREAK_ICON]
format = "[$env_value](bold yellow)"

[env_var.DIARYCTL_TEMPLATE]
format = "[($env_value)](dimmed)"
```

## Official Preset

```toml
# ~/.config/starship.toml
# diaryctl official preset

[custom.diaryctl]
command = "diaryctl status --format '{{.TodayIcon}}{{.Streak}}{{.StreakIcon}}'"
when = "command -v diaryctl"
shell = ["bash", "--nologin"]
format = "[◆ $output]($style) "
style = "bold yellow"
description = "diaryctl status in prompt"
```

## Advanced Configuration

### Conditional Styling

```toml
[custom.diaryctl]
command = """
if diaryctl status --format '{{.Today}}' | grep -q '✓'; then
  echo "journal"
else
  echo "no-entry"
fi
"""
when = "command -v diaryctl"
shell = ["bash", "--nologin"]
format = "[$symbol]($style) "
symbol = "◆"
style = "bold green"
style_no_entry = "bold red"
```

### With Context

```toml
[custom.diaryctl]
command = "diaryctl status"
when = "command -v diaryctl"
shell = ["bash", "--nologin"]
format = "[◆ $output](bold yellow) "
description = "diaryctl status"
```

## Installation Script

Future `diaryctl init` could support `--starship` flag:

```bash
# Install Starship preset
diaryctl init --starship >> ~/.config/starship.toml

# Preview without installing
diaryctl init --starship --preview
```

## Files

- Would add to `internal/shell/starship.go` — Preset generation

## Related Features

- [Shell Integration](shell-integration.md) — Core shell integration

---

*This is a proposed feature. No design document exists yet.*
