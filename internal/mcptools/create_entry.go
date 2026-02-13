package mcptools

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/shell"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/template"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateEntryHandler returns the handler function for the create_entry MCP tool.
func CreateEntryHandler(store storage.Storage, dataDir string) func(ctx context.Context, req *mcp.CallToolRequest, input CreateEntryInput) (*mcp.CallToolResult, CreateEntryOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CreateEntryInput) (*mcp.CallToolResult, CreateEntryOutput, error) {
		var content string
		var refs []entry.TemplateRef

		// Compose templates if specified
		if len(input.TemplateNames) > 0 {
			composed, templateRefs, err := template.Compose(store, input.TemplateNames)
			if err != nil {
				return nil, CreateEntryOutput{}, err
			}
			refs = templateRefs

			// Render variables if specified
			if len(input.TemplateVariables) > 0 {
				composed, err = template.Render(composed, input.TemplateVariables)
				if err != nil {
					return nil, CreateEntryOutput{}, err
				}
			}

			content = composed + "\n\n" + input.Content
		} else {
			content = input.Content
		}

		// Create entry (validates content + generates ID)
		e, err := entry.New(content, refs)
		if err != nil {
			return nil, CreateEntryOutput{}, err
		}

		if err := store.Create(e); err != nil {
			return nil, CreateEntryOutput{}, err
		}

		// Invalidate shell prompt cache (best-effort)
		if dataDir != "" {
			_ = shell.InvalidateCache(dataDir)
		}

		return nil, CreateEntryOutput{
			ID:      e.ID,
			Date:    e.CreatedAt.Format("2006-01-02"),
			Preview: e.Preview(200),
		}, nil
	}
}
