package context

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/mcptools"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// CompositeProvider implements Provider using the local MCP server.
type CompositeProvider struct {
	client *MCPClient
}

// NewCompositeProvider creates a provider backed by an in-memory MCP server.
func NewCompositeProvider(store storage.Storage) (*CompositeProvider, error) {
	_, transport := mcptools.NewDiaryMCPServer(store)
	client, err := NewMCPClient(transport)
	if err != nil {
		return nil, err
	}

	return &CompositeProvider{client: client}, nil
}

// Search implements Provider.Search.
func (p *CompositeProvider) Search(ctx context.Context, query string, limit int) ([]EntryResult, error) {
	return p.client.Search(ctx, query, limit)
}

// Filter implements Provider.Filter.
func (p *CompositeProvider) Filter(ctx context.Context, opts storage.ListOptions) ([]EntryResult, error) {
	input := mcptools.FilterInput{
		Limit: opts.Limit,
	}
	if opts.StartDate != nil {
		input.StartDate = opts.StartDate.Format("2006-01-02")
	}
	if opts.EndDate != nil {
		input.EndDate = opts.EndDate.Format("2006-01-02")
	}
	if opts.TemplateName != "" {
		input.TemplateNames = []string{opts.TemplateName}
	}
	return p.client.Filter(ctx, input)
}

// Ensure CompositeProvider implements Provider.
var _ Provider = (*CompositeProvider)(nil)
