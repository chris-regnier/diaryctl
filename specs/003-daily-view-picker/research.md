# Research: Daily Aggregated View with Interactive Picker

## R1: Storage Layer — Efficient Date Aggregation

### Decision
Add a `ListDays(opts ListDaysOptions) ([]DaySummary, error)` method to the `Storage` interface, where `DaySummary` contains `Date`, `Count`, and `Preview` (first entry's preview). Extend `ListOptions` with `StartDate` and `EndDate` fields for date range filtering.

### Rationale
- The existing `List` method returns full entries — unsuitable for a date picker that just needs dates and counts.
- The SQLite backend can compute this efficiently with `GROUP BY date(created_at)`.
- The Markdown backend can compute it by scanning the `YYYY/MM/DD/` directory tree without reading file contents (count files per directory, read one for preview).
- Keeping `ListDays` on the interface ensures both backends are tested via contract tests.

### Alternatives Considered
1. **Client-side aggregation**: Call `List()` with no filter, group in memory. Rejected — loads all entry content into memory, O(n) where n = total entries. Violates the <2s performance goal for large diaries.
2. **Add a `GroupBy` option to `List`**: Overloads the existing method, complicates the return type. Rejected for clarity.
3. **Separate `DayStore` interface**: Unnecessary indirection; all backends can implement date aggregation. Rejected per Simplicity principle.

---

## R2: Interactive Picker — Bubble Tea Architecture

### Decision
Build a multi-screen Bubble Tea model in `internal/ui/picker.go` with three states:
1. **Date list**: Scrollable list of dates with entry counts (using `bubbles/list`)
2. **Day detail**: Scrollable list of entries for a selected date
3. **Entry detail**: Full entry view (using existing viewport pattern from pager.go)

Navigation uses `left`/`right` arrows (or `n`/`p`) for prev/next day, `Enter` to drill in, `Esc`/`Backspace` to go back, `q` to quit.

### Rationale
- `charmbracelet/bubbles/list` is already an indirect dependency (via bubbles v0.21.0) and provides built-in scrolling, filtering, and keyboard navigation.
- The three-state approach maps directly to the spec's user stories (S1: date list → day view → entry view; S2: day-to-day navigation).
- A single Bubble Tea program with state switching avoids the complexity of nested programs.

### Alternatives Considered
1. **Two separate programs** (date picker → entry viewer): Requires re-launching Bubble Tea, causes screen flicker, loses alt-screen context. Rejected.
2. **Custom list rendering** (no bubbles/list): Reinvents scrolling and keyboard handling. Rejected — bubbles/list is battle-tested and already available.
3. **`huh` forms library**: Designed for form input, not browsing/navigation. Rejected.

---

## R3: Non-Interactive Output Format

### Decision
Non-interactive output groups entries by day with headers:

```
── 2026-01-31 (3 entries) ──────────
  abc12345  15:04  First line of content...
  def67890  14:30  Another entry preview...
  ghi11111  09:15  Morning thoughts...

── 2026-01-30 (1 entry) ───────────
  jkl22222  20:00  Evening reflection...
```

JSON output uses an array of day objects:
```json
[
  {
    "date": "2026-01-31",
    "count": 3,
    "entries": [
      {"id": "abc12345", "preview": "...", "created_at": "...", "updated_at": "..."}
    ]
  }
]
```

### Rationale
- The grouped text format is scannable and matches common CLI tool patterns (e.g., `git log --date-order`).
- JSON output nests entries within day objects, matching the "daily aggregate" entity from the spec.
- Both formats include the same information: date, count, entry ID, timestamp, preview.

### Alternatives Considered
1. **Flat table** (no grouping): Indistinguishable from `diaryctl list`. Rejected — the feature's value is the grouped view.
2. **Markdown output**: Adds formatting complexity without clear benefit for CLI piping. Rejected.

---

## R4: TTY Detection and Mode Selection

### Decision
Use `golang.org/x/term.IsTerminal(os.Stdout.Fd())` (already used in `pager.go`) to detect TTY. Logic:
1. If `--no-interactive` flag is set → non-interactive
2. If stdout is not a TTY → non-interactive
3. Otherwise → interactive (Bubble Tea)

### Rationale
- Matches the existing pattern in `PageOutput()` which already checks `term.IsTerminal`.
- The `--no-interactive` flag provides explicit override per FR-008.
- No new dependencies needed.

### Alternatives Considered
1. **`TERM` environment variable check**: Unreliable and non-standard for this purpose. Rejected.
2. **Always interactive unless piped**: Doesn't allow explicit non-interactive mode. Rejected per FR-008.

---

## R5: Date Range Filtering

### Decision
Add `--from` and `--to` flags (both `YYYY-MM-DD` format) to the `daily` command. These map to new `StartDate *time.Time` and `EndDate *time.Time` fields on `ListOptions` (reused from existing `List` method) and a corresponding `ListDaysOptions` struct.

### Rationale
- FR-010 requires date range filtering for both interactive and non-interactive modes.
- ISO 8601 date format is consistent with the existing `--date` flag on `list`.
- Both fields are optional — omitting `--from` means "no lower bound", omitting `--to` means "no upper bound".

### Alternatives Considered
1. **Single `--range` flag** (e.g., `--range 2026-01-01..2026-01-31`): Non-standard syntax, harder to parse, less composable. Rejected.
2. **Reuse `--date` for single-day filter**: Conflates single-day and range filtering. Rejected — separate flags are clearer.
