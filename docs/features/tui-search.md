# TUI Search and Filter

**Status:** Designed  
**Design Doc:** `docs/plans/2026-02-04-tui-features-design.md`

## Overview

Live search and filtering within the TUI, activated by the `/` key. This feature enables quick navigation through large diary histories without leaving the interactive interface.

## Current State

The TUI design document specifies search/filter functionality, but it is **not yet implemented**. Currently:

- The date list shows all days
- The day detail shows all entries for that day
- There is no way to filter within the TUI

## Proposed Interface

### Date List Filter

```
┌─ Daily View ─────────────┐
│                          │
│ /quarterly ▌            │  ← live filter as you type
│                          │
│  2025-01-15 (3 entries) │
│  2025-01-14 (1 entry)   │
│  2025-01-10 (2 entries) │  ← only days matching "quarterly"
│                          │
│  esc: clear  enter: open │
└──────────────────────────┘
```

**Filter criteria:**
- Date string (e.g., "2025-01")
- Content preview containing text
- Entry count

### Day Detail Filter

```
┌─ 2025-01-15 (3 entries) ─┐
│                          │
│ /standup ▌              │  ← filter entries on this day
│                          │
│  abc12345  09:00         │
│  def67890  14:00         │  ← only entries matching "standup"
│                          │
│  esc: clear  enter: view │
└──────────────────────────┘
```

**Filter criteria:**
- Entry content
- Template name
- Context name
- Entry ID

### Search Across All Entries

Global search from any screen:

```
┌─ Search ─────────────────┐
│                          │
│ Query: API design ▌     │
│                          │
│  abc12345  2025-01-15   │
│  def67890  2025-01-20   │
│  ghi12345  2025-02-01   │
│                          │
│  3 results found         │
│  esc: close  enter: view │
└──────────────────────────┘
```

## Key Bindings

| Key | Action |
|-----|--------|
| `/` | Start filtering on current screen |
| `esc` | Clear filter / close search |
| `enter` | Open selected result |
| `↑/↓` | Navigate results |
| `ctrl+f` | Global search (alternative to `/`) |

## Implementation

### Filter State

```go
type pickerModel struct {
    // ... existing fields ...

    // Filter state
    filterActive    bool
    filterInput     textinput.Model
    filterQuery     string
    filteredDays    []storage.DaySummary
    filteredEntries []entry.Entry
}
```

### Filter Logic

```go
func (m *pickerModel) applyFilter(query string) {
    query = strings.ToLower(query)

    switch m.screen {
    case screenDateList:
        m.filteredDays = filterDays(m.days, query)
        items := make([]list.Item, len(m.filteredDays))
        for i, d := range m.filteredDays {
            items[i] = dateItem{summary: d}
        }
        m.dateList.SetItems(items)

    case screenDayDetail:
        m.filteredEntries = filterEntries(m.dayEntries, query)
        items := make([]list.Item, len(m.filteredEntries))
        for i, e := range m.filteredEntries {
            items[i] = entryItem{entry: e}
        }
        m.dayList.SetItems(items)
    }
}

func filterDays(days []storage.DaySummary, query string) []storage.DaySummary {
    var result []storage.DaySummary
    for _, d := range days {
        // Match date string
        if strings.Contains(d.Date.Format("2006-01-02"), query) {
            result = append(result, d)
            continue
        }
        // Match preview content
        if strings.Contains(strings.ToLower(d.Preview), query) {
            result = append(result, d)
        }
    }
    return result
}

func filterEntries(entries []entry.Entry, query string) []entry.Entry {
    var result []entry.Entry
    for _, e := range entries {
        // Match content
        if strings.Contains(strings.ToLower(e.Content), query) {
            result = append(result, e)
            continue
        }
        // Match ID
        if strings.Contains(e.ID, query) {
            result = append(result, e)
            continue
        }
        // Match template names
        for _, t := range e.Templates {
            if strings.Contains(strings.ToLower(t.Name), query) {
                result = append(result, e)
                break
            }
        }
    }
    return result
}
```

### Update Handler

```go
func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... existing handlers ...

    case tea.KeyMsg:
        // Global filter key
        if msg.String() == "/" && !m.filterActive {
            m.filterActive = true
            m.filterInput = textinput.New()
            m.filterInput.Placeholder = "filter..."
            m.filterInput.Focus()
            return m, textinput.Blink
        }

        // Filter mode handling
        if m.filterActive {
            switch msg.String() {
            case "esc":
                m.filterActive = false
                m.filterQuery = ""
                m.clearFilter()
                return m, nil
            case "enter":
                m.filterActive = false
                return m, nil
            }

            // Update filter input and apply
            var cmd tea.Cmd
            m.filterInput, cmd = m.filterInput.Update(msg)
            m.filterQuery = m.filterInput.Value()
            m.applyFilter(m.filterQuery)
            return m, cmd
        }
    }
}
```

### View Rendering

```go
func (m pickerModel) View() string {
    // ... existing view logic ...

    if m.filterActive {
        filterBar := m.filterInput.View()
        return m.centerContent(result + "\n" + filterBar)
    }

    return m.centerContent(result)
}
```

## Performance Considerations

### Debouncing

For large datasets, debounce the filter application:

```go
func (m *pickerModel) applyFilterDebounced(query string) tea.Cmd {
    return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
        return filterMsg{query: query}
    })
}
```

### Caching

Cache the unfiltered lists to allow quick clearing:

```go
func (m *pickerModel) clearFilter() {
    // Restore original items
    items := make([]list.Item, len(m.days))
    for i, d := range m.days {
        items[i] = dateItem{summary: d}
    }
    m.dateList.SetItems(items)
}
```

## Future Enhancements

- **Fuzzy matching** — Use `github.com/sahilm/fuzzy` for typo-tolerant search
- **Saved filters** — Remember common searches
- **Advanced filters** — `template:standup context:work`
- **Highlight matches** — Show matched text in context

## Related Features

- [Search](search.md) — CLI search command with FTS5
- [Block-Based Model](block-based-model.md) — Filter by block attributes

---

*See `docs/plans/2026-02-04-tui-features-design.md` for TUI design specification.*
