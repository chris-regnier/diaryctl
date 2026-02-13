# diaryctl Feature Roadmap

This document tracks the current state of diaryctl features, organized by theme and implementation status. Use this as a guide for contribution opportunities and to understand the project's direction.

## Status Legend

| Status | Icon | Description |
|--------|------|-------------|
| Implemented | | Feature is complete and available |
| Designed | | Specification complete, ready for implementation |
| Proposed | | Idea documented, needs design/specification |

---

## Quick Reference

### Recently Implemented

- **Interactive TUI** — Bubble Tea-based interface with Today screen, date browser, and entry management
- **Shell Integration** — Prompt hooks, streak display, and Starship compatibility
- **MCP Server** — AI assistant integration via Model Context Protocol
- **Template System** — Custom templates with front-matter and variable substitution
- **Context Providers** — Automatic context from git branches and datetime

### High-Priority Next Steps

1. **Full-Text Search** — Documented in design docs but not implemented
2. **MCP Entry Creation** — `create_entry` and `list_templates` tools for AI assistants
3. **Custom Themes** — Config-driven TUI colors and markdown rendering presets
4. **Block-Based Data Model** — Complete design ready for implementation
5. **Entry Linking** — `[[id]]` syntax for bidirectional references
6. **Export/Import** — Data portability between storage backends

---

## Feature Categories

### 1. Data & Search

Features for organizing, finding, and connecting diary entries.

| Feature | Status | Notes |
|---------|--------|-------|
| [Full-Text Search](features/search.md) | Designed | Spec in `docs/plans/2025-02-01-workflow-features-design.md` |
| [Entry Linking](features/linking.md) | Proposed | `[[id]]` bidirectional reference syntax |
| [Export/Import](features/export-import.md) | Proposed | Cross-backend data portability |
| [Block-Based Model](features/block-based-model.md) | Designed | Day-centric atomic blocks architecture |
| Entry Statistics | Proposed | Word count, streaks, usage analytics |
| Recycle Bin/Undo | Proposed | Soft delete with restore capability |
| [Media Attachments](features/media-attachments.md) | Proposed | Attach images/files to entries |

### 2. TUI Enhancements

Improvements to the interactive terminal interface.

| Feature | Status | Notes |
|---------|--------|-------|
| [TUI Search/Filter](features/tui-search.md) | Designed | `/` key for live filtering per design doc |
| [Guided Capture](features/guided-capture.md) | Designed | Template-driven prompt flows (`--guided` flag) |
| [Custom Themes](features/custom-themes.md) | Designed | Config-driven TUI colors + markdown rendering presets |
| [Entry Pinning](features/entry-pinning.md) | Proposed | Pin important entries to top of day view |
| Reading Mode | Proposed | Distraction-free view with reading stats |
| Context Browser | Proposed | Browse entries by context (git branch, tags) |
| Draft Autosave | Proposed | Recovery from editor crashes |
| TUI Configuration | Proposed | Runtime theme/keybinding customization |

### 3. Automation & Intelligence

Features for reducing friction and adding smart behaviors.

| Feature | Status | Notes |
|---------|--------|-------|
| [Recurring Entries](features/recurring-entries.md) | Proposed | Auto-suggested templates at specific times |
| [Git Hook Integration](features/git-hooks.md) | Proposed | Auto-jot on commits |
| [MCP Entry Creation](features/mcp-entry-creation.md) | Designed | `create_entry` + `list_templates` MCP tools |
| [Weekly/Monthly Digest](features/weekly-digest.md) | Proposed | Summarize entries over configurable time periods |
| [Mood Sentiment](features/mood-sentiment.md) | Proposed | Auto-detect mood, enable mood-over-time queries |
| [Prompts of the Day](features/prompts-of-the-day.md) | Proposed | Rotating writing prompts in TUI and shell |
| [Streaks & Achievements](features/streaks-achievements.md) | Proposed | Gamification with configurable milestones |
| [Time Tracking Blocks](features/time-tracking.md) | Proposed | Start/stop timers, duration logging, reports |
| [Webhook Notifications](features/webhook-notifications.md) | Proposed | Fire webhooks on diary events for external integrations |
| Macro System | Proposed | User-defined command sequences |
| Smart Templates | Proposed | Template suggestions based on context |

### 4. Shell & Integration

Features for integrating diaryctl into the development environment.

| Feature | Status | Notes |
|---------|--------|-------|
| [Shell Integration](features/shell-integration.md) | Implemented | Bash/zsh prompt hooks, see `docs/plans/2026-02-02-shell-integration-design.md` |
| [Fish Shell Support](features/fish-shell.md) | Proposed | Extend init beyond bash/zsh |
| [Starship Preset](features/starship-preset.md) | Proposed | Official Starship configuration snippet |
| IDE Extensions | Proposed | VS Code, Vim plugins for entry creation |
| Calendar Integration | Proposed | Sync with external calendars |

### 5. Storage & Performance

Features for data storage, backends, and scalability.

| Feature | Status | Notes |
|---------|--------|-------|
| [Markdown Backend](features/markdown-backend.md) | Implemented | File-based storage with YAML frontmatter |
| [SQLite Backend](features/sqlite-backend.md) | Implemented | SQLite/Turso compatible database storage |
| PostgreSQL Backend | Proposed | Enterprise/multi-user scenarios |
| S3/Cloud Storage | Proposed | Remote backup and sync |
| [Sync Protocol](features/sync-protocol.md) | Proposed | Conflict-free sync between devices |
| Compression | Proposed | Automatic archival of old entries |
| Encryption | Proposed | At-rest encryption for sensitive entries |

---

## Architecture Evolution

### Current Model: Entry-Centric

The current implementation treats entries as independent entities:

```
Entry {
  ID, Content, CreatedAt, UpdatedAt
  Templates []TemplateRef
  Contexts  []ContextRef
}
```

### Proposed Model: Block-Based

A complete redesign documented in `docs/plans/2026-02-09-block-based-data-model.md`:

```
Day {
  Date, CreatedAt, UpdatedAt
  Blocks []Block
}

Block {
  ID, Content, CreatedAt, UpdatedAt
  Attributes map[string]string
}
```

**Benefits:**
- Days become the natural organizing unit
- Blocks are atomic and composable
- Attributes replace structured context/template references
- Immutable ordering (creation time) vs modification tracking
- Foundation for versioning and AI integration

**Migration Strategy:** Clean break (single known user, archive existing data).

---

## Design Documents Index

| Document | Date | Status | Description |
|----------|------|--------|-------------|
| `2025-02-01-jot-today-implementation.md` | 2025-02-01 | Implemented | Jot and today commands |
| `2025-02-01-templates-design.md` | 2025-02-01 | Implemented | Template system design |
| `2025-02-01-templates-implementation.md` | 2025-02-01 | Implemented | Template implementation plan |
| `2025-02-01-workflow-features-design.md` | 2025-02-01 | Partial | Search, guided capture, linking |
| `2026-02-02-context-providers-design.md` | 2026-02-02 | Implemented | Git/datetime context providers |
| `2026-02-02-context-providers-plan.md` | 2026-02-02 | Implemented | Implementation roadmap |
| `2026-02-02-shell-integration-design.md` | 2026-02-02 | Implemented | Shell prompt integration |
| `2026-02-04-tui-features-design.md` | 2026-02-04 | Partial | TUI screens and navigation |
| `2026-02-04-tui-features-plan.md` | 2026-02-04 | Partial | TUI implementation phases |
| `2026-02-09-block-based-data-model.md` | 2026-02-09 | Designed | Day/block architecture |
| `2026-02-11-tui-template-integration.md` | 2026-02-11 | Implemented | Template picker in TUI |

---

## Contributing to Features

### Picking Up a Designed Feature

1. Read the linked design document
2. Check for existing issues or PRs
3. Create a feature branch: `git checkout -b feature/search`
4. Follow the [Contributing Guide](../CONTRIBUTING.md)

### Proposing New Features

1. Open a GitHub Discussion to gauge interest
2. Create a design document in `docs/plans/` following existing patterns
3. Reference this roadmap in your proposal

### Design Document Template

New designs should include:

- **Overview** — What problem does this solve?
- **Usage Examples** — CLI commands and workflows
- **Behavior Specification** — Edge cases and error handling
- **Architecture** — New types, interfaces, file locations
- **Implementation Phases** — Breakdown of deliverables

See `docs/plans/2026-02-09-block-based-data-model.md` for a complete example.

---

## Success Metrics

This roadmap succeeds when:

1. Users can capture thoughts in <5 seconds (jot friction)
2. Users can find any entry in <10 seconds (retrieval)
3. The TUI feels like a natural journaling environment
4. Data is portable and never lost
5. The architecture enables future AI integrations

---

*Last updated: February 2026*
