package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/transcribe"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

// mockTranscriber returns a fixed transcription result.
type mockTranscriber struct {
	text string
	err  error
}

func (m *mockTranscriber) Name() string { return "mock" }

func (m *mockTranscriber) Transcribe(_ context.Context, _ string) (transcribe.Result, error) {
	if m.err != nil {
		return transcribe.Result{}, m.err
	}
	return transcribe.Result{Text: m.text}, nil
}

// newTestRootCmd creates a bare root command for testing, without any
// pre-registered subcommands that might conflict with test-injected deps.
func newTestRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diaryctl",
		Short: "A diary management CLI tool",
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

// createTestAudioFile creates a dummy audio file for testing.
func createTestAudioFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "test.wav")
	if err := os.WriteFile(p, []byte("fake wav data"), 0644); err != nil {
		t.Fatalf("creating test audio file: %v", err)
	}
	return p
}

func TestDictateCommand_FromFile(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "bought groceries at the store"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Created block") {
		t.Errorf("expected output to contain 'Created block', got: %s", outputStr)
	}

	// Verify block was created.
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "bought groceries at the store" {
		t.Errorf("expected content 'bought groceries at the store', got: %s", blocks[0].Content)
	}
	// Verify source attribute is set.
	if blocks[0].Attributes["source"] != "dictate" {
		t.Errorf("expected source=dictate attribute, got: %s", blocks[0].Attributes["source"])
	}
}

func TestDictateCommand_TranscribeOnly(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "this is a test transcription"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--transcribe-only"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should print transcription to stdout.
	if !strings.Contains(stdout.String(), "this is a test transcription") {
		t.Errorf("expected stdout to contain transcription, got: %s", stdout.String())
	}

	// Should NOT create a block.
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks in transcribe-only mode, got %d", len(blocks))
	}
}

func TestDictateCommand_WithAttributes(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "voice note content"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--attr", "type=voice-note", "--attr", "mood=happy"})

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

	b := blocks[0]
	if b.Attributes["type"] != "voice-note" {
		t.Errorf("expected type=voice-note, got: %s", b.Attributes["type"])
	}
	if b.Attributes["mood"] != "happy" {
		t.Errorf("expected mood=happy, got: %s", b.Attributes["mood"])
	}
	if b.Attributes["source"] != "dictate" {
		t.Errorf("expected source=dictate, got: %s", b.Attributes["source"])
	}
}

func TestDictateCommand_WithDate(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "dated entry"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--date", "2024-06-15"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	targetDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.Local)
	blocks, err := store.ListBlocks(targetDate)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "dated entry" {
		t.Errorf("expected content 'dated entry', got: %s", blocks[0].Content)
	}
}

func TestDictateCommand_InvalidDate(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "test"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--date", "not-a-date"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
	if !strings.Contains(err.Error(), "date") {
		t.Errorf("expected error about date, got: %v", err)
	}
}

func TestDictateCommand_NoEngine(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store: store,
		// No transcriber set.
		Theme: ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when no transcription engine configured")
	}
	if !strings.Contains(err.Error(), "engine") {
		t.Errorf("expected error about engine, got: %v", err)
	}
}

func TestDictateCommand_FileNotFound(t *testing.T) {
	store := setupTestStoreV2(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "test"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", "/nonexistent/path.wav"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error about file not found, got: %v", err)
	}
}

func TestDictateCommand_EmptyTranscription(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "   "},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty transcription")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error about empty text, got: %v", err)
	}
}

func TestDictateCommand_InvalidAttribute(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{text: "test content"},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--attr", "invalid"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid attribute format")
	}
	if !strings.Contains(err.Error(), "attribute") {
		t.Errorf("expected error about attribute, got: %v", err)
	}
}

func TestDictateCommand_TranscriberError(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	deps := DictateDeps{
		Store:       store,
		Transcriber: &mockTranscriber{err: fmt.Errorf("model not found")},
		Theme:       ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when transcriber fails")
	}
	if !strings.Contains(err.Error(), "transcription failed") {
		t.Errorf("expected 'transcription failed' error, got: %v", err)
	}
}

func TestDictateCommand_EngineFlag(t *testing.T) {
	store := setupTestStoreV2(t)
	audioFile := createTestAudioFile(t)

	// No Transcriber injected — command must create one from --engine flag.
	// Use "command" engine with an echo command so it works without external tools.
	deps := DictateDeps{
		Store: store,
		EngineConfig: transcribe.Config{
			Options: map[string]string{
				"command": "echo transcribed via engine flag",
			},
		},
		Theme: ui.Theme{},
	}

	rootCmd := newTestRootCmd()
	rootCmd.AddCommand(NewDictateCommand(deps))

	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	rootCmd.SetArgs([]string{"dictate", "--file", audioFile, "--engine", "command"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Created block") {
		t.Errorf("expected output to contain 'Created block', got: %s", outputStr)
	}

	// Verify block content is from the echo command.
	today := day.NormalizeDate(time.Now())
	blocks, err := store.ListBlocks(today)
	if err != nil {
		t.Fatalf("ListBlocks() error = %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "transcribed via engine flag" {
		t.Errorf("expected content 'transcribed via engine flag', got: %s", blocks[0].Content)
	}
}
