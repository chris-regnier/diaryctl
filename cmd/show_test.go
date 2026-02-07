package cmd

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
)

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func TestShowFullContent(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "Full diary entry content here", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	var buf bytes.Buffer
	ui.FormatEntryFull(&buf, got)
	output := buf.String()
	// Strip ANSI codes for testing since markdown rendering adds color codes
	outputStripped := stripANSI(output)

	if !strings.Contains(outputStripped, "Entry: "+id) {
		t.Error("missing entry ID in output")
	}
	if !strings.Contains(outputStripped, "Full diary entry content here") {
		t.Error("missing content in output")
	}
	if !strings.Contains(outputStripped, "Created:") {
		t.Error("missing Created header")
	}
}

func TestShowNotFound(t *testing.T) {
	s := setupTestStore(t)

	_, err := s.Get("nonexist")
	if err != storage.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestShowTemplateAttribution(t *testing.T) {
	setupTestEnv(t)

	tmpl := createTestTemplate(t, "daily", "# Daily")

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{
		ID: id, Content: "entry with templates", CreatedAt: now, UpdatedAt: now,
		Templates: []entry.TemplateRef{
			{TemplateID: tmpl.ID, TemplateName: "daily"},
		},
	}
	store.Create(e)

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	var buf bytes.Buffer
	ui.FormatEntryFull(&buf, got)
	output := buf.String()

	if !strings.Contains(output, "Templates: daily") {
		t.Errorf("expected 'Templates: daily' in output:\n%s", output)
	}
}

func TestShowNoTemplateAttribution(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "no templates", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	got, _ := s.Get(id)
	var buf bytes.Buffer
	ui.FormatEntryFull(&buf, got)

	if strings.Contains(buf.String(), "Templates:") {
		t.Errorf("should not show Templates line for entry without templates:\n%s", buf.String())
	}
}

func TestShowIDOnly(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "id-only test content", CreatedAt: now, UpdatedAt: now}
	store.Create(e)

	showIDOnly = true
	t.Cleanup(func() { showIDOnly = false })

	var buf bytes.Buffer
	showCmd.SetOut(&buf)
	if err := showCmd.RunE(showCmd, []string{id}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != id {
		t.Errorf("output = %q, want %q", output, id)
	}
}

func TestShowContentOnly(t *testing.T) {
	setupTestEnv(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "content-only test body", CreatedAt: now, UpdatedAt: now}
	store.Create(e)

	showContentOnly = true
	t.Cleanup(func() { showContentOnly = false })

	var buf bytes.Buffer
	showCmd.SetOut(&buf)
	if err := showCmd.RunE(showCmd, []string{id}); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != "content-only test body" {
		t.Errorf("output = %q, want %q", output, "content-only test body")
	}
}

func TestShowJSONOutput(t *testing.T) {
	s := setupTestStore(t)

	id, _ := entry.NewID()
	now := time.Now().UTC().Truncate(time.Second)
	e := entry.Entry{ID: id, Content: "JSON show content", CreatedAt: now, UpdatedAt: now}
	s.Create(e)

	got, _ := s.Get(id)
	var buf bytes.Buffer
	ui.FormatJSON(&buf, got)

	var result entry.Entry
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if result.Content != "JSON show content" {
		t.Errorf("content = %q, want %q", result.Content, "JSON show content")
	}
}
