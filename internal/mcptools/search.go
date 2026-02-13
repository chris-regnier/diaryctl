package mcptools

import (
	"context"
	"strings"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchHandler returns the handler function for the search_entries MCP tool.
func SearchHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 10
		}

		entries, err := store.List(storage.ListOptions{Limit: limit * 2})
		if err != nil {
			return nil, SearchOutput{}, err
		}

		query := strings.ToLower(input.Query)
		var results []EntryResult
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Content), query) {
				results = append(results, EntryResult{
					ID:      e.ID,
					Preview: e.Preview(100),
					Date:    e.CreatedAt.Format("2006-01-02"),
					Score:   1.0,
				})
				if len(results) >= limit {
					break
				}
			}
		}

		return nil, SearchOutput{Entries: results}, nil
	}
}
