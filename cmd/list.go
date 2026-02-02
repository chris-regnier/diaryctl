package cmd

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	dateFilter         string
	listTemplateFilter string
	listIDOnly         bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List diary entries",
	Long:  "List diary entries with preview, sorted by date (newest first).",
	Example: `  diaryctl list
  diaryctl list --date 2026-01-31
  diaryctl list --template daily
  diaryctl list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := storage.ListOptions{}

		if dateFilter != "" {
			t, err := time.ParseInLocation("2006-01-02", dateFilter, time.Local)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: invalid date format (use YYYY-MM-DD):", dateFilter)
				os.Exit(1)
			}
			opts.Date = &t
		}

		if listTemplateFilter != "" {
			opts.TemplateName = listTemplateFilter
		}

		entries, err := store.List(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		if listIDOnly {
			for _, e := range entries {
				fmt.Fprintln(cmd.OutOrStdout(), e.ID)
			}
			return nil
		}

		if jsonOutput {
			summaries := ui.ToSummaries(entries)
			ui.FormatJSON(os.Stdout, summaries)
		} else {
			var buf bytes.Buffer
			ui.FormatEntryList(&buf, entries)
			ui.OutputOrPage(os.Stdout, buf.String(), false)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&dateFilter, "date", "", "filter by date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&listTemplateFilter, "template", "", "filter by template name")
	listCmd.Flags().BoolVar(&listIDOnly, "id-only", false, "print just entry IDs, one per line")
	rootCmd.AddCommand(listCmd)
}
