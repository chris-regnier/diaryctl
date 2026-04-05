# Configuration

diaryctl works with zero configuration. All settings have sensible defaults. Customize behavior through a config file or environment variables.

## Config File

diaryctl looks for a TOML config file in this order:

1. Path passed via `--config` flag
2. `$XDG_CONFIG_HOME/diaryctl/config.toml` (if `XDG_CONFIG_HOME` is set)
3. `~/.diaryctl/config.toml`

### Full Example

```toml
storage = "markdown"           # "markdown" or "sqlite"
data_dir = "~/.diaryctl"       # data directory path
editor = "hx"                  # editor command (see Editor below)
default_template = ""          # auto-apply this template to new entries
max_width = 100                # max output width in columns
context_providers = ["git", "datetime"]   # content injected into editor
context_resolvers = ["git"]               # auto-detected contexts

[shell]
cache_ttl = "5m"               # prompt cache duration
today_icon = "✓"               # icon when today has an entry
no_today_icon = "✗"            # icon when today has no entry
streak_icon = "🔥"              # streak counter icon
show_context = true            # show active context in prompt
show_backend = false           # show storage backend in prompt

[theme]
preset = "default-dark"        # theme preset name
# Override individual colors (hex or ANSI 256 color codes):
# primary = "#F8F8F2"
# secondary = "#6272A4"
# accent = "#BD93F9"
# muted = "#6272A4"
# danger = "#FF5555"
# background = "#282A36"
# markdown_style = "dark"      # "dark" or "light"
```

## Environment Variables

Every config key can be set via environment variable with the `DIARYCTL_` prefix:

| Variable | Config Key | Example |
|----------|-----------|---------|
| `DIARYCTL_STORAGE` | `storage` | `sqlite` |
| `DIARYCTL_DATA_DIR` | `data_dir` | `/data/diary` |
| `DIARYCTL_EDITOR` | `editor` | `vim` |
| `DIARYCTL_DEFAULT_TEMPLATE` | `default_template` | `daily` |
| `DIARYCTL_MAX_WIDTH` | `max_width` | `120` |

Environment variables take precedence over the config file.

## Editor

diaryctl opens an external editor for `create`, `edit`, `today --edit`, and template editing. The editor is resolved in this order:

1. `editor` in config file
2. `$EDITOR` environment variable
3. `$VISUAL` environment variable
4. Fallback: `hx` (Helix)

The editor is launched as a subprocess with inherited stdin/stdout/stderr, so it works identically to your normal terminal editor workflow.

## Data Directory

All data is stored under `data_dir` (default `~/.diaryctl/`):

```
~/.diaryctl/
├── config.toml              # configuration (optional)
├── entries/                  # markdown entry files
│   └── YYYY/MM/DD/{id}.md
├── templates/                # template files
├── contexts/                 # context metadata
├── active-contexts.json      # manual context state
├── .prompt-cache             # shell prompt cache
└── diaryctl.db               # SQLite database (if using sqlite backend)
```

## Defaults Reference

| Key | Default |
|-----|---------|
| `storage` | `markdown` |
| `data_dir` | `~/.diaryctl/` |
| `editor` | `""` (falls back to `$EDITOR`, `$VISUAL`, then `hx`) |
| `default_template` | `""` (none) |
| `max_width` | `100` |
| `context_providers` | `[]` |
| `context_resolvers` | `[]` |
| `shell.cache_ttl` | `5m` |
| `shell.today_icon` | `✓` |
| `shell.no_today_icon` | `✗` |
| `shell.streak_icon` | `🔥` |
| `shell.show_context` | `true` |
| `shell.show_backend` | `false` |
| `theme.preset` | `default-dark` |
