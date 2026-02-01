package cmd

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

func TestEditContentUpdate(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "original", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	updated, err := s.Update(id, "edited content")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Content != "edited content" {
		t.Errorf("content = %q, want %q", updated.Content, "edited content")
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Error("updated_at should advance")
	}
}

func TestEditNotFound(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.Get("nonexist")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
