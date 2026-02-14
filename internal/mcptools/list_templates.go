package mcptools

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListTemplatesHandler returns the handler function for the list_templates MCP tool.
func ListTemplatesHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesInput) (*mcp.CallToolResult, ListTemplatesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesInput) (*mcp.CallToolResult, ListTemplatesOutput, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 20
		}

		templates, err := store.ListTemplates()
		if err != nil {
			return nil, ListTemplatesOutput{}, err
		}

		if len(templates) > limit {
			templates = templates[:limit]
		}

		var results []TemplateResult
		for _, t := range templates {
			results = append(results, TemplateResult{
				ID:      t.ID,
				Name:    t.Name,
				Preview: truncate(t.Content, 200),
			})
		}

		return nil, ListTemplatesOutput{Templates: results}, nil
	}
}
