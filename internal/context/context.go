package context

import "strings"

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
