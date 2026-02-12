# Recurring Entries

**Status:** Proposed

## Overview

Mark templates as "recurring" to automatically suggest or create entries at specific times. Useful for daily standups, weekly reviews, or habit tracking.

## Use Cases

- **Daily standup** — Prompt at 9 AM on weekdays
- **Weekly review** — Suggest Friday afternoon reflection
- **Habit tracking** — Remind to log morning mood
- **Meeting prep** — Auto-suggest template before recurring meetings

## Proposed Interface

### Template Configuration

```toml
# ~/.diaryctl/config.toml
[[recurring]]
template = "standup"
cron = "0 9 * * 1-5"  # 9 AM, Monday-Friday
action = "suggest"     # or "create"

[[recurring]]
template = "weekly-review"
cron = "0 17 * * 5"    # 5 PM Friday
action = "suggest"

[[recurring]]
template = "morning-mood"
cron = "0 8 * * *"     # 8 AM daily
action = "create"      # Auto-create with empty content
```

### Cron Expression Format

Standard cron syntax:

```
* * * * *
│ │ │ │ └─── Day of week (0-7, 0/7 = Sunday)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

Special strings:
- `@daily` — Once per day at midnight
- `@weekly` — Once per week (Sunday midnight)
- `@hourly` — Once per hour

### CLI Commands

```bash
# List configured recurring templates
diaryctl recurring list

# Manually trigger a recurring entry
diaryctl recurring run standup

# Skip today's recurring
diaryctl recurring skip morning-mood

# Check what would be created (dry run)
diaryctl recurring check
```

### Shell Hook Integration

The existing shell prompt hook can check for pending recurring entries:

```bash
$ diaryctl status
✓ 7🔥 morning  [standup pending]
```

## Implementation

### Data Model

```go
type Recurring struct {
    ID        string
    Template  string        // Template name
    Cron      string        // Cron expression
    Action    string        // "suggest" or "create"
    LastRun   *time.Time    // When last executed
    Enabled   bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Storage

```sql
-- SQLite
CREATE TABLE recurring (
    id TEXT PRIMARY KEY,
    template TEXT NOT NULL,
    cron TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT 'suggest',
    last_run TEXT,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

Markdown backend: Store in `~/.diaryctl/recurring.toml`.

### Scheduling

No background daemon — check at shell prompt time:

```go
func CheckRecurring(cfg Config) ([]Recurring, error) {
    var due []Recurring

    for _, r := range cfg.Recurring {
        if !r.Enabled {
            continue
        }

        next := cronexpr.MustParse(r.Cron).Next(r.LastRun)
        if next.Before(time.Now()) {
            due = append(due, r)
        }
    }

    return due, nil
}
```

### Auto-Create Behavior

For `action = "create"`:

```go
func AutoCreate(r Recurring, store Storage) error {
    // Check if entry already exists for this recurring today
    if ExistsToday(r.Template) {
        return nil
    }

    // Create entry with template
    entry := Entry{
        ID: NewID(),
        Content: "",  // Template content only
        Templates: []TemplateRef{{Name: r.Template}},
        CreatedAt: time.Now(),
    }

    return store.Create(entry)
}
```

### Suggest Behavior

For `action = "suggest"`, show in TUI and status:

```
┌─ Today ────────────────────────┐
│                                │
│  Nothing yet today.            │
│                                │
│  Pending recurring:            │
│    • standup (9 AM weekdays)   │
│                                │
│  r: run standup                │
│  j: jot  c: create             │
└────────────────────────────────┘
```

## TUI Integration

When launching `diaryctl`:

1. Check for pending recurring entries
2. If `action = "suggest"`, show in today screen with action key
3. If `action = "create"`, auto-create and refresh

## Design Decisions

### No Background Daemon

Keep it simple — use shell prompt hooks instead of a resident daemon. This avoids:
- Process management complexity
- Cross-platform issues
- Resource usage concerns

### Cron Over Natural Language

Use standard cron syntax rather than "every morning" or "weekdays":
- More precise
- Industry standard
- Extensive documentation available

### Per-Template, Not Per-Entry

Recurring is configured on templates, not entries:
- Templates define the structure
- Recurring defines when to apply it
- Separation of concerns

## Future Enhancements

- **Conditional recurring** — Only suggest if no entry matching criteria exists
- **Smart suggestions** — ML-based prediction of when user typically creates entries
- **Notification integration** — Desktop notifications for pending entries
- **Calendar sync** — Create entries based on calendar events

## Related Features

- [Guided Capture](guided-capture.md) — Recurring entries could use guided mode
- [Shell Integration](shell-integration.md) — Prompt hook integration

---

*This is a proposed feature. No design document exists yet.*
