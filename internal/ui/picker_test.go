package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// mockStorage implements StorageProvider for testing.
type mockStorage struct {
	days          []storage.DaySummary
	entries       map[string][]entry.Entry
	byID          map[string]entry.Entry
	attachError   error // Add this field
	contexts      []storage.Context
	entryContexts map[string][]string // entryID -> contextIDs
}

func (m *mockStorage) ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error) {
	return m.days, nil
}

func (m *mockStorage) List(opts storage.ListOptions) ([]entry.Entry, error) {
	if opts.Date != nil {
		key := opts.Date.Format("2006-01-02")
		return m.entries[key], nil
	}
	return nil, nil
}

func (m *mockStorage) Get(id string) (entry.Entry, error) {
	if e, ok := m.byID[id]; ok {
		// Add context references if any
		if contexts := m.entryContexts[id]; len(contexts) > 0 {
			e.Contexts = make([]entry.ContextRef, len(contexts))
			for i, ctxID := range contexts {
				e.Contexts[i] = entry.ContextRef{ContextID: ctxID}
			}
		}
		return e, nil
	}
	return entry.Entry{}, storage.ErrNotFound
}

// Write methods - now fully functional for testing
func (m *mockStorage) Create(e entry.Entry) error {
	// Add to byID map
	m.byID[e.ID] = e
	// Add to date-based entries map
	// Convert to local time for date key to match how entries are stored by date
	localTime := e.CreatedAt.In(time.Local)
	y, mo, d := localTime.Date()
	dateKey := time.Date(y, mo, d, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	m.entries[dateKey] = append(m.entries[dateKey], e)
	return nil
}

func (m *mockStorage) Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error) {
	e, ok := m.byID[id]
	if !ok {
		return entry.Entry{}, storage.ErrNotFound
	}
	e.Content = content
	e.Templates = templates
	e.UpdatedAt = time.Now().UTC()
	m.byID[id] = e

	// Update in date-based entries map
	// Convert to local time for date key to match how entries are stored by date
	dateKey := e.CreatedAt.In(time.Local).Truncate(24 * time.Hour).Format("2006-01-02")
	for i, existing := range m.entries[dateKey] {
		if existing.ID == id {
			m.entries[dateKey][i] = e
			break
		}
	}
	return e, nil
}

func (m *mockStorage) Delete(id string) error {
	_, ok := m.byID[id]
	if !ok {
		return storage.ErrNotFound
	}

	// Remove from byID map
	delete(m.byID, id)

	// Remove from date-based entries map
	// For test purposes, we'll just iterate through all date keys to find and remove the entry
	for dateKey, entries := range m.entries {
		filtered := []entry.Entry{}
		found := false
		for _, existing := range entries {
			if existing.ID == id {
				found = true
			} else {
				filtered = append(filtered, existing)
			}
		}
		if found {
			m.entries[dateKey] = filtered
			break
		}
	}

	return nil
}

// Context methods
func (m *mockStorage) ListContexts() ([]storage.Context, error) {
	return m.contexts, nil
}

func (m *mockStorage) CreateContext(c storage.Context) error {
	m.contexts = append(m.contexts, c)
	return nil
}

func (m *mockStorage) AttachContext(entryID string, contextID string) error {
	if m.attachError != nil {
		return m.attachError
	}
	if m.entryContexts == nil {
		m.entryContexts = make(map[string][]string)
	}
	m.entryContexts[entryID] = append(m.entryContexts[entryID], contextID)
	return nil
}

func (m *mockStorage) DetachContext(entryID string, contextID string) error {
	contexts := m.entryContexts[entryID]
	filtered := []string{}
	for _, ctx := range contexts {
		if ctx != contextID {
			filtered = append(filtered, ctx)
		}
	}
	m.entryContexts[entryID] = filtered
	return nil
}

// Template methods
func (m *mockStorage) ListTemplates() ([]storage.Template, error) {
	return nil, nil // Return empty list by default
}

func (m *mockStorage) GetTemplateByName(name string) (storage.Template, error) {
	return storage.Template{}, storage.ErrNotFound
}

func makeTestDays() (*mockStorage, []storage.DaySummary) {
	jan10 := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)
	jan12 := time.Date(2026, 1, 12, 0, 0, 0, 0, time.Local)
	jan15 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)

	days := []storage.DaySummary{
		{Date: jan15, Count: 2, Preview: "latest entry"},
		{Date: jan12, Count: 1, Preview: "mid entry"},
		{Date: jan10, Count: 1, Preview: "old entry"},
	}

	e1 := entry.Entry{ID: "entry001", Content: "jan15 morning", CreatedAt: jan15.Add(9 * time.Hour), UpdatedAt: jan15.Add(9 * time.Hour)}
	e2 := entry.Entry{ID: "entry002", Content: "jan15 afternoon", CreatedAt: jan15.Add(14 * time.Hour), UpdatedAt: jan15.Add(14 * time.Hour)}
	e3 := entry.Entry{ID: "entry003", Content: "jan12 entry", CreatedAt: jan12.Add(12 * time.Hour), UpdatedAt: jan12.Add(12 * time.Hour)}
	e4 := entry.Entry{ID: "entry004", Content: "jan10 entry", CreatedAt: jan10.Add(18 * time.Hour), UpdatedAt: jan10.Add(18 * time.Hour)}

	mock := &mockStorage{
		days: days,
		entries: map[string][]entry.Entry{
			"2026-01-15": {e2, e1},
			"2026-01-12": {e3},
			"2026-01-10": {e4},
		},
		byID: map[string]entry.Entry{
			"entry001": e1, "entry002": e2, "entry003": e3, "entry004": e4,
		},
	}
	return mock, days
}

func TestPickerDayNavigation(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])

	// Simulate window size
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Select first date (Jan 15, index 0) and enter day detail
	m.dayIdx = 0
	loaded, _ := m.loadDayDetail()
	m = loaded.(pickerModel)

	if m.screen != screenDayDetail {
		t.Fatalf("expected DayDetail screen, got %d", m.screen)
	}

	// Navigate to previous day (earlier) — should go to Jan 12 (index 1)
	updated, _ := m.updateDayDetail(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(pickerModel)
	if m.dayIdx != 1 {
		t.Errorf("after left: dayIdx = %d, want 1", m.dayIdx)
	}

	// Navigate to previous day again — should go to Jan 10 (index 2)
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(pickerModel)
	if m.dayIdx != 2 {
		t.Errorf("after second left: dayIdx = %d, want 2", m.dayIdx)
	}

	// Navigate to previous day at boundary — should stay at index 2
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyLeft})
	m = updated.(pickerModel)
	if m.dayIdx != 2 {
		t.Errorf("at boundary left: dayIdx = %d, want 2 (should stay)", m.dayIdx)
	}

	// Navigate forward (next/later day) — should go back to Jan 12 (index 1)
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(pickerModel)
	if m.dayIdx != 1 {
		t.Errorf("after n: dayIdx = %d, want 1", m.dayIdx)
	}

	// Navigate forward to Jan 15 (index 0)
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(pickerModel)
	if m.dayIdx != 0 {
		t.Errorf("after second n: dayIdx = %d, want 0", m.dayIdx)
	}

	// Navigate forward at boundary — should stay at index 0
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(pickerModel)
	if m.dayIdx != 0 {
		t.Errorf("at boundary n: dayIdx = %d, want 0 (should stay)", m.dayIdx)
	}
}

func TestPickerManyEntriesPerDay(t *testing.T) {
	// T024: Verify large number of entries per day (50+) works
	jan15 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)

	entries := make([]entry.Entry, 60)
	byID := make(map[string]entry.Entry)
	for i := 0; i < 60; i++ {
		e := entry.Entry{
			ID:        fmt.Sprintf("ent%05d", i),
			Content:   fmt.Sprintf("Entry number %d with some content", i),
			CreatedAt: jan15.Add(time.Duration(i) * time.Minute),
			UpdatedAt: jan15.Add(time.Duration(i) * time.Minute),
		}
		entries[59-i] = e // reverse chronological
		byID[e.ID] = e
	}

	mock := &mockStorage{
		days: []storage.DaySummary{
			{Date: jan15, Count: 60, Preview: "Entry number 59"},
		},
		entries: map[string][]entry.Entry{
			"2026-01-15": entries,
		},
		byID: byID,
	}

	m := newPickerModel(mock, mock.days, presets["default-dark"])
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Load day detail with 60 entries
	loaded, _ := m.loadDayDetail()
	m = loaded.(pickerModel)
	if m.screen != screenDayDetail {
		t.Fatalf("expected DayDetail, got %d", m.screen)
	}
	// Verify all items loaded in the list
	if len(m.dayList.Items()) != 60 {
		t.Errorf("expected 60 items, got %d", len(m.dayList.Items()))
	}
}

func TestPickerEmptyDateRange(t *testing.T) {
	// T026: Date range with no entries
	mock := &mockStorage{
		days:    []storage.DaySummary{},
		entries: map[string][]entry.Entry{},
		byID:    map[string]entry.Entry{},
	}

	m := newPickerModel(mock, mock.days, presets["default-dark"])
	// With empty days, RunPicker would print message and not launch Bubble Tea
	// Here we verify the model handles empty state gracefully
	if len(m.days) != 0 {
		t.Errorf("expected 0 days, got %d", len(m.days))
	}
}

func TestPickerScreenTransitions(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])

	// Simulate window size
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Start at DateList
	if m.screen != screenDateList {
		t.Fatalf("expected DateList screen, got %d", m.screen)
	}

	// Enter day detail
	m.dayIdx = 0
	loaded, _ := m.loadDayDetail()
	m = loaded.(pickerModel)
	if m.screen != screenDayDetail {
		t.Fatalf("expected DayDetail screen, got %d", m.screen)
	}

	// Enter entry detail
	loaded, _ = m.loadEntryDetail("entry002")
	m = loaded.(pickerModel)
	if m.screen != screenEntryDetail {
		t.Fatalf("expected EntryDetail screen, got %d", m.screen)
	}

	// Esc back to day detail
	updated, _ := m.updateEntryDetail(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(pickerModel)
	if m.screen != screenDayDetail {
		t.Fatalf("expected DayDetail screen after esc, got %d", m.screen)
	}

	// Esc back to date list
	updated, _ = m.updateDayDetail(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(pickerModel)
	if m.screen != screenDateList {
		t.Fatalf("expected DateList screen after esc, got %d", m.screen)
	}
}

func TestTodayScreenDataLoading(t *testing.T) {
	// Test loading today's data with multiple entries
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Create test entries for today (oldest to newest)
	e1 := entry.Entry{ID: "daily001", Content: "# Today's daily entry", CreatedAt: today.Add(8 * time.Hour), UpdatedAt: today.Add(8 * time.Hour)}
	e2 := entry.Entry{ID: "entry002", Content: "Morning note", CreatedAt: today.Add(10 * time.Hour), UpdatedAt: today.Add(10 * time.Hour)}
	e3 := entry.Entry{ID: "entry003", Content: "Afternoon note", CreatedAt: today.Add(14 * time.Hour), UpdatedAt: today.Add(14 * time.Hour)}

	mock := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {e3, e2, e1}, // newest-first order
		},
		byID: map[string]entry.Entry{
			"daily001": e1, "entry002": e2, "entry003": e3,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(mock, cfg)

	// Verify initial screen is today
	if m.screen != screenToday {
		t.Fatalf("expected screenToday, got %d", m.screen)
	}

	// Load today's data
	msg := m.loadTodayCmd()
	loadedMsg, ok := msg.(todayLoadedMsg)
	if !ok {
		t.Fatalf("expected todayLoadedMsg, got %T", msg)
	}

	if loadedMsg.err != nil {
		t.Fatalf("unexpected error: %v", loadedMsg.err)
	}

	// Verify daily entry is the oldest (e1)
	if loadedMsg.daily == nil {
		t.Fatal("expected daily entry, got nil")
	}
	if loadedMsg.daily.ID != "daily001" {
		t.Errorf("expected daily entry ID daily001, got %s", loadedMsg.daily.ID)
	}

	// Verify other entries are the rest (e2, e3) in newest-first order
	if len(loadedMsg.entries) != 2 {
		t.Fatalf("expected 2 other entries, got %d", len(loadedMsg.entries))
	}
	if loadedMsg.entries[0].ID != "entry003" {
		t.Errorf("expected first entry ID entry003, got %s", loadedMsg.entries[0].ID)
	}
	if loadedMsg.entries[1].ID != "entry002" {
		t.Errorf("expected second entry ID entry002, got %s", loadedMsg.entries[1].ID)
	}

	// Apply the message to the model
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)
	updated, _ := m.Update(loadedMsg)
	m = updated.(pickerModel)

	// Verify model state
	if m.dailyEntry == nil {
		t.Fatal("expected dailyEntry to be set")
	}
	if m.dailyEntry.ID != "daily001" {
		t.Errorf("expected dailyEntry ID daily001, got %s", m.dailyEntry.ID)
	}
	if len(m.todayEntries) != 2 {
		t.Errorf("expected 2 todayEntries, got %d", len(m.todayEntries))
	}
}

func TestTodayScreenEmptyDay(t *testing.T) {
	// Test loading today when there are no entries
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	mock := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {}, // no entries
		},
		byID: map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(mock, cfg)

	// Load today's data
	msg := m.loadTodayCmd()
	loadedMsg, ok := msg.(todayLoadedMsg)
	if !ok {
		t.Fatalf("expected todayLoadedMsg, got %T", msg)
	}

	if loadedMsg.err != nil {
		t.Fatalf("unexpected error: %v", loadedMsg.err)
	}

	// Verify empty state
	if loadedMsg.daily != nil {
		t.Errorf("expected nil daily entry, got %v", loadedMsg.daily)
	}
	if len(loadedMsg.entries) != 0 {
		t.Errorf("expected 0 other entries, got %d", len(loadedMsg.entries))
	}
}

func TestTodayScreenOnlyDailyEntry(t *testing.T) {
	// Test loading today when there's only one entry (the daily entry)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	e1 := entry.Entry{ID: "daily001", Content: "# Today's daily entry", CreatedAt: today.Add(8 * time.Hour), UpdatedAt: today.Add(8 * time.Hour)}

	mock := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {e1},
		},
		byID: map[string]entry.Entry{
			"daily001": e1,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(mock, cfg)

	// Load today's data
	msg := m.loadTodayCmd()
	loadedMsg, ok := msg.(todayLoadedMsg)
	if !ok {
		t.Fatalf("expected todayLoadedMsg, got %T", msg)
	}

	if loadedMsg.err != nil {
		t.Fatalf("unexpected error: %v", loadedMsg.err)
	}

	// Verify daily entry is set
	if loadedMsg.daily == nil {
		t.Fatal("expected daily entry, got nil")
	}
	if loadedMsg.daily.ID != "daily001" {
		t.Errorf("expected daily entry ID daily001, got %s", loadedMsg.daily.ID)
	}

	// Verify no other entries
	if len(loadedMsg.entries) != 0 {
		t.Errorf("expected 0 other entries, got %d", len(loadedMsg.entries))
	}
}

func TestContextPanelEscReturnsToCorrectScreen(t *testing.T) {
	tests := []struct {
		name       string
		fromScreen pickerScreen
	}{
		{"from today screen", screenToday},
		{"from date list", screenDateList},
		{"from day detail", screenDayDetail},
		{"from entry detail", screenEntryDetail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

			store := &mockStorage{
				entries: map[string][]entry.Entry{
					today.Format("2006-01-02"): {},
				},
				byID: map[string]entry.Entry{},
			}
			cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
			m := newTUIModel(store, cfg)
			m.screen = screenContextPanel
			m.prevScreen = tt.fromScreen

			// Simulate ESC key
			msg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, cmd := m.Update(msg)
			m = updatedModel.(pickerModel)

			if m.screen != tt.fromScreen {
				t.Errorf("ESC from context panel should return to %v, got %v", tt.fromScreen, m.screen)
			}

			// For non-today screens, loadTodayCmd should NOT be returned
			// This verifies the bug: currently it always returns loadTodayCmd
			if tt.fromScreen != screenToday && cmd != nil {
				// Execute the command to see what it does
				result := cmd()
				// If it's a todayLoadedMsg, that's the bug - we're loading today when we shouldn't
				if _, ok := result.(todayLoadedMsg); ok {
					t.Errorf("ESC from context panel to %v should not load today screen, but it did", tt.fromScreen)
				}
			}
		})
	}
}

func TestContextAutoAttachErrorHandling(t *testing.T) {
	store := &mockStorage{
		entries:     map[string][]entry.Entry{},
		byID:        map[string]entry.Entry{},
		attachError: fmt.Errorf("database connection failed"),
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenContextPanel
	m.contextCreating = true
	m.contextEntryID = "test123"
	m.contextInput.SetValue("work")

	// Simulate creating a new context
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateContextCreate(msg)
	m = updatedModel.(pickerModel)

	// Execute the command to trigger auto-attach
	if cmd == nil {
		t.Fatal("Expected command to be returned, got nil")
	}

	result := cmd()

	// The current implementation returns contextsLoadedMsg via loadContexts(),
	// but after the fix it should return contextCreatedMsg with the error
	if msg, ok := result.(contextCreatedMsg); ok {
		// The error should be captured, not ignored
		if msg.err == nil {
			t.Error("Expected error from failed auto-attach, got nil")
		}
	} else {
		// This is the current buggy behavior - it calls loadContexts() which
		// ignores the attach error
		t.Error("Expected contextCreatedMsg, but got different message type (bug: error was silently ignored)")
	}
}

func TestEditorTempFileCleanup(t *testing.T) {
	// This test verifies that temp file Close() errors are handled
	// We can't easily test the actual cleanup, but we can verify the pattern
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID: map[string]entry.Entry{
			"test": {ID: "test", Content: "test content", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)

	// Verify startCreate and startEdit don't panic with basic operations
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Editor action panicked: %v", r)
		}
	}()

	// Note: Full integration test would require mocking os.CreateTemp
	// which is beyond scope. This verifies the functions are callable.
	_, _ = m.startCreate()

	testEntry := entry.Entry{ID: "test", Content: "test content", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_, _ = m.startEdit(testEntry)
}

// Task 5: Add Tests for Interactive Actions

func TestJotActionCreatesNewEntry(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {}, // empty today
		},
		byID: map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Activate jot mode
	started, _ := m.startJot()
	m = started.(pickerModel)
	m.jotInput.SetValue("Meeting notes at 2pm")

	// Simulate Enter key to submit
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateJotInput(msg)
	m = updatedModel.(pickerModel)

	// Execute the jot command
	if cmd == nil {
		t.Fatal("Expected jot command to be returned, got nil")
	}

	result := cmd()
	if jotMsg, ok := result.(jotCompleteMsg); ok {
		if jotMsg.err != nil {
			t.Fatalf("Jot failed: %v", jotMsg.err)
		}
	} else {
		t.Fatalf("Expected jotCompleteMsg, got %T", result)
	}

	// Verify entry was created
	entries := store.entries[today.Format("2006-01-02")]
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry after jot, got %d", len(entries))
	}

	if !strings.Contains(entries[0].Content, "Meeting notes at 2pm") {
		t.Errorf("Entry content doesn't contain jot text: %s", entries[0].Content)
	}
}

func TestJotActionAppendsToExistingEntry(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Pre-populate with today's daily entry
	existingEntry := entry.Entry{
		ID:        "daily01",
		Content:   "# Daily Entry\n\nInitial content",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {existingEntry},
		},
		byID: map[string]entry.Entry{
			"daily01": existingEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = &existingEntry
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Activate jot mode
	started, _ := m.startJot()
	m = started.(pickerModel)
	m.jotInput.SetValue("Additional note")

	// Submit jot
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateJotInput(msg)
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected jot command to be returned, got nil")
	}

	result := cmd()
	if jotMsg, ok := result.(jotCompleteMsg); ok {
		if jotMsg.err != nil {
			t.Fatalf("Jot failed: %v", jotMsg.err)
		}
	} else {
		t.Fatalf("Expected jotCompleteMsg, got %T", result)
	}

	// Should still have 1 entry (updated, not created)
	entries := store.entries[today.Format("2006-01-02")]
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after jot append, got %d", len(entries))
	}

	// Should contain both original and new content
	content := store.byID["daily01"].Content
	if !strings.Contains(content, "Initial content") {
		t.Error("Original content lost during jot append")
	}
	if !strings.Contains(content, "Additional note") {
		t.Error("Jot content not appended")
	}
}

func TestDeleteActionConfirmation(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test entry", CreatedAt: today, UpdatedAt: today}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {testEntry},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenEntryDetail
	m.entry = testEntry
	m.deleteEntry = testEntry

	// Activate delete mode
	m.deleteActive = true

	// Simulate 'y' to confirm
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updatedModel, cmd := m.updateDeleteConfirm(msg)
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected delete command to be returned, got nil")
	}

	result := cmd()
	if delMsg, ok := result.(deleteCompleteMsg); ok {
		if delMsg.err != nil {
			t.Fatalf("Delete failed: %v", delMsg.err)
		}
	} else {
		t.Fatalf("Expected deleteCompleteMsg, got %T", result)
	}

	// Entry should be deleted
	dateKey := today.Format("2006-01-02")
	entries := store.entries[dateKey]
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after delete, got %d", len(entries))
	}

	// Should also be removed from byID
	if _, exists := store.byID["entry01"]; exists {
		t.Error("Entry still exists in byID map after delete")
	}
}

func TestDeleteActionCancellation(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test entry", CreatedAt: today, UpdatedAt: today}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {testEntry},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenEntryDetail
	m.entry = testEntry
	m.deleteEntry = testEntry
	m.deleteActive = true

	// Simulate 'n' to cancel
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updatedModel, _ := m.updateDeleteConfirm(msg)
	m = updatedModel.(pickerModel)

	// Entry should NOT be deleted
	entries := store.entries[today.Format("2006-01-02")]
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after cancel, got %d", len(entries))
	}

	// Delete mode should be deactivated
	if m.deleteActive {
		t.Error("Delete mode should be deactivated after cancellation")
	}
}

func TestHelpOverlayToggle(t *testing.T) {
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:    map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday

	// Initially help should be inactive
	if m.helpActive {
		t.Error("Help should be inactive initially")
	}

	// Press '?' to show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(pickerModel)

	if !m.helpActive {
		t.Error("Help should be active after pressing '?'")
	}

	// Press '?' again to hide help
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(pickerModel)

	if m.helpActive {
		t.Error("Help should be inactive after toggling off")
	}
}

// Task 6: Add Tests for Context Panel Actions

func TestContextPanelAttach(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test", CreatedAt: today, UpdatedAt: today}

	store := &mockStorage{
		contexts: []storage.Context{
			{ID: "ctx01", Name: "work", Source: "manual", CreatedAt: today, UpdatedAt: today},
			{ID: "ctx02", Name: "personal", Source: "manual", CreatedAt: today, UpdatedAt: today},
		},
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {testEntry},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
		entryContexts: make(map[string][]string),
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenContextPanel
	m.contextEntryID = "entry01"

	// Load contexts first
	loadResult := m.loadContexts()
	if loadMsg, ok := loadResult.(contextsLoadedMsg); ok {
		if loadMsg.err != nil {
			t.Fatalf("Failed to load contexts: %v", loadMsg.err)
		}
		updated, _ := m.Update(loadMsg)
		m = updated.(pickerModel)
	}

	// Select "work" context (index 0)
	m.contextList.Select(0)

	// Simulate Enter to attach
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateContextPanel(msg)
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected command to be returned, got nil")
	}

	result := cmd()
	if attachMsg, ok := result.(contextsLoadedMsg); ok {
		if attachMsg.err != nil {
			t.Fatalf("Attach failed: %v", attachMsg.err)
		}
	}

	// Verify context was attached
	attached := store.entryContexts["entry01"]
	found := false
	for _, ctx := range attached {
		if ctx == "ctx01" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Context 'work' (ctx01) was not attached to entry")
	}
}

func TestContextPanelDetach(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test", CreatedAt: today, UpdatedAt: today}

	store := &mockStorage{
		contexts: []storage.Context{
			{ID: "ctx01", Name: "work", Source: "manual", CreatedAt: today, UpdatedAt: today},
			{ID: "ctx02", Name: "personal", Source: "manual", CreatedAt: today, UpdatedAt: today},
		},
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {testEntry},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
		entryContexts: map[string][]string{
			"entry01": {"ctx01"}, // Pre-attach "work" context
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenContextPanel
	m.contextEntryID = "entry01"

	// Load contexts first
	loadResult := m.loadContexts()
	if loadMsg, ok := loadResult.(contextsLoadedMsg); ok {
		if loadMsg.err != nil {
			t.Fatalf("Failed to load contexts: %v", loadMsg.err)
		}
		updated, _ := m.Update(loadMsg)
		m = updated.(pickerModel)
	}

	// Select "work" context (index 0) which is already attached
	m.contextList.Select(0)

	// Simulate Enter to detach
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateContextPanel(msg)
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected command to be returned, got nil")
	}

	result := cmd()
	if detachMsg, ok := result.(contextsLoadedMsg); ok {
		if detachMsg.err != nil {
			t.Fatalf("Detach failed: %v", detachMsg.err)
		}
	}

	// Verify context was detached
	attached := store.entryContexts["entry01"]
	for _, ctx := range attached {
		if ctx == "ctx01" {
			t.Error("Context 'work' (ctx01) should have been detached")
		}
	}
}

func TestContextPanelCreate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test", CreatedAt: today, UpdatedAt: today}

	store := &mockStorage{
		contexts: []storage.Context{},
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {testEntry},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
		entryContexts: make(map[string][]string),
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenContextPanel
	m.contextEntryID = "entry01"

	// Simulate 'n' to initiate creation
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	updatedModel, _ := m.updateContextPanel(msg)
	m = updatedModel.(pickerModel)

	// Should be in context creation mode
	if !m.contextCreating {
		t.Error("Should be in context creation mode after pressing 'n'")
	}

	// Set the context name
	m.contextInput.SetValue("newcontext")

	// Simulate Enter to confirm creation
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.updateContextCreate(msg)
	m = updatedModel.(pickerModel)

	if cmd == nil {
		t.Fatal("Expected create command to be returned, got nil")
	}

	result := cmd()
	if createMsg, ok := result.(contextCreatedMsg); ok {
		if createMsg.err != nil {
			t.Fatalf("Create failed: %v", createMsg.err)
		}
	} else {
		t.Fatalf("Expected contextCreatedMsg, got %T", result)
	}

	// Verify context was created
	found := false
	var createdID string
	for _, ctx := range store.contexts {
		if ctx.Name == "newcontext" {
			found = true
			createdID = ctx.ID
			break
		}
	}
	if !found {
		t.Error("Context 'newcontext' was not created")
	}

	// Verify auto-attach happened
	attached := store.entryContexts["entry01"]
	attachedCorrectly := false
	for _, ctx := range attached {
		if ctx == createdID {
			attachedCorrectly = true
			break
		}
	}
	if !attachedCorrectly {
		t.Error("Context 'newcontext' was not auto-attached to entry")
	}
}

// TestJotMultiline verifies that Ctrl+J inserts newlines and Enter submits
func TestJotMultiline(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:        "daily-1",
		Content:   "# " + today.Format("2006-01-02"),
		CreatedAt: today,
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {dailyEntry},
		},
		byID: map[string]entry.Entry{
			"daily-1": dailyEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	m.dailyEntry = &dailyEntry
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Start jot
	started, _ := m.startJot()
	m = started.(pickerModel)
	if !m.jotActive {
		t.Fatal("Expected jotActive to be true")
	}

	// Type first line
	m.jotInput.SetValue("Line 1")

	// Press Ctrl+J to insert newline
	updated, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(pickerModel)
	if !m.jotActive {
		t.Error("Expected jotActive to still be true after Ctrl+J")
	}
	if cmd != nil {
		// If there's a command, it shouldn't be a submit
		result := cmd()
		if _, isJotComplete := result.(jotCompleteMsg); isJotComplete {
			t.Error("Ctrl+J should not submit the jot")
		}
	}

	// Verify newline was inserted
	value := m.jotInput.Value()
	if !strings.Contains(value, "\n") {
		t.Error("Expected newline to be inserted after Ctrl+J")
	}

	// Type second line
	m.jotInput.SetValue("Line 1\nLine 2")

	// Press Enter to submit
	updated, cmd = m.updateJotInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(pickerModel)
	if m.jotActive {
		t.Error("Expected jotActive to be false after Enter")
	}
	if cmd == nil {
		t.Fatal("Expected cmd to be set after Enter")
	}

	// Execute the jot command
	result := cmd()
	jotMsg, ok := result.(jotCompleteMsg)
	if !ok {
		t.Fatalf("Expected jotCompleteMsg, got %T", result)
	}
	if jotMsg.err != nil {
		t.Fatalf("doJot returned error: %v", jotMsg.err)
	}

	// Verify multiline content was saved
	updatedEntry := store.byID["daily-1"]
	if !strings.Contains(updatedEntry.Content, "Line 1") {
		t.Errorf("Expected content to contain 'Line 1', got: %s", updatedEntry.Content)
	}
	if !strings.Contains(updatedEntry.Content, "Line 2") {
		t.Errorf("Expected content to contain 'Line 2', got: %s", updatedEntry.Content)
	}
}

// TestJotEnterDoesNotInsertNewline verifies Enter submits, not inserts newline
func TestJotEnterDoesNotInsertNewline(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:        "daily-1",
		Content:   "# " + today.Format("2006-01-02"),
		CreatedAt: today,
	}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {dailyEntry},
		},
		byID: map[string]entry.Entry{
			"daily-1": dailyEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Start jot
	started, _ := m.startJot()
	m = started.(pickerModel)

	// Type single line
	m.jotInput.SetValue("Single line text")

	// Press Enter
	updated, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(pickerModel)

	// Verify it submits (jotActive becomes false)
	if m.jotActive {
		t.Error("Enter should deactivate jot mode (submit)")
	}

	// Verify a command was returned (the doJot command)
	if cmd == nil {
		t.Fatal("Enter should return a doJot command")
	}

	// Execute and verify it's a jotCompleteMsg
	result := cmd()
	if _, ok := result.(jotCompleteMsg); !ok {
		t.Errorf("Enter should submit jot, got %T instead", result)
	}
}

// TestJotEscape verifies Esc cancels jot input
func TestJotEscape(t *testing.T) {
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:    map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Start jot
	started, _ := m.startJot()
	m = started.(pickerModel)
	if !m.jotActive {
		t.Fatal("Expected jotActive to be true")
	}

	// Type some text
	m.jotInput.SetValue("Some text")

	// Press Escape
	updated, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(pickerModel)

	// Verify jot was cancelled
	if m.jotActive {
		t.Error("Esc should cancel jot mode")
	}

	// Verify no command was returned (no submission)
	if cmd != nil {
		t.Error("Esc should not return a command (no submission)")
	}
}

// TestJotEmptySubmit verifies empty jot is cancelled, not submitted
func TestJotEmptySubmit(t *testing.T) {
	store := &mockStorage{
		entries: map[string][]entry.Entry{},
		byID:    map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", DefaultTemplate: ""}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

	// Start jot
	started, _ := m.startJot()
	m = started.(pickerModel)

	// Don't type anything (empty)

	// Press Enter
	updated, cmd := m.updateJotInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(pickerModel)

	// Verify jot was cancelled (not submitted)
	if m.jotActive {
		t.Error("Empty jot should be cancelled")
	}

	// Verify no command was returned
	if cmd != nil {
		t.Error("Empty jot should not return a command")
	}
}

// --- Full-screen theme background tests ---

const testWidth = 80
const testHeight = 24

// assertViewFillsScreen verifies the View() output fills the terminal:
// - Exact height match (proves vertical filling by the full-screen wrapper)
// - Minimum width per line (proves horizontal filling; inner content may exceed it)
func assertViewFillsScreen(t *testing.T, output string, width, height int) {
	t.Helper()
	stripped := stripANSI(output)
	lines := strings.Split(stripped, "\n")
	if len(lines) != height {
		t.Errorf("expected %d lines, got %d", height, len(lines))
	}
	for i, line := range lines {
		if len(line) < width {
			t.Errorf("line %d: expected min width %d, got %d", i, width, len(line))
		}
	}
}

func TestViewFillsScreen_TodayEmpty(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {},
		},
		byID: map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)

	// Verify content is present
	stripped := stripANSI(output)
	if !strings.Contains(stripped, "Today") {
		t.Error("expected 'Today' in output")
	}
	if !strings.Contains(stripped, "Nothing yet today") {
		t.Error("expected empty state text in output")
	}
}

func TestViewFillsScreen_TodayWithEntries(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	e1 := entry.Entry{ID: "daily001", Content: "# Today's daily entry\nSome content here.", CreatedAt: today.Add(8 * time.Hour), UpdatedAt: today.Add(8 * time.Hour)}
	e2 := entry.Entry{ID: "entry002", Content: "Morning note", CreatedAt: today.Add(10 * time.Hour), UpdatedAt: today.Add(10 * time.Hour)}

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {e2, e1},
		},
		byID: map[string]entry.Entry{
			"daily001": e1, "entry002": e2,
		},
	}

	cfg := TUIConfig{Editor: "vi", Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	// Load today's entries
	msg := m.loadTodayCmd()
	updated, _ := m.Update(msg)
	m = updated.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_DateList(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_DayDetail(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	m.dayIdx = 0
	loaded, _ := m.loadDayDetail()
	m = loaded.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_EntryDetail(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	loaded, _ := m.loadEntryDetail("entry002")
	m = loaded.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_HelpOverlay(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days, presets["default-dark"])
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	m.helpActive = true
	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_DeletePrompt(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	testEntry := entry.Entry{ID: "entry01", Content: "Test entry", CreatedAt: today, UpdatedAt: today}
	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {},
		},
		byID: map[string]entry.Entry{
			"entry01": testEntry,
		},
	}

	cfg := TUIConfig{Editor: "vi", Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	m.deleteActive = true
	m.deleteEntry = testEntry

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)

	stripped := stripANSI(output)
	if !strings.Contains(stripped, "Delete entry") {
		t.Error("expected delete prompt text in output")
	}
}

func TestViewFillsScreen_JotInput(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {},
		},
		byID: map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	started, _ := m.startJot()
	m = started.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_ContextPanel(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	store := &mockStorage{
		contexts: []storage.Context{
			{ID: "ctx01", Name: "work", Source: "manual", CreatedAt: today, UpdatedAt: today},
		},
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {},
		},
		byID:          map[string]entry.Entry{},
		entryContexts: make(map[string][]string),
	}

	cfg := TUIConfig{Editor: "vi", Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = sized.(pickerModel)

	m.screen = screenContextPanel
	// Load contexts
	loadResult := m.loadContexts()
	if loadMsg, ok := loadResult.(contextsLoadedMsg); ok {
		updated, _ := m.Update(loadMsg)
		m = updated.(pickerModel)
	}

	output := m.View()
	assertViewFillsScreen(t, output, testWidth, testHeight)
}

func TestViewFillsScreen_AllThemes(t *testing.T) {
	// Verify that the full-screen wrapping works for all theme presets
	themeNames := []string{
		"default-dark", "default-light", "dracula",
		"ayu-dark", "ayu-light",
		"catppuccin-mocha", "catppuccin-latte",
		"gruvbox-dark", "gruvbox-light",
	}

	for _, name := range themeNames {
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

			store := &mockStorage{
				entries: map[string][]entry.Entry{
					today.Format("2006-01-02"): {},
				},
				byID: map[string]entry.Entry{},
			}

			cfg := TUIConfig{Editor: "vi", Theme: presets[name]}
			m := newTUIModel(store, cfg)
			m.screen = screenToday
			sized, _ := m.Update(tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
			m = sized.(pickerModel)

			output := m.View()
			assertViewFillsScreen(t, output, testWidth, testHeight)
		})
	}
}

func TestViewFillsScreen_WithMaxWidth(t *testing.T) {
	// When MaxWidth is set, the output should still fill the full terminal width
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	store := &mockStorage{
		entries: map[string][]entry.Entry{
			today.Format("2006-01-02"): {},
		},
		byID: map[string]entry.Entry{},
	}

	cfg := TUIConfig{Editor: "vi", MaxWidth: 60, Theme: presets["default-dark"]}
	m := newTUIModel(store, cfg)
	m.screen = screenToday
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: testHeight})
	m = sized.(pickerModel)

	output := m.View()
	assertViewFillsScreen(t, output, 100, testHeight)
}

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
	m.dayList = list.New(items, list.NewDefaultDelegate(), 80, 20)

	target := m.resolveJotTarget()
	if target == nil {
		t.Fatal("Expected jot target, got nil")
	}
	if target.ID != "target01" {
		t.Errorf("Expected target ID 'target01', got '%s'", target.ID)
	}
}

func TestResolveJotTarget_TodayDefaultsToDaily(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	dailyEntry := entry.Entry{
		ID:        "daily01",
		Content:   "# Daily\n\nContent",
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
		ID:        "daily01",
		Content:   "# Daily",
		CreatedAt: today.Add(8 * time.Hour),
		UpdatedAt: today.Add(8 * time.Hour),
	}
	otherEntry := entry.Entry{
		ID:        "other01",
		Content:   "# Other entry",
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
	m.todayList = list.New(items, list.NewDefaultDelegate(), 80, 20)

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

	// Set up day list BEFORE resize to avoid nil delegate panic
	items := []list.Item{entryItem{entry: otherEntry}, entryItem{entry: dailyEntry}}
	m.dayList = list.New(items, list.NewDefaultDelegate(), 80, 20)

	sized, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = sized.(pickerModel)

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
