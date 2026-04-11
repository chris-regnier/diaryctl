package transcribe

import (
	"context"
	"fmt"
	"time"
)

// Result holds the output of a transcription.
type Result struct {
	Text     string        // The transcribed text
	Duration time.Duration // Duration of the audio (zero if unavailable)
	Language string        // Detected language (empty if unavailable)
}

// Transcriber converts audio files to text using a locally-installed STT engine.
type Transcriber interface {
	// Transcribe converts the audio at the given path to text.
	Transcribe(ctx context.Context, audioPath string) (Result, error)

	// Name returns the human-readable name of this backend (e.g., "whisper", "parakeet").
	Name() string
}

// Config holds settings for creating a Transcriber.
type Config struct {
	ModelPath string            // Model name or path (e.g., "base", "large-v3")
	Language  string            // Language hint (e.g., "en")
	Options   map[string]string // Engine-specific options
}

// NewTranscriber creates a Transcriber based on the engine name and config.
func NewTranscriber(engine string, cfg Config) (Transcriber, error) {
	switch engine {
	case "whisper":
		return NewWhisperTranscriber(cfg), nil
	case "parakeet":
		return NewParakeetTranscriber(cfg), nil
	case "command":
		if cfg.Options == nil || cfg.Options["command"] == "" {
			return nil, fmt.Errorf("command engine requires dictate.command to be set in config")
		}
		return NewCommandTranscriber(cfg.Options["command"]), nil
	default:
		return nil, fmt.Errorf("unknown transcription engine %q: supported engines are whisper, parakeet, command", engine)
	}
}
