package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [content...]",
	Short: "Create a new diary entry",
	Long: `Create a new diary entry.

If content is provided as arguments, it is used directly.
If "-" is provided, content is read from stdin.
If no content is provided, your editor is opened.`,
	Example: `  diaryctl create "Today was great"
  diaryctl create Today was a good day
  echo "piped content" | diaryctl create -
  diaryctl create`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var content string

		switch {
		case len(args) == 1 && args[0] == "-":
			// Read from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
				os.Exit(2)
			}
			content = string(data)

		case len(args) > 0:
			// Inline content
			content = strings.Join(args, " ")

		default:
			// Open editor
			editorCmd := editor.ResolveEditor(appConfig.Editor)
			var err error
			var changed bool
			content, changed, err = editor.Edit(editorCmd, "")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Editor error:", err)
				os.Exit(3)
			}
			if !changed {
				fmt.Fprintln(os.Stderr, "Error: empty content")
				os.Exit(1)
			}
		}

		// Validate content
		if err := entry.ValidateContent(content); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		// Generate ID
		id, err := entry.NewID()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error generating ID:", err)
			os.Exit(2)
		}

		now := time.Now().UTC()
		e := entry.Entry{
			ID:        id,
			Content:   strings.TrimSpace(content),
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := store.Create(e); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, e)
		} else {
			ui.FormatEntryCreated(os.Stdout, e)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
