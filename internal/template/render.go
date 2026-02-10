package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// Render executes a Go text/template with the provided variables.
// It takes a template content string and a map of variable names to values,
// and returns the rendered result.
//
// The template uses Go's standard text/template syntax:
//   - {{.varname}} - inserts the value of a variable
//   - Missing variables render as "<no value>"
//
// Parameters:
//   - tmplContent: The template content using Go text/template syntax
//   - vars: A map of variable names to values for substitution
//
// Returns:
//   - The rendered template content, or an error if template parsing/execution fails
//
// Example:
//
//	content, err := Render("Hello {{.name}}", map[string]string{"name": "Alice"})
//	// content = "Hello Alice"
func Render(tmplContent string, vars map[string]string) (string, error) {
	// Parse the template
	tmpl, err := template.New("content").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	// Execute the template with vars
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
