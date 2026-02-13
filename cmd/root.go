package cmd

import (
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/config"
	"github.com/chris-regnier/diaryctl/internal/editor"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	"github.com/chris-regnier/diaryctl/internal/storage/sqlite"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			// Non-TTY: fall back to today's entry
			return todayRun(os.Stdout, false, false)
		}
		return ui.RunTUI(store, ui.TUIConfig{
			Editor:          editor.ResolveEditor(appConfig.Editor),
			DefaultTemplate: appConfig.DefaultTemplate,
			MaxWidth:        appConfig.MaxWidth,
			Theme:           ui.ResolveTheme(appConfig.Theme),
		})
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

// NewRootV2Command creates a root command for the v2 data model.
// This is a helper function for testing and programmatic usage of the CLI
// with the StorageV2 interface.
//
// Parameters:
//   - store: A StorageV2 implementation to use for all operations
//
// Returns:
//   - A configured cobra.Command with all v2 subcommands attached
func NewRootV2Command(store storage.StorageV2) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diaryctl",
		Short: "A diary management CLI tool",
		Long:  "diaryctl is a command-line tool for managing personal diary entries with a block-based data model.",
	}

	// Silence Cobra's built-in error and usage printing
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Add v2 commands
	cmd.AddCommand(NewJotV2Command(store))
	cmd.AddCommand(NewEditV2Command(store))

	return cmd
}
