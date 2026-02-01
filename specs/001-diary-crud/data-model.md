# Data Model: Basic Diary CRUD

**Feature**: 001-diary-crud
**Date**: 2026-01-31

## Entities

### Entry

The core entity. Represents a single diary entry.

| Field      | Type      | Constraints                          |
|------------|-----------|--------------------------------------|
| id         | string    | Primary key. Nanoid, 8 chars,        |
|            |           | lowercase alphanumeric. Globally     |
|            |           | unique. Immutable after creation.    |
| content    | string    | Required. Non-empty text body.       |
|            |           | No maximum length enforced.          |
| created_at | timestamp | Required. Set on creation. UTC.      |
|            |           | Immutable after creation.            |
| updated_at | timestamp | Required. Set on creation, updated   |
|            |           | on every modification. UTC.          |

**Identity**: `id` is the unique identifier across all backends and
devices. Generated via nanoid with alphabet `abcdefghijklmnopqrstuvwxyz0123456789` and length 8.

**Timestamps**: Stored as UTC. Displayed in local timezone.

### Validation Rules

- `content` MUST NOT be empty (whitespace-only counts as empty).
- `id` MUST match pattern `^[a-z0-9]{8}$`.
- `created_at` MUST be <= `updated_at`.
- `created_at` MUST NOT be modified after initial creation.

## Storage Interface

```text
Storage interface {
    Create(entry Entry) error
    Get(id string) (Entry, error)
    List(opts ListOptions) ([]Entry, error)
    Update(id string, content string) (Entry, error)
    Delete(id string) error
}

ListOptions {
    Date       *time.Time  // filter by date (local tz)
    OrderBy    string      // "created_at" (default: desc)
    Limit      int         // 0 = no limit
    Offset     int         // pagination offset
}
```

**Error types**:
- `ErrNotFound`: entry with given ID does not exist
- `ErrConflict`: concurrent write detected (optimistic locking)
- `ErrStorage`: underlying storage failure (disk, DB)
- `ErrValidation`: content empty or ID malformed

## Markdown File Backend Layout

```text
~/.diaryctl/
├── config.toml
└── entries/
    └── 2026/
        └── 01/
            └── 31/
                ├── a3kf9x2m.md
                └── b7np4q1w.md
```

**File format** (each `.md` file):

```markdown
---
id: a3kf9x2m
created_at: 2026-01-31T14:30:00Z
updated_at: 2026-01-31T14:30:00Z
---

Entry content goes here. This is the diary text
the user wrote. It can be multi-line.
```

**Atomic writes**: Write to a temporary file in the same directory,
then rename (atomic on POSIX). This ensures no partial writes.

**Concurrent access**: Use file-level locking (`flock`) during write
operations. Read operations do not require locks.

## SQLite Backend (Turso/libSQL) Schema

```sql
CREATE TABLE IF NOT EXISTS entries (
    id         TEXT PRIMARY KEY,
    content    TEXT NOT NULL CHECK(length(trim(content)) > 0),
    created_at TEXT NOT NULL,  -- ISO 8601 UTC
    updated_at TEXT NOT NULL,  -- ISO 8601 UTC
    CHECK(created_at <= updated_at)
);

CREATE INDEX idx_entries_created_at
    ON entries(created_at DESC);

CREATE INDEX idx_entries_date
    ON entries(date(created_at));
```

**Atomic writes**: SQLite transactions provide atomicity.

**Concurrent access**: SQLite's built-in locking handles concurrent
access. WAL mode enabled for better read concurrency.

## State Transitions

Entry has a simple lifecycle:

```text
[Created] → [Updated]* → [Deleted]
```

- **Created**: Entry is persisted with id, content, created_at,
  updated_at (all set).
- **Updated**: content and updated_at change. created_at preserved.
  Can happen 0 or more times.
- **Deleted**: Entry is permanently removed. No soft-delete.
