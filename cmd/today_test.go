package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

func TestTodayShowsExistingEntry(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}

	// Create today's entry first
	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}

	var buf bytes.Buffer
	err = todayRun(&buf, false, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	output := buf.String()
	// Strip ANSI codes for testing since markdown rendering adds color codes
	outputStripped := stripANSI(output)

	if !strings.Contains(outputStripped, e.ID) {
		t.Errorf("expected output to contain entry ID %q, got:\n%s", e.ID, outputStripped)
	}
	// Content might be transformed by markdown rendering, so just check for date pattern
	expectedDate := time.Now().Format("2006-01-02")
	if !strings.Contains(outputStripped, expectedDate) {
		t.Errorf("expected output to contain date %q, got:\n%s", expectedDate, outputStripped)
	}
}

func TestTodayCreatesEntryIfMissing(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}

	var buf bytes.Buffer
	err := todayRun(&buf, false, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty output after creating today's entry")
	}

	// Verify the entry was actually created
	e, created, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}
	if created {
		t.Error("expected entry to already exist (created=false), but got created=true")
	}
	if !strings.Contains(output, e.ID) {
		t.Errorf("expected output to contain entry ID %q, got:\n%s", e.ID, output)
	}
}

func TestTodayIDOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}

	// Create today's entry
	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}

	var buf bytes.Buffer
	err = todayRun(&buf, true, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != e.ID {
		t.Errorf("expected ID %q, got %q", e.ID, output)
	}
}

func TestTodayContentOnly(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}

	// Create today's entry
	e, _, err := daily.GetOrCreateToday(s, "")
	if err != nil {
		t.Fatalf("GetOrCreateToday: %v", err)
	}

	var buf bytes.Buffer
	err = todayRun(&buf, false, true)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	if output != strings.TrimSpace(e.Content) {
		t.Errorf("expected content %q, got %q", e.Content, output)
	}
}

func TestTodayAttachesContexts(t *testing.T) {
	s := setupTestStore(t)
	store = s
	appConfig = &config.Config{}
	appConfig.DataDir = t.TempDir()
	appConfig.ContextResolvers = []string{}

	ctx := storage.Context{
		ID:        "ctx001",
		Name:      "daily-work",
		Source:    "manual",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.CreateContext(ctx); err != nil {
		t.Fatalf("CreateContext: %v", err)
	}
	if err := context.SetManualContext(appConfig.DataDir, "daily-work"); err != nil {
		t.Fatalf("SetManualContext: %v", err)
	}

	var buf bytes.Buffer
	err := todayRun(&buf, false, false)
	if err != nil {
		t.Fatalf("todayRun: %v", err)
	}

	e, _, _ := daily.GetOrCreateToday(s, "")
	got, err := s.Get(e.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(got.Contexts))
	}
	if got.Contexts[0].ContextName != "daily-work" {
		t.Errorf("context = %q, want daily-work", got.Contexts[0].ContextName)
	}
}
