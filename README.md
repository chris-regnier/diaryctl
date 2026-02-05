# diaryctl

A powerful command-line diary management tool with pluggable storage backends and an elegant TUI.

## Features

- **Pluggable Storage**: Choose between Markdown files or SQLite/Turso database
- **Rich CLI**: Full-featured commands for creating, editing, listing, and managing diary entries
- **Interactive TUI**: Beautiful terminal user interface built with Bubble Tea
- **Context Tracking**: Automatic context detection from git branches, manual context tagging
- **Template Support**: Create and use custom entry templates
- **Shell Integration**: Export environment variables for shell prompts
- **Daily Summaries**: View entries by date range with filtering

## Installation

### From Source

```bash
go install github.com/chris-regnier/diaryctl@latest
```

### Building Locally

```bash
git clone https://github.com/chris-regnier/diaryctl.git
cd diaryctl
go build -o diaryctl
```

## Quick Start

### First-Time Setup

diaryctl automatically creates necessary directories and configuration on first use. No initialization command is required.

Configuration is optional - defaults work out of the box with markdown storage in `~/.diaryctl/`.

### Basic Usage

```bash
# Create a new entry interactively
diaryctl create

# Quick jot down a thought
diaryctl jot "Had a great idea for the project"

# List recent entries
diaryctl list

# Show today's entries
diaryctl today

# Launch interactive TUI
diaryctl
```

## Storage Backends

### Markdown (Default)

Stores entries as individual markdown files with YAML frontmatter:

```yaml
---
id: 01HQXYZ...
created: 2026-02-04T10:30:00Z
contexts:
  - feature/auth
  - sprint:23
template: daily
---

Entry content goes here...
```

### SQLite/Turso

Stores entries in a SQLite database compatible with Turso for remote sync.

## Context Tracking

diaryctl automatically tracks context from:

- **Git branches**: Current branch becomes a context
- **Manual tags**: Set custom contexts with `diaryctl context set project:auth`
- **Date/time**: Automatic datetime context provider

Filter entries by context:

```bash
diaryctl list --context feature/auth
```

## Templates

Create reusable entry templates:

```bash
# Create a new template
diaryctl template create standup

# Use template when creating entry
diaryctl create --template standup
```

## Configuration

Configuration file locations (searched in order):
- `$XDG_CONFIG_HOME/diaryctl/config.toml` (if XDG_CONFIG_HOME is set)
- `~/.diaryctl/config.toml`

```toml
storage = "markdown"  # or "sqlite"
data_dir = "~/.diaryctl/data"
editor = "hx"  # or vim, nano, etc.
context_providers = ["git", "datetime"]
context_resolvers = ["git"]
```

## Commands

| Command | Description |
|---------|-------------|
| `diaryctl` | Launch interactive TUI (when in TTY) |
| `diaryctl create` | Create a new entry |
| `diaryctl edit <id>` | Edit an entry |
| `diaryctl show <id>` | Display an entry |
| `diaryctl list` | List entries |
| `diaryctl delete <id>` | Delete an entry |
| `diaryctl jot <text>` | Quick entry creation |
| `diaryctl today` | Show today's entries |
| `diaryctl daily` | Show entries in date range |
| `diaryctl context` | Manage contexts |
| `diaryctl template` | Manage templates |
| `diaryctl status` | Show current status |

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
diaryctl/
├── cmd/              # Cobra commands
├── internal/
│   ├── config/       # Configuration management
│   ├── context/      # Context providers and resolvers
│   ├── daily/        # Daily view logic
│   ├── editor/       # Editor integration
│   ├── entry/        # Entry domain model
│   ├── shell/        # Shell integration
│   ├── storage/      # Storage interface
│   │   ├── markdown/ # Markdown backend
│   │   └── sqlite/   # SQLite backend
│   ├── template/     # Template management
│   └── ui/           # TUI components
└── main.go
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security policy and vulnerability reporting.
