package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
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
func FormatEntryFull(w io.Writer, e entry.Entry) {
	fmt.Fprintf(w, "Entry: %s\n", e.ID)
	fmt.Fprintf(w, "Created: %s\n", e.CreatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintf(w, "Modified: %s\n", e.UpdatedAt.Local().Format("2006-01-02 15:04"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, e.Content)
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
