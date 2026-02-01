# Feature Specification: Basic Diary CRUD

**Feature Branch**: `001-diary-crud`
**Created**: 2026-01-31
**Status**: Draft
**Input**: User description: "Create, read, update, and delete diary entries via CLI"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a Diary Entry (Priority: P1)

A user wants to capture a thought, event, or reflection by creating a
new diary entry. They run a command, provide the entry content (either
inline or via their text editor), and the entry is persisted with a
timestamp. This is the foundational action — without it, no other
operation is meaningful.

**Why this priority**: Creating entries is the core value proposition.
Without the ability to write entries, the tool has no purpose.

**Independent Test**: Can be fully tested by running the create command,
confirming the entry is persisted, and verifying it can be retrieved.

**Acceptance Scenarios**:

1. **Given** no existing entries, **When** the user runs the create
   command with inline content, **Then** a new entry is saved with
   the provided content, the current date-time as timestamp, and a
   unique identifier. The CLI confirms creation with the entry ID.
2. **Given** no existing entries, **When** the user runs the create
   command without inline content, **Then** the user's configured
   text editor opens for entry composition. Upon saving and closing
   the editor, the entry is persisted.
3. **Given** an existing entry from today, **When** the user creates
   another entry, **Then** both entries coexist with distinct
   identifiers and timestamps.

---

### User Story 2 - Read Diary Entries (Priority: P2)

A user wants to review their past entries. They can list all entries
(with summary/preview), view a single entry by its identifier, or
list entries for a specific date. Reading entries is the primary way
users derive value from the diary over time.

**Why this priority**: Reading is the complement to writing. Users
need to review their entries to get value from the tool.

**Independent Test**: Can be tested by pre-populating entries and
verifying list and detail views produce correct output.

**Acceptance Scenarios**:

1. **Given** multiple diary entries exist, **When** the user runs
   the list command, **Then** entries are displayed in reverse
   chronological order showing date, time, and a content preview
   (first 80 characters or first line). If output exceeds the
   terminal height, it is piped through the system pager.
2. **Given** an entry with a known identifier, **When** the user
   runs the show command with that identifier, **Then** the full
   entry content is displayed along with its metadata (created date,
   last modified date).
3. **Given** entries on multiple dates, **When** the user runs the
   list command with a date filter, **Then** only entries from that
   date are shown.
4. **Given** no entries exist, **When** the user runs the list
   command, **Then** a helpful message indicates no entries are
   found (not an error).

---

### User Story 3 - Update a Diary Entry (Priority: P3)

A user realizes they want to correct or expand an existing entry.
They specify the entry by identifier and provide new content or
open it in their text editor for modification.

**Why this priority**: Editing is important but less frequent than
creating or reading. Users can work around its absence by creating
new entries.

**Independent Test**: Can be tested by creating an entry, updating
it, and verifying the content and modified timestamp change while
the original created timestamp is preserved.

**Acceptance Scenarios**:

1. **Given** an entry exists with known content, **When** the user
   runs the update command with the entry identifier, **Then** the
   entry opens in the user's text editor pre-filled with existing
   content. Upon saving and closing, the entry is updated and the
   modified timestamp is set.
2. **Given** an entry exists, **When** the user runs the update
   command with inline replacement content, **Then** the entry
   content is replaced and the modified timestamp is updated.
3. **Given** a non-existent entry identifier, **When** the user
   runs the update command, **Then** a clear error message is
   shown indicating the entry was not found.

---

### User Story 4 - Delete a Diary Entry (Priority: P4)

A user wants to remove an entry they no longer wish to keep. They
specify the entry by identifier, confirm the deletion, and the
entry is permanently removed.

**Why this priority**: Deletion is the least frequent operation and
carries risk of data loss. It is important for completeness but the
tool is fully usable without it initially.

**Independent Test**: Can be tested by creating an entry, deleting
it, and verifying it no longer appears in listings.

**Acceptance Scenarios**:

1. **Given** an entry exists, **When** the user runs the delete
   command with the entry identifier, **Then** the system displays
   the entry content and asks for confirmation before deleting.
2. **Given** an entry exists, **When** the user confirms deletion,
   **Then** the entry is permanently removed and no longer appears
   in listings.
3. **Given** an entry exists, **When** the user declines the
   deletion confirmation, **Then** the entry is preserved unchanged.
4. **Given** an entry exists, **When** the user runs the delete
   command with a `--force` flag, **Then** the entry is deleted
   without confirmation.
5. **Given** a non-existent entry identifier, **When** the user
   runs the delete command, **Then** a clear error message is shown.

---

### Edge Cases

- What happens when the user's configured editor is not found or
  fails to launch? The system MUST display a clear error message
  suggesting how to configure the editor and MUST NOT create or
  modify the entry.
- What happens when storage is unavailable (e.g., disk full,
  permissions error)? The system MUST report the specific storage
  error and exit with a non-zero code.
- What happens when two processes attempt to write simultaneously?
  The system MUST NOT silently corrupt data. At minimum, the second
  write MUST fail with a clear error rather than overwriting.
- What happens when entry content is empty? The system MUST reject
  empty entries with a message indicating that content is required.
- What happens when the date filter format is invalid? The system
  MUST show an error with the expected date format.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create diary entries with
  text content, automatically assigning a unique identifier and
  timestamp.
- **FR-002**: System MUST support entry creation via inline text
  argument or by launching an external text editor.
- **FR-003**: System MUST list diary entries in reverse chronological
  order with date, time, and a content preview.
- **FR-004**: System MUST display the full content and metadata of a
  single entry when given its identifier.
- **FR-005**: System MUST support filtering the entry list by date.
- **FR-006**: System MUST allow users to update an existing entry's
  content, updating the modified timestamp while preserving the
  original creation timestamp.
- **FR-007**: System MUST allow users to delete an entry by
  identifier, requiring explicit confirmation unless `--force` is
  provided.
- **FR-008**: System MUST support both human-readable (default) and
  JSON output formats for all commands.
- **FR-009**: System MUST persist entries through the pluggable
  storage interface defined in the project constitution.
- **FR-010**: System MUST return meaningful exit codes: 0 for
  success, non-zero for failures.
- **FR-011**: System MUST reject empty entry content with a clear
  error message.
- **FR-012**: System MUST auto-page list and show output through the
  system pager when stdout is a TTY and output exceeds the terminal
  height. Pager is determined by `PAGER` environment variable
  (defaulting to `less`). When stdout is not a TTY, output MUST be
  written directly without paging.

### Key Entities

- **Entry**: A single diary entry. Attributes: unique identifier
  (short hash, globally unique for multi-device compatibility),
  content (text body), created timestamp, modified timestamp.

### Assumptions

- The editor is determined by the `EDITOR` environment variable
  (falling back to `VISUAL`, then a sensible platform default).
- Entry identifiers are short hash strings (e.g., nanoid-style such
  as `a3kf9`), optionally derived from the creation date. They MUST
  be globally unique to support future multi-device sync scenarios.
  Identifiers MUST be human-typeable (short alphanumeric, no special
  characters).
- Date filter accepts ISO 8601 date format (YYYY-MM-DD).
- Content preview in list view shows the first line of the entry,
  truncated to 80 characters if necessary.
- Default storage location is `~/.diaryctl/` (respecting
  `XDG_DATA_HOME` if set, e.g., `$XDG_DATA_HOME/diaryctl/`).
  The directory is created automatically on first use.
- There is no multi-user or authentication concern — diaryctl
  operates on the current user's data only.

## Clarifications

### Session 2026-01-31

- Q: What format should entry identifiers use? → A: Short hash/nanoid (e.g., `a3kf9`), optionally date-derived, globally unique to support future multi-device sync.
- Q: Where should diary data be stored by default? → A: Dedicated directory `~/.diaryctl/` (or XDG_DATA_HOME equivalent).
- Q: How should the list command handle large output? → A: Auto-page through system pager (like `git log`) when output exceeds terminal height.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a new diary entry in under 10 seconds
  (excluding editor time) from command invocation to confirmation.
- **SC-002**: Users can list 1,000 entries and see results within
  2 seconds.
- **SC-003**: All four CRUD operations (create, read, update, delete)
  are completable in a single command invocation each.
- **SC-004**: 100% of CRUD operations produce correct JSON output
  when the JSON flag is provided.
- **SC-005**: No data loss occurs during normal create, update, or
  delete operations — entries are either fully persisted or the
  operation fails with a clear error.
