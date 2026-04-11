package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Recorder captures audio from the microphone to a file.
type Recorder interface {
	// Record starts recording audio and blocks until the context is cancelled
	// or the child process exits. The output file is written in WAV format
	// (16kHz, mono, 16-bit).
	// The caller is responsible for cleaning up the output file.
	Record(ctx context.Context, outputPath string) error

	// Name returns the human-readable name of this recorder (e.g., "sox", "arecord").
	Name() string
}

// ResolveRecorder determines which recording tool to use.
// Priority: configRecorder > DIARYCTL_DICTATE_RECORDER env > auto-detect from PATH.
func ResolveRecorder(configRecorder string) (Recorder, error) {
	name := configRecorder
	if name == "" {
		name = os.Getenv("DIARYCTL_DICTATE_RECORDER")
	}

	if name != "" {
		return recorderByName(name)
	}

	// Auto-detect: try common recorders in order of preference.
	for _, candidate := range []string{"sox", "arecord", "ffmpeg"} {
		if _, err := exec.LookPath(recorderBinary(candidate)); err == nil {
			return recorderByName(candidate)
		}
	}

	return nil, fmt.Errorf("no recording tool found in PATH: install sox, arecord, or ffmpeg")
}

func recorderByName(name string) (Recorder, error) {
	switch strings.ToLower(name) {
	case "sox":
		return &SoxRecorder{}, nil
	case "arecord":
		return &ArecordRecorder{}, nil
	case "ffmpeg":
		return &FFmpegRecorder{}, nil
	default:
		return nil, fmt.Errorf("unknown recorder %q: supported recorders are sox, arecord, ffmpeg", name)
	}
}

func recorderBinary(name string) string {
	switch name {
	case "sox":
		return "rec" // sox's recording command
	default:
		return name
	}
}

// runRecordingCommand starts a recording command and waits for it to finish
// or for the context to be cancelled. On cancellation, it sends an interrupt
// signal to allow the tool to flush its buffer and write valid file headers.
func runRecordingCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", name, err)
	}

	// Wait for the process in a goroutine so we can handle context cancellation.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited on its own.
		if err != nil {
			// Context cancellation causes the process to be killed; that's expected.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%s exited with error: %w", name, err)
		}
		return nil
	case <-ctx.Done():
		// Context was cancelled — send interrupt to flush buffers.
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		// Wait for process to finish after interrupt.
		<-done
		return nil
	}
}

// SoxRecorder records audio using the sox/rec command.
type SoxRecorder struct{}

func (r *SoxRecorder) Name() string { return "sox" }

func (r *SoxRecorder) Record(ctx context.Context, outputPath string) error {
	// rec -q -r 16000 -c 1 -b 16 output.wav
	return runRecordingCommand(ctx, "rec", "-q", "-r", "16000", "-c", "1", "-b", "16", outputPath)
}

// ArecordRecorder records audio using ALSA's arecord (Linux).
type ArecordRecorder struct{}

func (r *ArecordRecorder) Name() string { return "arecord" }

func (r *ArecordRecorder) Record(ctx context.Context, outputPath string) error {
	// arecord -f S16_LE -r 16000 -c 1 -t wav output.wav
	return runRecordingCommand(ctx, "arecord", "-f", "S16_LE", "-r", "16000", "-c", "1", "-t", "wav", outputPath)
}

// FFmpegRecorder records audio using ffmpeg.
type FFmpegRecorder struct{}

func (r *FFmpegRecorder) Name() string { return "ffmpeg" }

func (r *FFmpegRecorder) Record(ctx context.Context, outputPath string) error {
	// Platform-specific input device.
	var inputArgs []string
	switch runtime.GOOS {
	case "darwin":
		inputArgs = []string{"-f", "avfoundation", "-i", ":default"}
	default: // linux
		inputArgs = []string{"-f", "pulse", "-i", "default"}
	}

	args := append(inputArgs, "-ar", "16000", "-ac", "1", "-sample_fmt", "s16", "-y", outputPath)
	return runRecordingCommand(ctx, "ffmpeg", args...)
}
