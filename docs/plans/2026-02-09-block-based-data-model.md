# Block-Based Data Model Design

**Date:** 2026-02-09
**Status:** Approved

## Overview

Simplify the data model from flat entries to a day-centric, block-based structure inspired by Notion and Coda. Days become the primary organizing unit (a "canvas"), and blocks are atomic content units ordered immutably by timestamp.

## Core Principles

- **Day as canvas**: Nothing exists without a day. Days are created implicitly when the first block is added.
- **Blocks as atoms**: Each block is a single piece of content with timestamp, content, and flat attributes.
- **Immutable ordering**: Blocks within a day are ordered by `CreatedAt` (never changes). `UpdatedAt` tracks modifications.
- **Flat attributes**: Key-value pairs on blocks enable filtering, searching, and rendering (like Jira labels).
- **Templates generate blocks**: Templates remain first-class entities that generate single atomic blocks.
- **Contexts become attributes**: No separate context entity - use attributes like `context: "work"`.

## Data Model

### Core Structures

```go
type Day struct {
    Date      time.Time    // Calendar date (normalized to midnight local time)
    Blocks    []Block      // Ordered by Block.CreatedAt
    CreatedAt time.Time    // When first block was added
    UpdatedAt time.Time    // When day was last modified
}

type Block struct {
    ID         string                 // 8-char nanoid
    Content    string                 // The actual content
    CreatedAt  time.Time             // Immutable - determines position
    UpdatedAt  time.Time             // Tracks modifications
    Attributes map[string]string     // Flat key-value pairs
}

type Template struct {
    ID         string
    Name       string
    Content    string              // Go template syntax
    Attributes map[string]string   // Default attributes for generated blocks
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### Key Changes from Current Model

| Current | New | Notes |
|---------|-----|-------|
| `Entry` is primary entity | `Day` is primary entity | Entries become Blocks within Days |
| Flat entry list | Blocks grouped by day | Days created implicitly |
| `Entry.Templates []TemplateRef` | `Block.Attributes["template"]` | Templates still first-class, but reference is just an attribute |
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

    // Template operations (unchanged - templates remain first-class)
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
    Attributes   map[string]string  // Filter by key-value attributes (AND logic)
    ContentQuery string             // Full-text search
    Limit        int
    Offset       int
}

type BlockResult struct {
    Block Block
    Day   time.Time  // Which day this block belongs to
}
```

### Storage File Structure

**Markdown backend:**
```
Current: ~/.diaryctl/entries/abc12345.md
New:     ~/.diaryctl/days/2026-02-09.md (all blocks for that day)
```

**SQLite backend:**
```
Current tables: entries, templates, contexts, entry_templates, entry_contexts
New tables:     days, blocks, templates
```

## Template Rendering

### Single Atomic Blocks

Each template application creates exactly one block. Users can apply the same template multiple times.

**Rendering process:**
1. User applies template → one block created with current timestamp
2. Template's `Content` is rendered using Go template syntax with variables
3. Template's `Attributes` are merged into the block's attributes
4. Block is added to the day's canvas

**Example:**
```
Template Name: "mood-check"
Content: "How am I feeling right now?\n\nMood: {{.Mood}}\nEnergy: {{.Energy}}"
Attributes: {
  "type": "reflection",
  "template": "mood-check"
}

Command: diaryctl edit --template mood-check --var Mood=great --var Energy=high

→ Creates one block with rendered content and merged attributes
```

### Template Variables

Templates support Go template syntax with user-provided variables:

```bash
diaryctl edit --template standup --var TEAM=backend --var SPRINT=23
diaryctl jot --template quick-note --var TOPIC=meeting
```

## Command Interface

### Core Block Creation

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

## Filtering and Search

### Attribute-Based Queries

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

### TUI Attribute Rendering

- Hidden by default in content view
- Can be displayed as "chips" below/above block content
- Toggle visibility with keybinding (e.g., `a` for attributes)
- Interactive filtering: toggle attribute filters shown as chips

## Migration Strategy

### Clean Break Approach

Since there's only one known user, make a clean break rather than complex migration:

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
- Implement for SQLite backend (new schema: `days`, `blocks` tables)
- Remove context-related methods and tables

### Phase 3: Template Rendering
- Templates remain unchanged structurally
- Add template variable support (Go template syntax)
- Render creates single block with merged attributes

### Phase 4: Commands
- Update `jot`, `edit`, `list`, `today`, `day` commands for blocks
- Add `--attr` flags for filtering
- Add `--var` flags for template variables
- Ensure bare `diaryctl` launches TUI

### Phase 5: TUI Updates
- Day picker shows days (not entries)
- Day detail shows ordered blocks with optional attribute chips
- Block filtering by attributes

## Future Considerations

### Block Versioning
The foundation is laid with `CreatedAt` + `UpdatedAt`. Future work:
- Track full edit history per block
- Version diffing and rollback
- Audit trail for content changes

### AI/LLM Integration
Attributes and blocks enable future integrations:
- MCP server integration for AI-generated blocks
- Macros that return blocks or block content
- Template generation from natural language
- Automated attribute extraction and tagging

### Export/Import
Block-based structure enables flexible export:
- Export day as markdown with block structure
- Import from other note-taking systems
- API for external integrations

## Design Decisions

### Why flat attributes instead of structured data?
- Simplicity: No schema to maintain
- Flexibility: Any attribute can be added at any time
- Proven pattern: Jira labels, GitHub labels, tags in many systems
- Sufficient for most use cases without over-engineering

### Why immutable block ordering?
- Diary entries are chronological by nature
- Edit history separate from creation time
- Clear mental model: when you wrote it vs when you changed it
- Prevents confusion from blocks "jumping around" on edit

### Why day as primary entity?
- Natural diary organization (people think in days)
- Canvas metaphor: day is the space where you add content
- Simplifies common queries (show me today, show me last week)
- Aligns with human time perception

### Why templates generate single blocks?
- Atomic units of content
- Composability: apply same template multiple times
- Simpler mental model than multi-block generation
- Easier to implement versioning per block in the future

### Why contexts become attributes instead of remaining entities?
- Reduces complexity: one fewer entity type
- Attributes provide same filtering/grouping capabilities
- More flexible: contexts can be ad-hoc, not predefined
- Templates remain first-class because they have their own content and lifecycle

## Success Criteria

This design succeeds if:
1. Creating and viewing diary entries becomes simpler and more intuitive
2. Filtering and searching by attributes provides sufficient flexibility
3. Template usage feels natural for both one-off and repeated use
4. The block structure enables future features (versioning, AI integration) without major refactoring
5. Storage backends can be implemented efficiently for the new model
