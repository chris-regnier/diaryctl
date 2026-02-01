package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var forceDelete bool

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a diary entry",
	Long:  "Permanently delete a diary entry. Requires confirmation unless --force is used.",
	Example: `  diaryctl delete a3kf9x2m
  diaryctl delete a3kf9x2m --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Fetch entry to confirm it exists and show preview
		e, err := store.Get(id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: entry %s not found\n", id)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		// Confirmation
		if !forceDelete {
			fmt.Fprintf(os.Stdout, "Entry: %s (%s)\n", e.ID, e.CreatedAt.Local().Format("2006-01-02 15:04"))
			fmt.Fprintf(os.Stdout, "Preview: %s\n\n", e.Preview(60))

			confirmed, err := ui.Confirm("Delete this entry? This cannot be undone.")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(2)
			}
			if !confirmed {
				fmt.Fprintln(os.Stdout, "Cancelled.")
				return nil
			}
		}

		if err := store.Delete(id); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, ui.DeleteResult{ID: id, Deleted: true})
		} else {
			ui.FormatEntryDeleted(os.Stdout, id)
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVar(&forceDelete, "force", false, "skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}
