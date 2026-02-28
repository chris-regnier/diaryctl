# Context-Aware Jot Target Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the TUI `j` (jot) key append to whichever entry is currently selected, not just the daily entry.

**Architecture:** Add a `jotTarget` field to `pickerModel` that `startJot()` resolves based on the current screen/selection. `doJot()` uses this target instead of always querying for the daily entry. A label above the jot textarea shows which entry is being targeted.

**Tech Stack:** Go, Bubble Tea (charmbracelet/bubbletea), Lipgloss

---

### Task 1: Add `jotTarget` field and `resolveJotTarget()` method

**Files:**
- Modify: `internal/ui/picker.go:160-162` (jot mode fields)
- Modify: `internal/ui/picker.go:922-935` (startJot)
- Test: `internal/ui/picker_test.go`

**Step 1: Write the failing test**

Add a test that verifies `resolveJotTarget` returns the selected entry on `screenDayDetail`.

```go
func TestResolveJotTarget_DayDetailSelectedEntry(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	targetEntry := entry.Entry{
		ID:        "target01",
		Content:   "# Target Entry\n\nSome content",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {targetEntry},
		},
		byID: map[string]entry.Entry{
			"target01": targetEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenDayDetail

	// Set up day list with the target entry selected
	items := []list.Item{entryItem{entry: targetEntry}}
	m.dayList = m.cfg.Theme.NewList(items, 80, 20)

	target := m.resolveJotTarget()
	if target == nil {
		t.Fatal("Expected jot target, got nil")
	}
	if target.ID != "target01" {
		t.Errorf("Expected target ID 'target01', got '%s'", target.ID)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestResolveJotTarget_DayDetailSelectedEntry -v`
Expected: FAIL — `resolveJotTarget` does not exist

**Step 3: Write `resolveJotTarget()` and add `jotTarget` field**

Add the `jotTarget` field to `pickerModel` (after `jotActive` on line 162):

```go
	jotTarget *entry.Entry // entry to append jot to (nil = create new daily)
```

Add the `resolveJotTarget()` method:

```go
// resolveJotTarget determines which entry to jot into based on the current screen
// and selection state. Returns nil if no target exists (will create new daily entry).
func (m *pickerModel) resolveJotTarget() *entry.Entry {
	switch m.screen {
	case screenToday:
		if m.todayFocus == focusEntryList {
			if item, ok := m.todayList.SelectedItem().(entryItem); ok {
				e := item.entry
				return &e
			}
		}
		// Default: daily entry (may be nil)
		return m.dailyEntry
	case screenDayDetail:
		if item, ok := m.dayList.SelectedItem().(entryItem); ok {
			e := item.entry
			return &e
		}
		return nil
	case screenEntryDetail:
		return &m.entry
	default:
		return nil
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/ -run TestResolveJotTarget_DayDetailSelectedEntry -v`
Expected: PASS

**Step 5: Commit**

```
feat(tui): add resolveJotTarget and jotTarget field
```

---

### Task 2: Additional `resolveJotTarget` tests

**Files:**
- Test: `internal/ui/picker_test.go`

**Step 1: Write tests for remaining screen states**

```go
func TestResolveJotTarget_TodayDefaultsToDaily(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:      "daily01",
		Content: "# Daily\n\nContent",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:   map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = &dailyEntry
	m.todayFocus = focusDailyViewport

	target := m.resolveJotTarget()
	if target == nil {
		t.Fatal("Expected daily entry as target, got nil")
	}
	if target.ID != "daily01" {
		t.Errorf("Expected target ID 'daily01', got '%s'", target.ID)
	}
}

func TestResolveJotTarget_TodaySelectedEntry(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:      "daily01",
		Content: "# Daily",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}
	otherEntry := entry.Entry{
		ID:      "other01",
		Content: "# Other entry",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:   map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = &dailyEntry
	m.todayFocus = focusEntryList
	items := []list.Item{entryItem{entry: otherEntry}}
	m.todayList = m.cfg.Theme.NewList(items, 80, 20)

	target := m.resolveJotTarget()
	if target == nil {
		t.Fatal("Expected selected entry as target, got nil")
	}
	if target.ID != "other01" {
		t.Errorf("Expected target ID 'other01', got '%s'", target.ID)
	}
}

func TestResolveJotTarget_EntryDetail(t *testing.T) {
	viewedEntry := entry.Entry{
		ID:      "viewed01",
		Content: "# Viewed Entry",
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:   map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenEntryDetail
	m.entry = viewedEntry

	target := m.resolveJotTarget()
	if target == nil {
		t.Fatal("Expected viewed entry as target, got nil")
	}
	if target.ID != "viewed01" {
		t.Errorf("Expected target ID 'viewed01', got '%s'", target.ID)
	}
}

func TestResolveJotTarget_TodayNoEntries(t *testing.T) {
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:   map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = nil
	m.todayFocus = focusDailyViewport

	target := m.resolveJotTarget()
	if target != nil {
		t.Errorf("Expected nil target for empty today, got %+v", target)
	}
}

func TestResolveJotTarget_DateListReturnsNil(t *testing.T) {
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:   map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenDateList

	target := m.resolveJotTarget()
	if target != nil {
		t.Errorf("Expected nil target for date list, got %+v", target)
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestResolveJotTarget -v`
Expected: All PASS

**Step 3: Commit**

```
test(tui): add resolveJotTarget tests for all screen states
```

---

### Task 3: Wire `resolveJotTarget` into `startJot` and refactor `doJot`

**Files:**
- Modify: `internal/ui/picker.go:922-935` (startJot)
- Modify: `internal/ui/picker.go:1387-1447` (doJot)

**Step 1: Write the failing test**

Test that jotting from `screenDayDetail` appends to the selected entry, not the daily.

```go
func TestJotIntoSelectedEntry_DayDetail(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:        "daily01",
		Content:   "# Daily\n\nDaily content",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}
	otherEntry := entry.Entry{
		ID:        "other01",
		Content:   "# Meeting Notes\n\nSome notes",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {otherEntry, dailyEntry},
		},
		byID: map[string]entry.Entry{
			"daily01": dailyEntry,
			"other01": otherEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenDayDetail
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Set up day list with the other entry selected
	items := []list.Item{entryItem{entry: otherEntry}, entryItem{entry: dailyEntry}}
	m.dayList = m.cfg.Theme.NewList(items, 80, 20)

	// Start jot — should resolve target to otherEntry
	started, _ := m.startJot()
	m = started.(pickerModel)
	m.jotInput.SetValue("Action item from meeting")

	// Submit
	updatedModel, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected jot command")
	}

	result := cmd()
	if jotMsg, ok := result.(jotCompleteMsg); ok {
		if jotMsg.err != nil {
			t.Fatalf("Jot failed: %v", jotMsg.err)
		}
	}

	// Verify the jot went to other01, not daily01
	updated := store.byID["other01"]
	if !strings.Contains(updated.Content, "Action item from meeting") {
		t.Errorf("Expected jot in other01, content: %s", updated.Content)
	}
	daily := store.byID["daily01"]
	if strings.Contains(daily.Content, "Action item from meeting") {
		t.Error("Jot should NOT have been appended to daily entry")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestJotIntoSelectedEntry_DayDetail -v`
Expected: FAIL — jot still targets daily entry

**Step 3: Modify `startJot` to resolve and store the target**

Replace `startJot`:

```go
func (m pickerModel) startJot() (tea.Model, tea.Cmd) {
	m.jotTarget = m.resolveJotTarget()

	ta := textarea.New()
	ta.Placeholder = "jot (^J=newline ↵=submit)..."
	ta.Focus()
	ta.CharLimit = maxJotInputLength
	ta.SetWidth(m.contentWidth() - 4)
	// Dynamic height: use 25% of screen height or max 5 lines
	height := max(min(m.height/4, 5), 3)
	ta.SetHeight(height)
	ta.ShowLineNumbers = false
	m.jotInput = ta
	m.jotActive = true
	return m, textarea.Blink
}
```

**Step 4: Refactor `doJot` to use `jotTarget`**

Replace `doJot`:

```go
func (m pickerModel) doJot(content string) tea.Msg {
	now := time.Now()
	timestamp := now.Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	if m.jotTarget != nil {
		// Append to the targeted entry
		target, err := m.store.Get(m.jotTarget.ID)
		if err != nil {
			return jotCompleteMsg{err: fmt.Errorf("jot target not found: %w", err)}
		}

		var newContent string
		if strings.TrimSpace(target.Content) == "" {
			newContent = jotLine
		} else {
			newContent = target.Content + "\n" + jotLine
		}
		_, err = m.store.Update(target.ID, newContent, nil)
		if err != nil {
			return jotCompleteMsg{err: err}
		}
	} else {
		// No target — create new daily entry (screenToday with no entries)
		id, err := entry.NewID()
		if err != nil {
			return jotCompleteMsg{err: err}
		}
		nowUTC := now.UTC()

		// Use default template if configured
		var templateRefs []entry.TemplateRef
		if m.cfg.DefaultTemplate != "" {
			names := template.ParseNames(m.cfg.DefaultTemplate)
			_, refs, err := template.Compose(m.store, names)
			if err != nil {
				// Continue without template refs (match CLI behavior)
			} else {
				templateRefs = refs
			}
		}

		e := entry.Entry{
			ID:        id,
			Content:   fmt.Sprintf("# %s\n\n%s", now.Format("2006-01-02"), jotLine),
			Templates: templateRefs,
			CreatedAt: nowUTC,
			UpdatedAt: nowUTC,
		}
		if err := m.store.Create(e); err != nil {
			return jotCompleteMsg{err: err}
		}
	}

	return jotCompleteMsg{}
}
```

**Step 5: Run tests to verify**

Run: `go test ./internal/ui/ -run "TestJot" -v`
Expected: All jot tests PASS (new and existing)

**Step 6: Commit**

```
feat(tui): wire resolveJotTarget into startJot and doJot
```

---

### Task 4: Add jot target indicator in the View

**Files:**
- Modify: `internal/ui/picker.go:886-891` (View method, jot rendering)
- Test: `internal/ui/picker_test.go`

**Step 1: Write the failing test**

```go
func TestJotTargetIndicator(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	otherEntry := entry.Entry{
		ID:        "other01",
		Content:   "# Meeting Notes\n\nSome notes",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {otherEntry},
		},
		byID: map[string]entry.Entry{
			"other01": otherEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: "", Theme: DefaultTheme}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.todayFocus = focusEntryList
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Set up entry list
	items := []list.Item{entryItem{entry: otherEntry}}
	m.todayList = m.cfg.Theme.NewList(items, 80, 20)

	// Start jot — target should be otherEntry
	started, _ := m.startJot()
	m = started.(pickerModel)

	view := m.View()
	stripped := stripANSI(view)
	if !strings.Contains(stripped, "Jotting into:") {
		t.Error("Expected 'Jotting into:' indicator in view")
	}
	if !strings.Contains(stripped, "Meeting Notes") {
		t.Error("Expected target entry preview in jot indicator")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestJotTargetIndicator -v`
Expected: FAIL — no indicator rendered

**Step 3: Add `jotTargetLabel()` helper and update View**

Add helper method:

```go
// jotTargetLabel returns a short label describing the current jot target.
func (m *pickerModel) jotTargetLabel() string {
	if m.jotTarget == nil {
		return "Jotting into: new entry"
	}
	preview := strings.SplitN(m.jotTarget.Content, "\n", 2)[0]
	if len(preview) > 40 {
		preview = preview[:37] + "..."
	}
	return fmt.Sprintf("Jotting into: %s", preview)
}
```

In the `View()` method, replace the jot rendering block (lines 889-891):

```go
	} else if m.jotActive {
		label := m.cfg.Theme.HelpStyle().Width(cw).Render(m.jotTargetLabel())
		result = result + "\n" + label + "\n" + m.jotInput.View()
	}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/ -run TestJotTargetIndicator -v`
Expected: PASS

**Step 5: Commit**

```
feat(tui): show jot target indicator above textarea
```

---

### Task 5: Fix `jotCompleteMsg` refresh for `screenEntryDetail`

**Files:**
- Modify: `internal/ui/picker.go:219-236` (jotCompleteMsg handler)
- Test: `internal/ui/picker_test.go`

**Step 1: Write the failing test**

```go
func TestJotFromEntryDetail_RefreshesEntry(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	viewedEntry := entry.Entry{
		ID:        "viewed01",
		Content:   "# Viewed Entry\n\nOriginal content",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {viewedEntry},
		},
		byID: map[string]entry.Entry{
			"viewed01": viewedEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenEntryDetail
	m.entry = viewedEntry
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Simulate jot complete
	updated, cmd := m.Update(jotCompleteMsg{})
	m = updated.(pickerModel)

	// Should have returned a command (to reload entry detail)
	if cmd == nil {
		t.Error("Expected refresh command after jot on entry detail, got nil")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/ -run TestJotFromEntryDetail_RefreshesEntry -v`
Expected: FAIL — `cmd` is nil because `screenEntryDetail` falls into `default`

**Step 3: Add `screenEntryDetail` case to `jotCompleteMsg` handler**

In the `jotCompleteMsg` handler (around line 225), add a case before `default`:

```go
		case screenEntryDetail:
			return m, func() tea.Msg {
				e, err := m.store.Get(m.entry.ID)
				if err != nil {
					return todayLoadedMsg{err: err}
				}
				m.entry = e
				m.viewport.SetContent(m.formatEntry())
				return nil
			}
```

Actually, a simpler approach: reload the entry via `loadEntryDetail`:

```go
		case screenEntryDetail:
			return m.loadEntryDetail(m.entry.ID)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/ui/ -run TestJotFromEntryDetail_RefreshesEntry -v`
Expected: PASS

**Step 5: Run all tests**

Run: `go test ./internal/ui/ -v`
Expected: All PASS

**Step 6: Commit**

```
fix(tui): refresh entry detail after jot
```

---

### Task 6: Test jot into selected entry on today screen

**Files:**
- Test: `internal/ui/picker_test.go`

**Step 1: Write the test**

```go
func TestJotIntoSelectedEntry_TodayScreen(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:        "daily01",
		Content:   "# Daily\n\nDaily content",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}
	otherEntry := entry.Entry{
		ID:        "other01",
		Content:   "# Side Notes",
		CreatedAt: today.Add(10 * time.Hour),
		UpdatedAt: today.Add(10 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {otherEntry, dailyEntry},
		},
		byID: map[string]entry.Entry{
			"daily01": dailyEntry,
			"other01": otherEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = &dailyEntry
	m.todayFocus = focusEntryList
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Set up entry list with other entry selected
	items := []list.Item{entryItem{entry: otherEntry}}
	m.todayList = m.cfg.Theme.NewList(items, 80, 20)

	// Start jot
	started, _ := m.startJot()
	m = started.(pickerModel)
	m.jotInput.SetValue("Quick thought")

	// Submit
	updatedModel, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected jot command")
	}

	result := cmd()
	if jotMsg, ok := result.(jotCompleteMsg); ok {
		if jotMsg.err != nil {
			t.Fatalf("Jot failed: %v", jotMsg.err)
		}
	}

	// Verify jot went to other01
	updated := store.byID["other01"]
	if !strings.Contains(updated.Content, "Quick thought") {
		t.Errorf("Expected jot in other01, content: %s", updated.Content)
	}

	// Verify daily was NOT modified
	daily := store.byID["daily01"]
	if strings.Contains(daily.Content, "Quick thought") {
		t.Error("Jot should NOT have gone to daily entry")
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/ui/ -run TestJotIntoSelectedEntry_TodayScreen -v`
Expected: PASS

**Step 3: Commit**

```
test(tui): add jot-into-selected-entry test for today screen
```

---

### Task 7: Update help overlay and run full test suite

**Files:**
- Modify: `internal/ui/picker.go:896-920` (helpOverlay)

**Step 1: Update help text**

In the help overlay, change the jot description from:

```
  j          jot a note (^J for newline)
```

to:

```
  j          jot into selected entry (^J for newline)
```

**Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Manual smoke test checklist**

- [ ] `diaryctl` — press `j` with no entries: creates new daily (unchanged)
- [ ] `diaryctl` — press `j` with daily focused: appends to daily (unchanged)
- [ ] `diaryctl` — Tab to entry list, select entry, press `j`: appends to selected entry
- [ ] `diaryctl` — Browse to day detail, select entry, press `j`: appends to that entry
- [ ] `diaryctl` — View entry detail, press `j`: appends to viewed entry
- [ ] Jot target indicator visible above textarea
- [ ] Entry detail refreshes after jot

**Step 4: Commit**

```
docs(tui): update help overlay for context-aware jot
```
