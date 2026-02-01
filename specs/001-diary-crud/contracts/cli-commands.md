# CLI Command Contracts: Basic Diary CRUD

**Feature**: 001-diary-crud
**Date**: 2026-01-31

## Command Structure

```text
diaryctl
├── create [content]           # Create a new entry
├── list [--date DATE]         # List entries
├── show <id>                  # Show a single entry
├── edit <id>                  # Update an entry (editor)
├── update <id> <content>      # Update an entry (inline)
└── delete <id> [--force]      # Delete an entry
```

Global flags (all commands):
- `--json` — Output in JSON format
- `--config <path>` — Config file path override
- `--storage <backend>` — Storage backend override (markdown|sqlite)
- `--help` — Show help with usage examples

## `diaryctl create`

Create a new diary entry.

**Usage**:
```text
diaryctl create [content...]
diaryctl create                    # opens editor
diaryctl create "Today was great"  # inline
echo "piped" | diaryctl create -   # stdin
```

**Arguments**:
- `content` (optional): Entry text. If omitted, opens editor.
  If `-`, reads from stdin.

**Output (human)**:
```text
Created entry a3kf9x2m (2026-01-31 14:30)
```

**Output (JSON)**:
```json
{
  "id": "a3kf9x2m",
  "content": "Today was great",
  "created_at": "2026-01-31T14:30:00Z",
  "updated_at": "2026-01-31T14:30:00Z"
}
```

**Exit codes**:
- `0`: Success
- `1`: Validation error (empty content)
- `2`: Storage error
- `3`: Editor error (not found, crashed, cancelled)

---

## `diaryctl list`

List diary entries with preview.

**Usage**:
```text
diaryctl list
diaryctl list --date 2026-01-31
diaryctl list --json
```

**Flags**:
- `--date DATE` (optional): Filter by date (YYYY-MM-DD)

**Output (human)** — auto-paged when TTY:
```text
a3kf9x2m  2026-01-31 14:30  Today was great and I went for a long walk...
b7np4q1w  2026-01-31 09:15  Morning thoughts about the project deadline...
x2mp9a3k  2026-01-30 22:00  Reflecting on the week. Lots of progress on...
```

**Output (JSON)**:
```json
[
  {
    "id": "a3kf9x2m",
    "preview": "Today was great and I went for a long walk...",
    "created_at": "2026-01-31T14:30:00Z",
    "updated_at": "2026-01-31T14:30:00Z"
  }
]
```

**Output (empty)**:
```text
No diary entries found.
```

**Exit codes**:
- `0`: Success (including empty results)
- `1`: Invalid date format
- `2`: Storage error

---

## `diaryctl show`

Display full entry content and metadata.

**Usage**:
```text
diaryctl show a3kf9x2m
diaryctl show a3kf9x2m --json
```

**Arguments**:
- `id` (required): Entry identifier

**Output (human)** — auto-paged when TTY:
```text
Entry: a3kf9x2m
Created: 2026-01-31 14:30
Modified: 2026-01-31 15:45

Today was great and I went for a long walk in the park.
The weather was perfect — crisp air, blue sky. I should
do this more often.
```

**Output (JSON)**:
```json
{
  "id": "a3kf9x2m",
  "content": "Today was great and I went for a long walk...",
  "created_at": "2026-01-31T14:30:00Z",
  "updated_at": "2026-01-31T15:45:00Z"
}
```

**Exit codes**:
- `0`: Success
- `1`: Entry not found
- `2`: Storage error

---

## `diaryctl edit`

Open an entry in the editor for modification.

**Usage**:
```text
diaryctl edit a3kf9x2m
```

**Arguments**:
- `id` (required): Entry identifier

**Behavior**: Opens the entry content in the configured editor.
On save and close, updates the entry. If content unchanged, skips
update and reports "No changes detected."

**Output (human)**:
```text
Updated entry a3kf9x2m (2026-01-31 15:45)
```
or
```text
No changes detected for entry a3kf9x2m.
```

**Exit codes**:
- `0`: Success (updated or no changes)
- `1`: Entry not found
- `2`: Storage error
- `3`: Editor error

---

## `diaryctl update`

Replace entry content inline (without editor).

**Usage**:
```text
diaryctl update a3kf9x2m "Updated content here"
echo "new content" | diaryctl update a3kf9x2m -
```

**Arguments**:
- `id` (required): Entry identifier
- `content` (required): New content text. If `-`, reads from stdin.

**Output (human)**:
```text
Updated entry a3kf9x2m (2026-01-31 15:45)
```

**Output (JSON)**:
```json
{
  "id": "a3kf9x2m",
  "content": "Updated content here",
  "created_at": "2026-01-31T14:30:00Z",
  "updated_at": "2026-01-31T15:45:00Z"
}
```

**Exit codes**:
- `0`: Success
- `1`: Entry not found or validation error
- `2`: Storage error

---

## `diaryctl delete`

Delete an entry permanently.

**Usage**:
```text
diaryctl delete a3kf9x2m
diaryctl delete a3kf9x2m --force
```

**Arguments**:
- `id` (required): Entry identifier

**Flags**:
- `--force`: Skip confirmation prompt

**Behavior (without --force)**:
```text
Entry: a3kf9x2m (2026-01-31 14:30)
Preview: Today was great and I went for a long walk...

Delete this entry? This cannot be undone. [y/N]:
```

**Output (human, after confirmation)**:
```text
Deleted entry a3kf9x2m.
```

**Output (JSON)**:
```json
{
  "id": "a3kf9x2m",
  "deleted": true
}
```

**Exit codes**:
- `0`: Success (deleted or user cancelled)
- `1`: Entry not found
- `2`: Storage error

---

## Storage Contract Tests

Each storage backend MUST pass these contract tests:

1. **Create**: Create an entry → Get by ID → content matches
2. **Create empty**: Create with empty content → returns validation error
3. **Create sets timestamps**: created_at and updated_at are set and equal
4. **Get not found**: Get non-existent ID → returns ErrNotFound
5. **List empty**: List with no entries → returns empty slice, no error
6. **List order**: Create 3 entries → List → reverse chronological order
7. **List date filter**: Create entries on 2 dates → filter → correct subset
8. **Update content**: Update → content changes, updated_at advances, created_at preserved
9. **Update not found**: Update non-existent ID → returns ErrNotFound
10. **Delete**: Delete → Get → returns ErrNotFound
11. **Delete not found**: Delete non-existent ID → returns ErrNotFound
12. **ID uniqueness**: Create 100 entries → all IDs distinct
