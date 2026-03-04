package cmd

import (
	"fmt"
	"os"

	"github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/entry"
)

// buildContentProviders creates ContentProviders from config names, skipping unknown.
func buildContentProviders(names []string) []context.ContentProvider {
	var providers []context.ContentProvider
	for _, name := range names {
		p := context.LookupContentProvider(name)
		if p != nil {
			providers = append(providers, p)
		}
	}
	return providers
}

// buildContextResolvers creates ContextResolvers from config names, skipping unknown.
func buildContextResolvers(names []string) []context.ContextResolver {
	var resolvers []context.ContextResolver
	for _, name := range names {
		r := context.LookupContextResolver(name)
		if r != nil {
			resolvers = append(resolvers, r)
		}
	}
	return resolvers
}

// resolveContexts is a convenience function that builds resolvers from config,
// loads manual contexts, calls ResolveActiveContexts, and prints warnings.
// Returns the resolved context refs.
func resolveContexts() []entry.ContextRef {
	resolvers := buildContextResolvers(appConfig.ContextResolvers)
	manual, err := context.LoadManualContexts(appConfig.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load manual contexts: %v\n", err)
	}
	refs, warnings := context.ResolveActiveContexts(resolvers, manual, store)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}
	return refs
}
