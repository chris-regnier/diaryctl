# Shell Integration

diaryctl integrates with your shell prompt to show diary status at a glance — whether you've written today, your current streak, and active context.

## Setup

Add one line to your shell configuration:

### Bash

```bash
# ~/.bashrc
eval "$(diaryctl init bash)"
```

### Zsh

```bash
# ~/.zshrc
eval "$(diaryctl init zsh)"
```

This sets up:
- Shell completions
- A prompt hook that refreshes diary status environment variables
- The `diaryctl_prompt_info` helper function

## Prompt Helper

After initialization, use `diaryctl_prompt_info` in your prompt:

### Bash

```bash
PS1='$(diaryctl_prompt_info) \w \$ '
```

### Zsh

```bash
RPROMPT='$(diaryctl_prompt_info)'
```

## Environment Variables

The prompt hook exports these variables on each prompt:

| Variable | Description | Example |
|----------|-------------|---------|
| `DIARYCTL_TODAY` | `1` if today has an entry, `0` otherwise | `1` |
| `DIARYCTL_STREAK` | Consecutive days with entries | `5` |
| `DIARYCTL_TEMPLATE` | Template used for today's entry | `daily` |
| `DIARYCTL_CONTEXT` | Active context name | `feature/auth` |
| `DIARYCTL_BACKEND` | Storage backend in use | `markdown` |

Access these directly in your prompt or scripts:

```bash
diaryctl status --env
# export DIARYCTL_TODAY="1"
# export DIARYCTL_STREAK="5"
# ...
```

## Status Command

Query diary status directly:

```bash
diaryctl status              # "✓ 5🔥 daily"
diaryctl status --env        # shell variable assignments
diaryctl status --refresh    # force cache refresh
```

### Custom Format

Use a Go template for custom output:

```bash
diaryctl status --format "{{.TodayIcon}} streak:{{.Streak}}"
```

## Caching

Status results are cached at `~/.diaryctl/.prompt-cache` to keep prompts fast. The cache is:

- Refreshed when the TTL expires (default: 5 minutes)
- Automatically invalidated after `create`, `edit`, `delete`, `update`, and `jot` commands
- Rolled over at midnight

### Configuration

```toml
[shell]
cache_ttl = "5m"           # how long to cache status
today_icon = "✓"           # shown when today has an entry
no_today_icon = "✗"        # shown when today has no entry
streak_icon = "🔥"          # appended to streak count
show_context = true        # include active context in output
show_backend = false       # include storage backend in output
```

## Streaks

The streak counter shows how many consecutive days (up to and including today) have at least one diary entry. A missed day resets the streak to zero.
