package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// TestEditV2CommandHelp verifies the command structure and help text
func TestEditV2CommandHelp(t *testing.T) {
	store := setupTestStoreV2(t)

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"edit", "--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	helpText := output.String()

	// Verify command description is present
	if !strings.Contains(helpText, "edit") {
		t.Errorf("expected help text to contain 'edit', got: %s", helpText)
	}

	// Verify flags are documented
	requiredFlags := []string{"--template", "--var", "--attr", "--date"}
	for _, flag := range requiredFlags {
		if !strings.Contains(helpText, flag) {
			t.Errorf("expected help text to contain %q flag, got: %s", flag, helpText)
		}
	}
}

// TestEditV2CommandFlags verifies that flags are properly registered
func TestEditV2CommandFlags(t *testing.T) {
	store := setupTestStoreV2(t)

	cmd := NewEditV2Command(store)

	// Check that all expected flags exist
	flags := []string{"template", "var", "attr", "date"}
	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag %q to be registered", flagName)
		}
	}
}

// TestEditV2Command_WithTemplate is skipped in CI as it requires an interactive editor.
// To test locally: run with environment variable CI unset or set EDITOR to a test script.
func TestEditV2Command_WithTemplate(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("skipping interactive editor test in CI")
	}

	store := setupTestStoreV2(t)

	// Create a test template
	tmpl := storage.Template{
		ID:        "tmpl0001",
		Name:      "test-template",
		Content:   "# Meeting Notes\n\nDate: {{.date}}\nTopic: {{.topic}}",
		Attributes: map[string]string{"type": "meeting"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate() error = %v", err)
	}

	// Set up a test editor that just saves the file unchanged
	// This allows us to test template rendering without manual interaction
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "true") // 'true' command exits with 0, does nothing
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--template", "test-template",
		"--var", "date=2024-01-15",
		"--var", "topic=Project Review",
		"--attr", "priority=high",
	})

	err := rootCmd.Execute()
	// When using 'true' as editor, the temp file is preserved with the initial content
	// The editor.Edit function reads it back and sees the template-rendered content
	// This is valid - the user could save without changes
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify block was created with rendered template content
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	b := blocks[0]
	// Check that template was rendered
	if !strings.Contains(b.Content, "2024-01-15") {
		t.Errorf("expected content to contain rendered date '2024-01-15', got: %s", b.Content)
	}
	if !strings.Contains(b.Content, "Project Review") {
		t.Errorf("expected content to contain rendered topic 'Project Review', got: %s", b.Content)
	}

	// Check that attributes were merged (template + user)
	if b.Attributes["type"] != "meeting" {
		t.Errorf("expected type=meeting from template, got: %s", b.Attributes["type"])
	}
	if b.Attributes["priority"] != "high" {
		t.Errorf("expected priority=high from user, got: %s", b.Attributes["priority"])
	}
}

// TestEditV2CommandInvalidTemplate tests error handling for non-existent templates
func TestEditV2CommandInvalidTemplate(t *testing.T) {
	store := setupTestStoreV2(t)

	// Set editor to skip interaction
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "true")
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--template", "nonexistent-template",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent template, got nil")
	}

	if !strings.Contains(err.Error(), "template") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error about template not found, got: %v", err)
	}
}

// TestEditV2CommandInvalidVarFormat tests validation of --var flags
func TestEditV2CommandInvalidVarFormat(t *testing.T) {
	store := setupTestStoreV2(t)

	// Set editor to skip interaction
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "true")
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--var", "invalid-format",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid var format, got nil")
	}

	if !strings.Contains(err.Error(), "=") && !strings.Contains(err.Error(), "var") {
		t.Errorf("expected error about invalid var format, got: %v", err)
	}
}

// TestEditV2CommandInvalidAttrFormat tests validation of --attr flags
func TestEditV2CommandInvalidAttrFormat(t *testing.T) {
	store := setupTestStoreV2(t)

	// Set editor to skip interaction
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "true")
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--attr", "invalid-format",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid attr format, got nil")
	}

	if !strings.Contains(err.Error(), "=") && !strings.Contains(err.Error(), "attribute") {
		t.Errorf("expected error about invalid attribute format, got: %v", err)
	}
}

// TestEditV2CommandInvalidDateFormat tests date validation
func TestEditV2CommandInvalidDateFormat(t *testing.T) {
	store := setupTestStoreV2(t)

	// Set editor to skip interaction
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", "true")
	defer func() {
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--date", "invalid-date",
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid date format, got nil")
	}

	if !strings.Contains(err.Error(), "date") {
		t.Errorf("expected error about invalid date, got: %v", err)
	}
}

// mockEditor is a helper to simulate an editor that writes specific content
func mockEditor(t *testing.T, content string) func() {
	t.Helper()

	// Create a temporary script that writes content to the file
	script := "#!/bin/sh\necho '" + content + "' > \"$1\"\n"
	tmpfile, err := os.CreateTemp("", "test-editor-*.sh")
	if err != nil {
		t.Fatalf("failed to create temp editor script: %v", err)
	}

	if _, err := tmpfile.WriteString(script); err != nil {
		t.Fatalf("failed to write editor script: %v", err)
	}

	if err := tmpfile.Chmod(0755); err != nil {
		t.Fatalf("failed to chmod editor script: %v", err)
	}

	tmpfile.Close()

	// Set EDITOR to the script
	originalEditor := os.Getenv("EDITOR")
	os.Setenv("EDITOR", tmpfile.Name())

	// Return cleanup function
	return func() {
		os.Remove(tmpfile.Name())
		if originalEditor != "" {
			os.Setenv("EDITOR", originalEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}
}

// TestEditV2CommandSuccess tests successful block creation via edit command
func TestEditV2CommandSuccess(t *testing.T) {
	// Skip in CI environments since editor behavior may vary
	if os.Getenv("CI") != "" {
		t.Skip("skipping interactive editor test in CI")
	}

	store := setupTestStoreV2(t)

	cleanup := mockEditor(t, "Test content from editor")
	defer cleanup()

	rootCmd := NewRootV2Command(store)

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{
		"edit",
		"--attr", "type=note",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify output
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
	if !strings.Contains(b.Content, "Test content from editor") {
		t.Errorf("expected content to contain 'Test content from editor', got: %s", b.Content)
	}

	if b.Attributes["type"] != "note" {
		t.Errorf("expected type=note, got: %s", b.Attributes["type"])
	}

	// Verify block ID is valid
	if err := block.ValidateID(b.ID); err != nil {
		t.Errorf("invalid block ID %q: %v", b.ID, err)
	}
}
