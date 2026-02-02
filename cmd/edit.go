package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	tmpl "github.com/chris-regnier/diaryctl/internal/template"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a diary entry in your editor",
	Long: `Open an existing diary entry in your configured editor for modification.

Use --template to append template content to the entry.`,
	Example: `  diaryctl edit a3kf9x2m
  diaryctl edit a3kf9x2m --template prompts`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		templateFlag, _ := cmd.Flags().GetString("template")

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

		editorContent := e.Content
		var newRefs []entry.TemplateRef

		// If --template specified, compose and append
		if templateFlag != "" {
			names := tmpl.ParseNames(templateFlag)
			if len(names) > 0 {
				tc, refs, err := tmpl.Compose(store, names)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
					os.Exit(1)
				}
				editorContent = e.Content + "\n\n" + tc
				newRefs = refs
			}
		}

		// Open editor
		editorCmd := editor.ResolveEditor(appConfig.Editor)
		content, changed, err := editor.Edit(editorCmd, editorContent)
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

		// Merge template refs: existing + new, deduplicated by TemplateID
		var mergedTemplates []entry.TemplateRef
		if len(newRefs) > 0 {
			seen := make(map[string]bool)
			for _, ref := range e.Templates {
				seen[ref.TemplateID] = true
				mergedTemplates = append(mergedTemplates, ref)
			}
			for _, ref := range newRefs {
				if !seen[ref.TemplateID] {
					mergedTemplates = append(mergedTemplates, ref)
				}
			}
		}

		// Determine what to pass for templates parameter
		var templatesArg []entry.TemplateRef
		if len(newRefs) > 0 {
			templatesArg = mergedTemplates
		}
		// nil preserves existing refs when no --template flag used

		// Update entry
		updated, err := store.Update(id, content, templatesArg)
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
	editCmd.Flags().String("template", "", "template(s) to append (comma-separated)")
	rootCmd.AddCommand(editCmd)
}
