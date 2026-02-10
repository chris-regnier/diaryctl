package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

// setupTestStoreV2 creates a test MarkdownV2 storage instance
func setupTestStoreV2(t *testing.T) storage.StorageV2 {
	t.Helper()
	dir := t.TempDir()
	s, err := markdown.NewV2(dir)
	if err != nil {
		t.Fatalf("creating test storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestJotV2Command tests the jot command with the v2 data model
func TestJotV2Command(t *testing.T) {
	store := setupTestStoreV2(t)

	// Create root command with jot subcommand
	rootCmd := NewRootV2Command(store)

	// Set up command arguments
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "bought groceries"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify output contains block ID and date
	outputStr := output.String()
	if !strings.Contains(outputStr, "Created block") {
		t.Errorf("expected output to contain 'Created block', got: %s", outputStr)
	}

	// Verify block was created
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.Content != "bought groceries" {
		t.Errorf("expected content 'bought groceries', got: %s", b.Content)
	}

	// Verify block ID is valid
	if err := block.ValidateID(b.ID); err != nil {
		t.Errorf("invalid block ID %q: %v", b.ID, err)
	}

	// Verify timestamps are set
	if b.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if b.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if !b.CreatedAt.Equal(b.UpdatedAt) {
		t.Error("expected CreatedAt and UpdatedAt to be equal for new block")
	}
}

// TestJotV2CommandWithDate tests jot with custom date
func TestJotV2CommandWithDate(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "--date", "2024-01-15", "test content"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify block was created on specified date
	targetDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local)
	blocks, err := store.ListBlocks(targetDate)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	if blocks[0].Content != "test content" {
		t.Errorf("expected content 'test content', got: %s", blocks[0].Content)
	}
}

// TestJotV2CommandWithAttributes tests jot with custom attributes
func TestJotV2CommandWithAttributes(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "--attr", "type=note", "--attr", "mood=happy", "feeling good today"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify block has attributes
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	if b.Attributes["type"] != "note" {
		t.Errorf("expected type=note, got: %s", b.Attributes["type"])
	}
	if b.Attributes["mood"] != "happy" {
		t.Errorf("expected mood=happy, got: %s", b.Attributes["mood"])
	}
}

// TestJotV2CommandMultipleWords tests that multiple args are joined
func TestJotV2CommandMultipleWords(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "this", "is", "a", "multi", "word", "note"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	expectedContent := "this is a multi word note"
	if blocks[0].Content != expectedContent {
		t.Errorf("expected content %q, got: %q", expectedContent, blocks[0].Content)
	}
}

// TestJotV2CommandNoArgs tests that jot requires content
func TestJotV2CommandNoArgs(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for jot without content, got nil")
	}

	if !strings.Contains(err.Error(), "content") {
		t.Errorf("expected error about missing content, got: %v", err)
	}
}

// TestJotV2CommandInvalidDateFormat tests date validation
func TestJotV2CommandInvalidDateFormat(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "--date", "invalid-date", "test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid date format, got nil")
	}

	if !strings.Contains(err.Error(), "date") && !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected error about invalid date, got: %v", err)
	}
}

// TestJotV2CommandInvalidAttributeFormat tests attribute validation
func TestJotV2CommandInvalidAttributeFormat(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"jot", "--attr", "invalid", "test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid attribute format, got nil")
	}

	if !strings.Contains(err.Error(), "=") && !strings.Contains(err.Error(), "attribute") {
		t.Errorf("expected error about invalid attribute format, got: %v", err)
	}
}
