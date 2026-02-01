package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

func TestDeleteConfirmed(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "delete me", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := s.Get(id)
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := setupTestStore(t)
	err := s.Delete("nonexist")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteJSONOutput(t *testing.T) {
	result := ui.DeleteResult{ID: "abc12345", Deleted: true}
	var buf bytes.Buffer
	ui.FormatJSON(&buf, result)

	var got ui.DeleteResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if !got.Deleted {
		t.Error("expected deleted=true")
	}
	if got.ID != "abc12345" {
		t.Errorf("id = %q, want %q", got.ID, "abc12345")
	}
}
