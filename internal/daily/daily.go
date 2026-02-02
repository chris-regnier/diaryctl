package daily

import (
	"fmt"
	"os"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/template"
)

// GetOrCreateToday finds today's entry or creates a new one.
// If defaultTemplate is non-empty, it composes the template content for new entries.
// If the default template is not found, a warning is printed to stderr and an entry
// with default content is created.
// Returns the entry, whether it was newly created, and any error.
func GetOrCreateToday(store storage.Storage, defaultTemplate string) (entry.Entry, bool, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Try to find today's entry
	entries, err := store.List(storage.ListOptions{
		Date:  &today,
		Limit: 1,
	})
	if err != nil {
		return entry.Entry{}, false, fmt.Errorf("listing today's entries: %w", err)
	}
	if len(entries) > 0 {
		return entries[0], false, nil
	}

	// No entry for today — create one
	content := fmt.Sprintf("# %s", now.Format("2006-01-02"))
	var refs []entry.TemplateRef

	if defaultTemplate != "" {
		names := template.ParseNames(defaultTemplate)
		c, r, err := template.Compose(store, names)
		if err != nil {
			// Warn but don't block entry creation
			fmt.Fprintf(os.Stderr, "Warning: default template %q not found, skipping\n", defaultTemplate)
		} else {
			content = c
			refs = r
		}
	}

	id, err := entry.NewID()
	if err != nil {
		return entry.Entry{}, false, fmt.Errorf("generating entry ID: %w", err)
	}

	nowUTC := now.UTC()
	e := entry.Entry{
		ID:        id,
		Content:   content,
		CreatedAt: nowUTC,
		UpdatedAt: nowUTC,
		Templates: refs,
	}
	if err := store.Create(e); err != nil {
		return entry.Entry{}, false, fmt.Errorf("creating today's entry: %w", err)
	}
	return e, true, nil
}
