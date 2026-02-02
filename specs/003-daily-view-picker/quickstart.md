# Quickstart: Daily Aggregated View with Interactive Picker

## Prerequisites

- Go 1.24+ installed
- Existing `diaryctl` binary built from the `003-daily-view-picker` branch
- Some diary entries already created (`diaryctl create "..."`)

## Usage

### Interactive Mode (default when in a terminal)

```bash
# Launch the interactive daily picker
diaryctl daily

# With date range filter
diaryctl daily --from 2026-01-01 --to 2026-01-31
```

**Keyboard controls** (displayed in the picker):
| Key | Action |
|-----|--------|
| `↑`/`↓` | Navigate date/entry list |
| `Enter` | Select date → view entries; select entry → view full |
| `←`/`p` | Previous day (from day detail) |
| `→`/`n` | Next day (from day detail) |
| `Esc`/`Backspace` | Go back to previous screen |
| `q`/`Ctrl+C` | Quit |

### Non-Interactive Mode

```bash
# Forced non-interactive (plain text)
diaryctl daily --no-interactive

# JSON output
diaryctl daily --no-interactive --json

# Piped output (auto-detects non-TTY)
diaryctl daily | head -20

# With date range
diaryctl daily --no-interactive --from 2026-01-15 --to 2026-01-31
```

### Output Examples

**Plain text** (`--no-interactive`):
```
── 2026-01-31 (3 entries) ──────────
  abc12345  15:04  Had a productive meeting about...
  def67890  14:30  Lunch at the new place downtown...
  ghi11111  09:15  Morning run felt great today...

── 2026-01-30 (1 entry) ───────────
  jkl22222  20:00  Quiet evening reading by the...
```

**JSON** (`--no-interactive --json`):
```json
[
  {
    "date": "2026-01-31",
    "count": 3,
    "entries": [
      {
        "id": "abc12345",
        "preview": "Had a productive meeting about...",
        "created_at": "2026-01-31T15:04:00Z",
        "updated_at": "2026-01-31T15:04:00Z"
      }
    ]
  }
]
```

## Development

### Build and test

```bash
go build -o diaryctl .
go test ./...
```

### Key files

| File | Purpose |
|------|---------|
| `cmd/daily.go` | Command definition, flag parsing, mode dispatch |
| `internal/ui/picker.go` | Bubble Tea interactive picker model |
| `internal/ui/output.go` | Non-interactive formatting + JSON types |
| `internal/storage/storage.go` | `ListDays` interface + `DaySummary` type |
| `internal/storage/contract_test.go` | Contract tests for `ListDays` |
