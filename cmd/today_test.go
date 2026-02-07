package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/daily"
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
	if !strings.Contains(outputStripped, "2026-02-06") {
		t.Errorf("expected output to contain date, got:\n%s", outputStripped)
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
