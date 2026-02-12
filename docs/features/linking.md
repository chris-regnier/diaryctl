# Entry Linking

**Status:** Proposed

## Overview

Enable entries to reference each other using wiki-style `[[id]]` syntax. This creates a lightweight knowledge graph within your diary, allowing you to:

- Reference related entries without duplication
- Navigate between connected thoughts
- Discover emergent patterns through backlinks

## Proposed Interface

### Link Syntax

In entry content, use double brackets to reference another entry:

```markdown
# 2025-02-01

Had a great meeting with the team. We decided to go with
JWT over sessions. See [[abc12345]] for the initial research.

Also discussed the timeline — [[def67890]] has the project plan.
```

### Backlinks Command

```bash
# Find all entries that reference a given entry
diaryctl backlinks abc12345

# Output shows referencing entries with context snippets
diaryctl backlinks abc12345 --show-context
```

### Show Command Enhancement

```bash
# Render links as clickable/annotated references
diaryctl show abc12345 --render-links

# Output:
# Entry: abc12345
# Created: 2025-02-01 09:00
#
# Had a great meeting with the team. We decided to go with
# JWT over sessions. See [abc12345 → "JWT Research" 2025-01-28] for the initial research.
```

## Implementation

### Link Parsing

```go
var linkRegex = regexp.MustCompile(`\[\[([a-zA-Z0-9]+)\]\]`)

func ExtractLinks(content string) []string {
    matches := linkRegex.FindAllStringSubmatch(content, -1)
    var ids []string
    for _, m := range matches {
        if len(m) > 1 {
            ids = append(ids, m[1])
        }
    }
    return ids
}
```

### Backlink Indexing

Two implementation strategies:

**Option 1: On-Demand Scan**
```go
func (s *Storage) FindBacklinks(id string) ([]entry.Entry, error) {
    // Scan all entries
    // Check content for [[id]]
    // Return matching entries
}
```

**Option 2: Backlink Cache Table (SQLite)**
```sql
CREATE TABLE backlinks (
    source_id TEXT REFERENCES entries(id),
    target_id TEXT REFERENCES entries(id),
    PRIMARY KEY (source_id, target_id)
);

-- Update on entry create/update/delete
```

### TUI Integration

In the entry detail view:

```
┌─ Entry: abc12345 ───────────────┐
│                                  │
│  Had a great meeting...         │
│  See [[def67890]] for research. │
│                                  │
│  ── Linked References ──        │
│  ← Linked from: 2025-02-01      │
│  ← Linked from: 2025-02-05      │
│                                  │
│  l: view links  enter: follow   │
└──────────────────────────────────┘
```

## Design Decisions

### No Validation Required

Links are purely conventional — no requirement that the referenced entry exists. This allows:
- Referencing future entries
- Graceful handling of deleted entries
- Cross-diary references (with prefixes like `[[other.diary:abc123]]`)

### No Link Types

Unlike some wiki systems, links have no type/category. The context of the sentence provides the relationship semantics.

### Short IDs

Entry IDs are already short (8 characters), so the link syntax stays readable. No title-based linking to avoid ambiguity and renaming issues.

## Future Enhancements

- **Link types** — `[[abc123|type:related]]` for semantic relationships
- **Transclusion** — Embed linked entry content inline
- **Graph visualization** — Export to DOT/Graphviz for visualization
- **Daily backlink summary** — Show entries linking to today's entries

## Related Features

- [Search](search.md) — Find entries to link to
- [Block-Based Model](block-based-model.md) — Links could reference specific blocks

---

*This is a proposed feature. No design document exists yet.*
