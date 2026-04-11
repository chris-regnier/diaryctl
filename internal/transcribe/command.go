package transcribe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
)

// CommandTranscriber executes a user-configured command to transcribe audio.
// The command template receives the audio file path via {{.AudioPath}}.
// The command must write the transcription text to stdout.
type CommandTranscriber struct {
	commandTemplate string
}

// NewCommandTranscriber creates a new CommandTranscriber with the given command template.
func NewCommandTranscriber(cmdTemplate string) *CommandTranscriber {
	return &CommandTranscriber{commandTemplate: cmdTemplate}
}

func (t *CommandTranscriber) Name() string { return "command" }

func (t *CommandTranscriber) Transcribe(ctx context.Context, audioPath string) (Result, error) {
	// Expand the command template with the audio path.
	tmpl, err := template.New("cmd").Parse(t.commandTemplate)
	if err != nil {
		return Result{}, fmt.Errorf("parsing command template: %w", err)
	}

	var cmdBuf bytes.Buffer
	data := struct{ AudioPath string }{AudioPath: audioPath}
	if err := tmpl.Execute(&cmdBuf, data); err != nil {
		return Result{}, fmt.Errorf("executing command template: %w", err)
	}

	expanded := cmdBuf.String()
	parts := strings.Fields(expanded)
	if len(parts) == 0 {
		return Result{}, fmt.Errorf("command template expanded to empty string")
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("transcription command failed: %w: %s", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	return Result{Text: text}, nil
}
