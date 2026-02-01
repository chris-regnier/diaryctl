package storage_test

import (
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
			today := time.Now().UTC().Truncate(time.Second)
			yesterday := today.Add(-24 * time.Hour)

			e1 := makeEntry(t, "today entry")
			e1.CreatedAt = today
			e1.UpdatedAt = today

			e2 := makeEntry(t, "yesterday entry")
			e2.CreatedAt = yesterday
			e2.UpdatedAt = yesterday

			if err := s.Create(e1); err != nil {
				t.Fatalf("Create e1: %v", err)
			}
			if err := s.Create(e2); err != nil {
				t.Fatalf("Create e2: %v", err)
			}

			filterDate := today.Local()
			entries, err := s.List(storage.ListOptions{Date: &filterDate})
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected 1 entry for today, got %d", len(entries))
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

			updated, err := s.Update(e.ID, "new content")
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
			_, err := s.Update("nonexist", "new content")
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
}

func TestSQLiteStorage(t *testing.T) {
	runContractTests(t, "SQLite", sqliteFactory)
}
