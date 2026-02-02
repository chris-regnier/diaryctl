package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <id> <content>",
	Short: "Update a diary entry inline",
	Long:  "Replace the content of an existing diary entry with new text.",
	Example: `  diaryctl update a3kf9x2m "Updated content here"
  echo "new content" | diaryctl update a3kf9x2m -`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		var content string

		if args[1] == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
				os.Exit(2)
			}
			content = string(data)
		} else {
			content = strings.Join(args[1:], " ")
		}

		// Validate
		if err := entry.ValidateContent(content); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		updated, err := store.Update(id, strings.TrimSpace(content), nil)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: entry %s not found\n", id)
				os.Exit(1)
			}
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
	rootCmd.AddCommand(updateCmd)
}
