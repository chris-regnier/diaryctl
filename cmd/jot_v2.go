package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/spf13/cobra"
)

// NewJotV2Command creates a new jot command for the v2 data model.
// The command accepts content as arguments (joined with spaces) and creates
// a new block in the storage backend.
//
// Flags:
//   - --date: Specify the date for the block (YYYY-MM-DD format, default: today)
//   - --attr: Add attributes to the block (key=value format, repeatable)
//
// The command:
//   - Generates a block ID using block.NewID()
//   - Sets CreatedAt and UpdatedAt to the current timestamp
//   - Creates the block in storage
//   - Prints "Created block {id} on {date}"
func NewJotV2Command(store storage.StorageV2) *cobra.Command {
	var dateStr string
	var attrs []string

	cmd := &cobra.Command{
		Use:   "jot [text...]",
		Short: "Create a new block with the given content",
		Long: `Create a new block in your diary with the given content.

The content is provided as arguments and will be joined with spaces.
You can optionally specify a date and add attributes to the block.`,
		Example: `  diaryctl jot "bought groceries"
  diaryctl jot --date 2024-01-15 "meeting notes"
  diaryctl jot --attr type=note --attr mood=happy "feeling great today"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that content is provided
			if len(args) == 0 {
				return fmt.Errorf("jot requires content: provide text as arguments")
			}

			// Join all arguments into content
			content := strings.Join(args, " ")

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

			// Generate block ID
			blockID := block.NewID()

			// Create block with current timestamps
			now := time.Now()
			b := block.Block{
				ID:         blockID,
				Content:    content,
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

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&dateStr, "date", "", "date for the block (YYYY-MM-DD, default: today)")
	cmd.Flags().StringArrayVar(&attrs, "attr", []string{}, "block attributes in key=value format (repeatable)")

	return cmd
}
