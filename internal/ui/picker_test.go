package ui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// mockStorage implements StorageProvider for testing.
type mockStorage struct {
	days        []storage.DaySummary
	entries     map[string][]entry.Entry
	byID        map[string]entry.Entry
	attachError error // Add this field
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
		return e, nil
	}
	return entry.Entry{}, storage.ErrNotFound
}

// Write methods (stubs for test - not used by current tests)
func (m *mockStorage) Create(e entry.Entry) error {
	return nil
}

func (m *mockStorage) Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error) {
	return entry.Entry{}, nil
}

func (m *mockStorage) Delete(id string) error {
	return nil
}

// Context methods (stubs for test - not used by current tests)
func (m *mockStorage) ListContexts() ([]storage.Context, error) {
	return nil, nil
}

func (m *mockStorage) CreateContext(c storage.Context) error {
	return nil
}

func (m *mockStorage) AttachContext(entryID string, contextID string) error {
	if m.attachError != nil {
		return m.attachError
	}
	// ... existing implementation ...
	return nil
}

func (m *mockStorage) DetachContext(entryID string, contextID string) error {
	return nil
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
	m := newPickerModel(mock, days)

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

	m := newPickerModel(mock, mock.days)
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

	m := newPickerModel(mock, mock.days)
	// With empty days, RunPicker would print message and not launch Bubble Tea
	// Here we verify the model handles empty state gracefully
	if len(m.days) != 0 {
		t.Errorf("expected 0 days, got %d", len(m.days))
	}
}

func TestPickerScreenTransitions(t *testing.T) {
	mock, days := makeTestDays()
	m := newPickerModel(mock, days)

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
		entries: map[string][]entry.Entry{},
		byID:    map[string]entry.Entry{},
	}
	// Make AttachContext return an error
	store.attachError = fmt.Errorf("database connection failed")

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
