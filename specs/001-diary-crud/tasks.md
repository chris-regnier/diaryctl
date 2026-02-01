# Tasks: Basic Diary CRUD

**Input**: Design documents from `/specs/001-diary-crud/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included per constitution principle III (Test-Alongside). Tests are written alongside implementation within each phase.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `cmd/`, `internal/`, `main.go` at repository root (Go with Cobra)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, Go module, dependencies, directory structure

- [X] T001 Initialize Go module with `go mod init` and create `main.go` entry point in main.go
- [X] T002 Install all dependencies: cobra, viper, go-nanoid, frontmatter, go-libsql, bubbletea, lipgloss, bubbles in go.mod
- [X] T003 Create directory structure: cmd/, internal/entry/, internal/storage/, internal/storage/markdown/, internal/storage/sqlite/, internal/editor/, internal/config/, internal/ui/
- [X] T004 [P] Implement root Cobra command with global flags (--json, --config, --storage, --help) in cmd/root.go
- [X] T005 [P] Implement config loading with Viper (TOML at ~/.diaryctl/config.toml, XDG support, env var binding) in internal/config/config.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types, storage interface, and backends that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Define Entry struct with fields (id, content, created_at, updated_at), validation methods (ValidateContent, ValidateID), and NewID() nanoid generator in internal/entry/entry.go
- [X] T007 Define Storage interface (Create, Get, List, Update, Delete), ListOptions struct, and error types (ErrNotFound, ErrConflict, ErrStorage, ErrValidation) in internal/storage/storage.go
- [X] T008 [P] Implement Markdown file backend: Create/Get/List/Update/Delete with YAML front-matter, date-based directory structure (~/.diaryctl/entries/YYYY/MM/DD/<id>.md), atomic writes (temp+rename), file locking (flock) in internal/storage/markdown/markdown.go
- [X] T009 [P] Implement SQLite/Turso backend: Create/Get/List/Update/Delete with libSQL, schema creation (entries table, indexes), WAL mode, transaction-based writes in internal/storage/sqlite/sqlite.go
- [X] T010 Write storage contract tests (12 test cases from contracts/cli-commands.md) parameterized to run against both Markdown and SQLite backends in internal/storage/contract_test.go
- [X] T011 [P] Implement human-readable and JSON output formatting helpers (entry formatting, list formatting, preview truncation) in internal/ui/output.go
- [X] T012 [P] Implement Bubble Tea viewport pager (auto-page when TTY and output exceeds terminal height, direct stdout when not TTY) in internal/ui/pager.go
- [X] T013 [P] Implement editor resolution (diaryctl config > EDITOR > VISUAL > vi fallback) and editor session management (temp file create, launch editor, read result, detect empty/unchanged, cleanup) in internal/editor/editor.go
- [X] T014 Wire storage backend selection in root command: read config/flag, instantiate correct backend, pass to subcommands in cmd/root.go

**Checkpoint**: Foundation ready — storage works with both backends, contract tests pass, UI helpers and editor ready

---

## Phase 3: User Story 1 — Create a Diary Entry (Priority: P1) MVP

**Goal**: Users can create diary entries via inline text, editor, or stdin

**Independent Test**: Run `diaryctl create "test"`, verify entry is persisted and retrievable via storage

### Implementation for User Story 1

- [X] T015 [US1] Implement `diaryctl create` command: parse inline content args, handle `-` for stdin, open editor when no args, call storage.Create, print confirmation with entry ID in cmd/create.go
- [X] T016 [US1] Add --json output support to create command, returning full entry JSON on success in cmd/create.go
- [X] T017 [US1] Add error handling: empty content validation (exit 1), storage errors (exit 2), editor errors (exit 3) with contextual error messages to stderr in cmd/create.go
- [X] T018 [US1] Write tests for create command: inline creation, stdin creation, empty content rejection, JSON output, exit codes in cmd/create_test.go

**Checkpoint**: `diaryctl create "text"` works end-to-end with both backends. MVP is functional.

---

## Phase 4: User Story 2 — Read Diary Entries (Priority: P2)

**Goal**: Users can list entries (with preview, date filter, auto-paging) and view single entries

**Independent Test**: Pre-populate entries, run `diaryctl list` and `diaryctl show <id>`, verify output format and content

### Implementation for User Story 2

- [X] T019 [P] [US2] Implement `diaryctl list` command: fetch entries via storage.List, format as table (id, date, preview), apply --date filter, auto-page via Bubble Tea viewport when TTY in cmd/list.go
- [X] T020 [P] [US2] Implement `diaryctl show` command: fetch entry via storage.Get, display full content with metadata header (id, created, modified), auto-page when TTY in cmd/show.go
- [X] T021 [US2] Add --json output support to list (array of entry summaries) and show (full entry object) commands in cmd/list.go and cmd/show.go
- [X] T022 [US2] Add error handling: not-found (exit 1), invalid date format (exit 1), storage errors (exit 2), empty list message in cmd/list.go and cmd/show.go
- [X] T023 [US2] Write tests for list command: reverse chronological order, date filtering, empty list message, JSON output, auto-pager TTY detection in cmd/list_test.go
- [X] T024 [US2] Write tests for show command: full content display, not-found error, JSON output in cmd/show_test.go

**Checkpoint**: `diaryctl list` and `diaryctl show <id>` work. Entries created in US1 are browsable.

---

## Phase 5: User Story 3 — Update a Diary Entry (Priority: P3)

**Goal**: Users can modify existing entries via editor or inline replacement

**Independent Test**: Create an entry, update it, verify content changed and updated_at advanced while created_at preserved

### Implementation for User Story 3

- [X] T025 [P] [US3] Implement `diaryctl edit` command: fetch entry, open editor pre-filled with content, detect unchanged content, call storage.Update on change in cmd/edit.go
- [X] T026 [P] [US3] Implement `diaryctl update` command: fetch entry, accept inline content or stdin, call storage.Update in cmd/update.go
- [X] T027 [US3] Add --json output support to edit and update commands in cmd/edit.go and cmd/update.go
- [X] T028 [US3] Add error handling: not-found (exit 1), empty content (exit 1), storage errors (exit 2), editor errors (exit 3), "no changes detected" message in cmd/edit.go and cmd/update.go
- [X] T029 [US3] Write tests for edit command: content update, unchanged detection, not-found error, editor failure in cmd/edit_test.go
- [X] T030 [US3] Write tests for update command: inline update, stdin update, timestamp preservation, not-found error, JSON output in cmd/update_test.go

**Checkpoint**: `diaryctl edit <id>` and `diaryctl update <id> "new"` work. Timestamps behave correctly.

---

## Phase 6: User Story 4 — Delete a Diary Entry (Priority: P4)

**Goal**: Users can permanently delete entries with confirmation safeguard

**Independent Test**: Create an entry, delete it with confirmation, verify it no longer appears in listings

### Implementation for User Story 4

- [X] T031 [US4] Implement `diaryctl delete` command: fetch entry, display preview, prompt for confirmation via Bubble Tea confirm component, call storage.Delete on confirmation in cmd/delete.go
- [X] T032 [US4] Implement Bubble Tea confirmation prompt component (y/N input, styled with Lip Gloss) in internal/ui/confirm.go
- [X] T033 [US4] Add --force flag to skip confirmation, add --json output support in cmd/delete.go
- [X] T034 [US4] Add error handling: not-found (exit 1), storage errors (exit 2), cancellation message (exit 0) in cmd/delete.go
- [X] T035 [US4] Write tests for delete command: confirmed deletion, declined deletion, --force flag, not-found error, JSON output in cmd/delete_test.go

**Checkpoint**: Full CRUD is complete. All four operations work end-to-end.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Quality, documentation, and validation across all stories

- [X] T036 [P] Add --help text with usage examples to all commands (create, list, show, edit, update, delete) in cmd/*.go
- [X] T037 [P] Run `go vet`, `staticcheck`, `gofmt`/`goimports` and fix all findings across codebase
- [X] T038 [P] Add GoDoc comments to all exported types and functions in internal/**/*.go
- [X] T039 Write editor integration tests: editor resolution fallback chain, temp file cleanup, cancelled session handling in internal/editor/editor_test.go
- [X] T040 Run quickstart.md validation: execute all quickstart commands end-to-end and verify expected output
- [X] T041 Verify `go test ./...` passes with both storage backends

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Phase 2 completion
  - US1 (Phase 3): No dependencies on other stories
  - US2 (Phase 4): No dependencies on other stories (can run parallel with US1)
  - US3 (Phase 5): No dependencies on other stories (can run parallel with US1/US2)
  - US4 (Phase 6): No dependencies on other stories (can run parallel with US1/US2/US3)
- **Polish (Phase 7)**: Depends on all user stories being complete

### Within Each User Story

- Implementation tasks before test tasks within the same story (test-alongside)
- Commands depend on the foundational storage interface and UI helpers
- Error handling integrated into command implementation

### Parallel Opportunities

- T004, T005 can run in parallel (Setup phase)
- T008, T009 can run in parallel (both storage backends)
- T011, T012, T013 can run in parallel (UI helpers and editor)
- T019, T020 can run in parallel (list and show commands)
- T025, T026 can run in parallel (edit and update commands)
- T036, T037, T038 can run in parallel (Polish phase)
- All four user story phases can run in parallel if team capacity allows

---

## Parallel Example: Phase 2

```bash
# Launch both storage backends together:
Task: "Implement Markdown file backend in internal/storage/markdown/markdown.go"
Task: "Implement SQLite/Turso backend in internal/storage/sqlite/sqlite.go"

# Launch UI helpers together:
Task: "Implement output formatting in internal/ui/output.go"
Task: "Implement Bubble Tea pager in internal/ui/pager.go"
Task: "Implement editor management in internal/editor/editor.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (Create)
4. **STOP and VALIDATE**: `diaryctl create "test"` works with both backends
5. Deployable MVP — users can create entries

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Create) → Test independently → MVP!
3. Add US2 (Read) → Test independently → Users can create + browse
4. Add US3 (Update) → Test independently → Users can create + browse + edit
5. Add US4 (Delete) → Test independently → Full CRUD
6. Polish → Production-ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Constitution compliance: test-alongside (III), atomic writes (V), CLI-first (I), pluggable storage (II)
