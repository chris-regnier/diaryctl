package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/daily"
	"github.com/spf13/cobra"
)

var jotTemplate string

var jotCmd = &cobra.Command{
	Use:   "jot [text...]",
	Short: "Append a timestamped note to today's entry",
	Long: `Append a timestamped note to today's daily entry.

The note is formatted as a bullet with a timestamp: - **HH:MM** text
If no entry exists for today, one is created automatically.`,
	Example: `  diaryctl jot "bought groceries"
  diaryctl jot meeting went well
  echo "note from pipe" | diaryctl jot -`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var content string

		switch {
		case len(args) == 1 && args[0] == "-":
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			content = strings.TrimSpace(string(data))
		case len(args) > 0:
			content = strings.Join(args, " ")
		default:
			return fmt.Errorf("jot requires text: diaryctl jot \"some text\"")
		}

		templateName := jotTemplate
		if templateName == "" {
			templateName = appConfig.DefaultTemplate
		}

		return jotRun(content, templateName)
	},
}

func jotRun(content string, templateName string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("jot: empty content")
	}

	e, _, err := daily.GetOrCreateToday(store, templateName)
	if err != nil {
		return fmt.Errorf("getting today's entry: %w", err)
	}

	timestamp := time.Now().Format("15:04")
	jotLine := fmt.Sprintf("- **%s** %s", timestamp, content)

	var newContent string
	if strings.TrimSpace(e.Content) == "" {
		newContent = jotLine
	} else {
		newContent = e.Content + "\n" + jotLine
	}

	if _, err := store.Update(e.ID, newContent); err != nil {
		return fmt.Errorf("updating entry: %w", err)
	}

	fmt.Fprintln(os.Stderr, jotLine)
	return nil
}

func init() {
	jotCmd.Flags().StringVar(&jotTemplate, "template", "", "template to use when creating today's entry")
	rootCmd.AddCommand(jotCmd)
}
