package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  "Manage reusable content templates for diary entries.",
}

var templateListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all templates",
	Example: `  diaryctl template list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templates, err := store.ListTemplates()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, templates)
		} else {
			ui.FormatTemplateList(os.Stdout, templates)
		}
		return nil
	},
}

var templateShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a template",
	Example: `  diaryctl template show daily`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		tmpl, err := store.GetTemplateByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: template %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, tmpl)
		} else {
			ui.FormatTemplateFull(os.Stdout, tmpl)
		}
		return nil
	},
}

var templateCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new template",
	Long: `Create a new template with the given name.

If "-" is provided as content, it is read from stdin.
Otherwise, your editor is opened.`,
	Example: `  diaryctl template create daily
  echo "# Daily Entry" | diaryctl template create daily -`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := entry.ValidateTemplateName(name); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		var content string

		if len(args) == 2 && args[1] == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
				os.Exit(2)
			}
			content = strings.TrimSpace(string(data))
		} else {
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

		if strings.TrimSpace(content) == "" {
			fmt.Fprintln(os.Stderr, "Error: template content must not be empty")
			os.Exit(1)
		}

		id, err := entry.NewID()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error generating ID:", err)
			os.Exit(2)
		}

		now := time.Now().UTC()
		tmpl := storage.Template{
			ID:        id,
			Name:      name,
			Content:   content,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := store.CreateTemplate(tmpl); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				fmt.Fprintf(os.Stderr, "Error: template %q already exists\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, tmpl)
		} else {
			fmt.Fprintf(os.Stdout, "Created template %q (%s)\n", tmpl.Name, tmpl.ID)
		}
		return nil
	},
}

var templateEditCmd = &cobra.Command{
	Use:     "edit <name>",
	Short:   "Edit an existing template",
	Example: `  diaryctl template edit daily`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		tmpl, err := store.GetTemplateByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: template %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		editorCmd := editor.ResolveEditor(appConfig.Editor)
		content, changed, err := editor.Edit(editorCmd, tmpl.Content)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Editor error:", err)
			os.Exit(3)
		}

		if !changed {
			fmt.Fprintf(os.Stdout, "No changes detected for template %q.\n", name)
			return nil
		}

		updated, err := store.UpdateTemplate(tmpl.ID, tmpl.Name, content)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, updated)
		} else {
			fmt.Fprintf(os.Stdout, "Updated template %q (%s)\n", updated.Name, updated.ID)
		}
		return nil
	},
}

var forceDeleteTemplate bool

var templateDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete a template",
	Long:    "Permanently delete a template. Requires --force flag.",
	Example: `  diaryctl template delete daily --force`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		tmpl, err := store.GetTemplateByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: template %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if !forceDeleteTemplate {
			fmt.Fprintf(os.Stdout, "Template: %s (%s)\n", tmpl.Name, tmpl.ID)
			confirmed, err := ui.Confirm("Delete this template? This cannot be undone.")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(2)
			}
			if !confirmed {
				fmt.Fprintln(os.Stdout, "Cancelled.")
				return nil
			}
		}

		if err := store.DeleteTemplate(tmpl.ID); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, map[string]interface{}{"name": name, "deleted": true})
		} else {
			fmt.Fprintf(os.Stdout, "Deleted template %q.\n", name)
		}
		return nil
	},
}

func init() {
	templateDeleteCmd.Flags().BoolVar(&forceDeleteTemplate, "force", false, "skip confirmation prompt")

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateEditCmd)
	templateCmd.AddCommand(templateDeleteCmd)

	rootCmd.AddCommand(templateCmd)
}
