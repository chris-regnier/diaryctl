# diaryctl - Project Overview

## Purpose
CLI/TUI diary management tool with pluggable storage backends.

## Tech Stack
- Go 1.24 (toolchain go1.24.12)
- Cobra (CLI), Bubble Tea + Bubbles + Lipgloss (TUI), Viper (config)
- Storage: Markdown files (`~/.diaryctl/entries/`) and SQLite/libSQL backends via `storage.Storage` interface

## Key Packages
- `cmd/` - Cobra commands (create, show, list, update, delete, edit, daily, today, jot, template, context, status, seed)
- `internal/ui/` - TUI components (picker.go, pager.go, output.go, confirm.go)
- `internal/storage/` - Storage interface + markdown/sqlite backends
- `internal/entry/` - Entry model (ID, Content, CreatedAt, UpdatedAt, Templates, Contexts)
- `internal/config/` - TOML configuration
- `internal/editor/` - External editor integration
- `internal/template/` - Template composition
- `internal/context/` - Context providers/resolvers
- `internal/daily/` - Daily entry utilities
- `internal/shell/` - Shell integration

## Conventions
- 8-char nanoid IDs (lowercase alphanumeric)
- UTC timestamps internally, local timezone for display
- `storage.Storage` interface for all persistence
- Bubble Tea for interactive TUI, with alt screen mode
