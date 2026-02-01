<!--
Sync Impact Report
===================
Version change: 0.0.0 → 1.0.0 (initial ratification)
Modified principles: N/A (initial creation)
Added sections:
  - Core Principles (5): CLI-First, Pluggable Storage, Test-Alongside,
    Simplicity, Data Integrity
  - Development Workflow
  - Quality Standards
  - Governance
Removed sections: N/A
Templates requiring updates:
  - .specify/templates/plan-template.md — ✅ no changes needed
    (Constitution Check section is generic; principles align)
  - .specify/templates/spec-template.md — ✅ no changes needed
    (user story structure compatible with all principles)
  - .specify/templates/tasks-template.md — ✅ no changes needed
    (phase structure supports test-alongside discipline)
  - .specify/templates/agent-file-template.md — ✅ no changes needed
    (template is generic, populated per-feature)
Follow-up TODOs: none
-->

# diaryctl Constitution

## Core Principles

### I. CLI-First

Every feature MUST be accessible through the `diaryctl` command-line
interface. The CLI is the primary and authoritative interface.

- All operations accept structured input via arguments and stdin.
- Output MUST go to stdout; errors and diagnostics MUST go to stderr.
- Support both human-readable (default) and JSON (`--json`) output
  formats for every command that produces output.
- Commands MUST return meaningful exit codes (0 = success, non-zero =
  specific failure category).

**Rationale**: A CLI-first tool ensures scriptability, composability
with other Unix tools, and predictable automation.

### II. Pluggable Storage

Diary data MUST be accessible through a storage interface that supports
multiple backends. The initial backends are local Markdown files and
SQLite.

- A `Storage` interface MUST define all data access operations.
- Backend selection MUST be configurable (config file or CLI flag).
- Each backend MUST pass the same contract tests.
- Adding a new backend MUST NOT require changes to commands or business
  logic.

**Rationale**: Users have different needs — some prefer plain-text
files for portability and version control; others prefer structured
databases for querying. A pluggable design serves both without
coupling the core to either.

### III. Test-Alongside

Tests MUST be written alongside implementation code. Every exported
function and every CLI command MUST have corresponding test coverage
before the feature is considered complete.

- Unit tests live next to the code they test (`*_test.go` in the same
  package).
- Integration tests for CLI commands live in a dedicated `tests/`
  or `cmd/*_test.go` structure.
- Storage backend contract tests MUST exist and MUST be run against
  every backend implementation.
- Tests MUST be runnable with `go test ./...` and MUST pass in CI.

**Rationale**: Test-alongside balances velocity with confidence. It
avoids the rigidity of strict TDD while ensuring no feature ships
without verified behavior.

### IV. Simplicity

Prefer the simplest solution that meets current requirements. Do not
add abstractions, configuration options, or extension points for
hypothetical future needs.

- YAGNI: features and code paths MUST be justified by a current
  requirement.
- No more than one level of indirection unless the complexity is
  justified and documented in a Complexity Tracking table.
- Standard library preferred over third-party dependencies. External
  dependencies MUST be justified.
- Flat package structure until package size or cohesion demands
  splitting.

**Rationale**: Simplicity reduces maintenance burden, speeds up
onboarding, and makes the codebase easier to reason about.

### V. Data Integrity

User diary data MUST NOT be silently lost, corrupted, or overwritten.
All write operations MUST be safe by default.

- Write operations MUST be atomic or clearly documented as non-atomic
  with recovery guidance.
- Destructive operations (delete, overwrite) MUST require explicit
  confirmation or a `--force` flag.
- The Markdown file backend MUST preserve content it does not
  understand (e.g., custom front-matter keys, embedded HTML).
- Backup/export functionality MUST be available before any migration
  or schema change.

**Rationale**: Diary entries are personal and irreplaceable. The tool
MUST prioritize data safety above convenience or performance.

## Quality Standards

- Code MUST pass `go vet`, `golint` (or `staticcheck`), and
  `gofmt`/`goimports` with zero findings before merge.
- Public API surface (exported types, functions) MUST have GoDoc
  comments.
- Error messages MUST include enough context for the user to
  understand what went wrong and how to fix it.
- All CLI commands MUST include `--help` text with usage examples.

## Development Workflow

- Feature work follows the speckit workflow: spec, plan, tasks,
  implement.
- Commits MUST be scoped and descriptive (conventional commits
  encouraged but not enforced).
- The `main` branch MUST always build and pass tests.
- Code review is required for all changes (self-review acceptable
  for single-contributor phases).

## Governance

This constitution is the authoritative source for project principles
and development standards. All implementation decisions, code reviews,
and architectural choices MUST be consistent with these principles.

- **Amendments**: Any change to this constitution MUST be documented
  with a version bump, rationale, and updated Sync Impact Report.
- **Versioning**: MAJOR for principle removal/redefinition, MINOR for
  new principles or material expansion, PATCH for clarifications.
- **Compliance**: Every plan.md Constitution Check MUST verify
  alignment with the current principles. Violations MUST be justified
  in the Complexity Tracking table.
- **Runtime guidance**: See `CLAUDE.md` for agent-specific development
  guidance.

**Version**: 1.0.0 | **Ratified**: 2026-01-31 | **Last Amended**: 2026-01-31
