package cmd

import (
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/context"
)

// resolveAndAttachContexts resolves active contexts (from resolvers + manual state)
// and attaches them to the given entry. Failures are logged as warnings but never
// block entry creation.
func resolveAndAttachContexts(entryID string) {
	var resolvers []context.ContextResolver
	for _, name := range appConfig.ContextResolvers {
		r := context.LookupContextResolver(name)
		if r != nil {
			resolvers = append(resolvers, r)
		}
	}

	manualContexts, _ := context.LoadManualContexts(appConfig.DataDir)

	refs, warnings := context.ResolveActiveContexts(resolvers, manualContexts, store)

	// Print any warnings from context resolution
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	for _, ref := range refs {
		if err := store.AttachContext(entryID, ref.ContextID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: attaching context %q: %v\n", ref.ContextName, err)
		}
	}
}

// buildContentProviders returns content providers from config for editor buffer composition.
func buildContentProviders() []context.ContentProvider {
	var providers []context.ContentProvider
	for _, name := range appConfig.ContextProviders {
		p := context.LookupContentProvider(name)
		if p != nil {
			providers = append(providers, p)
		}
	}
	return providers
}
