package cmd

import (
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/shell"
	"github.com/spf13/cobra"
)

var initShellCmd = &cobra.Command{
	Use:   "init <shell>",
	Short: "Output shell integration script",
	Long: `Output shell integration script for eval.

Generates shell-specific initialization code that sets up:
- Shell completions
- Prompt hook for diary status env vars
- diaryctl_prompt_info helper function

Supported shells: bash, zsh`,
	Example: `  # Add to ~/.bashrc
  eval "$(diaryctl init bash)"

  # Add to ~/.zshrc
  eval "$(diaryctl init zsh)"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			shell.WriteBashInit(os.Stdout)
		case "zsh":
			shell.WriteZshInit(os.Stdout)
		default:
			fmt.Fprintf(os.Stderr, "Error: unsupported shell %q (supported: bash, zsh)\n", args[0])
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initShellCmd)
}
