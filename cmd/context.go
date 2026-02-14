package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/ui"
	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage contexts",
	Long:  "Manage semantic contexts for grouping diary entries.",
}

var contextListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all contexts",
	Example: `  diaryctl context list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		contexts, err := store.ListContexts()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if jsonOutput {
			ui.FormatJSON(os.Stdout, contexts)
		} else {
			ui.FormatContextList(os.Stdout, contexts)
		}
		return nil
	},
}

var contextShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a context",
	Example: `  diaryctl context show feature/auth`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ctx, err := store.GetContextByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: context %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if jsonOutput {
			ui.FormatJSON(os.Stdout, ctx)
		} else {
			ui.FormatContextFull(os.Stdout, ctx)
		}
		return nil
	},
}

var forceDeleteContext bool

var contextDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete a context",
	Example: `  diaryctl context delete feature/auth`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ctx, err := store.GetContextByName(name)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "Error: context %q not found\n", name)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		if !forceDeleteContext {
			fmt.Fprintf(os.Stdout, "Context: %s (%s)\n", ctx.Name, ctx.ID)
			confirmed, err := ui.Confirm("Delete this context? This cannot be undone.", ui.ResolveTheme(appConfig.Theme))
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(2)
			}
			if !confirmed {
				fmt.Fprintln(os.Stdout, "Cancelled.")
				return nil
			}
		}
		if err := store.DeleteContext(ctx.ID); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Deleted context %q.\n", name)
		return nil
	},
}

var contextSetCmd = &cobra.Command{
	Use:     "set <name>",
	Short:   "Activate a manual context",
	Example: `  diaryctl context set sprint:23`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := context.SetManualContext(appConfig.DataDir, name); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Activated context %q.\n", name)
		return nil
	},
}

var contextUnsetCmd = &cobra.Command{
	Use:     "unset <name>",
	Short:   "Deactivate a manual context",
	Example: `  diaryctl context unset sprint:23`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := context.UnsetManualContext(appConfig.DataDir, name); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "Deactivated context %q.\n", name)
		return nil
	},
}

var contextActiveCmd = &cobra.Command{
	Use:     "active",
	Short:   "Show currently active contexts",
	Example: `  diaryctl context active`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manual, err := context.LoadManualContexts(appConfig.DataDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(2)
		}

		var autoContexts []string
		for _, name := range appConfig.ContextResolvers {
			r := context.LookupContextResolver(name)
			if r == nil {
				continue
			}
			names, err := r.Resolve()
			if err != nil {
				continue
			}
			autoContexts = append(autoContexts, names...)
		}

		if jsonOutput {
			ui.FormatJSON(os.Stdout, map[string][]string{
				"manual": manual,
				"auto":   autoContexts,
			})
		} else {
			ui.FormatActiveContexts(os.Stdout, manual, autoContexts)
		}
		return nil
	},
}

func init() {
	contextDeleteCmd.Flags().BoolVar(&forceDeleteContext, "force", false, "skip confirmation prompt")

	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	contextCmd.AddCommand(contextSetCmd)
	contextCmd.AddCommand(contextUnsetCmd)
	contextCmd.AddCommand(contextActiveCmd)

	rootCmd.AddCommand(contextCmd)
}
