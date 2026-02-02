package daily

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

func testStore(t *testing.T) storage.Storage {
	t.Helper()
	s, err := markdown.New(t.TempDir())
	if err != nil {
		t.Fatalf("markdown.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestGetOrCreateToday_CreatesNewEntry(t *testing.T) {
	s := testStore(t)
	e, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !created {
		t.Error("expected created=true for new entry")
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	// Default content should be a date header
	today := time.Now().Format("2006-01-02")
	wantContent := fmt.Sprintf("# %s", today)
	if e.Content != wantContent {
		t.Errorf("expected default content %q, got %q", wantContent, e.Content)
	}
	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("got ID=%q, want %q", got.ID, e.ID)
	}
}

func TestGetOrCreateToday_FindsExistingEntry(t *testing.T) {
	s := testStore(t)
	now := time.Now().UTC()
	id, err := entry.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	e := entry.Entry{
		ID:        id,
		Content:   "existing daily entry",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if created {
		t.Error("expected created=false for existing entry")
	}
	if got.ID != e.ID {
		t.Errorf("got ID=%q, want %q", got.ID, e.ID)
	}
}

func TestGetOrCreateToday_WithDefaultTemplate(t *testing.T) {
	s := testStore(t)
	tmplID, err := entry.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	tmpl := storage.Template{
		ID:        tmplID,
		Name:      "daily",
		Content:   "# Daily Log\n\n",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	e, created, err := GetOrCreateToday(s, "daily")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	// Storage trims whitespace on read, so check trimmed content
	if !strings.Contains(e.Content, "# Daily Log") {
		t.Errorf("expected template content containing '# Daily Log', got %q", e.Content)
	}
	if len(e.Templates) != 1 || e.Templates[0].TemplateName != "daily" {
		t.Errorf("expected template attribution, got %v", e.Templates)
	}
}

func TestGetOrCreateToday_BadDefaultTemplateWarns(t *testing.T) {
	s := testStore(t)
	e, created, err := GetOrCreateToday(s, "nonexistent")
	if err != nil {
		t.Fatalf("GetOrCreateToday should not error on bad default template: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	// When template not found, falls back to default date-header content
	today := time.Now().Format("2006-01-02")
	wantContent := fmt.Sprintf("# %s", today)
	if e.Content != wantContent {
		t.Errorf("expected default content %q, got %q", wantContent, e.Content)
	}
}

func TestGetOrCreateToday_MultipleEntries_ReturnsNewest(t *testing.T) {
	s := testStore(t)

	now := time.Now().UTC()
	oldID, err := entry.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	old := entry.Entry{
		ID:        oldID,
		Content:   "older entry",
		CreatedAt: now.Add(-1 * time.Hour),
		UpdatedAt: now.Add(-1 * time.Hour),
	}
	newerID, err := entry.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	newer := entry.Entry{
		ID:        newerID,
		Content:   "newer entry",
		CreatedAt: now,
		UpdatedAt: now,
	}
	_ = s.Create(old)
	_ = s.Create(newer)

	got, created, err := GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if created {
		t.Error("expected created=false")
	}
	// Should return the newest entry for today (List returns DESC by created_at, Limit 1)
	if got.ID != newer.ID {
		t.Errorf("expected newest entry %q, got %q", newer.ID, got.ID)
	}
}
