package mcptools

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FilterHandler returns the handler function for the filter_entries MCP tool.
func FilterHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input FilterInput) (*mcp.CallToolResult, FilterOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input FilterInput) (*mcp.CallToolResult, FilterOutput, error) {
		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		if input.StartDate != "" {
			t, err := parseDate(input.StartDate)
			if err == nil {
				opts.StartDate = &t
			}
		}
		if input.EndDate != "" {
			t, err := parseDate(input.EndDate)
			if err == nil {
				opts.EndDate = &t
			}
		}

		if len(input.TemplateNames) > 0 {
			opts.TemplateName = input.TemplateNames[0]
		}

		entries, err := store.List(opts)
		if err != nil {
			return nil, FilterOutput{}, err
		}

		var results []EntryResult
		for _, e := range entries {
			results = append(results, EntryResult{
				ID:      e.ID,
				Preview: e.Preview(100),
				Date:    e.CreatedAt.Format("2006-01-02"),
				Score:   1.0,
			})
		}

		return nil, FilterOutput{Entries: results}, nil
	}
}
