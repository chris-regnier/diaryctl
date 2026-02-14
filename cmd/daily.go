package cmd

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	dailyFrom           string
	dailyTo             string
	dailyNoInteractive  bool
	dailyTemplateFilter string
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Browse diary entries by day",
	Long: `Display a day-over-day aggregated view of diary entries.

In interactive mode (default in a terminal), launches a date picker
to browse and drill into entries by day.

In non-interactive mode (piped output, --no-interactive, or --json),
prints a grouped-by-day summary to stdout.`,
	Example: `  diaryctl daily
  diaryctl daily --from 2026-01-01 --to 2026-01-31
  diaryctl daily --template daily
  diaryctl daily --no-interactive
  diaryctl daily --no-interactive --json
  diaryctl daily | head -20`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse date flags
		var startDate, endDate *time.Time
		if dailyFrom != "" {
			t, err := time.ParseInLocation("2006-01-02", dailyFrom, time.Local)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: invalid --from date format (use YYYY-MM-DD):", dailyFrom)
				os.Exit(1)
			}
			startDate = &t
		}
		if dailyTo != "" {
			t, err := time.ParseInLocation("2006-01-02", dailyTo, time.Local)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: invalid --to date format (use YYYY-MM-DD):", dailyTo)
				os.Exit(1)
			}
			endDate = &t
		}

		// Mode selection
		nonInteractive := dailyNoInteractive || jsonOutput || !term.IsTerminal(int(os.Stdout.Fd()))

		if nonInteractive {
			return runDailyNonInteractive(startDate, endDate, dailyTemplateFilter)
		}

		return ui.RunPicker(store, storage.ListDaysOptions{
			StartDate:    startDate,
			EndDate:      endDate,
			TemplateName: dailyTemplateFilter,
		}, ui.ResolveTheme(appConfig.Theme))
	},
}

func runDailyNonInteractive(startDate, endDate *time.Time, templateFilter string) error {
	days, err := store.ListDays(storage.ListDaysOptions{
		StartDate:    startDate,
		EndDate:      endDate,
		TemplateName: templateFilter,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(2)
	}

	// Build DayEntries for each day
	dayEntries := make([]ui.DayEntries, 0, len(days))
	for _, d := range days {
		date := d.Date
		entries, err := store.List(storage.ListOptions{Date: &date})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		dayEntries = append(dayEntries, ui.DayEntries{
			Date:    d.Date,
			Entries: entries,
		})
	}

	if jsonOutput {
		if len(dayEntries) == 0 {
			fmt.Println("[]")
			return nil
		}
		groups := ui.BuildDayGroups(dayEntries)
		return ui.FormatJSON(os.Stdout, groups)
	}

	var buf bytes.Buffer
	ui.FormatDailySummary(&buf, dayEntries)
	return ui.OutputOrPage(os.Stdout, buf.String(), false, ui.ResolveTheme(appConfig.Theme))
}

func init() {
	dailyCmd.Flags().StringVar(&dailyFrom, "from", "", "start date filter (YYYY-MM-DD, inclusive)")
	dailyCmd.Flags().StringVar(&dailyTo, "to", "", "end date filter (YYYY-MM-DD, inclusive)")
	dailyCmd.Flags().BoolVar(&dailyNoInteractive, "no-interactive", false, "force non-interactive output")
	dailyCmd.Flags().StringVar(&dailyTemplateFilter, "template", "", "filter by template name")
	rootCmd.AddCommand(dailyCmd)
}
