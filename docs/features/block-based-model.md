# Block-Based Data Model

**Status:** Designed  
**Design Doc:** `docs/plans/2026-02-09-block-based-data-model.md`

## Overview

A fundamental redesign of the diaryctl data model, shifting from flat entries to a **day-centric, block-based structure**. Days become the primary organizing unit (a "canvas"), and blocks are atomic content units ordered immutably by timestamp.

This design enables:
- More natural diary organization (people think in days)
- Immutable content history
- Flexible attribute-based filtering
- Foundation for versioning and AI integration

## Core Concepts

### Day as Canvas

Nothing exists without a day. Days are created implicitly when the first block is added.

```go
type Day struct {
    Date      time.Time    // Calendar date (normalized to midnight local)
    Blocks    []Block      // Ordered by Block.CreatedAt
    CreatedAt time.Time    // When first block was added
    UpdatedAt time.Time    // When day was last modified
}
```

### Blocks as Atoms

Each block is a single piece of content with timestamp, content, and flat attributes.

```go
type Block struct {
    ID         string                 // 8-char nanoid
    Content    string                 // The actual content
    CreatedAt  time.Time             // Immutable - determines position
    UpdatedAt  time.Time             // Tracks modifications
    Attributes map[string]string     // Flat key-value pairs
}
```

### Templates Generate Blocks

Templates remain first-class entities that generate single atomic blocks.

```go
type Template struct {
    ID         string
    Name       string
    Content    string              // Go template syntax
    Attributes map[string]string   // Default attributes for generated blocks
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

## Key Changes from Current Model

| Current | New | Notes |
|---------|-----|-------|
| `Entry` is primary entity | `Day` is primary entity | Entries become Blocks within Days |
| Flat entry list | Blocks grouped by day | Days created implicitly |
| `Entry.Templates []TemplateRef` | `Block.Attributes["template"]` | Template reference is just an attribute |
| `Entry.Contexts []ContextRef` | `Block.Attributes["context"]` | Contexts removed as entities |
| Single timestamp per entry | `CreatedAt` + `UpdatedAt` | Position vs modification tracking |

## Storage Interface

```go
type Storage interface {
    // Day operations
    GetDay(date time.Time) (Day, error)           // Get or create day
    ListDays(opts ListDaysOptions) ([]DaySummary, error)
    DeleteDay(date time.Time) error

    // Block operations (scoped to a day)
    CreateBlock(date time.Time, block Block) error
    GetBlock(blockID string) (Block, time.Time, error)  // Returns block + its day
    UpdateBlock(blockID string, content string, attributes map[string]string) error
    DeleteBlock(blockID string) error
    ListBlocks(date time.Time) ([]Block, error)

    // Template operations (unchanged)
    CreateTemplate(t Template) error
    GetTemplate(id string) (Template, error)
    GetTemplateByName(name string) (Template, error)
    ListTemplates() ([]Template, error)
    UpdateTemplate(id string, name string, content string) (Template, error)
    DeleteTemplate(id string) error

    // Search across days
    SearchBlocks(opts SearchOptions) ([]BlockResult, error)

    Close() error
}

type SearchOptions struct {
    StartDate    *time.Time
    EndDate      *time.Time
    Attributes   map[string]string  // Filter by key-value (AND logic)
    ContentQuery string             // Full-text search
    Limit        int
    Offset       int
}

type BlockResult struct {
    Block Block
    Day   time.Time  // Which day this block belongs to
}
```

## Storage File Structure

### Markdown Backend

```
Current: ~/.diaryctl/entries/abc12345.md
New:     ~/.diaryctl/days/2026-02-09.md (all blocks for that day)
```

Day file format:
```markdown
---
date: 2026-02-09
created_at: 2026-02-09T08:00:00Z
updated_at: 2026-02-09T18:00:00Z
---

## Block: a1b2c3d4 (08:00)

type: morning-pages
template: morning-pages

Content of the morning pages block...

---

## Block: e5f6g7h8 (12:30)

type: jot

Quick lunch note...

---

## Block: i9j0k1l2 (17:00)

type: standup
template: standup
context: work

What I worked on today...
```

### SQLite Backend

```sql
-- Current tables: entries, templates, contexts, entry_templates, entry_contexts
-- New tables:

CREATE TABLE days (
    date TEXT PRIMARY KEY,  -- ISO 8601 date
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE blocks (
    id TEXT PRIMARY KEY,
    day_date TEXT NOT NULL REFERENCES days(date),
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE block_attributes (
    block_id TEXT NOT NULL REFERENCES blocks(id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (block_id, key)
);

-- Templates table unchanged (still first-class)
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Template default attributes
CREATE TABLE template_attributes (
    template_id TEXT NOT NULL REFERENCES templates(id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (template_id, key)
);
```

## Command Interface

### Block Creation

```bash
# Quick jot - creates block with content
diaryctl jot "Met with team, discussed Q2 goals"

# Long-form edit - opens editor with optional template
diaryctl edit                           # Blank block
diaryctl edit --template morning-pages  # Pre-filled from template
diaryctl edit --template standup --var TEAM=backend --var SPRINT=23

# Create block on specific day (default: today)
diaryctl jot "Yesterday's insight" --date 2026-02-08
```

### Day Viewing

```bash
# Default: launch TUI
diaryctl

# View today's blocks
diaryctl today

# View specific day
diaryctl day 2026-02-08

# Interactive day picker (TUI)
diaryctl daily
```

### Block Operations

```bash
# List blocks (with filtering)
diaryctl list                           # All blocks (recent first)
diaryctl list --date 2026-02-08        # Specific day
diaryctl list --attr type=reflection   # Filter by attributes

# Update/delete blocks
diaryctl update <block-id> --content "new content"
diaryctl update <block-id> --attr mood=great --attr energy=high
diaryctl delete <block-id>

# Search
diaryctl search "quarterly goals" --from 2026-01-01
diaryctl search --attr template=standup --last 30d
```

## Attribute-Based Filtering

Attributes enable flexible filtering:

```go
// Find all mood-check blocks in January
SearchBlocks(SearchOptions{
    StartDate: jan1,
    EndDate: jan31,
    Attributes: map[string]string{"template": "mood-check"},
})

// Find work-related reflections (AND logic)
SearchBlocks(SearchOptions{
    Attributes: map[string]string{
        "context": "work",
        "type": "reflection",
    },
})
```

## Migration Strategy

**Clean Break Approach:** Since there's only one known user, make a clean break rather than complex migration:

1. Archive existing data structure (optional export)
2. Implement new day/block model from scratch
3. Start fresh with new storage structure

No backwards compatibility needed.

## Implementation Phases

### Phase 1: Core Data Model
- Define `Day` and `Block` structs in new package
- Keep flat `Attributes map[string]string` on blocks
- Preserve `Template` as first-class entity

### Phase 2: Storage Layer
- Update `storage.Storage` interface for day/block operations
- Implement for markdown backend (one file per day)
- Implement for SQLite backend (new schema)
- Remove context-related methods and tables

### Phase 3: Template Rendering
- Templates remain unchanged structurally
- Add template variable support
- Render creates single block with merged attributes

### Phase 4: Commands
- Update `jot`, `edit`, `list`, `today`, `day` commands
- Add `--attr` flags for filtering
- Add `--var` flags for template variables
- Ensure bare `diaryctl` launches TUI

### Phase 5: TUI Updates
- Day picker shows days (not entries)
- Day detail shows ordered blocks with attribute chips
- Block filtering by attributes

## Design Decisions

### Why flat attributes instead of structured data?
- **Simplicity:** No schema to maintain
- **Flexibility:** Any attribute can be added at any time
- **Proven pattern:** Jira labels, GitHub labels work similarly
- **Sufficient:** Covers most use cases without over-engineering

### Why immutable block ordering?
- **Chronological:** Diary entries are naturally time-ordered
- **Clear model:** When you wrote it vs when you changed it
- **Prevents confusion:** Blocks don't "jump around" on edit

### Why day as primary entity?
- **Natural organization:** People think in days
- **Canvas metaphor:** Day is the space for content
- **Simpler queries:** "Show me today" is a common operation

### Why templates generate single blocks?
- **Atomic units:** Composable, versionable individually
- **Simpler mental model:** One template = one block
- **Future-proof:** Easier to implement per-block versioning

### Why contexts become attributes?
- **Reduced complexity:** One fewer entity type
- **Same capability:** Attributes provide filtering/grouping
- **More flexible:** Contexts can be ad-hoc, not predefined

## Success Criteria

This design succeeds if:
1. Creating and viewing diary entries becomes simpler
2. Filtering and searching by attributes is flexible enough
3. Template usage feels natural for both one-off and repeated use
4. The block structure enables future features (versioning, AI)
5. Storage backends can be implemented efficiently

## Related Features

- [Search](search.md) — Attribute-based search is central to the new model
- [Export/Import](export-import.md) — Would need to handle new format
- [Guided Capture](guided-capture.md) — Template-driven entry creation

---

*See `docs/plans/2026-02-09-block-based-data-model.md` for the complete design.*
