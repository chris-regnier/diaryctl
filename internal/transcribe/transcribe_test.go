package transcribe

import (
	"context"
	"testing"
)

func TestNewTranscriber_Whisper(t *testing.T) {
	tr, err := NewTranscriber("whisper", Config{ModelPath: "base", Language: "en"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "whisper" {
		t.Errorf("expected name %q, got %q", "whisper", tr.Name())
	}
}

func TestNewTranscriber_Parakeet(t *testing.T) {
	tr, err := NewTranscriber("parakeet", Config{Language: "en"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "parakeet" {
		t.Errorf("expected name %q, got %q", "parakeet", tr.Name())
	}
}

func TestNewTranscriber_Command(t *testing.T) {
	cfg := Config{
		Options: map[string]string{
			"command": "echo hello {{.AudioPath}}",
		},
	}
	tr, err := NewTranscriber("command", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Name() != "command" {
		t.Errorf("expected name %q, got %q", "command", tr.Name())
	}
}

func TestNewTranscriber_CommandMissing(t *testing.T) {
	_, err := NewTranscriber("command", Config{})
	if err == nil {
		t.Fatal("expected error for command engine without command config")
	}
}

func TestNewTranscriber_Unknown(t *testing.T) {
	_, err := NewTranscriber("nonexistent", Config{})
	if err == nil {
		t.Fatal("expected error for unknown engine")
	}
}

func TestCommandTranscriber_Transcribe(t *testing.T) {
	tr := NewCommandTranscriber("echo hello world")
	result, err := tr.Transcribe(context.Background(), "/tmp/test.wav")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", result.Text)
	}
}

func TestCommandTranscriber_TemplateExpansion(t *testing.T) {
	tr := NewCommandTranscriber("echo {{.AudioPath}}")
	result, err := tr.Transcribe(context.Background(), "/tmp/test.wav")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "/tmp/test.wav" {
		t.Errorf("expected text %q, got %q", "/tmp/test.wav", result.Text)
	}
}

func TestCommandTranscriber_InvalidTemplate(t *testing.T) {
	tr := NewCommandTranscriber("echo {{.Invalid")
	_, err := tr.Transcribe(context.Background(), "/tmp/test.wav")
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestCommandTranscriber_FailedCommand(t *testing.T) {
	tr := NewCommandTranscriber("false")
	_, err := tr.Transcribe(context.Background(), "/tmp/test.wav")
	if err == nil {
		t.Fatal("expected error for failed command")
	}
}
