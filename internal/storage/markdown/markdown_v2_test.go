package markdown_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

// TestMarkdownV2_GetDay verifies that GetDay creates an empty day if it doesn't exist.
func TestMarkdownV2_GetDay(t *testing.T) {
	// Setup: Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "diaryctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	storage, err := markdown.NewV2(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test: Get a day that doesn't exist
	date := time.Date(2026, 2, 9, 0, 0, 0, 0, time.Local)
	d, err := storage.GetDay(date)
	if err != nil {
		t.Fatalf("GetDay failed: %v", err)
	}

	// Verify: Day is empty but initialized
	if !d.Date.Equal(date) {
		t.Errorf("Expected date %v, got %v", date, d.Date)
	}
	if len(d.Blocks) != 0 {
		t.Errorf("Expected 0 blocks, got %d", len(d.Blocks))
	}

	// Verify: File was not created (lazy creation)
	dayPath := filepath.Join(tmpDir, "days", "2026-02-09.json")
	if _, err := os.Stat(dayPath); err == nil {
		t.Error("Expected day file to NOT exist yet (lazy creation)")
	}
}

// TestMarkdownV2_CreateBlock verifies that CreateBlock saves a block and it can be retrieved.
func TestMarkdownV2_CreateBlock(t *testing.T) {
	// Setup: Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "diaryctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	storage, err := markdown.NewV2(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Test: Create a block
	date := time.Date(2026, 2, 9, 0, 0, 0, 0, time.Local)
	now := time.Now()
	blk := block.Block{
		ID:         block.NewID(),
		Content:    "Test block content",
		CreatedAt:  now,
		UpdatedAt:  now,
		Attributes: map[string]string{"type": "note"},
	}

	err = storage.CreateBlock(date, blk)
	if err != nil {
		t.Fatalf("CreateBlock failed: %v", err)
	}

	// Verify: Block can be retrieved via GetDay
	d, err := storage.GetDay(date)
	if err != nil {
		t.Fatalf("GetDay failed: %v", err)
	}

	if len(d.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(d.Blocks))
	}

	retrievedBlock := d.Blocks[0]
	if retrievedBlock.ID != blk.ID {
		t.Errorf("Expected block ID %s, got %s", blk.ID, retrievedBlock.ID)
	}
	if retrievedBlock.Content != blk.Content {
		t.Errorf("Expected content %q, got %q", blk.Content, retrievedBlock.Content)
	}
	if retrievedBlock.Attributes["type"] != "note" {
		t.Errorf("Expected type attribute 'note', got %q", retrievedBlock.Attributes["type"])
	}

	// Verify: Day file was created
	dayPath := filepath.Join(tmpDir, "days", "2026-02-09.json")
	if _, err := os.Stat(dayPath); err != nil {
		t.Errorf("Expected day file to exist at %s", dayPath)
	}
}
