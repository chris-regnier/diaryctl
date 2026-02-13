# Entry Pinning

**Status:** Proposed

## Overview

Pin important entries to the top of the TUI day view for quick reference. Useful for keeping active tasks, reminders, or key decisions visible.

## Proposed Interface

```bash
# Pin an entry
diaryctl pin <entry-id>

# Unpin an entry
diaryctl unpin <entry-id>

# List pinned entries
diaryctl list --pinned
```

### TUI Integration

```
┌─ 2026-02-12 (5 entries) ──────┐
│                                 │
│  📌 abc12345  Sprint goals     │  ← pinned entries at top
│  📌 def67890  Team agreements  │
│  ─────────────────────────────  │
│  ghi12345  14:30  standup      │  ← regular entries below
│  jkl67890  09:00  morning      │
│  mno12345  08:00  jot          │
│                                 │
│  p: pin/unpin  enter: view     │
└─────────────────────────────────┘
```

## Core Concepts

- Pinned state stored as entry attribute or metadata
- Pins are per-entry, not per-day (a pinned entry shows on its day)
- TUI `p` keybinding to toggle pin on selected entry
- Pinned entries sort to top within their day, preserving relative order
- Pin icon configurable in theme/config

## Related Features

- [Block-Based Model](block-based-model.md) — Pin as block attribute
- [Custom Themes](custom-themes.md) — Pin icon styling

---

*Proposed February 2026.*
