# diaryctl

> A powerful command-line diary management tool with pluggable storage backends and an elegant TUI.

## Features

- **Pluggable Storage** — Markdown files or SQLite/Turso database
- **Rich CLI** — Create, edit, list, search, and manage diary entries
- **Interactive TUI** — Terminal UI built with Bubble Tea
- **Context Tracking** — Automatic detection from git branches, manual tagging
- **Template Support** — Reusable entry templates with variables
- **Shell Integration** — Prompt status, streaks, environment variables
- **Theming** — 9 built-in color presets plus custom colors
- **MCP Server** — AI assistant integration via Model Context Protocol

## Quick Start

```bash
# Install
go install github.com/chris-regnier/diaryctl@latest

# Create your first entry
diaryctl create "Hello, diary!"

# Jot a quick note
diaryctl jot "Had a great idea"

# Launch the TUI
diaryctl
```

See the [Getting Started](getting-started.md) guide for a full walkthrough.
