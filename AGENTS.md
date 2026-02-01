# AGENTS.md

Agent guidance for diaryctl — a diary management CLI tool.

## Project Status

**Pre-implementation** — Specifications complete, no source code yet.

## Project Overview

A CLI-first diary management tool written in Go 1.22+. Features:
- CRUD operations for diary entries via `diaryctl` command
- Pluggable storage backends (Markdown files, SQLite/Turso)
- Human-readable and JSON output formats
- Editor integration for composing/editing entries
- Auto-paging for long output

## Speckit Workflow

This project uses a specification-driven workflow. Features go through phases:

1. **Specify** (`/speckit.specify`) — Define user stories, requirements, acceptance criteria
2. **Clarify** (`/speckit.clarify`) — Resolve ambiguities with stakeholder Q&A
3. **Plan** (`/speckit.plan`) — Technical design, architecture, dependencies
4. **Tasks** (`/speckit.tasks`) — Break down into executable implementation tasks
5. **Implement** (`/speckit.implement`) — Execute tasks with TDD approach

### Key Directories

```
specs/<feature-id>/
├── spec.md              # Feature specification (user stories, requirements)
├── plan.md              # Technical implementation plan
├── data-model.md        # Entities, storage schemas
├── contracts/           # CLI command contracts, API specs
│   └── cli-commands.md
├── research.md          # Technical research notes
├── quickstart.md        # Integration scenarios
└── checklists/
    └── requirements.md  # Pre-implementation quality gates

.specify/
├── memory/
│   └── constitution.md  # Project principles (source of truth)
├── templates/           # Document templates for speckit
└── scripts/bash/        # Workflow automation scripts

.claude/commands/        # Speckit slash commands
```

## Constitution Principles

Read `.specify/memory/constitution.md` before any work. Key principles:

| # | Principle | Summary |
|---|-----------|---------|
| I | CLI-First | All ops via CLI. stdout/stderr separation. `--json` flag. Meaningful exit codes. |
| II | Pluggable Storage | `Storage` interface. Markdown + SQLite backends. Contract tests for each. |
| III | Test-Alongside | Tests written with implementation. `*_test.go` next to code. `go test ./...` |
| IV | Simplicity | YAGNI. Stdlib preferred. Flat packages. Minimal indirection. |
| V | Data Integrity | Atomic writes. Confirm destructive ops. No silent data loss. |

## Planned Tech Stack

From `specs/001-diary-crud/plan.md`:

- **Language**: Go 1.22+
- **CLI Framework**: `github.com/spf13/cobra`
- **Config**: `github.com/spf13/viper`
- **ID Generation**: `github.com/matoous/go-nanoid/v2` (8-char alphanumeric)
- **Markdown Parsing**: `github.com/adrg/frontmatter`
- **SQLite**: `github.com/tursodatabase/go-libsql`
- **TUI**: `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss`

## Planned Project Structure

```
cmd/
├── root.go              # Root command, global flags
├── create.go            # diaryctl create
├── list.go              # diaryctl list
├── show.go              # diaryctl show
├── edit.go              # diaryctl edit (editor)
├── update.go            # diaryctl update (inline)
└── delete.go            # diaryctl delete

internal/
├── entry/               # Entry type, validation, ID generation
├── storage/             # Storage interface + backends
│   ├── markdown/        # Markdown file backend
│   └── sqlite/          # Turso/libSQL backend
├── editor/              # Editor resolution & sessions
├── config/              # Config loading
└── ui/                  # Pager, confirm prompts, output formatting

storage_test/
└── contract_test.go     # Shared tests for all backends

main.go
```

## Commands (Once Implemented)

```bash
# Build
go build -o diaryctl .

# Test
go test ./...

# Lint (quality standards require)
go vet ./...
staticcheck ./...
gofmt -d .
goimports -d .
```

## CLI Contract

From `specs/001-diary-crud/contracts/cli-commands.md`:

```
diaryctl create [content]        # Create entry (inline or editor)
diaryctl list [--date DATE]      # List entries
diaryctl show <id>               # Show full entry
diaryctl edit <id>               # Update via editor
diaryctl update <id> <content>   # Update inline
diaryctl delete <id> [--force]   # Delete with confirmation
```

Global flags: `--json`, `--config <path>`, `--storage <backend>`

Exit codes: `0` success, `1` validation/not-found, `2` storage error, `3` editor error

## Data Model

**Entry** (from `specs/001-diary-crud/data-model.md`):
- `id`: 8-char nanoid (lowercase alphanumeric, globally unique, immutable)
- `content`: non-empty text body
- `created_at`: UTC timestamp (immutable after creation)
- `updated_at`: UTC timestamp (updated on every modification)

**Storage backends**:
- **Markdown**: `~/.diaryctl/entries/YYYY/MM/DD/<id>.md` with YAML frontmatter
- **SQLite**: Single table `entries` with ISO 8601 timestamps

## Development Guidelines

1. **Read the spec first** — Check `specs/<feature>/spec.md` for requirements
2. **Check the plan** — Technical decisions are in `plan.md`
3. **Follow contracts** — CLI behavior is specified in `contracts/cli-commands.md`
4. **Test alongside** — Write `*_test.go` with implementation, not after
5. **Contract tests** — All storage backends must pass identical contract tests
6. **Atomic writes** — Use temp file + rename (files) or transactions (SQLite)
7. **No silent data loss** — Confirm destructive operations or require `--force`

## Implementation Notes

- **Editor**: Resolve via `$EDITOR` → `$VISUAL` → platform default
- **Paging**: Auto-page when stdout is TTY and output exceeds terminal height
- **Date format**: ISO 8601 (`YYYY-MM-DD`) for filters
- **Storage location**: `~/.diaryctl/` (or `$XDG_DATA_HOME/diaryctl/`)
- **Preview length**: First line, truncated to 80 chars

## Running Speckit Commands

```bash
# Check prerequisites before any speckit command
.specify/scripts/bash/check-prerequisites.sh --json

# Create new feature
.specify/scripts/bash/create-new-feature.sh <feature-id>
```

Or use slash commands in Claude: `/speckit.specify`, `/speckit.plan`, etc.
