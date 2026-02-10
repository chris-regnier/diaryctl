package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/template"
	"github.com/spf13/cobra"
)

// NewEditV2Command creates a new edit command for the v2 data model.
// The command opens an editor for creating new diary blocks with optional
// template support and custom attributes.
//
// Flags:
//   - --template: Template name to use (renders with --var flags)
//   - --var: Template variables in KEY=value format (repeatable)
//   - --attr: Block attributes in key=value format (repeatable)
//   - --date: Target date for the block (YYYY-MM-DD format, default: today)
//
// Template Behavior:
//   - If --template is provided, the template is loaded and rendered with vars
//   - Template attributes are merged into block attributes (user attrs override)
//   - The rendered content is used as initial editor content
//
// Editor Behavior:
//   - Opens the user's configured editor with initial content
//   - If content is empty after editing, the command aborts
//   - Otherwise, creates a block with the edited content
//
// The command:
//   - Generates a block ID using block.NewID()
//   - Sets CreatedAt and UpdatedAt to the current timestamp
//   - Creates the block in storage
//   - Prints "Created block {id} on {date}"
func NewEditV2Command(store storage.StorageV2) *cobra.Command {
	var templateName string
	var vars []string
	var attrs []string
	var dateStr string

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Create a new block using an editor",
		Long: `Create a new block in your diary by opening an editor.

You can optionally use a template, provide template variables, set block
attributes, and specify the target date. If a template is provided, it will
be rendered with the given variables and used as initial content in the editor.`,
		Example: `  diaryctl edit
  diaryctl edit --template meeting --var date=2024-01-15 --var topic=Planning
  diaryctl edit --attr type=note --attr mood=happy
  diaryctl edit --date 2024-01-15 --template daily`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse date (default to today)
			var targetDate time.Time
			if dateStr != "" {
				parsedDate, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					return fmt.Errorf("invalid date format %q: use YYYY-MM-DD (e.g., 2024-01-15)", dateStr)
				}
				targetDate = day.NormalizeDate(parsedDate)
			} else {
				targetDate = day.NormalizeDate(time.Now())
			}

			// Parse variables
			templateVars := make(map[string]string)
			for _, v := range vars {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid var format %q: use KEY=value (e.g., date=2024-01-15)", v)
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key == "" {
					return fmt.Errorf("invalid var format %q: KEY cannot be empty", v)
				}
				templateVars[key] = value
			}

			// Parse attributes
			attributes := make(map[string]string)
			for _, attr := range attrs {
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

			// Prepare initial content
			var initialContent string

			// If template is provided, load and render it
			if templateName != "" {
				tmpl, err := store.GetTemplateByName(templateName)
				if err != nil {
					return fmt.Errorf("getting template %q: %w", templateName, err)
				}

				// Render template with variables
				rendered, err := template.Render(tmpl.Content, templateVars)
				if err != nil {
					return fmt.Errorf("rendering template: %w", err)
				}
				initialContent = rendered

				// Merge template attributes into block attributes (user attrs override)
				for k, v := range tmpl.Attributes {
					if _, exists := attributes[k]; !exists {
						attributes[k] = v
					}
				}
			}

			// Resolve editor
			editorCmd := editor.ResolveEditor(os.Getenv("EDITOR"))

			// Open editor
			editedContent, changed, err := editor.Edit(editorCmd, initialContent)
			if err != nil {
				return fmt.Errorf("opening editor: %w", err)
			}

			// Check if content is empty
			if strings.TrimSpace(editedContent) == "" {
				return fmt.Errorf("no content provided (empty after editing)")
			}

			// Generate block ID
			blockID := block.NewID()

			// Create block with current timestamps
			now := time.Now()
			b := block.Block{
				ID:         blockID,
				Content:    editedContent,
				CreatedAt:  now,
				UpdatedAt:  now,
				Attributes: attributes,
			}

			// Create block in storage
			if err := store.CreateBlock(targetDate, b); err != nil {
				return fmt.Errorf("creating block: %w", err)
			}

			// Print success message
			dateFormatted := targetDate.Format("2006-01-02")
			fmt.Fprintf(cmd.OutOrStdout(), "Created block %s on %s\n", blockID, dateFormatted)

			// Suppress "changed" warning - we always save non-empty content
			_ = changed

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&templateName, "template", "", "template name to use")
	cmd.Flags().StringArrayVar(&vars, "var", []string{}, "template variables in KEY=value format (repeatable)")
	cmd.Flags().StringArrayVar(&attrs, "attr", []string{}, "block attributes in key=value format (repeatable)")
	cmd.Flags().StringVar(&dateStr, "date", "", "date for the block (YYYY-MM-DD, default: today)")

	return cmd
}
