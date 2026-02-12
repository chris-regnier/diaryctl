# Full-Text Search

**Status:** Designed  
**Design Doc:** `docs/plans/2025-02-01-workflow-features-design.md` (Feature 5)

## Overview

Full-text search addresses the retrieval problem — finding diary entries after they've been written. This feature enables users to search across all entry content with filters for date ranges and templates.

## Proposed Interface

### CLI Commands

```bash
# Basic search
diaryctl search "API design"

# Date-filtered search
diaryctl search "blocker" --from 2025-01-01
diaryctl search "decision" --from 2025-01-01 --to 2025-01-31

# Template-filtered search
diaryctl search "blocker" --template standup

# Combined filters
diaryctl search "quarterly" --from 2025-01-01 --template work

# Scripting output modes
diaryctl search "decision" --id-only | xargs -I{} diaryctl show {}
diaryctl search "API" --json
```

### Output Format

Default (table):
```
ID        DATE        PREVIEW
abc12345  2025-01-15  Working on API design for the new...
xyz67890  2025-01-20  Finalized API contract with team...
```

## Storage Implementation

### Markdown Backend

Simple substring scan across all entry files:

```go
func (s *Storage) Search(opts SearchOptions) ([]entry.Entry, error) {
    // Walk entries directory
    // Read each file
    // Check if content contains opts.Query (case-insensitive)
    // Apply date/template filters
    // Return matches
}
```

**Complexity:** O(n) where n = total entries. Acceptable for moderate volumes (<10k entries).

### SQLite Backend

Use FTS5 (Full-Text Search) virtual table:

```sql
-- New virtual table
CREATE VIRTUAL TABLE entries_fts USING fts5(
    content,
    content_rowid=rowid
);

-- Index existing entries
INSERT INTO entries_fts(rowid, content) SELECT rowid, content FROM entries;

-- Search query
SELECT entries.* FROM entries_fts
JOIN entries ON entries.rowid = entries_fts.rowid
WHERE entries_fts MATCH 'API design'
ORDER BY rank;
```

**Benefits:**
- Efficient full-text indexing
- Ranking by relevance
- Prefix matching
- Tokenization

## Storage Interface Addition

```go
type SearchOptions struct {
    Query        string
    StartDate    *time.Time
    EndDate      *time.Time
    TemplateName string
    Limit        int
    Offset       int
}

// New method on Storage interface
Search(opts SearchOptions) ([]entry.Entry, error)
```

## TUI Integration

The TUI design document specifies `/` as the search/filter key:

```
┌─ Daily View ─────────────┐
│  2025-01-15 (3 entries)  │
│  2025-01-14 (1 entry)    │
│ >2025-01-13 (5 entries)  │
└──────────────────────────┘
/search: API ▌           ← live filter as you type
```

## Implementation Considerations

1. **Case Sensitivity:** Case-insensitive by default, optional `--case-sensitive` flag
2. **Pagination:** Use `--limit` and `--offset` for large result sets
3. **Ranking:** SQLite FTS5 provides relevance; Markdown backend returns chronological
4. **Special Characters:** Escape regex-special characters in substring search
5. **Performance:** Consider caching file list for Markdown backend

## Future Enhancements

- **Fuzzy matching** — Handle typos with Levenshtein distance
- **Regex search** — `--regex` flag for pattern matching
- **Search history** — Save and replay common searches
- **Saved searches** — Named queries for repeated use

## Related Features

- [Entry Linking](linking.md) — Cross-reference entries in search results
- [TUI Search](tui-search.md) — Interactive search/filter interface

---

*See design doc for original specification.*
