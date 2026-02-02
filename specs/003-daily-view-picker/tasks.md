# Tasks: Daily Aggregated View with Interactive Picker

**Input**: Design documents from `/specs/003-daily-view-picker/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Included per constitution principle III (Test-Alongside). Contract tests written alongside storage implementation; command tests alongside command implementation.

**Organization**: Tasks grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Go single project: `cmd/`, `internal/` at repository root
- Tests: `*_test.go` alongside source files

---

## Phase 1: Setup

**Purpose**: No new project setup needed — this feature extends an existing codebase. Phase 1 is a no-op.

**Checkpoint**: Existing project builds and tests pass (`go test ./...`).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend the storage interface and both backends with `ListDays` + date range support. All user stories depend on this.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T001 Add `DaySummary`, `ListDaysOptions` types and `ListDays` method to `Storage` interface; add `StartDate`/`EndDate` fields to `ListOptions` in `internal/storage/storage.go`
- [X] T002 [P] Implement `ListDays` for markdown backend in `internal/storage/markdown/markdown.go` — scan `YYYY/MM/DD/` directory tree, count files per directory, read newest for preview, apply date range filter, return reverse chronological order
- [X] T003 [P] Implement `ListDays` for sqlite backend in `internal/storage/sqlite/sqlite.go` — use `GROUP BY date(created_at)` query with date range `WHERE` clauses, return reverse chronological order
- [X] T004 [P] Extend `List` in markdown backend to support `StartDate`/`EndDate` date range filtering in `internal/storage/markdown/markdown.go` — `Date` takes precedence if set
- [X] T005 [P] Extend `List` in sqlite backend to support `StartDate`/`EndDate` date range filtering in `internal/storage/sqlite/sqlite.go` — `Date` takes precedence if set
- [X] T006 Add `ListDays` contract tests (TC-01 through TC-08) to `internal/storage/contract_test.go` — empty store, single day, multiple days, StartDate, EndDate, both bounds, empty range, preview content
- [X] T007 Add date range `List` contract tests (TC-09 through TC-12) to `internal/storage/contract_test.go` — StartDate only, EndDate only, range, Date precedence over range
- [X] T008 Add `DayGroupJSON` type and `FormatDailySummary` function to `internal/ui/output.go` — grouped plain-text format with day headers; `DayGroupJSON` struct for JSON output; helper to build `[]DayGroupJSON` from entries grouped by day

**Checkpoint**: `go test ./...` passes. Both backends implement `ListDays` and date range `List`. Contract tests cover all 12 test cases.

---

## Phase 3: User Story 1 — Browse Entries by Day (Priority: P1) 🎯 MVP

**Goal**: Users can launch `diaryctl daily`, see an interactive date picker, select a date to view entries, and drill into individual entries.

**Independent Test**: Pre-populate entries across multiple days, launch daily view, verify dates listed with correct counts, select a date, confirm entries displayed, select an entry, confirm full content shown.

### Implementation for User Story 1

- [X] T009 [US1] Create the Bubble Tea picker model scaffold in `internal/ui/picker.go` — define `pickerModel` struct with three screen states (DateList, DayDetail, EntryDetail), `Init`, `Update`, `View` methods, and `RunPicker` public entry point that creates a `tea.Program` with `WithAltScreen`
- [X] T010 [US1] Implement DateList screen in `internal/ui/picker.go` — use `bubbles/list` to display `[]DaySummary` as list items formatted as `"YYYY-MM-DD  (N entries)  <preview>"`; handle `↑`/`↓`/`j`/`k` navigation, `Enter` to transition to DayDetail, `q`/`Ctrl+C` to quit; show help hint footer
- [X] T011 [US1] Implement DayDetail screen in `internal/ui/picker.go` — fetch entries for selected date via storage callback, display as scrollable list with `"<id>  HH:MM  <preview>"` items; handle `Enter` to transition to EntryDetail, `Esc`/`Backspace` to return to DateList; show help hint footer
- [X] T012 [US1] Implement EntryDetail screen in `internal/ui/picker.go` — display full entry using `bubbles/viewport` (following pager.go pattern), show ID/Created/Modified header then content; handle `Esc`/`Backspace` to return to DayDetail, `q` to quit; handle `WindowSizeMsg` for adaptive layout
- [X] T013 [US1] Handle empty state in picker — if `ListDays` returns empty slice, print `"No diary entries found."` to stdout and exit without launching Bubble Tea; handle in the `RunPicker` function
- [X] T014 [US1] Create `diaryctl daily` command in `cmd/daily.go` — register with `rootCmd.AddCommand`; add `--from`, `--to`, `--no-interactive` flags; parse date flags as `YYYY-MM-DD` in local timezone; implement mode selection logic (JSON → non-interactive, --no-interactive → non-interactive, non-TTY → non-interactive, else → interactive); call `ui.RunPicker` for interactive mode passing storage reference and date range options
- [X] T015 [US1] Add `--help` text with usage examples to `daily` command in `cmd/daily.go` — include `Use`, `Short`, `Long`, `Example` fields per existing command patterns (list.go, show.go)
- [X] T016 [US1] Add command tests for `daily` in `cmd/daily_test.go` — test date flag parsing, test empty store message, test that storage is called with correct `ListDaysOptions` for date range

**Checkpoint**: `diaryctl daily` launches interactive picker, dates are listed, entries viewable per-date, full entry drilldown works. `go test ./...` passes.

---

## Phase 4: User Story 2 — Navigate Between Days (Priority: P2)

**Goal**: Users can navigate to the next/previous day directly from the DayDetail screen without returning to the date list.

**Independent Test**: Select a date in the picker, press `→`/`n` to go to next day, verify entries update; press `←`/`p` for previous day; verify boundary behavior (stays on current day at edges).

### Implementation for User Story 2

- [X] T017 [US2] Add prev/next day navigation to DayDetail screen in `internal/ui/picker.go` — handle `←`/`p` and `→`/`n` key events; find the adjacent day index in the `[]DaySummary` list; reload entries for the new date; stay on current day if at boundary (no error, no wrap)
- [X] T018 [US2] Update DayDetail help hint footer in `internal/ui/picker.go` to include `←/p prev day` and `→/n next day` key bindings alongside existing bindings
- [X] T019 [US2] Add navigation test cases in `cmd/daily_test.go` — verify that prev/next day correctly updates the selected date index and handles boundaries

**Checkpoint**: Day-to-day navigation works. Boundary behavior correct. Help hints updated. `go test ./...` passes.

---

## Phase 5: User Story 3 — Non-Interactive Daily Summary (Priority: P3)

**Goal**: Users get a grouped-by-day summary printed to stdout when using `--no-interactive`, `--json`, or piping. Supports date range filtering.

**Independent Test**: Run `diaryctl daily --no-interactive` with pre-populated entries, verify grouped plain-text output. Run with `--json`, verify JSON structure. Pipe to `cat`, verify non-interactive mode auto-detected. Use `--from`/`--to`, verify filtered output.

### Implementation for User Story 3

- [X] T020 [US3] Implement non-interactive plain-text output path in `cmd/daily.go` — call `store.ListDays` with date range options, then for each day call `store.List` with that date to get entries; format using `ui.FormatDailySummary` writing to a buffer; pipe through `ui.OutputOrPage`; handle empty result with `"No diary entries found."`
- [X] T021 [US3] Implement non-interactive JSON output path in `cmd/daily.go` — build `[]ui.DayGroupJSON` from days + entries, call `ui.FormatJSON` to write to stdout; handle empty result as `[]`
- [X] T022 [US3] Implement TTY auto-detection in `cmd/daily.go` — use `term.IsTerminal(os.Stdout.Fd())` to fall back to non-interactive when stdout is not a TTY; ensure `--json` implies non-interactive
- [X] T023 [US3] Add non-interactive output tests in `cmd/daily_test.go` — test plain-text grouped format, test JSON structure matches `DayGroupJSON` schema, test empty output, test date range filtering, test TTY fallback logic

**Checkpoint**: Non-interactive output works for plain text and JSON. TTY auto-detection works. Date range filtering works. `go test ./...` passes.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, performance, and final validation.

- [X] T024 [P] Handle edge case: large number of entries per day (50+) — verify `bubbles/list` scrolling handles this; verify non-interactive mode prints all entries; add test case in `cmd/daily_test.go`
- [X] T025 [P] Handle edge case: hundreds of days — verify `bubbles/list` date list scrolling is responsive; no performance regression
- [X] T026 [P] Handle edge case: date range with no entries — verify both interactive (empty message + exit) and non-interactive modes handle gracefully; add test case
- [X] T027 Run `go vet ./...` and `gofmt` to ensure quality standards per constitution
- [X] T028 Run quickstart.md validation — manually test all commands from `specs/003-daily-view-picker/quickstart.md` and verify expected output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No-op — existing project
- **Phase 2 (Foundational)**: BLOCKS all user stories — storage interface + implementations + contract tests
- **Phase 3 (US1)**: Depends on Phase 2 — interactive picker core
- **Phase 4 (US2)**: Depends on Phase 3 — adds day navigation to existing picker
- **Phase 5 (US3)**: Depends on Phase 2 only — non-interactive path is independent of interactive picker
- **Phase 6 (Polish)**: Depends on Phases 3, 4, 5

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2. No dependencies on other stories.
- **US2 (P2)**: Depends on US1 (extends the DayDetail screen built in US1).
- **US3 (P3)**: Can start after Phase 2. Independent of US1/US2 (non-interactive path only).

### Within Each Phase

- T001 must complete before T002–T005 (interface definition before implementation)
- T002–T005 are all parallelizable (different files)
- T006–T007 can run after T002–T005 (contract tests need implementations)
- T008 is independent of storage tasks (UI layer only)
- T009 must complete before T010–T012 (scaffold before screens)
- T010–T012 are sequential (each screen builds on prior state transitions)
- T017–T018 are sequential (navigation logic before help hints)
- T020–T022 can be developed in sequence within daily.go

### Parallel Opportunities

- **Phase 2**: T002+T003 in parallel (markdown + sqlite ListDays); T004+T005 in parallel (markdown + sqlite date range); T008 in parallel with all storage tasks
- **Phase 3 + Phase 5**: US1 (interactive) and US3 (non-interactive) can run in parallel after Phase 2 since they touch different code paths
- **Phase 6**: T024+T025+T026 all in parallel (independent edge cases)

---

## Parallel Example: Phase 2 (Foundational)

```bash
# After T001 (interface) completes, launch in parallel:
Task: "Implement ListDays for markdown backend in internal/storage/markdown/markdown.go"
Task: "Implement ListDays for sqlite backend in internal/storage/sqlite/sqlite.go"
Task: "Extend List in markdown backend for date range in internal/storage/markdown/markdown.go"
Task: "Extend List in sqlite backend for date range in internal/storage/sqlite/sqlite.go"
Task: "Add DayGroupJSON and FormatDailySummary to internal/ui/output.go"
```

## Parallel Example: US1 + US3

```bash
# After Phase 2 completes, these two story tracks can run in parallel:
# Track A (interactive):
Task: "Create Bubble Tea picker model scaffold in internal/ui/picker.go"
# Track B (non-interactive):
Task: "Implement non-interactive plain-text output in cmd/daily.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Foundational (storage interface + both backends + contract tests)
2. Complete Phase 3: User Story 1 (interactive picker with date list → day detail → entry detail)
3. **STOP and VALIDATE**: `diaryctl daily` launches, dates shown, entries browsable
4. This delivers the core value — day-over-day browsing in the terminal

### Incremental Delivery

1. Phase 2 → Foundational storage layer ready
2. Phase 3 (US1) → Interactive picker MVP → Validate
3. Phase 4 (US2) → Day-to-day navigation → Validate
4. Phase 5 (US3) → Non-interactive + JSON output → Validate
5. Phase 6 → Edge cases + polish → Final validation
6. Each phase adds value without breaking previous phases

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Constitution: Test-Alongside requires contract tests in Phase 2 and command tests per user story
- All Bubble Tea UI code goes in `internal/ui/picker.go`; all command logic in `cmd/daily.go`
- No new dependencies — uses existing `bubbles/list`, `bubbles/viewport`, `lipgloss`, `term`
- Commit after each task or logical group
- Stop at any checkpoint to validate independently
