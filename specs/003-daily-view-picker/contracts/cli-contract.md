# CLI Contract: `diaryctl daily`

## Command Definition

```
diaryctl daily [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--from` | string | `""` | Start date filter (YYYY-MM-DD, inclusive) |
| `--to` | string | `""` | End date filter (YYYY-MM-DD, inclusive) |
| `--no-interactive` | bool | `false` | Force non-interactive output |
| `--json` | bool | (global) | JSON output (implies non-interactive) |

### Mode Selection Logic

```
if --json is set          → non-interactive JSON
if --no-interactive       → non-interactive plain text
if stdout is not a TTY    → non-interactive plain text
else                      → interactive Bubble Tea picker
```

## Non-Interactive Output

### Plain Text (stdout)

```
── YYYY-MM-DD (N entries) ──────────
  <id>  HH:MM  <preview up to 80 chars>
  <id>  HH:MM  <preview up to 80 chars>

── YYYY-MM-DD (N entry) ───────────
  <id>  HH:MM  <preview up to 80 chars>
```

- Days in reverse chronological order
- Entries within each day in reverse chronological order
- Empty result: `"No diary entries found."`
- Singular "entry" when count is 1

### JSON Output (stdout)

```json
[
  {
    "date": "YYYY-MM-DD",
    "count": <int>,
    "entries": [
      {
        "id": "<string>",
        "preview": "<string, max 80 chars>",
        "created_at": "<RFC3339>",
        "updated_at": "<RFC3339>"
      }
    ]
  }
]
```

- Empty result: `[]`
- Days in reverse chronological order
- Entries within each day in reverse chronological order

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Invalid flag/argument (bad date format) |
| 2 | Storage error |

## Interactive Mode

### Screens

1. **Date List**: Scrollable list of `"YYYY-MM-DD  (N entries)  <preview>"` items
2. **Day Detail**: Scrollable list of entries for the selected date
3. **Entry Detail**: Full entry content in a viewport

### Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `↑`/`k` | Date list, Day detail | Move cursor up |
| `↓`/`j` | Date list, Day detail | Move cursor down |
| `Enter` | Date list | Open day detail |
| `Enter` | Day detail | Open entry detail |
| `←`/`p` | Day detail | Navigate to previous day (earlier) |
| `→`/`n` | Day detail | Navigate to next day (later) |
| `Esc`/`Backspace` | Day detail | Back to date list |
| `Esc`/`Backspace` | Entry detail | Back to day detail |
| `q`/`Ctrl+C` | Any | Quit |

### Help Hint

A footer line displays available key bindings for the current screen.

### Edge Cases

- **No entries**: Show "No diary entries found." and exit (no interactive UI)
- **Terminal too small**: Bubble Tea handles adaptive layout via `WindowSizeMsg`
- **Prev/next day at boundary**: Stay on current day (no error, no wrap)
