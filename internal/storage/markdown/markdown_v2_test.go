package markdown_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/storage"
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

// TestMarkdownV2_CreateTemplate verifies that CreateTemplate saves a template and it can be retrieved.
func TestMarkdownV2_CreateTemplate(t *testing.T) {
	// Setup: Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "diaryctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	store, err := markdown.NewV2(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Test: Create a template
	now := time.Now()
	tmpl := storage.Template{
		ID:         "tmpl0001",
		Name:       "Daily Standup",
		Content:    "What did I accomplish?\n\nWhat will I do today?\n\nAny blockers?",
		Attributes: map[string]string{"type": "standup"},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = store.CreateTemplate(tmpl)
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}

	// Verify: Template can be retrieved via GetTemplate
	retrieved, err := store.GetTemplate("tmpl0001")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	if retrieved.ID != tmpl.ID {
		t.Errorf("Expected template ID %s, got %s", tmpl.ID, retrieved.ID)
	}
	if retrieved.Name != tmpl.Name {
		t.Errorf("Expected template name %q, got %q", tmpl.Name, retrieved.Name)
	}
	if retrieved.Content != tmpl.Content {
		t.Errorf("Expected template content %q, got %q", tmpl.Content, retrieved.Content)
	}
	if retrieved.Attributes["type"] != "standup" {
		t.Errorf("Expected type attribute 'standup', got %q", retrieved.Attributes["type"])
	}

	// Verify: Template file was created
	tmplPath := filepath.Join(tmpDir, "templates", "tmpl0001.json")
	if _, err := os.Stat(tmplPath); err != nil {
		t.Errorf("Expected template file to exist at %s", tmplPath)
	}
}

// TestMarkdownV2_GetTemplateByName verifies that GetTemplateByName finds a template by name.
func TestMarkdownV2_GetTemplateByName(t *testing.T) {
	// Setup: Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "diaryctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	store, err := markdown.NewV2(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Setup: Create multiple templates
	now := time.Now()
	templates := []storage.Template{
		{
			ID:         "tmpl0001",
			Name:       "Daily Standup",
			Content:    "Standup content",
			Attributes: map[string]string{"type": "standup"},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "tmpl0002",
			Name:       "Weekly Review",
			Content:    "Review content",
			Attributes: map[string]string{"type": "review"},
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}

	for _, tmpl := range templates {
		if err := store.CreateTemplate(tmpl); err != nil {
			t.Fatalf("CreateTemplate failed: %v", err)
		}
	}

	// Test: Get template by name
	retrieved, err := store.GetTemplateByName("Weekly Review")
	if err != nil {
		t.Fatalf("GetTemplateByName failed: %v", err)
	}

	if retrieved.ID != "tmpl0002" {
		t.Errorf("Expected template ID 'tmpl0002', got %s", retrieved.ID)
	}
	if retrieved.Name != "Weekly Review" {
		t.Errorf("Expected template name 'Weekly Review', got %q", retrieved.Name)
	}

	// Test: Get template by non-existent name
	_, err = store.GetTemplateByName("Non-existent")
	if err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent template, got %v", err)
	}
}
