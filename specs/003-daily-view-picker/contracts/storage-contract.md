# Storage Contract: ListDays

## Method Signature

```go
ListDays(opts ListDaysOptions) ([]DaySummary, error)
```

## Input

```go
type ListDaysOptions struct {
    StartDate *time.Time // inclusive lower bound; nil = no lower bound
    EndDate   *time.Time // inclusive upper bound; nil = no upper bound
}
```

## Output

```go
type DaySummary struct {
    Date    time.Time // Calendar date (time zeroed, local timezone)
    Count   int       // Number of entries on this date
    Preview string    // Content preview of most recent entry (≤80 chars, single line)
}
```

Returns `[]DaySummary` sorted in **reverse chronological order** (newest date first).

## Contract Test Cases

### TC-01: ListDays empty store
- **Given**: No entries exist
- **When**: `ListDays(ListDaysOptions{})`
- **Then**: Returns empty slice, no error

### TC-02: ListDays single day
- **Given**: 3 entries on 2026-01-15
- **When**: `ListDays(ListDaysOptions{})`
- **Then**: Returns 1 DaySummary with Count=3, Date=2026-01-15, Preview from newest entry

### TC-03: ListDays multiple days
- **Given**: Entries on 2026-01-10, 2026-01-12, 2026-01-15
- **When**: `ListDays(ListDaysOptions{})`
- **Then**: Returns 3 DaySummary items in order: Jan 15, Jan 12, Jan 10

### TC-04: ListDays with StartDate
- **Given**: Entries on Jan 10, 12, 15
- **When**: `ListDays(ListDaysOptions{StartDate: Jan 12})`
- **Then**: Returns 2 items: Jan 15, Jan 12

### TC-05: ListDays with EndDate
- **Given**: Entries on Jan 10, 12, 15
- **When**: `ListDays(ListDaysOptions{EndDate: Jan 12})`
- **Then**: Returns 2 items: Jan 12, Jan 10

### TC-06: ListDays with both StartDate and EndDate
- **Given**: Entries on Jan 10, 12, 15
- **When**: `ListDays(ListDaysOptions{StartDate: Jan 11, EndDate: Jan 14})`
- **Then**: Returns 1 item: Jan 12

### TC-07: ListDays date range with no entries
- **Given**: Entries on Jan 10, Jan 15
- **When**: `ListDays(ListDaysOptions{StartDate: Jan 11, EndDate: Jan 14})`
- **Then**: Returns empty slice, no error

### TC-08: ListDays preview content
- **Given**: 2 entries on same day — earlier entry "First", later entry "Second entry with more text"
- **When**: `ListDays(ListDaysOptions{})`
- **Then**: Preview contains text from the **most recent** entry on that day

---

## Extended List Contract: Date Range Filtering

### Method Signature (existing, extended)

```go
List(opts ListOptions) ([]entry.Entry, error)
```

### Extended ListOptions

```go
type ListOptions struct {
    Date      *time.Time // existing: single date filter
    StartDate *time.Time // NEW: inclusive lower bound
    EndDate   *time.Time // NEW: inclusive upper bound
    OrderBy   string
    Limit     int
    Offset    int
}
```

### TC-09: List with StartDate only
- **Given**: Entries on Jan 10 (1 entry), Jan 15 (2 entries)
- **When**: `List(ListOptions{StartDate: Jan 12})`
- **Then**: Returns 2 entries from Jan 15

### TC-10: List with EndDate only
- **Given**: Entries on Jan 10, Jan 15
- **When**: `List(ListOptions{EndDate: Jan 12})`
- **Then**: Returns 1 entry from Jan 10

### TC-11: List with date range
- **Given**: Entries on Jan 10, Jan 12, Jan 15
- **When**: `List(ListOptions{StartDate: Jan 11, EndDate: Jan 13})`
- **Then**: Returns entry from Jan 12

### TC-12: Date takes precedence over range
- **Given**: Entries on Jan 10, Jan 12, Jan 15
- **When**: `List(ListOptions{Date: Jan 12, StartDate: Jan 10, EndDate: Jan 15})`
- **Then**: Returns only entries from Jan 12 (Date filter wins)
