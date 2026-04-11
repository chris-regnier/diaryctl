package transcribe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ParakeetTranscriber uses NVIDIA NeMo Parakeet to transcribe audio.
// It shells out to python3 to invoke the NeMo ASR model.
type ParakeetTranscriber struct {
	model    string
	language string
}

// NewParakeetTranscriber creates a new ParakeetTranscriber with the given config.
func NewParakeetTranscriber(cfg Config) *ParakeetTranscriber {
	model := cfg.ModelPath
	if model == "" {
		model = "nvidia/parakeet-tdt-0.6b-v2"
	}
	return &ParakeetTranscriber{
		model:    model,
		language: cfg.Language,
	}
}

func (t *ParakeetTranscriber) Name() string { return "parakeet" }

func (t *ParakeetTranscriber) Transcribe(ctx context.Context, audioPath string) (Result, error) {
	// Python script that loads the model and transcribes the audio file.
	script := fmt.Sprintf(`
import nemo.collections.asr as nemo_asr
import sys

model = nemo_asr.models.ASRModel.from_pretrained('%s')
result = model.transcribe(['%s'])
if hasattr(result[0], 'text'):
    print(result[0].text)
else:
    print(result[0])
`, t.model, audioPath)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "python3", "-c", script)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("parakeet transcription failed: %w: %s", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	return Result{Text: text, Language: t.language}, nil
}
