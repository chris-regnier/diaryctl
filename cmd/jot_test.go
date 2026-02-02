package cmd

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

func TestJotCreatesNewDailyEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	err := jotRun(io.Discard, "bought groceries", "")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !strings.Contains(e.Content, "bought groceries") {
		t.Errorf("expected content to contain 'bought groceries', got:\n%s", e.Content)
	}
}

func TestJotAppendsToExistingEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	// Create today's entry with some existing content
	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	originalContent := e.Content

	err = jotRun(io.Discard, "appended note", "")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	updated, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !strings.HasPrefix(updated.Content, originalContent) {
		t.Errorf("expected content to start with original %q, got:\n%s", originalContent, updated.Content)
	}
	if !strings.Contains(updated.Content, "appended note") {
		t.Errorf("expected content to contain 'appended note', got:\n%s", updated.Content)
	}
}

func TestJotTimestampFormat(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	err := jotRun(io.Discard, "check timestamp", "")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}

	// Expect a line like: - **HH:MM** check timestamp
	now := time.Now()
	expectedTime := now.Format("15:04")
	pattern := regexp.MustCompile(`- \*\*\d{2}:\d{2}\*\* check timestamp`)
	if !pattern.MatchString(e.Content) {
		t.Errorf("expected timestamp pattern in content, got:\n%s", e.Content)
	}
	if !strings.Contains(e.Content, "**"+expectedTime+"**") {
		t.Errorf("expected timestamp %s in content, got:\n%s", expectedTime, e.Content)
	}
}

func TestJotEmptyContentRejected(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	err := jotRun(io.Discard, "", "")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty content, got: %v", err)
	}
}

func TestJotFromStdin(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	// Test that jotRun works with content (simulating what stdin would provide)
	err := jotRun(io.Discard, "stdin content here", "")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if !strings.Contains(e.Content, "stdin content here") {
		t.Errorf("expected content to contain 'stdin content here', got:\n%s", e.Content)
	}
}

func TestJotWithTemplateFlag(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}

	// Create a template
	tmplID, err := entry.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	tmpl := storage.Template{
		ID:        tmplID,
		Name:      "worklog",
		Content:   "# Work Log\n\n",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = s.CreateTemplate(tmpl)

	// Jot with explicit template
	err = jotRun(io.Discard, "hello world", "worklog")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	entries, _ := s.List(storage.ListOptions{Date: &today})
	if !strings.Contains(entries[0].Content, "# Work Log") {
		t.Error("expected template content in entry")
	}
	if !strings.Contains(entries[0].Content, "hello world") {
		t.Error("expected jot content appended")
	}
}

func TestJotJSONOutput(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	jsonOutput = true
	defer func() { jsonOutput = false }()

	var buf bytes.Buffer
	err := jotRun(&buf, "test note", "")
	if err != nil {
		t.Fatalf("jotRun: %v", err)
	}
	if !strings.Contains(buf.String(), `"content"`) {
		t.Error("expected JSON output")
	}
	if !strings.Contains(buf.String(), "test note") {
		t.Error("expected jot content in JSON")
	}
}
