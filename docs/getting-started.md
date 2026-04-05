# Getting Started

This guide walks you through installing diaryctl and writing your first diary entry.

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

## First-Time Setup

diaryctl works out of the box with no configuration required. On first use it creates its data directory at `~/.diaryctl/` and uses the Markdown storage backend by default.

To customize behavior, create a config file at `~/.diaryctl/config.toml` (see [Configuration](configuration.md)).

## Basic Workflow

### Create an Entry

Open your editor to write a new entry:

```bash
diaryctl create
```

Or pass content inline:

```bash
diaryctl create "Shipped the new auth module today"
```

You can also pipe content from stdin:

```bash
echo "Quick thought" | diaryctl create -
```

### Jot a Quick Note

Append a timestamped note to today's entry without opening an editor:

```bash
diaryctl jot "Had a productive standup"
```

Each jot is appended as a timestamped bullet (`- **HH:MM** text`) to today's entry.

### View Today's Entry

```bash
diaryctl today
```

### List Recent Entries

```bash
diaryctl list
```

### Browse Interactively

Launch the TUI by running diaryctl with no arguments in an interactive terminal:

```bash
diaryctl
```

This opens an interactive date picker where you can browse entries by day, view full content, edit, delete, and manage contexts. See [TUI Guide](tui.md) for details.

## What's Next

- [CLI Reference](cli-reference.md) — full command and flag reference
- [Configuration](configuration.md) — config file, environment variables, editor setup
- [Storage Backends](storage-backends.md) — Markdown files vs. SQLite/Turso
- [Templates](templates.md) — reusable entry templates
- [Contexts](contexts.md) — automatic and manual context tagging
- [Shell Integration](shell-integration.md) — prompt status and environment variables
- [Themes](themes.md) — built-in presets and custom colors
- [TUI Guide](tui.md) — interactive terminal interface
