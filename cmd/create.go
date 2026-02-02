package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	tmpl "github.com/chris-regnier/diaryctl/internal/template"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [content...]",
	Short: "Create a new diary entry",
	Long: `Create a new diary entry.

If content is provided as arguments, it is used directly.
If "-" is provided, content is read from stdin.
If no content is provided, your editor is opened.

Use --template to pre-fill the editor with template content.
Use --no-template to skip the default template.`,
	Example: `  diaryctl create "Today was great"
  diaryctl create Today was a good day
  echo "piped content" | diaryctl create -
  diaryctl create
  diaryctl create --template daily
  diaryctl create --template daily,prompts`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templateFlag, _ := cmd.Flags().GetString("template")
		noTemplate, _ := cmd.Flags().GetBool("no-template")

		// Check for conflicting flags
		if templateFlag != "" && noTemplate {
			fmt.Fprintln(os.Stderr, "Error: --template and --no-template cannot be used together")
			os.Exit(1)
		}

		var content string
		var templateRefs []entry.TemplateRef

		switch {
		case len(args) == 1 && args[0] == "-":
			// Read from stdin — no template applied
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
				os.Exit(2)
			}
			content = string(data)

		case len(args) > 0:
			// Inline content — no template applied
			content = strings.Join(args, " ")

		default:
			// Resolve template names
			var templateContent string
			if !noTemplate {
				names := tmpl.ParseNames(templateFlag)
				if len(names) == 0 && appConfig.DefaultTemplate != "" {
					// Use config default
					names = tmpl.ParseNames(appConfig.DefaultTemplate)
					// Default template: graceful fallback on error
					tc, refs, err := tmpl.Compose(store, names)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: default template %q not found, skipping\n", appConfig.DefaultTemplate)
					} else {
						templateContent = tc
						templateRefs = refs
					}
				} else if len(names) > 0 {
					// Explicit --template: fail fast on error
					tc, refs, err := tmpl.Compose(store, names)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error:", err)
						os.Exit(1)
					}
					templateContent = tc
					templateRefs = refs
				}
			}

			// Open editor
			editorCmd := editor.ResolveEditor(appConfig.Editor)
			var err error
			var changed bool
			content, changed, err = editor.Edit(editorCmd, templateContent)
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
			Templates: templateRefs,
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
	createCmd.Flags().String("template", "", "template(s) to use (comma-separated)")
	createCmd.Flags().Bool("no-template", false, "skip default template")
	rootCmd.AddCommand(createCmd)
}
