package cmd

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/daily"
)

func TestJotCreatesNewDailyEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DefaultTemplate = ""

	err := jotRun("bought groceries")
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

	err = jotRun("appended note")
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

	err := jotRun("check timestamp")
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

	err := jotRun("")
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
	err := jotRun("stdin content here")
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
