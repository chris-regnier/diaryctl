# Context-Aware Jot Target

**Date:** 2026-02-18
**Status:** Approved

## Problem

The TUI `j` (jot) key always appends timestamped bullets to the daily entry (oldest entry of the day). There is no way to jot into a specific non-daily entry from the TUI.

## Design

### Core Change: `resolveJotTarget()`

Introduce a `resolveJotTarget()` method on `pickerModel` that returns the entry to append to based on the current screen and selection state:

| Screen | Selection State | Jot Target |
|--------|----------------|------------|
| `screenToday` | No list item selected (viewport focus) | Daily entry (existing behavior) |
| `screenToday` | List item selected in "other entries" | Selected entry |
| `screenDayDetail` | Entry selected in list | Selected entry |
| `screenEntryDetail` | Viewing a specific entry | That entry |
| `screenDateList` | No entry context | Jot disabled (existing behavior) |

The jot input textarea, timestamped bullet format (`- **HH:MM** text`), and the append-then-update flow all remain unchanged. Only the target entry changes.

### Jot Target Indicator

When the jot textarea opens, display a label above it showing the target:

```
Jotting into: "# 2026-02-18" ...        (daily)
Jotting into: "Meeting notes with..."   (non-daily)
```

Truncated first-line preview of the target entry's content. Disappears when jot closes.

### Edge Cases

- **No entries exist (screenToday, no selection):** Create a new daily entry as today (preserve existing behavior).
- **Target entry missing (all other cases):** Show error status message and cancel jot. This handles the rare case where an entry was deleted between render and jot.

### No Changes To

- CLI `jot` command (still targets the daily)
- Jot format (timestamped bullets)
- Template handling on new entry creation
- The `c` key (create) or `e` key (edit) flows
