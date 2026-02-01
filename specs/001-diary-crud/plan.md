# Implementation Plan: Basic Diary CRUD

**Branch**: `001-diary-crud` | **Date**: 2026-01-31 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-diary-crud/spec.md`

## Summary

Implement a complete CRUD CLI for diary entries using Go with Cobra.
Entries are stored through a pluggable storage interface with two
backends: Markdown files (with YAML front-matter) and SQLite via
Turso/libSQL. The CLI supports human-readable and JSON output, editor
integration for composition, and auto-paging via Bubble Tea viewport.
Entry IDs are globally-unique nanoid short hashes.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**:
- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — Configuration management
- `github.com/matoous/go-nanoid/v2` — ID generation
- `github.com/adrg/frontmatter` — Markdown front-matter parsing
- `github.com/tursodatabase/go-libsql` — SQLite/Turso backend
- `github.com/charmbracelet/bubbletea` — TUI framework (pager, prompts)
- `github.com/charmbracelet/lipgloss` — Terminal styling
- `github.com/charmbracelet/bubbles` — TUI components (viewport)

**Storage**: Pluggable interface with Markdown file + SQLite (Turso)
backends. Default: Markdown files at `~/.diaryctl/entries/`.
**Testing**: `go test ./...` with contract tests for both backends
**Target Platform**: macOS, Linux (cross-platform CLI)
**Project Type**: Single project
**Performance Goals**: List 1,000 entries in <2s; create in <10s
(excluding editor)
**Constraints**: Single-user, offline-capable, data integrity priority
**Scale/Scope**: Personal diary, 1,000s of entries over years

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. CLI-First | PASS | All operations via `diaryctl` subcommands. stdout/stderr separation. `--json` flag on all commands. Meaningful exit codes (0-3). |
| II. Pluggable Storage | PASS | `Storage` interface with Markdown + SQLite backends. Backend selection via config/flag. Contract tests for both. New backends require no command changes. |
| III. Test-Alongside | PASS | `*_test.go` files alongside code. Contract tests for storage backends. Integration tests for CLI commands. `go test ./...` |
| IV. Simplicity | PASS | Flat package structure. Stdlib preferred where possible. Third-party deps justified (Cobra for CLI, Turso per user requirement, Bubble Tea per user requirement, nanoid for global uniqueness). |
| V. Data Integrity | PASS | Atomic writes (temp+rename for files, transactions for SQLite). Delete requires confirmation or `--force`. No silent data loss. |

**Post-design re-check**: All principles still satisfied. No
Complexity Tracking violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-diary-crud/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli-commands.md  # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
├── root.go              # Root command, global flags, config loading
├── create.go            # diaryctl create
├── list.go              # diaryctl list
├── show.go              # diaryctl show
├── edit.go              # diaryctl edit (editor)
├── update.go            # diaryctl update (inline)
└── delete.go            # diaryctl delete

internal/
├── entry/
│   └── entry.go         # Entry type, validation, ID generation
├── storage/
│   ├── storage.go       # Storage interface, errors, ListOptions
│   ├── markdown/
│   │   └── markdown.go  # Markdown file backend
│   └── sqlite/
│       └── sqlite.go    # Turso/libSQL backend
├── editor/
│   └── editor.go        # Editor resolution & session management
├── config/
│   └── config.go        # Config loading (Viper wrapper)
└── ui/
    ├── pager.go          # Bubble Tea viewport pager
    ├── confirm.go        # Bubble Tea confirmation prompt
    └── output.go         # Human/JSON output formatting

main.go                   # Entry point

storage_test/
└── contract_test.go      # Shared contract tests run against all backends
```

**Structure Decision**: Single project with `cmd/` for CLI commands
and `internal/` for business logic. This follows standard Go CLI
conventions (Cobra pattern). `internal/` enforces encapsulation.
The `storage_test/` directory holds contract tests that are
parameterized across backends.

## Complexity Tracking

> No violations. All design decisions align with constitution.
