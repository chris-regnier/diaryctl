package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/chris-regnier/diaryctl/internal/shell"
	"github.com/spf13/cobra"
)

// statusData holds the template data for status formatting.
type statusData struct {
	TodayIcon  string
	Streak     int
	StreakIcon  string
	Template   string
	Backend    string
	HasToday   bool
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show diary prompt status",
	Long: `Show diary status for shell prompt integration.

Outputs today indicator, streak count, and context information.
Reads from cache when fresh, queries storage when stale.

Use --env to output shell environment variable assignments.
Use --refresh to force a cache refresh.
Use --format with a Go template for custom output.`,
	Example: `  diaryctl status
  diaryctl status --env
  diaryctl status --refresh
  diaryctl status --format "{{.TodayIcon}} {{.Streak}}{{.StreakIcon}}"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		envFlag, _ := cmd.Flags().GetBool("env")
		refreshFlag, _ := cmd.Flags().GetBool("refresh")
		formatFlag, _ := cmd.Flags().GetString("format")

		// Parse cache TTL
		ttl, err := time.ParseDuration(appConfig.Shell.CacheTTL)
		if err != nil {
			ttl = 5 * time.Minute
		}

		// Read or refresh cache
		cache := shell.ReadCache(appConfig.DataDir)
		if refreshFlag || !cache.IsFresh(ttl) {
			todayExists, streak, err := shell.ComputeStatus(store)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error computing status:", err)
				os.Exit(2)
			}

			cache = &shell.PromptCache{
				Today:           todayExists,
				Streak:          streak,
				TodayDate:       time.Now().Format("2006-01-02"),
				DefaultTemplate: appConfig.DefaultTemplate,
				StorageBackend:  appConfig.Storage,
				UpdatedAt:       time.Now(),
			}

			if err := shell.WriteCache(appConfig.DataDir, cache); err != nil {
				// Non-fatal: cache write failure shouldn't break the prompt
				fmt.Fprintln(os.Stderr, "Warning: could not write cache:", err)
			}
		}

		// Build status data
		data := buildStatusData(cache)

		if envFlag {
			return outputEnv(data)
		}

		if formatFlag != "" {
			return outputTemplate(data, formatFlag)
		}

		return outputDefault(data)
	},
}

func buildStatusData(cache *shell.PromptCache) statusData {
	icon := appConfig.Shell.NoTodayIcon
	if cache.Today {
		icon = appConfig.Shell.TodayIcon
	}

	return statusData{
		TodayIcon: icon,
		Streak:    cache.Streak,
		StreakIcon: appConfig.Shell.StreakIcon,
		Template:  cache.DefaultTemplate,
		Backend:   cache.StorageBackend,
		HasToday:  cache.Today,
	}
}

func outputEnv(data statusData) error {
	fmt.Printf("export DIARYCTL_TODAY=%q\n", data.TodayIcon)
	fmt.Printf("export DIARYCTL_STREAK=%q\n", fmt.Sprintf("%d", data.Streak))
	fmt.Printf("export DIARYCTL_STREAK_ICON=%q\n", data.StreakIcon)
	if data.Template != "" {
		fmt.Printf("export DIARYCTL_TEMPLATE=%q\n", data.Template)
	}
	if data.Backend != "" {
		fmt.Printf("export DIARYCTL_BACKEND=%q\n", data.Backend)
	}
	return nil
}

func outputTemplate(data statusData, format string) error {
	tmpl, err := template.New("status").Parse(format)
	if err != nil {
		return fmt.Errorf("invalid format template: %w", err)
	}
	if err := tmpl.Execute(os.Stdout, data); err != nil {
		return fmt.Errorf("executing format template: %w", err)
	}
	fmt.Println()
	return nil
}

func outputDefault(data statusData) error {
	var parts []string

	// Today indicator + streak
	parts = append(parts, fmt.Sprintf("%s %d%s", data.TodayIcon, data.Streak, data.StreakIcon))

	// Context: template name
	if appConfig.Shell.ShowContext && data.Template != "" {
		parts = append(parts, data.Template)
	}

	// Optional: backend
	if appConfig.Shell.ShowBackend && data.Backend != "" {
		parts = append(parts, data.Backend)
	}

	fmt.Println(strings.Join(parts, " "))
	return nil
}

func init() {
	statusCmd.Flags().Bool("env", false, "output shell environment variable assignments")
	statusCmd.Flags().Bool("refresh", false, "force cache refresh")
	statusCmd.Flags().String("format", "", "Go template format string")
	rootCmd.AddCommand(statusCmd)
}
