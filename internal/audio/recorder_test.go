package audio

import (
	"testing"
)

func TestRecorderByName_Sox(t *testing.T) {
	r, err := recorderByName("sox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "sox" {
		t.Errorf("expected name %q, got %q", "sox", r.Name())
	}
}

func TestRecorderByName_Arecord(t *testing.T) {
	r, err := recorderByName("arecord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "arecord" {
		t.Errorf("expected name %q, got %q", "arecord", r.Name())
	}
}

func TestRecorderByName_FFmpeg(t *testing.T) {
	r, err := recorderByName("ffmpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "ffmpeg" {
		t.Errorf("expected name %q, got %q", "ffmpeg", r.Name())
	}
}

func TestRecorderByName_Unknown(t *testing.T) {
	_, err := recorderByName("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown recorder")
	}
}

func TestRecorderByName_CaseInsensitive(t *testing.T) {
	r, err := recorderByName("SOX")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "sox" {
		t.Errorf("expected name %q, got %q", "sox", r.Name())
	}
}

func TestResolveRecorder_FromConfig(t *testing.T) {
	// When config specifies a known recorder, it should be used.
	r, err := ResolveRecorder("sox")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name() != "sox" {
		t.Errorf("expected name %q, got %q", "sox", r.Name())
	}
}

func TestResolveRecorder_FromConfig_Invalid(t *testing.T) {
	_, err := ResolveRecorder("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown recorder in config")
	}
}

func TestRecorderBinary(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"sox", "rec"},
		{"arecord", "arecord"},
		{"ffmpeg", "ffmpeg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recorderBinary(tt.name)
			if got != tt.expected {
				t.Errorf("recorderBinary(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}
