# Storage Backends

diaryctl supports pluggable storage backends. Both backends implement the same `Storage` interface, so all CLI commands and TUI features work identically regardless of which you choose.

## Markdown (Default)

Entries are stored as individual `.md` files with YAML frontmatter.

### Directory Structure

```
~/.diaryctl/entries/
└── 2026/
    └── 02/
        └── 04/
            ├── a3kf9x2m.md
            └── b7yn4p1q.md
```

### File Format

```yaml
---
id: a3kf9x2m
created_at: 2026-02-04T10:30:00Z
updated_at: 2026-02-04T10:30:00Z
templates:
  - daily
contexts:
  - feature/auth
  - sprint:23
---

Entry content goes here...
```

### When to Use Markdown

- You want human-readable, portable files
- You version-control your diary with git
- You want to edit entries with external tools
- You don't need remote sync

### Configuration

```toml
storage = "markdown"
data_dir = "~/.diaryctl"
```

## SQLite / Turso

Entries are stored in a SQLite database, compatible with [Turso](https://turso.tech/) for remote sync.

### Database Location

```
~/.diaryctl/diaryctl.db
```

### Schema

The database uses the following tables:

- `entries` — diary entries (id, content, created_at, updated_at)
- `templates` — reusable templates (id, name, content)
- `entry_templates` — many-to-many relationship
- `contexts` — context definitions (id, name)
- `entry_contexts` — many-to-many relationship

WAL mode is enabled for concurrent read access.

### When to Use SQLite

- You want fast queries and filtering across many entries
- You need remote sync via Turso
- You prefer a single-file database over many small files

### Configuration

```toml
storage = "sqlite"
data_dir = "~/.diaryctl"
```

### Turso Remote Sync

The SQLite backend uses libSQL, which supports Turso's embedded replicas. This allows local-first operation with background sync to a remote database. Refer to the [Turso documentation](https://docs.turso.tech/) for setting up a remote database.

## Switching Backends

Change the `storage` setting in your config file or use the `--storage` flag:

```bash
# One-off override
diaryctl list --storage sqlite

# Permanent switch
# Edit ~/.diaryctl/config.toml:
# storage = "sqlite"
```

Note: switching backends does not migrate existing data. Each backend maintains its own data store independently.
