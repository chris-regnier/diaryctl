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

func TestUpdateInline(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "original", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	updated, err := s.Update(id, "updated inline")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Content != "updated inline" {
		t.Errorf("content = %q, want %q", updated.Content, "updated inline")
	}
}

func TestUpdateTimestampPreservation(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	created := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "original", CreatedAt: created, UpdatedAt: created}
	s.Create(e)

	updated, _ := s.Update(id, "new content")
	if !updated.CreatedAt.Equal(created) {
		t.Errorf("created_at changed: got %v, want %v", updated.CreatedAt, created)
	}
	if !updated.UpdatedAt.After(created) {
		t.Error("updated_at should advance")
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.Update("nonexist", "content")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateJSONOutput(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "original", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	updated, _ := s.Update(id, "json update test")

	var buf bytes.Buffer
	ui.FormatJSON(&buf, updated)

	var result entry.Entry
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if result.Content != "json update test" {
		t.Errorf("content = %q, want %q", result.Content, "json update test")
	}
}
