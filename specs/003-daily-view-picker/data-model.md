# Data Model: Daily Aggregated View with Interactive Picker

## New Types

### DaySummary

Represents one calendar day's aggregation of diary entries. Returned by the new `ListDays` storage method.

```go
// In internal/storage/storage.go

// DaySummary represents an aggregated view of entries for a single calendar day.
type DaySummary struct {
    Date    time.Time // Calendar date (time part zeroed, local timezone)
    Count   int       // Number of entries on this day
    Preview string    // Content preview of the most recent entry
}
```

| Field   | Type        | Description                                              |
|---------|-------------|----------------------------------------------------------|
| Date    | `time.Time` | Calendar date with time zeroed, in local timezone        |
| Count   | `int`       | Total number of entries created on this date             |
| Preview | `string`    | First 80 chars of the newest entry's content (single line) |

### ListDaysOptions

Controls filtering for the `ListDays` operation.

```go
// In internal/storage/storage.go

// ListDaysOptions controls filtering for ListDays operations.
type ListDaysOptions struct {
    StartDate *time.Time // inclusive lower bound (nil = no lower bound)
    EndDate   *time.Time // inclusive upper bound (nil = no upper bound)
}
```

| Field     | Type         | Description                                   |
|-----------|--------------|-----------------------------------------------|
| StartDate | `*time.Time` | Inclusive start date; nil means unbounded      |
| EndDate   | `*time.Time` | Inclusive end date; nil means unbounded        |

## Modified Types

### ListOptions (existing)

Add date range fields alongside the existing single-date filter.

```go
// In internal/storage/storage.go (modified)

type ListOptions struct {
    Date      *time.Time // filter by single date (existing)
    StartDate *time.Time // inclusive lower bound (new)
    EndDate   *time.Time // inclusive upper bound (new)
    OrderBy   string     // "created_at" (default: desc)
    Limit     int        // 0 = no limit
    Offset    int        // pagination offset
}
```

**Validation**: `Date` and `StartDate`/`EndDate` are mutually exclusive. If `Date` is set, range fields are ignored.

## Interface Extension

```go
// In internal/storage/storage.go (modified)

type Storage interface {
    Create(e entry.Entry) error
    Get(id string) (entry.Entry, error)
    List(opts ListOptions) ([]entry.Entry, error)
    ListDays(opts ListDaysOptions) ([]DaySummary, error)  // NEW
    Update(id string, content string) (entry.Entry, error)
    Delete(id string) error
    Close() error
}
```

## Output Types (UI layer)

### DayGroupJSON

JSON output structure for non-interactive `--json` mode.

```go
// In internal/ui/output.go

// DayGroupJSON is the JSON representation of a daily aggregate.
type DayGroupJSON struct {
    Date    string         `json:"date"`    // "YYYY-MM-DD"
    Count   int            `json:"count"`
    Entries []EntrySummary `json:"entries"`
}
```

## Relationships

```
Storage.ListDays() → []DaySummary
    Used by: date list screen, non-interactive summary

Storage.List(opts with Date) → []entry.Entry
    Used by: day detail screen (fetch entries for selected date)

Storage.Get(id) → entry.Entry
    Used by: entry detail screen (fetch full entry)

DaySummary.Date → ListOptions.Date
    Selecting a DaySummary feeds its Date into List to fetch that day's entries
```

## State Transitions (Interactive Picker)

```
┌──────────────┐    Enter     ┌──────────────┐    Enter    ┌──────────────┐
│  Date List   │ ──────────→  │  Day Detail   │ ────────→  │ Entry Detail  │
│  (ListDays)  │              │ (List w/date) │            │   (Get by id) │
└──────────────┘  ← Esc/Back  └──────────────┘ ← Esc/Back └──────────────┘
                                  │       ↑
                          ←/p ────┘       └──── →/n
                         (prev day)      (next day)
```

States:
- **DateList**: Shows `[]DaySummary` in a scrollable list
- **DayDetail**: Shows `[]entry.Entry` for one date, with prev/next day navigation
- **EntryDetail**: Shows one `entry.Entry` in full, via viewport
