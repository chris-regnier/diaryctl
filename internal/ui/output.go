package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// FormatEntryCreated formats a creation confirmation message.
func FormatEntryCreated(w io.Writer, e entry.Entry) {
	fmt.Fprintf(w, "Created entry %s (%s)\n", e.ID, e.CreatedAt.Local().Format("2006-01-02 15:04"))
}

// FormatEntryUpdated formats an update confirmation message.
func FormatEntryUpdated(w io.Writer, e entry.Entry) {
	fmt.Fprintf(w, "Updated entry %s (%s)\n", e.ID, e.UpdatedAt.Local().Format("2006-01-02 15:04"))
}

// FormatEntryDeleted formats a deletion confirmation message.
func FormatEntryDeleted(w io.Writer, id string) {
	fmt.Fprintf(w, "Deleted entry %s.\n", id)
}

// FormatNoChanges formats a "no changes" message.
func FormatNoChanges(w io.Writer, id string) {
	fmt.Fprintf(w, "No changes detected for entry %s.\n", id)
}

// FormatEntryFull formats a full entry display with metadata header.
// The markdownStyle parameter controls glamour rendering (e.g. "dark", "light").
func FormatEntryFull(w io.Writer, e entry.Entry, markdownStyle string) {
	fmt.Fprintf(w, "Entry: %s\n", e.ID)
	fmt.Fprintf(w, "Created: %s\n", e.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Modified: %s\n", e.UpdatedAt.Local().Format("2006-01-02 15:04"))
	if len(e.Templates) > 0 {
		names := make([]string, len(e.Templates))
		for i, ref := range e.Templates {
			names[i] = ref.TemplateName
		}
		fmt.Fprintf(w, "Templates: %s\n", strings.Join(names, ", "))
	}
	if len(e.Contexts) > 0 {
		names := make([]string, len(e.Contexts))
		for i, ref := range e.Contexts {
			names[i] = ref.ContextName
		}
		fmt.Fprintf(w, "Contexts: %s\n", strings.Join(names, ", "))
	}
	fmt.Fprintln(w)

	// Render markdown content as rich text
	// Use a reasonable default width (80 chars) which will be adjusted by the pager if used
	rendered := RenderMarkdownWithStyle(e.Content, 80, markdownStyle)
	fmt.Fprintln(w, rendered)
}

// FormatEntryList formats a list of entries as a table.
func FormatEntryList(w io.Writer, entries []entry.Entry) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No diary entries found.")
		return
	}
	for _, e := range entries {
		fmt.Fprintf(w, "%s  %s  %s\n",
			e.ID,
			e.CreatedAt.Local().Format("2006-01-02 15:04"),
			e.Preview(60),
		)
	}
}

// FormatTemplateList formats a list of templates.
func FormatTemplateList(w io.Writer, templates []storage.Template) {
	if len(templates) == 0 {
		fmt.Fprintln(w, "No templates found.")
		return
	}
	for _, t := range templates {
		fmt.Fprintf(w, "%s  %s  %s\n", t.Name, t.ID, t.UpdatedAt.Local().Format("2006-01-02 15:04"))
	}
}

// FormatTemplateFull formats a full template display.
func FormatTemplateFull(w io.Writer, t storage.Template) {
	fmt.Fprintf(w, "Template: %s\n", t.Name)
	fmt.Fprintf(w, "ID: %s\n", t.ID)
	fmt.Fprintf(w, "Created: %s\n", t.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Modified: %s\n", t.UpdatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, t.Content)
}

// FormatJSON writes any value as JSON to the writer.
func FormatJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// EntrySummary is a JSON representation for list output.
type EntrySummary struct {
	ID        string    `json:"id"`
	Preview   string    `json:"preview"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToSummaries converts entries to summary format for JSON list output.
func ToSummaries(entries []entry.Entry) []EntrySummary {
	summaries := make([]EntrySummary, len(entries))
	for i, e := range entries {
		summaries[i] = EntrySummary{
			ID:        e.ID,
			Preview:   e.Preview(60),
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		}
	}
	return summaries
}

// DeleteResult is a JSON representation for delete output.
type DeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// DayGroupJSON is the JSON representation of a daily aggregate.
type DayGroupJSON struct {
	Date    string         `json:"date"`
	Count   int            `json:"count"`
	Entries []EntrySummary `json:"entries"`
}

// BuildDayGroups creates DayGroupJSON slices from entries grouped by day.
func BuildDayGroups(days []DayEntries) []DayGroupJSON {
	groups := make([]DayGroupJSON, len(days))
	for i, d := range days {
		entries := make([]EntrySummary, len(d.Entries))
		for j, e := range d.Entries {
			entries[j] = EntrySummary{
				ID:        e.ID,
				Preview:   e.Preview(80),
				CreatedAt: e.CreatedAt,
				UpdatedAt: e.UpdatedAt,
			}
		}
		groups[i] = DayGroupJSON{
			Date:    d.Date.Format("2006-01-02"),
			Count:   len(d.Entries),
			Entries: entries,
		}
	}
	return groups
}

// DayEntries pairs a date with its entries for formatting.
type DayEntries struct {
	Date    time.Time
	Entries []entry.Entry
}

// FormatContextList formats a list of contexts.
func FormatContextList(w io.Writer, contexts []storage.Context) {
	if len(contexts) == 0 {
		fmt.Fprintln(w, "No contexts found.")
		return
	}
	for _, c := range contexts {
		fmt.Fprintf(w, "%s  %s  %s  %s\n", c.Name, c.Source, c.ID, c.UpdatedAt.Local().Format("2006-01-02 15:04"))
	}
}

// FormatContextFull formats full details of a context.
func FormatContextFull(w io.Writer, c storage.Context) {
	fmt.Fprintf(w, "Context: %s\n", c.Name)
	fmt.Fprintf(w, "ID: %s\n", c.ID)
	fmt.Fprintf(w, "Source: %s\n", c.Source)
	fmt.Fprintf(w, "Created: %s\n", c.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Modified: %s\n", c.UpdatedAt.Local().Format("2006-01-02 15:04"))
}

// FormatActiveContexts formats the currently active contexts.
func FormatActiveContexts(w io.Writer, manual []string, auto []string) {
	if len(manual) == 0 && len(auto) == 0 {
		fmt.Fprintln(w, "No active contexts.")
		return
	}
	if len(manual) > 0 {
		fmt.Fprintf(w, "manual:  %s\n", strings.Join(manual, ", "))
	}
	if len(auto) > 0 {
		fmt.Fprintf(w, "auto:    %s\n", strings.Join(auto, ", "))
	}
}

// FormatDailySummary formats grouped-by-day entries as plain text.
func FormatDailySummary(w io.Writer, days []DayEntries) {
	if len(days) == 0 {
		fmt.Fprintln(w, "No diary entries found.")
		return
	}
	for i, d := range days {
		label := "entries"
		if len(d.Entries) == 1 {
			label = "entry"
		}
		fmt.Fprintf(w, "── %s (%d %s) ──────────\n",
			d.Date.Format("2006-01-02"), len(d.Entries), label)
		for _, e := range d.Entries {
			fmt.Fprintf(w, "  %s  %s  %s\n",
				e.ID,
				e.CreatedAt.Local().Format("15:04"),
				e.Preview(80),
			)
		}
		if i < len(days)-1 {
			fmt.Fprintln(w)
		}
	}
}
