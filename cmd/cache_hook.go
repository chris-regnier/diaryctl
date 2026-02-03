package cmd

import (
	"github.com/chris-regnier/diaryctl/internal/shell"
	"github.com/spf13/cobra"
)

// invalidateCachePostRun is a PostRunE hook that invalidates the prompt cache
// after mutating commands (create, jot, edit, update, delete).
func invalidateCachePostRun(cmd *cobra.Command, args []string) error {
	if appConfig == nil {
		return nil
	}
	// Best-effort: ignore errors — cache invalidation should never
	// prevent a successful command from reporting success.
	_ = shell.InvalidateCache(appConfig.DataDir)
	return nil
}
