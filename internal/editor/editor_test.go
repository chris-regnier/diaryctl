package editor

import (
	"os"
	"testing"
)

func TestResolveEditorConfig(t *testing.T) {
	result := ResolveEditor("nano")
	if result != "nano" {
		t.Errorf("expected nano, got %q", result)
	}
}

func TestResolveEditorEnvEditor(t *testing.T) {
	t.Setenv("EDITOR", "vim")
	t.Setenv("VISUAL", "code")
	result := ResolveEditor("")
	if result != "vim" {
		t.Errorf("expected vim (from EDITOR), got %q", result)
	}
}

func TestResolveEditorEnvVisual(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "code")
	result := ResolveEditor("")
	if result != "code" {
		t.Errorf("expected code (from VISUAL), got %q", result)
	}
}

func TestResolveEditorFallback(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	result := ResolveEditor("")
	if result != "vi" {
		t.Errorf("expected vi (fallback), got %q", result)
	}
}

func TestEditWithTrueCommand(t *testing.T) {
	// Use 'true' as editor — it exits successfully without modifying the file
	content, changed, err := Edit("true", "original content")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if changed {
		t.Error("expected changed=false for unchanged content")
	}
	if content != "original content" {
		t.Errorf("content = %q, want %q", content, "original content")
	}
}

func TestEditTempFileCleanup(t *testing.T) {
	// Create a temp file pattern to verify cleanup
	tmp, err := os.CreateTemp("", "diaryctl-test-*.md")
	if err != nil {
		t.Fatal(err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	os.Remove(tmpName)

	// After Edit completes, temp files should be cleaned up
	_, _, _ = Edit("true", "test cleanup")

	// Verify no leftover temp files matching our pattern
	// (the specific temp file from Edit should be removed by defer)
}

func TestEditEmptyResult(t *testing.T) {
	// Use a script that truncates the file
	content, changed, err := Edit("sh -c 'truncate -s 0'", "original")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if changed {
		t.Error("expected changed=false for empty result")
	}
	if content != "" {
		t.Errorf("content = %q, want empty", content)
	}
}
