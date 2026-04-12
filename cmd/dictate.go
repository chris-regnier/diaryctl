package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/audio"
	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/transcribe"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// DictateDeps holds dependencies for the dictate command, allowing
// injection of mock implementations in tests.
type DictateDeps struct {
	Store       storage.StorageV2
	Recorder    audio.Recorder         // nil = resolve from config at runtime
	Transcriber transcribe.Transcriber // nil = resolve from config at runtime
	Editor      string                 // editor command (empty = resolve from config)
	Theme       ui.Theme               // TUI theme

	// Config fields used for runtime resolution when Recorder/Transcriber are nil.
	RecorderName string // config recorder name for audio.ResolveRecorder
	Engine       string // config engine name (e.g. "whisper", "parakeet", "command")
	EngineConfig transcribe.Config // config for creating a transcriber
}

// NewDictateCommand creates a new dictate command for voice-to-text diary entry.
func NewDictateCommand(deps DictateDeps) *cobra.Command {
	var dateStr string
	var attrs []string
	var filePath string
	var transcribeOnly bool
	var engineOverride string
	var editTranscription bool

	cmd := &cobra.Command{
		Use:   "dictate",
		Short: "Record audio and transcribe it into a diary block",
		Long: `Record audio from your microphone, transcribe it using a locally-installed
speech-to-text engine, and save the transcription as a diary block.

Requires a recording tool (sox, arecord, or ffmpeg) and a transcription
engine (whisper, parakeet, or a custom command) to be installed locally.

Configure the engine in config.toml under [dictate].`,
		Example: `  diaryctl dictate
  diaryctl dictate --file recording.wav
  diaryctl dictate --transcribe-only
  diaryctl dictate --file recording.wav --edit
  diaryctl dictate --attr type=voice-note --attr mood=reflective
  diaryctl dictate --date 2026-04-10
  diaryctl dictate --engine whisper`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDictate(cmd, deps, dictateOpts{
				dateStr:           dateStr,
				attrs:             attrs,
				filePath:          filePath,
				transcribeOnly:    transcribeOnly,
				engineOverride:    engineOverride,
				editTranscription: editTranscription,
			})
		},
	}

	cmd.Flags().StringVar(&dateStr, "date", "", "date for the block (YYYY-MM-DD, default: today)")
	cmd.Flags().StringArrayVar(&attrs, "attr", []string{}, "block attributes in key=value format (repeatable)")
	cmd.Flags().StringVar(&filePath, "file", "", "path to an existing audio file (skip recording)")
	cmd.Flags().BoolVar(&transcribeOnly, "transcribe-only", false, "print transcription to stdout without saving")
	cmd.Flags().StringVar(&engineOverride, "engine", "", "override transcription engine from config")
	cmd.Flags().BoolVar(&editTranscription, "edit", false, "open transcription in editor before saving")

	return cmd
}

type dictateOpts struct {
	dateStr           string
	attrs             []string
	filePath          string
	transcribeOnly    bool
	engineOverride    string
	editTranscription bool
}

func runDictate(cmd *cobra.Command, deps DictateDeps, opts dictateOpts) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	audioPath := opts.filePath

	// Step 1: Record audio if no file provided.
	if audioPath == "" {
		rec := deps.Recorder
		if rec == nil {
			// Try to resolve from config.
			var err error
			rec, err = audio.ResolveRecorder(deps.RecorderName)
			if err != nil {
				return fmt.Errorf("no recorder configured: set dictate.recorder in config.toml or install sox, arecord, or ffmpeg")
			}
		}

		tmpFile, err := os.CreateTemp("", "diaryctl-dictate-*.wav")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		audioPath = tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(audioPath)

		// If interactive, show recording indicator.
		isTTY := term.IsTerminal(int(os.Stdin.Fd()))
		if isTTY {
			// Start recording in a goroutine, controlled by context cancellation.
			recordCtx, cancelRecord := context.WithCancel(ctx)

			recordErr := make(chan error, 1)
			go func() {
				recordErr <- rec.Record(recordCtx, audioPath)
			}()

			// Show the recording TUI — blocks until user presses Enter/Esc.
			result, err := ui.RunRecordingIndicator(deps.Theme)
			cancelRecord()

			if err != nil {
				return fmt.Errorf("recording indicator: %w", err)
			}
			if result.Cancelled {
				return fmt.Errorf("recording cancelled")
			}

			// Wait for recorder to finish after cancellation.
			if recErr := <-recordErr; recErr != nil && recErr != context.Canceled {
				return fmt.Errorf("recording failed: %w", recErr)
			}
		} else {
			// Non-interactive: record until interrupted.
			fmt.Fprintln(cmd.ErrOrStderr(), "Recording... Press Ctrl+C to stop.")
			if err := rec.Record(ctx, audioPath); err != nil {
				return fmt.Errorf("recording failed: %w", err)
			}
		}
	} else {
		// Verify the provided file exists.
		if _, err := os.Stat(audioPath); os.IsNotExist(err) {
			return fmt.Errorf("audio file %q not found", audioPath)
		}
	}

	// Step 2: Transcribe the audio.
	tr := deps.Transcriber
	if tr == nil {
		// Resolve transcriber from --engine flag or config.
		engine := deps.Engine
		if opts.engineOverride != "" {
			engine = opts.engineOverride
		}
		if engine == "" {
			return fmt.Errorf("no transcription engine configured: set dictate.engine in config.toml or use --engine flag")
		}
		cfg := deps.EngineConfig
		if engine == "command" && (cfg.Options == nil || cfg.Options["command"] == "") {
			return fmt.Errorf("command engine requires dictate.command to be set in config.toml")
		}
		var err error
		tr, err = transcribe.NewTranscriber(engine, cfg)
		if err != nil {
			return fmt.Errorf("creating transcription engine: %w", err)
		}
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Transcribing...")
	result, err := tr.Transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	if strings.TrimSpace(result.Text) == "" {
		return fmt.Errorf("transcription produced empty text")
	}

	content := result.Text

	// Step 3: Optionally open in editor.
	if opts.editTranscription {
		editorCmd := deps.Editor
		if editorCmd == "" {
			editorCmd = editor.ResolveEditor("")
		}
		edited, changed, err := editor.Edit(editorCmd, content)
		if err != nil {
			return fmt.Errorf("editor: %w", err)
		}
		if changed {
			content = edited
		}
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("transcription discarded: editor returned empty content")
		}
	}

	// Step 4: If transcribe-only, print and exit.
	if opts.transcribeOnly {
		fmt.Fprintln(cmd.OutOrStdout(), content)
		return nil
	}

	// Step 5: Parse date.
	var targetDate time.Time
	if opts.dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", opts.dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format %q: use YYYY-MM-DD (e.g., 2024-01-15)", opts.dateStr)
		}
		targetDate = day.NormalizeDate(parsedDate)
	} else {
		targetDate = day.NormalizeDate(time.Now())
	}

	// Step 6: Parse attributes.
	attributes := map[string]string{
		"source": "dictate",
	}
	for _, attr := range opts.attrs {
		parts := strings.SplitN(attr, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid attribute format %q: use key=value (e.g., type=note)", attr)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("invalid attribute format %q: key cannot be empty", attr)
		}
		attributes[key] = value
	}

	// Step 7: Create and save block.
	blockID := block.NewID()
	now := time.Now()
	b := block.Block{
		ID:         blockID,
		Content:    content,
		CreatedAt:  now,
		UpdatedAt:  now,
		Attributes: attributes,
	}

	if err := deps.Store.CreateBlock(targetDate, b); err != nil {
		return fmt.Errorf("creating block: %w", err)
	}

	dateFormatted := targetDate.Format("2006-01-02")
	fmt.Fprintf(cmd.OutOrStdout(), "Created block %s on %s\n", blockID, dateFormatted)

	return nil
}
