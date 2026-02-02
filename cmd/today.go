package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	todayEdit        bool
	todayIDOnly      bool
	todayContentOnly bool
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "View or edit today's daily entry",
	Long: `View or edit today's daily entry.

If no entry exists for today, one is created automatically.`,
	Example: `  diaryctl today
  diaryctl today --edit
  diaryctl today --id-only
  diaryctl today --content-only
  diaryctl today --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if todayEdit {
			return todayEditRun()
		}
		return todayRun(os.Stdout, todayIDOnly, todayContentOnly)
	},
}

func todayRun(w io.Writer, idOnly bool, contentOnly bool) error {
	e, _, err := daily.GetOrCreateToday(store, appConfig.DefaultTemplate)
	if err != nil {
		return fmt.Errorf("getting today's entry: %w", err)
	}

	if jsonOutput {
		return ui.FormatJSON(w, e)
	}

	if idOnly {
		fmt.Fprintln(w, e.ID)
		return nil
	}

	if contentOnly {
		fmt.Fprintln(w, e.Content)
		return nil
	}

	var buf bytes.Buffer
	ui.FormatEntryFull(&buf, e)
	ui.OutputOrPage(w, buf.String(), false)
	return nil
}

func todayEditRun() error {
	e, _, err := daily.GetOrCreateToday(store, appConfig.DefaultTemplate)
	if err != nil {
		return fmt.Errorf("getting today's entry: %w", err)
	}

	editorCmd := editor.ResolveEditor(appConfig.Editor)
	content, changed, err := editor.Edit(editorCmd, e.Content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Editor error:", err)
		os.Exit(3)
	}

	if !changed {
		if jsonOutput {
			ui.FormatJSON(os.Stdout, e)
		} else {
			ui.FormatNoChanges(os.Stdout, e.ID)
		}
		return nil
	}

	updated, err := store.Update(e.ID, content)
	if err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}

	if jsonOutput {
		ui.FormatJSON(os.Stdout, updated)
	} else {
		ui.FormatEntryUpdated(os.Stdout, updated)
	}
	return nil
}

func init() {
	todayCmd.Flags().BoolVar(&todayEdit, "edit", false, "Open today's entry in the editor")
	todayCmd.Flags().BoolVar(&todayIDOnly, "id-only", false, "Print just the entry ID")
	todayCmd.Flags().BoolVar(&todayContentOnly, "content-only", false, "Print just the content")
	rootCmd.AddCommand(todayCmd)
}
