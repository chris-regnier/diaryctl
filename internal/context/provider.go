package context

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
)

// EntryResult represents a search result from the context provider.
type EntryResult struct {
	ID      string  `json:"id"`
	Preview string  `json:"preview"`
	Date    string  `json:"date"`
	Score   float64 `json:"score"`
}

// EnrichedContent holds LLM-enriched template output.
type EnrichedContent struct {
	Content         string   `json:"content"`
	SuggestedTags   []string `json:"suggested_tags"`
	Mood            string   `json:"mood,omitempty"`
	FollowUpPrompts []string `json:"follow_up_prompts"`
}

// Provider defines the interface for context enrichment operations.
type Provider interface {
	// Search performs fuzzy search over diary entries.
	Search(ctx context.Context, query string, limit int) ([]EntryResult, error)

	// Filter retrieves entries matching date/template criteria.
	Filter(ctx context.Context, opts storage.ListOptions) ([]EntryResult, error)
}
