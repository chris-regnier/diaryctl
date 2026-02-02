package entry

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	idAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	idLength   = 8
)

var idPattern = regexp.MustCompile(`^[a-z0-9]{8}$`)
var templateNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// TemplateRef is a lightweight reference to a template, stored on entries for attribution.
type TemplateRef struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
}

// Entry represents a single diary entry.
type Entry struct {
	ID        string        `json:"id"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Templates []TemplateRef `json:"templates,omitempty"`
}

// NewID generates a new nanoid for an entry.
func NewID() (string, error) {
	return gonanoid.Generate(idAlphabet, idLength)
}

// ValidateID checks whether an ID matches the expected pattern.
func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid entry ID: %q (must be 8 lowercase alphanumeric characters)", id)
	}
	return nil
}

// ValidateContent checks whether content is non-empty.
func ValidateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("entry content must not be empty")
	}
	return nil
}

// ValidateTemplateName checks whether a template name is valid.
func ValidateTemplateName(name string) error {
	if !templateNamePattern.MatchString(name) {
		return fmt.Errorf("invalid template name %q: must be lowercase alphanumeric, hyphens, underscores", name)
	}
	return nil
}

// Preview returns a truncated preview of the entry content.
func (e *Entry) Preview(maxLen int) string {
	content := strings.ReplaceAll(e.Content, "\n", " ")
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}
