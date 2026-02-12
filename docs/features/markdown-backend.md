# Markdown Backend

**Status:** Implemented

## Overview

The default storage backend for diaryctl. Stores entries as individual Markdown files with YAML frontmatter in a hierarchical directory structure.

## File Structure

```
~/.diaryctl/
├── entries/
│   ├── 2026/
│   │   ├── 01/
│   │   │   ├── 15-abc12345.md
│   │   │   └── 20-def67890.md
│   │   └── 02/
│   │       └── 01-ghi12345.md
├── templates/
│   ├── daily.md
│   └── standup.md
├── contexts.json
└── config.toml
```

## Entry File Format

```markdown
---
id: abc12345
created_at: 2026-01-15T09:00:00Z
updated_at: 2026-01-15T09:00:00Z
templates:
  - name: daily
    template_id: tpl1
contexts:
  - name: work
    context_id: ctx1
    source: manual
---

Entry content here...

Can include **markdown** formatting.
```

## Configuration

```toml
# ~/.diaryctl/config.toml
storage = "markdown"
data_dir = "~/.diaryctl"
```

## Implementation

- `internal/storage/markdown/` — Markdown storage implementation
- `internal/storage/markdown/markdown.go` — Core storage logic
- `internal/storage/markdown/markdown_v2.go` — V2 data model support

## Atomic Writes

Uses temp file + rename for atomic writes:

1. Write to temp file in same directory
2. Sync to disk
3. Rename to target filename

This ensures data integrity even on crashes.

## Advantages

- Human-readable files
- Version control friendly
- Easy to browse and edit manually
- No database dependencies
- Simple backup (just copy files)

## Limitations

- No full-text search index
- O(n) operations for listing/filtering
- File handle limits for large datasets

## Related Features

- [SQLite Backend](sqlite-backend.md) — Alternative database backend
- [Export/Import](export-import.md) — Migrate between backends

---

*Feature is implemented and stable.*
