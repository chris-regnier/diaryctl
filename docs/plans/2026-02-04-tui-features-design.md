# TUI Features Design

**Date:** 2026-02-04
**Status:** Approved

## Overview

Enhance the existing `diaryctl daily` picker into a full-featured TUI that becomes the default action when users run bare `diaryctl`. The TUI should feel like a journal — today-first, low-friction, with fewer than 4 keystrokes for any primary action.

## Approach

Progressive enhancement of the existing picker (`internal/ui/picker.go`). No new entry point command — the root `diaryctl` command launches the TUI when in a TTY, falls back to plain text output when piped.

## Screen Inventory

6 screen states, expanded from the current 3:

| Screen | Purpose | Entry point |
|--------|---------|-------------|
| **Today** (home) | Today's daily entry inline + other today entries | Default on launch |
| **Date List** (browse) | Calendar list of days, filterable | `b` from today |
| **Day Detail** | Entries for a selected day | `enter` from date list |
| **Entry View** | Full entry in scrollable viewport | `enter` from day/today list |
| **Context Panel** | Overlay for managing contexts | `x` from any screen |
| **Help Overlay** | Keybinding reference | `?` from any screen |

## Navigation Flow

```
                    diaryctl (bare)
                         │
                         ▼
                   ┌───────────┐
                   │   Today   │  home screen
                   │           │  j: jot, c: create, enter: edit daily
                   └─────┬─────┘
                         │ b
                         ▼
                   ┌───────────┐
              ┌───►│ Date List │  browse history
              │    │           │  /: filter, enter: select day
              │    └─────┬─────┘
              │          │ enter
              │          ▼
              │    ┌───────────┐
              │    │Day Detail │  entries for selected day
              │    │           │  ←/→: prev/next day, e/d on selected
              │    └─────┬─────┘
              │          │ enter
              │          ▼
              │    ┌───────────┐
              │    │Entry View │  full entry content (viewport)
              │    │           │  e: edit, d: delete
              │    └─────┬─────┘
              │          │
              └──────────┘  esc goes back one level

         x (from any screen) → Context Panel
         ? (from any screen) → Help Overlay
```

Navigation rules:
- `esc`/`backspace` always goes back one level
- `q`/`ctrl+c` always quits
- `?` shows help overlay from any screen
- `x` opens context panel from any screen
- `j` and `c` work from any screen (global actions)
- After editor returns (from `c`, `e`, `j`), land back on same screen with data refreshed

## Screen Designs

### Today Screen (Home)

Three zones:

**Header:** `Today — 2026-02-04` with entry count, styled bold.

**Content zone — two parts:**
- **Daily entry inline** — the first entry of the day (identified via `daily.GetOrCreateToday()` logic) rendered directly in a viewport. Jot bullets render naturally. If content exceeds ~60% of terminal height, it scrolls. `enter` opens `$EDITOR` on this entry.
- **Other entries list** — remaining entries for today shown as a `bubbles/list` below. Each shows ID, time, preview. `↑/↓` navigates, `enter` opens entry view (read-only).

**Focus model:** `tab` switches focus between daily entry viewport and entry list. `↑/↓` scrolls or navigates depending on focus.

**Empty state:**
```
Nothing yet today.

  j  jot a quick note
  c  create a new entry
```

**Footer:**
```
j jot  c create  e edit  b browse  x ctx  ? help
```

### Date List, Day Detail, Entry View

These are the existing 3 screens from the picker, enhanced with write action keybindings.

### Context Panel

Overlay triggered by `x`. Behavior depends on whether an entry is selected:

**With entry selected:**
```
┌─ Contexts for abc12345 ─────────────────────┐
│  ● feature/auth          (git)               │
│  ○ daily-reflection      (manual)            │
│  ○ work                  (manual)            │
│                                              │
│  ● = attached, ○ = available                 │
│──────────────────────────────────────────────│
│  enter toggle  n new  / filter  esc close    │
└──────────────────────────────────────────────┘
```

- `enter` toggles context attachment (attach/detach)
- `n` creates a new context and attaches it
- `/` filters the context list
- `esc` closes panel

**Without entry selected (from date list):**
- Browse all contexts
- `enter` filters the date list to entries with that context

### Help Overlay

Semi-transparent overlay showing all keybindings, grouped by category (Navigation, Actions). Dismissed with `?`, `esc`, or any key.

## Write Actions

All actions ≤3 keystrokes:

| Key | Action | Keystrokes | Implementation |
|-----|--------|------------|----------------|
| `j` | Jot | 2 (j, type, enter) | Inline `bubbles/textinput` at bottom. Calls `daily.GetOrCreateToday()` + `store.Update()` |
| `c` | Create | 1+editor | Suspends TUI via `tea.ExecProcess`, opens `$EDITOR` with template. Calls `store.Create()` |
| `e` | Edit | 1+editor | Suspends TUI, opens `$EDITOR` with entry content. Calls `store.Update()` |
| `d` | Delete | 2 (d, y) | Inline confirmation `Delete entry abc12345? [y/N]`. Calls `store.Delete()` |
| `enter` | Edit daily | 1+editor | On today screen daily entry only. Suspends TUI, opens `$EDITOR` |

After any write action, a `tea.Cmd` triggers data refresh for the current screen.

## Interface Changes

The `StorageProvider` interface expands to the full `storage.Storage` interface (or a larger subset) to support write operations:

```go
type StorageProvider interface {
    // Read (existing)
    ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error)
    List(opts storage.ListOptions) ([]entry.Entry, error)
    Get(id string) (entry.Entry, error)

    // Write (new)
    Create(e entry.Entry) error
    Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error)
    Delete(id string) error

    // Context (new)
    ListContexts() ([]storage.Context, error)
    CreateContext(c storage.Context) error
    AttachContext(entryID string, contextID string) error
    DetachContext(entryID string, contextID string) error
}
```

## Default `diaryctl` Behavior

The root command (`cmd/root.go`) gains a `RunE`:
- TTY detected → launch TUI (today screen)
- Non-TTY → print today's entry (same as `diaryctl today` non-interactive)

The existing `daily` command continues to work, launching the date list screen.

## Implementation Sequence

1. Expand `StorageProvider` interface and wire up full storage
2. Add `screenToday` — today screen with inline daily entry + entry list
3. Add jot action (inline text input)
4. Add create/edit actions (editor suspension)
5. Add delete action (inline confirmation)
6. Add context panel overlay
7. Add help overlay
8. Wire root command to launch TUI as default
9. Refresh logic after write actions
