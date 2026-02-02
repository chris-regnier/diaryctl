package template

import (
	"fmt"
	"strings"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// TemplateLoader is the subset of storage.Storage needed for template composition.
type TemplateLoader interface {
	GetTemplateByName(name string) (storage.Template, error)
}

// ParseNames splits a comma-separated template names string into a slice.
func ParseNames(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

// Compose loads the named templates and concatenates their content.
// Returns the combined content and a slice of TemplateRefs for attribution.
// If names is empty, returns ("", nil, nil).
// If any template is not found, returns an error immediately (fail fast).
func Compose(loader TemplateLoader, names []string) (string, []entry.TemplateRef, error) {
	if len(names) == 0 {
		return "", nil, nil
	}

	var parts []string
	var refs []entry.TemplateRef

	for _, name := range names {
		tmpl, err := loader.GetTemplateByName(name)
		if err != nil {
			return "", nil, fmt.Errorf("template %q: %w", name, err)
		}
		parts = append(parts, tmpl.Content)
		refs = append(refs, entry.TemplateRef{
			TemplateID:   tmpl.ID,
			TemplateName: tmpl.Name,
		})
	}

	return strings.Join(parts, "\n\n"), refs, nil
}
