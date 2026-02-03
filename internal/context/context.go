package context

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/context/datetime"
	gitctx "github.com/chris-regnier/diaryctl/internal/context/git"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// ContentProvider generates text for the editor buffer.
type ContentProvider interface {
	Name() string
	Generate() (string, error)
}

// ContextResolver detects contexts from the environment.
type ContextResolver interface {
	Name() string
	Resolve() ([]string, error)
}

// ContextStore is the subset of storage.Storage needed for context resolution.
type ContextStore interface {
	GetContextByName(name string) (storage.Context, error)
	CreateContext(c storage.Context) error
}

// ResolveActiveContexts gathers contexts from resolvers and manual list,
// deduplicates, and ensures each exists in storage (creating if needed).
// Returns resolved context refs and a slice of warning messages.
// Warnings are informational only and never block entry creation.
func ResolveActiveContexts(
	resolvers []ContextResolver,
	manualContexts []string,
	store ContextStore,
) ([]entry.ContextRef, []string) {
	seen := make(map[string]string) // name -> source
	var warnings []string

	for _, r := range resolvers {
		names, err := r.Resolve()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("resolver %q failed: %v", r.Name(), err))
			continue
		}
		for _, name := range names {
			if name != "" {
				seen[name] = r.Name()
			}
		}
	}

	for _, name := range manualContexts {
		if name != "" {
			if _, exists := seen[name]; !exists {
				seen[name] = "manual"
			}
		}
	}

	if len(seen) == 0 {
		return nil, warnings
	}

	var refs []entry.ContextRef
	for name, source := range seen {
		c, err := store.GetContextByName(name)
		if err != nil {
			// Auto-create
			id, idErr := entry.NewID()
			if idErr != nil {
				warnings = append(warnings, fmt.Sprintf("failed to generate ID for context %q: %v", name, idErr))
				continue
			}
			now := time.Now().UTC()
			c = storage.Context{
				ID:        id,
				Name:      name,
				Source:    source,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if createErr := store.CreateContext(c); createErr != nil {
				warnings = append(warnings, fmt.Sprintf("failed to create context %q: %v", name, createErr))
				continue
			}
		}
		refs = append(refs, entry.ContextRef{
			ContextID:   c.ID,
			ContextName: c.Name,
		})
	}

	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ContextName < refs[j].ContextName
	})

	return refs, warnings
}

// ComposeContent assembles the editor buffer from provider output and template content.
// Order: provider outputs (in order, separated by newlines) -> blank line -> template content.
// Providers that return empty strings or errors are silently skipped.
func ComposeContent(providers []ContentProvider, templateContent string) string {
	var parts []string
	for _, p := range providers {
		output, err := p.Generate()
		if err != nil || output == "" {
			continue
		}
		parts = append(parts, output)
	}

	providerText := strings.Join(parts, "\n")

	switch {
	case providerText == "" && templateContent == "":
		return ""
	case providerText == "":
		return templateContent
	case templateContent == "":
		return providerText
	default:
		return providerText + "\n\n" + templateContent
	}
}

var contentProviders = map[string]func() ContentProvider{
	"datetime": func() ContentProvider { return datetime.New() },
	"git":      func() ContentProvider { return gitctx.NewContentProvider() },
}

var contextResolvers = map[string]func() ContextResolver{
	"git": func() ContextResolver { return gitctx.NewContextResolver() },
}

// LookupContentProvider returns a content provider by name, or nil if unknown.
func LookupContentProvider(name string) ContentProvider {
	factory, ok := contentProviders[name]
	if !ok {
		return nil
	}
	return factory()
}

// LookupContextResolver returns a context resolver by name, or nil if unknown.
func LookupContextResolver(name string) ContextResolver {
	factory, ok := contextResolvers[name]
	if !ok {
		return nil
	}
	return factory()
}
