package transcribe

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WhisperTranscriber uses the whisper CLI to transcribe audio.
type WhisperTranscriber struct {
	model    string
	language string
}

// NewWhisperTranscriber creates a new WhisperTranscriber with the given config.
func NewWhisperTranscriber(cfg Config) *WhisperTranscriber {
	model := cfg.ModelPath
	if model == "" {
		model = "base"
	}
	return &WhisperTranscriber{
		model:    model,
		language: cfg.Language,
	}
}

func (t *WhisperTranscriber) Name() string { return "whisper" }

func (t *WhisperTranscriber) Transcribe(ctx context.Context, audioPath string) (Result, error) {
	// Create a temp directory for whisper output.
	outDir, err := os.MkdirTemp("", "diaryctl-whisper-*")
	if err != nil {
		return Result{}, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(outDir)

	args := []string{
		audioPath,
		"--model", t.model,
		"--output_format", "txt",
		"--output_dir", outDir,
	}
	if t.language != "" {
		args = append(args, "--language", t.language)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "whisper", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("whisper failed: %w: %s", err, stderr.String())
	}

	// Whisper writes output as <basename>.txt in the output directory.
	base := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	txtPath := filepath.Join(outDir, base+".txt")

	data, err := os.ReadFile(txtPath)
	if err != nil {
		return Result{}, fmt.Errorf("reading whisper output: %w", err)
	}

	text := strings.TrimSpace(string(data))
	return Result{Text: text, Language: t.language}, nil
}
