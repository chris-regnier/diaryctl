# Implementation Plan: Daily Aggregated View with Interactive Picker

**Branch**: `003-daily-view-picker` | **Date**: 2026-01-31 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-daily-view-picker/spec.md`

## Summary

Add a `diaryctl daily` command that provides a day-over-day aggregated view of diary entries. In interactive mode, users navigate a Bubble Tea date picker to browse and drill into entries by day. In non-interactive mode (piped output or `--no-interactive`), the command prints a grouped-by-day summary to stdout in plain text or JSON. A new `ListDays` storage method provides efficient date-grouped aggregation, while the existing `List` method (with date range extensions) handles per-day entry retrieval.

## Technical Context

**Language/Version**: Go 1.24 (go1.24.12 toolchain)
**Primary Dependencies**: Cobra (CLI), Bubble Tea + Bubbles + Lipgloss (TUI), Viper (config)
**Storage**: Pluggable — Markdown files and SQLite/libSQL backends via `storage.Storage` interface
**Testing**: `go test ./...` with contract tests for both storage backends
**Target Platform**: macOS, Linux (terminal CLI)
**Project Type**: Single Go project
**Performance Goals**: Interactive launch <2s for 1,000 entries / 365 days; day navigation <0.5s; non-interactive <3s
**Constraints**: Must work with existing storage interface; no new external dependencies beyond what's already in go.mod
**Scale/Scope**: Up to 1,000+ entries across 365+ days

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First | ✅ PASS | New `daily` subcommand; `--json`, `--no-interactive` flags; stdout/stderr separation; exit codes |
| II. Pluggable Storage | ✅ PASS | New `ListDays` method added to `Storage` interface; both backends implement it; contract tests cover it |
| III. Test-Alongside | ✅ PASS | Contract tests for `ListDays`, unit tests for aggregation logic, command tests for `daily` |
| IV. Simplicity | ✅ PASS | Uses existing Bubble Tea dependency; `charmbracelet/bubbles/list` already available in the dependency tree; no new deps. Single `daily.go` command file + one new TUI model file |
| V. Data Integrity | ✅ PASS | Read-only feature — no writes, no mutations. No data integrity risk |

**Pre-design gate: PASSED**

### Post-Design Re-Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First | ✅ PASS | `daily` command with `--from`, `--to`, `--no-interactive`, `--json` flags. Plain text and JSON output. Exit codes 0/1/2. Errors to stderr. |
| II. Pluggable Storage | ✅ PASS | `ListDays` added to `Storage` interface with `DaySummary` return type. Both markdown and sqlite backends implement it. Contract tests defined (TC-01 through TC-12). |
| III. Test-Alongside | ✅ PASS | Storage contract tests for `ListDays` + date range `List`. Command tests for `daily`. TUI model testable via Bubble Tea test utilities. |
| IV. Simplicity | ✅ PASS | No new dependencies. One new interface method. One new command file. One new TUI model file. Uses existing `bubbles/list` and `viewport` components. |
| V. Data Integrity | ✅ PASS | Read-only feature. No writes, updates, or deletes. |

**Post-design gate: PASSED**

## Project Structure

### Documentation (this feature)

```text
specs/003-daily-view-picker/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
├── daily.go             # New: daily view command (interactive + non-interactive)
└── daily_test.go        # New: tests for daily command

internal/
├── entry/
│   └── entry.go         # Unchanged
├── storage/
│   ├── storage.go       # Modified: add DaySummary, ListDays to Storage interface; add date range to ListOptions
│   ├── contract_test.go # Modified: add ListDays contract tests + date range tests
│   ├── markdown/
│   │   └── markdown.go  # Modified: implement ListDays + date range filter
│   └── sqlite/
│       └── sqlite.go    # Modified: implement ListDays + date range filter
├── config/
│   └── config.go        # Unchanged
├── editor/
│   └── editor.go        # Unchanged
└── ui/
    ├── output.go        # Modified: add FormatDailySummary, DailySummaryJSON types
    ├── pager.go          # Unchanged
    ├── confirm.go        # Unchanged
    └── picker.go         # New: Bubble Tea interactive daily picker model
```

**Structure Decision**: Follows existing flat `cmd/` + `internal/` layout. The new TUI picker model goes in `internal/ui/picker.go` alongside existing pager and confirm models. The command file `cmd/daily.go` handles flag parsing, TTY detection, and dispatching to interactive vs non-interactive paths.

## Complexity Tracking

> No constitution violations. Table intentionally left empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
