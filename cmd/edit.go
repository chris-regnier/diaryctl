package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a diary entry in your editor",
	Long:  "Open an existing diary entry in your configured editor for modification.",
	Example: `  diaryctl edit a3kf9x2m`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Fetch existing entry
		e, err := store.Get(id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: entry %s not found\n", id)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		// Open editor
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
				ui.FormatNoChanges(os.Stdout, id)
			}
			return nil
		}

		// Update entry
		updated, err := store.Update(id, content)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, updated)
		} else {
			ui.FormatEntryUpdated(os.Stdout, updated)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
