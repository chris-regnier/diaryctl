package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var showIDOnly bool
var showContentOnly bool

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a diary entry",
	Long:  "Display the full content and metadata of a diary entry.",
	Example: `  diaryctl show a3kf9x2m
  diaryctl show a3kf9x2m --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		e, err := store.Get(id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: entry %s not found\n", id)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if showIDOnly {
			fmt.Fprintln(cmd.OutOrStdout(), e.ID)
			return nil
		}
		if showContentOnly {
			fmt.Fprintln(cmd.OutOrStdout(), e.Content)
			return nil
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, e)
		} else {
			var buf bytes.Buffer
			ui.FormatEntryFull(&buf, e)
			ui.OutputOrPage(os.Stdout, buf.String(), false)
		}

		return nil
	},
}

func init() {
	showCmd.Flags().BoolVar(&showIDOnly, "id-only", false, "print just the entry ID")
	showCmd.Flags().BoolVar(&showContentOnly, "content-only", false, "print just the entry content")
	rootCmd.AddCommand(showCmd)
}
