package storage_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	"github.com/chris-regnier/diaryctl/internal/storage/sqlite"
)

type storageFactory func(t *testing.T) storage.Storage

func markdownFactory(t *testing.T) storage.Storage {
	t.Helper()
	dir := t.TempDir()
	s, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("creating markdown storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func sqliteFactory(t *testing.T) storage.Storage {
	t.Helper()
	dir := t.TempDir()
	s, err := sqlite.New(dir)
	if err != nil {
		t.Fatalf("creating sqlite storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeEntry(t *testing.T, content string) entry.Entry {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return entry.Entry{
		ID:        id,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func makeEntryAt(t *testing.T, content string, at time.Time) entry.Entry {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	at = at.UTC().Truncate(time.Second)
	return entry.Entry{
		ID:        id,
		Content:   content,
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func dateLocal(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func dateLocalAt(year int, month time.Month, day, hour, min int) time.Time {
	return time.Date(year, month, day, hour, min, 0, 0, time.Local)
}

func runContractTests(t *testing.T, name string, factory storageFactory) {
	t.Run(name, func(t *testing.T) {
		t.Run("Create and Get", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "Hello diary")
			if err := s.Create(e); err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.Content != e.Content {
				t.Errorf("content = %q, want %q", got.Content, e.Content)
			}
		})

		t.Run("Create empty content", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "   ")
			err := s.Create(e)
			if err == nil {
				t.Fatal("expected validation error for empty content")
			}
			if !isValidationError(err) {
				t.Errorf("expected ErrValidation, got: %v", err)
			}
		})

		t.Run("Create sets timestamps", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "timestamps test")
			if err := s.Create(e); err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if !got.CreatedAt.Equal(got.UpdatedAt) {
				t.Errorf("created_at (%v) != updated_at (%v) on new entry", got.CreatedAt, got.UpdatedAt)
			}
		})

		t.Run("Get not found", func(t *testing.T) {
			s := factory(t)
			_, err := s.Get("nonexist")
			if err != storage.ErrNotFound {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})

		t.Run("List empty", func(t *testing.T) {
			s := factory(t)
			entries, err := s.List(storage.ListOptions{})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 0 {
				t.Errorf("expected empty list, got %d entries", len(entries))
			}
		})

		t.Run("List order", func(t *testing.T) {
			s := factory(t)
			var ids []string
			for i := 0; i < 3; i++ {
				e := makeEntry(t, "entry "+string(rune('A'+i)))
				// Stagger creation times
				e.CreatedAt = e.CreatedAt.Add(time.Duration(i) * time.Second)
				e.UpdatedAt = e.CreatedAt
				if err := s.Create(e); err != nil {
					t.Fatalf("Create %d: %v", i, err)
				}
				ids = append(ids, e.ID)
			}
			entries, err := s.List(storage.ListOptions{})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 3 {
				t.Fatalf("expected 3 entries, got %d", len(entries))
			}
			// Should be reverse chronological (newest first)
			if entries[0].ID != ids[2] {
				t.Errorf("first entry = %s, want %s (newest)", entries[0].ID, ids[2])
			}
		})

		t.Run("List date filter", func(t *testing.T) {
			s := factory(t)
			jan15 := dateLocalAt(2026, 1, 15, 12, 0)
			jan16 := dateLocalAt(2026, 1, 16, 12, 0)

			e1 := makeEntryAt(t, "jan15 entry", jan15)
			e2 := makeEntryAt(t, "jan16 entry", jan16)

			if err := s.Create(e1); err != nil {
				t.Fatalf("Create e1: %v", err)
			}
			if err := s.Create(e2); err != nil {
				t.Fatalf("Create e2: %v", err)
			}

			filterDate := dateLocal(2026, 1, 15)
			entries, err := s.List(storage.ListOptions{Date: &filterDate})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry for jan15, got %d", len(entries))
			}
			if entries[0].ID != e1.ID {
				t.Errorf("expected entry %s, got %s", e1.ID, entries[0].ID)
			}
		})

		t.Run("Update content", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "original content")
			e.CreatedAt = e.CreatedAt.Add(-time.Hour) // created an hour ago
			e.UpdatedAt = e.CreatedAt
			if err := s.Create(e); err != nil {
				t.Fatalf("Create: %v", err)
			}

			updated, err := s.Update(e.ID, "new content", nil)
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if updated.Content != "new content" {
				t.Errorf("content = %q, want %q", updated.Content, "new content")
			}
			if !updated.UpdatedAt.After(updated.CreatedAt) {
				t.Error("updated_at should be after created_at")
			}
			if !updated.CreatedAt.Equal(e.CreatedAt) {
				t.Error("created_at should be preserved")
			}
		})

		t.Run("Update not found", func(t *testing.T) {
			s := factory(t)
			_, err := s.Update("nonexist", "new content", nil)
			if err != storage.ErrNotFound {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})

		t.Run("Delete", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "delete me")
			if err := s.Create(e); err != nil {
				t.Fatalf("Create: %v", err)
			}
			if err := s.Delete(e.ID); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			_, err := s.Get(e.ID)
			if err != storage.ErrNotFound {
				t.Errorf("expected ErrNotFound after delete, got: %v", err)
			}
		})

		t.Run("Delete not found", func(t *testing.T) {
			s := factory(t)
			err := s.Delete("nonexist")
			if err != storage.ErrNotFound {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
		})

		t.Run("ID uniqueness", func(t *testing.T) {
			s := factory(t)
			seen := make(map[string]bool)
			for i := 0; i < 100; i++ {
				e := makeEntry(t, "uniqueness test")
				e.CreatedAt = e.CreatedAt.Add(time.Duration(i) * time.Millisecond)
				e.UpdatedAt = e.CreatedAt
				if seen[e.ID] {
					t.Fatalf("duplicate ID: %s", e.ID)
				}
				seen[e.ID] = true
				if err := s.Create(e); err != nil {
					t.Fatalf("Create %d: %v", i, err)
				}
			}
		})

		// TC-01: ListDays empty store
		t.Run("ListDays empty store", func(t *testing.T) {
			s := factory(t)
			days, err := s.ListDays(storage.ListDaysOptions{})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 0 {
				t.Errorf("expected empty slice, got %d", len(days))
			}
		})

		// TC-02: ListDays single day
		t.Run("ListDays single day", func(t *testing.T) {
			s := factory(t)
			jan15_9 := dateLocalAt(2026, 1, 15, 9, 0)
			jan15_12 := dateLocalAt(2026, 1, 15, 12, 0)
			jan15_15 := dateLocalAt(2026, 1, 15, 15, 0)

			for _, at := range []time.Time{jan15_9, jan15_12, jan15_15} {
				e := makeEntryAt(t, "entry at "+at.Format("15:04"), at)
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}

			days, err := s.ListDays(storage.ListDaysOptions{})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 1 {
				t.Fatalf("expected 1 day, got %d", len(days))
			}
			if days[0].Count != 3 {
				t.Errorf("count = %d, want 3", days[0].Count)
			}
			if days[0].Date.Format("2006-01-02") != "2026-01-15" {
				t.Errorf("date = %s, want 2026-01-15", days[0].Date.Format("2006-01-02"))
			}
		})

		// TC-03: ListDays multiple days
		t.Run("ListDays multiple days", func(t *testing.T) {
			s := factory(t)
			dates := []time.Time{
				dateLocalAt(2026, 1, 10, 12, 0),
				dateLocalAt(2026, 1, 12, 12, 0),
				dateLocalAt(2026, 1, 15, 12, 0),
			}
			for _, at := range dates {
				e := makeEntryAt(t, "entry on "+at.Format("2006-01-02"), at)
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}

			days, err := s.ListDays(storage.ListDaysOptions{})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 3 {
				t.Fatalf("expected 3 days, got %d", len(days))
			}
			// Reverse chronological
			if days[0].Date.Format("2006-01-02") != "2026-01-15" {
				t.Errorf("first day = %s, want 2026-01-15", days[0].Date.Format("2006-01-02"))
			}
			if days[1].Date.Format("2006-01-02") != "2026-01-12" {
				t.Errorf("second day = %s, want 2026-01-12", days[1].Date.Format("2006-01-02"))
			}
			if days[2].Date.Format("2006-01-02") != "2026-01-10" {
				t.Errorf("third day = %s, want 2026-01-10", days[2].Date.Format("2006-01-02"))
			}
		})

		// TC-04: ListDays with StartDate
		t.Run("ListDays with StartDate", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 12, 15} {
				e := makeEntryAt(t, "entry", dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			start := dateLocal(2026, 1, 12)
			days, err := s.ListDays(storage.ListDaysOptions{StartDate: &start})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 2 {
				t.Fatalf("expected 2 days, got %d", len(days))
			}
			if days[0].Date.Format("2006-01-02") != "2026-01-15" {
				t.Errorf("first = %s, want 2026-01-15", days[0].Date.Format("2006-01-02"))
			}
		})

		// TC-05: ListDays with EndDate
		t.Run("ListDays with EndDate", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 12, 15} {
				e := makeEntryAt(t, "entry", dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			end := dateLocal(2026, 1, 12)
			days, err := s.ListDays(storage.ListDaysOptions{EndDate: &end})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 2 {
				t.Fatalf("expected 2 days, got %d", len(days))
			}
			if days[0].Date.Format("2006-01-02") != "2026-01-12" {
				t.Errorf("first = %s, want 2026-01-12", days[0].Date.Format("2006-01-02"))
			}
		})

		// TC-06: ListDays with both StartDate and EndDate
		t.Run("ListDays with date range", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 12, 15} {
				e := makeEntryAt(t, "entry", dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			start := dateLocal(2026, 1, 11)
			end := dateLocal(2026, 1, 14)
			days, err := s.ListDays(storage.ListDaysOptions{StartDate: &start, EndDate: &end})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 1 {
				t.Fatalf("expected 1 day, got %d", len(days))
			}
			if days[0].Date.Format("2006-01-02") != "2026-01-12" {
				t.Errorf("day = %s, want 2026-01-12", days[0].Date.Format("2006-01-02"))
			}
		})

		// TC-07: ListDays date range with no entries
		t.Run("ListDays empty range", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 15} {
				e := makeEntryAt(t, "entry", dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			start := dateLocal(2026, 1, 11)
			end := dateLocal(2026, 1, 14)
			days, err := s.ListDays(storage.ListDaysOptions{StartDate: &start, EndDate: &end})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 0 {
				t.Errorf("expected empty, got %d", len(days))
			}
		})

		// TC-08: ListDays preview content
		t.Run("ListDays preview content", func(t *testing.T) {
			s := factory(t)
			earlier := makeEntryAt(t, "First", dateLocalAt(2026, 1, 15, 9, 0))
			later := makeEntryAt(t, "Second entry with more text", dateLocalAt(2026, 1, 15, 15, 0))

			if err := s.Create(earlier); err != nil {
				t.Fatalf("Create earlier: %v", err)
			}
			if err := s.Create(later); err != nil {
				t.Fatalf("Create later: %v", err)
			}

			days, err := s.ListDays(storage.ListDaysOptions{})
			if err != nil {
				t.Fatalf("ListDays: %v", err)
			}
			if len(days) != 1 {
				t.Fatalf("expected 1 day, got %d", len(days))
			}
			if !strings.Contains(days[0].Preview, "Second") {
				t.Errorf("preview = %q, want content from most recent entry", days[0].Preview)
			}
		})

		// TC-09: List with StartDate only
		t.Run("List with StartDate", func(t *testing.T) {
			s := factory(t)
			e1 := makeEntryAt(t, "jan10", dateLocalAt(2026, 1, 10, 12, 0))
			e2 := makeEntryAt(t, "jan15a", dateLocalAt(2026, 1, 15, 10, 0))
			e3 := makeEntryAt(t, "jan15b", dateLocalAt(2026, 1, 15, 14, 0))
			for _, e := range []entry.Entry{e1, e2, e3} {
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			start := dateLocal(2026, 1, 12)
			entries, err := s.List(storage.ListOptions{StartDate: &start})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 2 {
				t.Fatalf("expected 2 entries, got %d", len(entries))
			}
		})

		// TC-10: List with EndDate only
		t.Run("List with EndDate", func(t *testing.T) {
			s := factory(t)
			e1 := makeEntryAt(t, "jan10", dateLocalAt(2026, 1, 10, 12, 0))
			e2 := makeEntryAt(t, "jan15", dateLocalAt(2026, 1, 15, 12, 0))
			for _, e := range []entry.Entry{e1, e2} {
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			end := dateLocal(2026, 1, 12)
			entries, err := s.List(storage.ListOptions{EndDate: &end})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(entries))
			}
			if entries[0].ID != e1.ID {
				t.Errorf("expected %s, got %s", e1.ID, entries[0].ID)
			}
		})

		// TC-11: List with date range
		t.Run("List with date range", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 12, 15} {
				e := makeEntryAt(t, "jan"+string(rune('0'+d)), dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			start := dateLocal(2026, 1, 11)
			end := dateLocal(2026, 1, 13)
			entries, err := s.List(storage.ListOptions{StartDate: &start, EndDate: &end})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(entries))
			}
		})

		// TC-12: Date takes precedence over range
		t.Run("List Date precedence over range", func(t *testing.T) {
			s := factory(t)
			for _, d := range []int{10, 12, 15} {
				e := makeEntryAt(t, "entry", dateLocalAt(2026, 1, d, 12, 0))
				if err := s.Create(e); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			dateFilter := dateLocal(2026, 1, 12)
			start := dateLocal(2026, 1, 10)
			end := dateLocal(2026, 1, 15)
			entries, err := s.List(storage.ListOptions{Date: &dateFilter, StartDate: &start, EndDate: &end})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry (Date wins), got %d", len(entries))
			}
		})
	})
}

func makeTemplate(t *testing.T, name, content string) storage.Template {
	t.Helper()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("generating ID: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return storage.Template{
		ID:        id,
		Name:      name,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func runTemplateContractTests(t *testing.T, name string, factory storageFactory) {
	t.Run(name+" Template CRUD", func(t *testing.T) {

		t.Run("CreateTemplate and GetTemplate", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "daily", "# Daily Entry")
			err := s.CreateTemplate(tmpl)
			if err != nil {
				t.Fatalf("CreateTemplate: %v", err)
			}
			got, err := s.GetTemplate(tmpl.ID)
			if err != nil {
				t.Fatalf("GetTemplate: %v", err)
			}
			if got.Name != "daily" || got.Content != "# Daily Entry" {
				t.Errorf("got name=%q content=%q", got.Name, got.Content)
			}
		})

		t.Run("GetTemplateByName", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "prompts", "## Prompts\n\n- What happened today?\n")
			_ = s.CreateTemplate(tmpl)
			got, err := s.GetTemplateByName("prompts")
			if err != nil {
				t.Fatalf("GetTemplateByName: %v", err)
			}
			if got.ID != tmpl.ID {
				t.Errorf("got ID=%q want %q", got.ID, tmpl.ID)
			}
		})

		t.Run("GetTemplateByName not found", func(t *testing.T) {
			s := factory(t)
			_, err := s.GetTemplateByName("nonexistent")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("CreateTemplate duplicate name", func(t *testing.T) {
			s := factory(t)
			tmpl1 := makeTemplate(t, "daily", "content1")
			tmpl2 := makeTemplate(t, "daily", "content2")
			_ = s.CreateTemplate(tmpl1)
			err := s.CreateTemplate(tmpl2)
			if !errors.Is(err, storage.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})

		t.Run("ListTemplates", func(t *testing.T) {
			s := factory(t)
			_ = s.CreateTemplate(makeTemplate(t, "alpha", "a"))
			_ = s.CreateTemplate(makeTemplate(t, "beta", "b"))
			_ = s.CreateTemplate(makeTemplate(t, "gamma", "c"))
			list, err := s.ListTemplates()
			if err != nil {
				t.Fatalf("ListTemplates: %v", err)
			}
			if len(list) != 3 {
				t.Errorf("got %d templates, want 3", len(list))
			}
		})

		t.Run("UpdateTemplate", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "daily", "old content")
			_ = s.CreateTemplate(tmpl)
			updated, err := s.UpdateTemplate(tmpl.ID, "daily-v2", "new content")
			if err != nil {
				t.Fatalf("UpdateTemplate: %v", err)
			}
			if updated.Name != "daily-v2" || updated.Content != "new content" {
				t.Errorf("got name=%q content=%q", updated.Name, updated.Content)
			}
		})

		t.Run("UpdateTemplate not found", func(t *testing.T) {
			s := factory(t)
			_, err := s.UpdateTemplate("nonexist", "name", "content")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})

		t.Run("DeleteTemplate", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "daily", "content")
			_ = s.CreateTemplate(tmpl)
			err := s.DeleteTemplate(tmpl.ID)
			if err != nil {
				t.Fatalf("DeleteTemplate: %v", err)
			}
			_, err = s.GetTemplate(tmpl.ID)
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound after delete, got %v", err)
			}
		})

		t.Run("DeleteTemplate not found", func(t *testing.T) {
			s := factory(t)
			err := s.DeleteTemplate("nonexist")
			if !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})
}

func runAttributionContractTests(t *testing.T, name string, factory storageFactory) {
	t.Run(name+" Template Attribution", func(t *testing.T) {

		t.Run("Create entry with template refs", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "daily", "# Daily\n")
			_ = s.CreateTemplate(tmpl)

			e := makeEntry(t, "hello world")
			e.Templates = []entry.TemplateRef{
				{TemplateID: tmpl.ID, TemplateName: tmpl.Name},
			}
			err := s.Create(e)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if len(got.Templates) != 1 || got.Templates[0].TemplateName != "daily" {
				t.Errorf("expected 1 template ref 'daily', got %v", got.Templates)
			}
		})

		t.Run("Create entry without template refs", func(t *testing.T) {
			s := factory(t)
			e := makeEntry(t, "no templates")
			err := s.Create(e)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := s.Get(e.ID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if len(got.Templates) != 0 {
				t.Errorf("expected 0 template refs, got %v", got.Templates)
			}
		})

		t.Run("List entries filtered by template name", func(t *testing.T) {
			s := factory(t)
			tmpl := makeTemplate(t, "daily", "# Daily\n")
			_ = s.CreateTemplate(tmpl)

			e1 := makeEntry(t, "with template")
			e1.Templates = []entry.TemplateRef{
				{TemplateID: tmpl.ID, TemplateName: tmpl.Name},
			}
			_ = s.Create(e1)

			e2 := makeEntry(t, "without template")
			_ = s.Create(e2)

			opts := storage.ListOptions{TemplateName: "daily"}
			results, err := s.List(opts)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(results) != 1 || results[0].ID != e1.ID {
				t.Errorf("expected 1 entry with template 'daily', got %d", len(results))
			}
		})
	})
}

func isValidationError(err error) bool {
	return err != nil && (err == storage.ErrValidation ||
		(err.Error() != "" && containsValidation(err)))
}

func containsValidation(err error) bool {
	for e := err; e != nil; {
		if e == storage.ErrValidation {
			return true
		}
		if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapper.Unwrap()
		} else {
			return false
		}
	}
	return false
}

func TestMarkdownStorage(t *testing.T) {
	runContractTests(t, "Markdown", markdownFactory)
	runTemplateContractTests(t, "Markdown", markdownFactory)
	runAttributionContractTests(t, "Markdown", markdownFactory)
}

func TestSQLiteStorage(t *testing.T) {
	runContractTests(t, "SQLite", sqliteFactory)
	runTemplateContractTests(t, "SQLite", sqliteFactory)
	runAttributionContractTests(t, "SQLite", sqliteFactory)
}
