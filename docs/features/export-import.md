# Export/Import

**Status:** Proposed

## Overview

Enable data portability between storage backends and external systems. Users should be able to:

- Backup diary data to a portable format
- Migrate between Markdown and SQLite backends
- Archive old entries
- Import from other journaling systems

## Proposed Interface

### Export

```bash
# Export all entries to JSON
diaryctl export --format json > diary-backup-2025.json

# Export to Markdown archive
diaryctl export --format markdown --output ./diary-export/

# Export specific date range
diaryctl export --from 2025-01-01 --to 2025-01-31 --format json

# Export entries matching search
diaryctl export --search "project-x" --format json

# Export specific template only
diaryctl export --template standup --format markdown
```

### Import

```bash
# Import from JSON
diaryctl import diary-backup-2025.json

# Import from Markdown directory
diaryctl import ./old-diary/ --format markdown

# Merge strategy for duplicates
diaryctl import backup.json --strategy merge  # skip existing, add new
diaryctl import backup.json --strategy overwrite  # replace existing
diaryctl import backup.json --strategy rename  # rename duplicates

# Dry run to preview changes
diaryctl import backup.json --dry-run
```

## Export Formats

### JSON Format

```json
{
  "version": "1.0",
  "exported_at": "2025-02-01T10:00:00Z",
  "storage_backend": "markdown",
  "entries": [
    {
      "id": "abc12345",
      "content": "Entry content here...",
      "created_at": "2025-01-15T09:00:00Z",
      "updated_at": "2025-01-15T09:00:00Z",
      "templates": [{"template_id": "tpl1", "name": "daily"}],
      "contexts": [{"context_id": "ctx1", "name": "work"}]
    }
  ],
  "templates": [
    {
      "id": "tpl1",
      "name": "daily",
      "content": "## Today\n\n## Tomorrow"
    }
  ],
  "contexts": [
    {"id": "ctx1", "name": "work", "source": "manual"}
  ]
}
```

### Markdown Format

Directory structure:
```
diary-export/
├── entries/
│   ├── 2025/
│   │   ├── 01/
│   │   │   ├── 15-abc12345.md
│   │   │   └── 20-def67890.md
│   │   └── 02/
│   │       └── 01-ghi12345.md
│   └── metadata.json
├── templates/
│   ├── daily.md
│   └── standup.md
└── contexts.json
```

Entry file format:
```markdown
---
id: abc12345
created_at: 2025-01-15T09:00:00Z
updated_at: 2025-01-15T09:00:00Z
templates:
  - daily
contexts:
  - work
---

Entry content here...
```

## Implementation

### Export Process

```go
func Export(opts ExportOptions) error {
    // 1. Query all entries (with filters if specified)
    // 2. Query templates and contexts
    // 3. Serialize to target format
    // 4. Write to output (file or stdout)
}
```

### Import Process

```go
func Import(opts ImportOptions) error {
    // 1. Parse input file
    // 2. Validate format version
    // 3. Check for duplicates
    // 4. Apply merge strategy
    // 5. Write to storage
}
```

### ID Collision Handling

Options for handling duplicate IDs during import:

| Strategy | Behavior |
|----------|----------|
| `skip` | Skip entries with existing IDs |
| `overwrite` | Replace existing entries |
| `rename` | Generate new IDs for duplicates |
| `merge` | Append content to existing entries |

## Use Cases

### Backup

```bash
# Weekly backup cron job
0 0 * * 0 diaryctl export --format json > ~/backups/diary-$(date +%Y%m%d).json
```

### Migration

```bash
# Markdown → SQLite migration
diaryctl --storage markdown export --format json > /tmp/diary.json
diaryctl --storage sqlite import /tmp/diary.json
```

### Archive

```bash
# Archive entries older than 1 year
diaryctl export --to 2024-01-01 --format json > ~/archive/diary-2023.json
diaryctl delete --before 2024-01-01  # with confirmation
```

### External Integration

```bash
# Import from Day One export
diaryctl import ~/Downloads/DayOneExport.json --format dayone

# Export for analysis in Python
diaryctl export --format json | python3 analyze-diary.py
```

## Future Enhancements

- **Incremental export** — Only export entries changed since last backup
- **Compression** — Gzip output automatically for large exports
- **Encryption** — Encrypt exports with password or GPG key
- **Cloud sync** — Direct export to S3, Dropbox, etc.
- **Import formats** — Day One, Jrnl, plain text, etc.

## Related Features

- [Block-Based Model](block-based-model.md) — Export format would change with new data model

---

*This is a proposed feature. No design document exists yet.*
