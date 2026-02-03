# Shell Integration Design

## Summary

Add shell integration to diaryctl via `eval "$(diaryctl init <shell>)"` pattern, exposing diary status (today indicator, streak, current template) in the terminal prompt. Supports bash and zsh, with starship compatibility via env vars and a standalone `diaryctl status` command.

## New Commands

### `diaryctl init <shell>`

Outputs shell initialization script for eval. Supported shells: `bash`, `zsh`.

```bash
# In ~/.bashrc or ~/.zshrc
eval "$(diaryctl init bash)"
eval "$(diaryctl init zsh)"
```

The output includes:
- Cobra-generated shell completions
- A `__diaryctl_prompt_hook` function registered via `PROMPT_COMMAND` (bash) or `precmd_functions` (zsh)
- The hook reads cached status and sets env vars
- A `diaryctl_prompt_info` shell function for embedding in custom PS1

### `diaryctl status`

Standalone command that outputs formatted diary status.

```
$ diaryctl status
✓ 7🔥 morning

$ diaryctl status --format "{{.TodayIcon}} {{.Streak}}{{.StreakIcon}} {{.Template}}"
✓ 7🔥 morning
```

Flags:
- `--format` — Go template string for custom output
- `--env` — Output env var assignments instead of formatted text (for eval in shell hooks)
- `--refresh` — Force cache refresh, bypassing TTL

`--env` output:
```bash
export DIARYCTL_TODAY="✓"
export DIARYCTL_STREAK="7"
export DIARYCTL_STREAK_ICON="🔥"
export DIARYCTL_TEMPLATE="morning"
```

## Cache Design

### Cache file

Location: `~/.diaryctl/.prompt-cache` (JSON)

```json
{
  "today": true,
  "streak": 7,
  "today_date": "2026-02-02",
  "default_template": "morning",
  "storage_backend": "markdown",
  "updated_at": "2026-02-02T14:30:00Z"
}
```

### Cache logic

1. On prompt hook or `diaryctl status`: read cache file
2. Check if `updated_at` is within TTL (default 5m) AND `today_date` matches current date
3. If fresh: use cached values
4. If stale or date changed: query storage, compute values, write cache
5. Date change (midnight rollover) always triggers refresh

### Cache invalidation

- Mutating commands (`create`, `jot`, `edit`, `update`, `delete`) invalidate the cache via a Cobra `PostRun` hook
- This is a direct function call within the same process, no extra subprocess
- `diaryctl status --refresh` exposes manual refresh

### Streak computation

- Walk backwards from today through storage using `ListDays`
- Count consecutive days with at least one entry
- Stop at first gap
- Cached for 5 minutes so this runs rarely

## Configuration

New `[shell]` section in `config.toml`:

```toml
[shell]
cache_ttl = "5m"         # duration string, default 5m
today_icon = "✓"         # shown when today entry exists
no_today_icon = "✗"      # shown when no today entry
streak_icon = "🔥"       # shown after streak count
show_context = true      # show template name in prompt
show_backend = false     # show storage backend (hidden by default)
```

## Starship Integration

### Option A: Custom command (standalone, no init required)

```toml
[custom.diaryctl]
command = "diaryctl status"
when = "command -v diaryctl"
shell = ["bash", "--nologin"]
format = "[$output]($style) "
style = "bold yellow"
```

### Option B: Env vars (requires init eval, faster)

```toml
[env_var.DIARYCTL_TODAY]
format = "[$env_value](bold green)"

[env_var.DIARYCTL_STREAK]
format = "[$env_value](bold yellow)"
```

## Shell Init Scripts

### Bash (`diaryctl init bash`)

```bash
# diaryctl shell integration
__diaryctl_prompt_hook() {
  eval "$(command diaryctl status --env 2>/dev/null)"
}

diaryctl_prompt_info() {
  command diaryctl status 2>/dev/null
}

if [[ -z "$PROMPT_COMMAND" ]]; then
  PROMPT_COMMAND="__diaryctl_prompt_hook"
else
  PROMPT_COMMAND="__diaryctl_prompt_hook;${PROMPT_COMMAND}"
fi

eval "$(command diaryctl completion bash 2>/dev/null)"
```

### Zsh (`diaryctl init zsh`)

```zsh
# diaryctl shell integration
__diaryctl_prompt_hook() {
  eval "$(command diaryctl status --env 2>/dev/null)"
}

diaryctl_prompt_info() {
  command diaryctl status 2>/dev/null
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd __diaryctl_prompt_hook

eval "$(command diaryctl completion zsh 2>/dev/null)"
```

## Implementation

### New files

- `cmd/init.go` — `diaryctl init <shell>` command
- `cmd/status.go` — `diaryctl status` command
- `internal/shell/cache.go` — prompt cache read/write/invalidation
- `internal/shell/streak.go` — streak computation logic
- `internal/shell/init_bash.go` — bash init script template
- `internal/shell/init_zsh.go` — zsh init script template

### Modified files

- `cmd/root.go` — register new commands
- `cmd/create.go`, `cmd/jot.go`, `cmd/edit.go`, `cmd/update.go`, `cmd/delete.go` — add PostRun hook for cache invalidation
- `internal/config/config.go` — add shell config section

### Env vars set by prompt hook

| Variable | Example | Description |
|----------|---------|-------------|
| `DIARYCTL_TODAY` | `✓` or `✗` | Today entry indicator |
| `DIARYCTL_STREAK` | `7` | Consecutive days count |
| `DIARYCTL_STREAK_ICON` | `🔥` | Streak suffix icon |
| `DIARYCTL_TEMPLATE` | `morning` | Current default template |
| `DIARYCTL_BACKEND` | `markdown` | Storage backend (opt-in) |
