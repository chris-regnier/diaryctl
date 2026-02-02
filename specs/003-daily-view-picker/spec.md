# Feature Specification: Daily Aggregated View with Interactive Picker

**Feature Branch**: `003-daily-view-picker`
**Created**: 2026-01-31
**Status**: Draft
**Input**: User description: "View entries day over day in aggregated view. Expose an interactive selector / picker for fast viewing and navigation."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse Entries by Day (Priority: P1)

A user wants to review their diary entries organized by day. They
launch an interactive daily view that shows a list of dates (with
entry counts or previews), and they can select a date to see all
entries for that day. This gives a calendar-like browsing experience
directly in the terminal.

**Why this priority**: The core value of this feature is day-over-day
browsing. Without it, users must manually filter by date or scroll
through a flat chronological list, which is tedious for long-term
diary use.

**Independent Test**: Can be tested by pre-populating entries across
multiple days, launching the daily view, verifying dates are listed
with correct entry counts, selecting a date, and confirming all
entries for that day are displayed.

**Acceptance Scenarios**:

1. **Given** entries exist across multiple days, **When** the user
   launches the daily view command, **Then** an interactive list of
   dates is displayed in reverse chronological order, each showing
   the date and the number of entries for that day.
2. **Given** the interactive date list is displayed, **When** the
   user selects a date using arrow keys and presses Enter, **Then**
   all entries for that date are displayed with their timestamps
   and content previews.
3. **Given** the user is viewing entries for a selected date,
   **When** they select a specific entry, **Then** the full entry
   content and metadata are displayed.
4. **Given** no entries exist, **When** the user launches the daily
   view, **Then** a helpful message indicates no entries are found.

---

### User Story 2 - Navigate Between Days (Priority: P2)

A user is viewing entries for a particular day and wants to quickly
move to the next or previous day without returning to the date list.
The interactive picker supports day-to-day navigation so the user
can flip through days fluidly.

**Why this priority**: Seamless navigation between days is what
makes the interactive view genuinely useful compared to repeated
list-and-filter commands. Without it, the user must back out and
re-select each time.

**Independent Test**: Can be tested by selecting a date, then using
navigation controls to move to adjacent days and verifying the
displayed entries update correctly.

**Acceptance Scenarios**:

1. **Given** the user is viewing entries for a specific date and
   entries exist on an earlier date, **When** they press the
   "previous day" navigation key, **Then** the view updates to
   show entries from the nearest earlier date that has entries.
2. **Given** the user is viewing entries for a specific date and
   entries exist on a later date, **When** they press the "next
   day" navigation key, **Then** the view updates to show entries
   from the nearest later date that has entries.
3. **Given** the user is viewing the earliest date with entries,
   **When** they press the "previous day" key, **Then** the view
   indicates there are no earlier entries (no error, stays on
   current day).
4. **Given** the user is viewing entries for a day, **When** they
   press the "back" key, **Then** they return to the date list
   picker.

---

### User Story 3 - Non-Interactive Daily Summary (Priority: P3)

A user wants a quick, non-interactive summary of entries grouped by
day — suitable for piping to other tools or scripting. This is the
non-interactive counterpart to the picker, outputting the same
day-over-day aggregation to stdout.

**Why this priority**: The interactive picker serves hands-on
browsing, but the CLI-first constitution requires that functionality
also be accessible non-interactively for scripting and automation.

**Independent Test**: Can be tested by running the command with a
non-interactive flag and verifying structured output (human-readable
or JSON) is written to stdout without launching any interactive UI.

**Acceptance Scenarios**:

1. **Given** entries exist across multiple days, **When** the user
   runs the daily view command with a `--no-interactive` flag (or
   when stdout is not a terminal), **Then** a plain-text summary is
   printed showing each date, entry count, and content previews,
   grouped by day in reverse chronological order.
2. **Given** entries exist, **When** the user runs the command with
   `--json` and `--no-interactive`, **Then** the output is a JSON
   array of day objects, each containing the date, entry count, and
   an array of entry summaries.
3. **Given** a date range is specified, **When** the user runs the
   non-interactive daily view, **Then** only days within the
   specified range are included in the output.

---

### Edge Cases

- What happens when a day has a very large number of entries (e.g.,
  50+)? The entry list for that day MUST be scrollable or paginated
  within the interactive view. Non-interactive output MUST include
  all entries.
- What happens when the terminal window is too small to display the
  picker? The system MUST gracefully degrade — either adapt to the
  terminal size or display an error suggesting a minimum terminal
  size.
- What happens when the user pipes the output or redirects stdout?
  The system MUST automatically fall back to non-interactive mode
  when stdout is not a TTY. No interactive UI MUST be rendered.
- What happens when entries span hundreds of days? The date list
  MUST be scrollable and responsive, not rendering all dates at
  once if it would degrade performance.
- What happens when the user provides a date range that contains no
  entries? The system MUST display a message indicating no entries
  were found in the specified range.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an interactive terminal picker
  that lists dates with diary entries in reverse chronological
  order, showing the entry count per day.
- **FR-002**: System MUST allow the user to select a date from the
  picker to view all entries for that day.
- **FR-003**: System MUST allow the user to select an individual
  entry from the day view to see its full content.
- **FR-004**: System MUST support navigating to the next and
  previous day (that has entries) without returning to the date
  list.
- **FR-005**: System MUST support returning from the day detail view
  back to the date list picker.
- **FR-006**: System MUST support exiting the interactive view at
  any point via a quit key.
- **FR-007**: System MUST automatically use non-interactive mode
  when stdout is not a TTY.
- **FR-008**: System MUST support a `--no-interactive` flag to force
  non-interactive output.
- **FR-009**: System MUST support `--json` output in non-interactive
  mode, producing a structured JSON representation of the daily
  aggregation.
- **FR-010**: System MUST support an optional date range filter
  (start date and/or end date) to limit which days are shown.
- **FR-011**: The interactive picker MUST display keyboard shortcuts
  or a help hint so users can discover navigation controls.
- **FR-012**: System MUST show content previews (first line or first
  80 characters) for each entry in both the day view and
  non-interactive output.

### Key Entities

- **Daily Aggregate**: A group of diary entries sharing the same
  calendar date. Attributes: date, entry count, list of entry
  summaries (identifier, timestamp, content preview).

### Assumptions

- The interactive picker uses standard terminal input (arrow keys,
  Enter, q to quit). No mouse interaction is required.
- Navigation keys for next/previous day use standard conventions
  (e.g., left/right arrows or n/p keys). Exact bindings are an
  implementation decision, but MUST be documented in the help hint.
- Date range filter accepts ISO 8601 date format (YYYY-MM-DD),
  consistent with the diary-crud feature.
- The daily view operates on the same storage backend as other
  diaryctl commands — it reads entries through the pluggable storage
  interface.
- "Day" is defined by the user's local timezone. Entries are grouped
  by their creation date in local time.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can launch the interactive daily view, select a
  date, and see that day's entries within 2 seconds for a diary
  with up to 1,000 entries across 365 days.
- **SC-002**: Users can navigate between adjacent days in under
  0.5 seconds per navigation action.
- **SC-003**: Non-interactive output for 1,000 entries across 365
  days completes within 3 seconds.
- **SC-004**: 100% of non-TTY invocations produce non-interactive
  output without rendering any interactive UI elements.
- **SC-005**: Users can discover all navigation controls from the
  help hint displayed within the interactive view without consulting
  external documentation.
