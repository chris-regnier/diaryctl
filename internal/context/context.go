package context

import (
	"sort"
	"strings"
	"time"

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
// Failed resolvers are silently skipped.
func ResolveActiveContexts(
	resolvers []ContextResolver,
	manualContexts []string,
	store ContextStore,
) ([]entry.ContextRef, error) {
	seen := make(map[string]string) // name -> source

	for _, r := range resolvers {
		names, err := r.Resolve()
		if err != nil {
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
		return nil, nil
	}

	var refs []entry.ContextRef
	for name, source := range seen {
		c, err := store.GetContextByName(name)
		if err != nil {
			// Auto-create
			id, idErr := entry.NewID()
			if idErr != nil {
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

	return refs, nil
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
