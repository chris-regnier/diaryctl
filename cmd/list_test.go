package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

func TestListReverseChronological(t *testing.T) {
	s := setupTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		id, _ := entry.NewID()
		e := entry.Entry{
			ID:        id,
			Content:   "Entry " + string(rune('A'+i)),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := s.Create(e); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	entries, err := s.List(storage.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Newest first
	for i := 1; i < len(entries); i++ {
		if entries[i].CreatedAt.After(entries[i-1].CreatedAt) {
			t.Errorf("entries not in reverse chronological order at index %d", i)
		}
	}
}

func TestListDateFilter(t *testing.T) {
	s := setupTestStore(t)

	today := time.Now().UTC().Truncate(time.Second)
	yesterday := today.Add(-24 * time.Hour)

	id1, _ := entry.NewID()
	e1 := entry.Entry{ID: id1, Content: "today", CreatedAt: today, UpdatedAt: today}
	s.Create(e1)

	id2, _ := entry.NewID()
	e2 := entry.Entry{ID: id2, Content: "yesterday", CreatedAt: yesterday, UpdatedAt: yesterday}
	s.Create(e2)

	filterDate := today.Local()
	entries, err := s.List(storage.ListOptions{Date: &filterDate})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "today" {
		t.Errorf("expected today entry, got %q", entries[0].Content)
	}
}

func TestListEmptyMessage(t *testing.T) {
	s := setupTestStore(t)
	entries, _ := s.List(storage.ListOptions{})

	var buf bytes.Buffer
	ui.FormatEntryList(&buf, entries)
	if !strings.Contains(buf.String(), "No diary entries found") {
		t.Errorf("expected empty message, got %q", buf.String())
	}
}

func TestListTemplateFilter(t *testing.T) {
	setupTestEnv(t)

	tmpl := createTestTemplate(t, "daily", "# Daily")

	// Entry with template
	id1, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e1 := entry.Entry{
		ID: id1, Content: "with template", CreatedAt: now, UpdatedAt: now,
		Templates: []entry.TemplateRef{{TemplateID: tmpl.ID, TemplateName: "daily"}},
	}
	store.Create(e1)

	// Entry without template
	id2, _ := entry.NewID()
	e2 := entry.Entry{ID: id2, Content: "without template", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}
	store.Create(e2)

	// Filter by template
	entries, err := store.List(storage.ListOptions{TemplateName: "daily"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != id1 {
		t.Errorf("expected 1 entry with template 'daily', got %d", len(entries))
	}

	// Without filter returns both
	all, _ := store.List(storage.ListOptions{})
	if len(all) != 2 {
		t.Errorf("expected 2 entries without filter, got %d", len(all))
	}
}

func TestListIDOnly(t *testing.T) {
	setupTestEnv(t)

	now := time.Now().UTC().Truncate(time.Second)
	var ids []string
	for i := 0; i < 3; i++ {
		id, _ := entry.NewID()
		ids = append(ids, id)
		e := entry.Entry{
			ID:        id,
			Content:   "Entry " + string(rune('A'+i)),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := store.Create(e); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	listIDOnly = true
	t.Cleanup(func() { listIDOnly = false })

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	if err := listCmd.RunE(listCmd, []string{}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Each line should be one of the IDs we created
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	for _, line := range lines {
		if !idSet[strings.TrimSpace(line)] {
			t.Errorf("unexpected ID in output: %q", line)
		}
	}
}

func TestListJSONOutput(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "json list test", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	entries, _ := s.List(storage.ListOptions{})
	summaries := ui.ToSummaries(entries)

	var buf bytes.Buffer
	ui.FormatJSON(&buf, summaries)

	var result []ui.EntrySummary
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}
	if result[0].ID != id {
		t.Errorf("id = %q, want %q", result[0].ID, id)
	}
}
