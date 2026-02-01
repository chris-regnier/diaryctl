package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	store = setupTestStore(t)
	appConfig = &config.Config{}
	jsonOutput = false
}

func TestCreateInline(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{ID: id, Content: "Test inline entry", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Content != "Test inline entry" {
		t.Errorf("content = %q, want %q", got.Content, "Test inline entry")
	}
}

func TestCreateEmptyContentRejected(t *testing.T) {
	err := entry.ValidateContent("   ")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestCreateJSONOutput(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC()
	e := entry.Entry{ID: id, Content: "JSON test", CreatedAt: now, UpdatedAt: now}
	if err := store.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var buf bytes.Buffer
	ui.FormatJSON(&buf, e)

	var result entry.Entry
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if result.ID != id {
		t.Errorf("id = %q, want %q", result.ID, id)
	}
	if result.Content != "JSON test" {
		t.Errorf("content = %q, want %q", result.Content, "JSON test")
	}
}
