# SQLite Backend

**Status:** Implemented

## Overview

SQLite storage backend for diaryctl. Stores entries in a SQLite database with support for Turso remote sync.

## Configuration

```toml
# ~/.diaryctl/config.toml
storage = "sqlite"
data_dir = "~/.diaryctl"
```

## Database Schema

```sql
-- Current schema (entry-centric)
CREATE TABLE entries (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE contexts (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE entry_templates (
    entry_id TEXT REFERENCES entries(id),
    template_id TEXT REFERENCES templates(id),
    PRIMARY KEY (entry_id, template_id)
);

CREATE TABLE entry_contexts (
    entry_id TEXT REFERENCES entries(id),
    context_id TEXT REFERENCES contexts(id),
    PRIMARY KEY (entry_id, context_id)
);
```

## Turso Support

Uses `github.com/tursodatabase/go-libsql` for Turso compatibility:

```toml
# Turso configuration
storage = "sqlite"
database_url = "libsql://your-db.turso.io"
auth_token = "your-token"
```

## Implementation

- `internal/storage/sqlite/` — SQLite storage implementation
- `internal/storage/sqlite/sqlite.go` — Core storage logic

## FTS5 for Search

Future enhancement for full-text search:

```sql
-- Future FTS5 virtual table
CREATE VIRTUAL TABLE entries_fts USING fts5(
    content,
    content_rowid=rowid
);
```

## Advantages

- Fast queries with indexes
- Full-text search ready (FTS5)
- ACID transactions
- Remote sync via Turso
- Better performance for large datasets

## Limitations

- Binary format (not human-readable)
- Requires SQLite/library
- Backup requires export

## Related Features

- [Markdown Backend](markdown-backend.md) — Alternative file-based backend
- [Export/Import](export-import.md) — Migrate between backends
- [Search](search.md) — FTS5 full-text search

---

*Feature is implemented and stable.*
