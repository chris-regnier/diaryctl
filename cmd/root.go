package cmd

import (
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	"github.com/chris-regnier/diaryctl/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

var (
	cfgFile        string
	jsonOutput     bool
	storageBackend string
	appConfig      *config.Config
	store          storage.Storage
)

var rootCmd = &cobra.Command{
	Use:   "diaryctl",
	Short: "A diary management CLI tool",
	Long:  "diaryctl is a command-line tool for managing personal diary entries with pluggable storage backends.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		appConfig = cfg

		// Override storage backend from flag
		if storageBackend != "" {
			appConfig.Storage = storageBackend
		}

		// Initialize storage backend
		switch appConfig.Storage {
		case "markdown":
			store, err = markdown.New(appConfig.DataDir)
			if err != nil {
				return fmt.Errorf("initializing markdown storage: %w", err)
			}
		case "sqlite":
			store, err = sqlite.New(appConfig.DataDir)
			if err != nil {
				return fmt.Errorf("initializing sqlite storage: %w", err)
			}
		default:
			return fmt.Errorf("unknown storage backend: %s", appConfig.Storage)
		}

		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().StringVar(&storageBackend, "storage", "", "storage backend (markdown|sqlite)")

	// Silence Cobra's built-in error and usage printing so we control stderr output
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
}

func exitWithError(msg string, code int) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
