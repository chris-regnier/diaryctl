package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

func dateLocal(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func dateLocalAt(year int, month time.Month, day, hour, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.Local)
}

func createEntryAt(t *testing.T, s storage.Storage, content string, at time.Time) entry.Entry {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	utc := at.UTC().Truncate(time.Second)
	e := entry.Entry{
		ID:        id,
		Content:   content,
		CreatedAt: utc,
		UpdatedAt: utc,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return e
}

func TestDailyDateFlagParsing(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{"valid from", "2026-01-15", "", false},
		{"valid to", "", "2026-01-31", false},
		{"valid both", "2026-01-01", "2026-01-31", false},
		{"empty both", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var start, end *time.Time
			if tt.from != "" {
				parsed, err := time.ParseInLocation("2006-01-02", tt.from, time.Local)
				if err != nil {
					if !tt.wantErr {
						t.Fatalf("unexpected parse error: %v", err)
					}
					return
				}
				start = &parsed
			}
			if tt.to != "" {
				parsed, err := time.ParseInLocation("2006-01-02", tt.to, time.Local)
				if err != nil {
					if !tt.wantErr {
						t.Fatalf("unexpected parse error: %v", err)
					}
					return
				}
				end = &parsed
			}
			if tt.wantErr {
				t.Fatal("expected error but none occurred")
			}
			// Verify parsed values
			if start != nil && start.Format("2006-01-02") != tt.from {
				t.Errorf("start = %s, want %s", start.Format("2006-01-02"), tt.from)
			}
			if end != nil && end.Format("2006-01-02") != tt.to {
				t.Errorf("end = %s, want %s", end.Format("2006-01-02"), tt.to)
			}
		})
	}
}

func TestDailyEmptyStore(t *testing.T) {
	setupTestEnv(t)

	days, err := store.ListDays(storage.ListDaysOptions{})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}
	if len(days) != 0 {
		t.Errorf("expected empty days, got %d", len(days))
	}
}

func TestDailyListDaysWithDateRange(t *testing.T) {
	setupTestEnv(t)

	createEntryAt(t, store, "jan10", dateLocalAt(2026, 1, 10, 12, 0))
	createEntryAt(t, store, "jan15", dateLocalAt(2026, 1, 15, 12, 0))
	createEntryAt(t, store, "jan20", dateLocalAt(2026, 1, 20, 12, 0))

	start := dateLocal(2026, 1, 12)
	end := dateLocal(2026, 1, 18)
	days, err := store.ListDays(storage.ListDaysOptions{StartDate: &start, EndDate: &end})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(days))
	}
	if days[0].Date.Format("2006-01-02") != "2026-01-15" {
		t.Errorf("day = %s, want 2026-01-15", days[0].Date.Format("2006-01-02"))
	}
}

func TestDailyNonInteractivePlainText(t *testing.T) {
	setupTestEnv(t)

	createEntryAt(t, store, "Morning thoughts", dateLocalAt(2026, 1, 15, 9, 0))
	createEntryAt(t, store, "Afternoon notes", dateLocalAt(2026, 1, 15, 14, 0))
	createEntryAt(t, store, "Yesterday entry", dateLocalAt(2026, 1, 14, 18, 0))

	days, err := store.ListDays(storage.ListDaysOptions{})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}

	dayEntries := make([]ui.DayEntries, 0, len(days))
	for _, d := range days {
		date := d.Date
		entries, err := store.List(storage.ListOptions{Date: &date})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		dayEntries = append(dayEntries, ui.DayEntries{Date: d.Date, Entries: entries})
	}

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, dayEntries)
	output := buf.String()

	if !strings.Contains(output, "2026-01-15") {
		t.Errorf("expected date 2026-01-15 in output:\n%s", output)
	}
	if !strings.Contains(output, "2026-01-14") {
		t.Errorf("expected date 2026-01-14 in output:\n%s", output)
	}
	if !strings.Contains(output, "(2 entries)") {
		t.Errorf("expected (2 entries) in output:\n%s", output)
	}
	if !strings.Contains(output, "(1 entry)") {
		t.Errorf("expected (1 entry) in output:\n%s", output)
	}
}

func TestDailyNonInteractiveJSON(t *testing.T) {
	setupTestEnv(t)

	createEntryAt(t, store, "Entry one", dateLocalAt(2026, 1, 15, 10, 0))
	createEntryAt(t, store, "Entry two", dateLocalAt(2026, 1, 15, 14, 0))

	days, err := store.ListDays(storage.ListDaysOptions{})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}

	dayEntries := make([]ui.DayEntries, 0, len(days))
	for _, d := range days {
		date := d.Date
		entries, err := store.List(storage.ListOptions{Date: &date})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		dayEntries = append(dayEntries, ui.DayEntries{Date: d.Date, Entries: entries})
	}

	groups := ui.BuildDayGroups(dayEntries)
	var buf bytes.Buffer
	ui.FormatJSON(&buf, groups)

	var result []ui.DayGroupJSON
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 day group, got %d", len(result))
	}
	if result[0].Date != "2026-01-15" {
		t.Errorf("date = %q, want 2026-01-15", result[0].Date)
	}
	if result[0].Count != 2 {
		t.Errorf("count = %d, want 2", result[0].Count)
	}
	if len(result[0].Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(result[0].Entries))
	}
}

func TestDailyNonInteractiveEmpty(t *testing.T) {
	setupTestEnv(t)

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, nil)
	if !strings.Contains(buf.String(), "No diary entries found") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}

func TestDailyNonInteractiveDateRangeFilter(t *testing.T) {
	setupTestEnv(t)

	createEntryAt(t, store, "jan10", dateLocalAt(2026, 1, 10, 12, 0))
	createEntryAt(t, store, "jan15", dateLocalAt(2026, 1, 15, 12, 0))
	createEntryAt(t, store, "jan20", dateLocalAt(2026, 1, 20, 12, 0))

	start := dateLocal(2026, 1, 12)
	end := dateLocal(2026, 1, 18)
	days, err := store.ListDays(storage.ListDaysOptions{StartDate: &start, EndDate: &end})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}

	dayEntries := make([]ui.DayEntries, 0, len(days))
	for _, d := range days {
		date := d.Date
		entries, err := store.List(storage.ListOptions{Date: &date})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		dayEntries = append(dayEntries, ui.DayEntries{Date: d.Date, Entries: entries})
	}

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, dayEntries)
	output := buf.String()

	if !strings.Contains(output, "2026-01-15") {
		t.Errorf("expected 2026-01-15 in output:\n%s", output)
	}
	if strings.Contains(output, "2026-01-10") {
		t.Errorf("should not contain 2026-01-10 in output:\n%s", output)
	}
	if strings.Contains(output, "2026-01-20") {
		t.Errorf("should not contain 2026-01-20 in output:\n%s", output)
	}
}

func TestDailyNonInteractiveManyEntries(t *testing.T) {
	// T024: Non-interactive mode prints all entries for days with 50+ entries
	setupTestEnv(t)

	for i := 0; i < 55; i++ {
		createEntryAt(t, store, fmt.Sprintf("Entry %d", i),
			dateLocalAt(2026, 1, 15, 8, i))
	}

	days, err := store.ListDays(storage.ListDaysOptions{})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(days))
	}
	if days[0].Count != 55 {
		t.Errorf("count = %d, want 55", days[0].Count)
	}

	dayEntries := make([]ui.DayEntries, 0, len(days))
	for _, d := range days {
		date := d.Date
		entries, err := store.List(storage.ListOptions{Date: &date})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		dayEntries = append(dayEntries, ui.DayEntries{Date: d.Date, Entries: entries})
	}

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, dayEntries)
	output := buf.String()

	if !strings.Contains(output, "(55 entries)") {
		t.Errorf("expected (55 entries) in output:\n%s", output[:min(200, len(output))])
	}
	// Count entry lines (lines starting with spaces containing entry IDs)
	lines := strings.Split(output, "\n")
	entryLines := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "  ") && len(strings.TrimSpace(l)) > 0 {
			entryLines++
		}
	}
	if entryLines != 55 {
		t.Errorf("expected 55 entry lines, got %d", entryLines)
	}
}

func TestDailyEmptyDateRange(t *testing.T) {
	// T026: Date range with no matching entries
	setupTestEnv(t)

	createEntryAt(t, store, "jan10", dateLocalAt(2026, 1, 10, 12, 0))
	createEntryAt(t, store, "jan20", dateLocalAt(2026, 1, 20, 12, 0))

	start := dateLocal(2026, 1, 12)
	end := dateLocal(2026, 1, 18)
	days, err := store.ListDays(storage.ListDaysOptions{StartDate: &start, EndDate: &end})
	if err != nil {
		t.Fatalf("ListDays: %v", err)
	}
	if len(days) != 0 {
		t.Errorf("expected 0 days, got %d", len(days))
	}

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, nil)
	if !strings.Contains(buf.String(), "No diary entries found") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}
