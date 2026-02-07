package context

import (
	"context"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewDiaryMCPServer creates an in-memory MCP server exposing diary tools.
// Returns the server and a client transport for connecting to it.
func NewDiaryMCPServer(store storage.Storage) (*mcp.Server, mcp.Transport) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := createMCPServer(store)

	// Start server in background
	go func() {
		_, _ = server.Connect(context.Background(), serverTransport, nil)
	}()

	return server, clientTransport
}

// CreateMCPServer creates an MCP server with registered diary tools.
// Use this to create a server that can be connected with any transport.
func CreateMCPServer(store storage.Storage) *mcp.Server {
	return createMCPServer(store)
}

func createMCPServer(store storage.Storage) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "diaryctl",
		Version: "1.0.0",
	}, nil)

	// Register search_entries tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_entries",
		Description: "Fuzzy search diary entries by content",
	}, SearchHandler(store))

	// Register filter_entries tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "filter_entries",
		Description: "Filter diary entries by date range and template",
	}, FilterHandler(store))

	return server
}

// SearchHandler returns the handler function for the search_entries MCP tool.
func SearchHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
		limit := input.Limit
		if limit <= 0 {
			limit = 10
		}

		entries, err := store.List(storage.ListOptions{Limit: limit * 2}) // fetch extra for filtering
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

// FilterHandler returns the handler function for the filter_entries MCP tool.
func FilterHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input FilterInput) (*mcp.CallToolResult, FilterOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input FilterInput) (*mcp.CallToolResult, FilterOutput, error) {
		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		// Parse dates if provided
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

		// Note: TemplateNames filtering would need List to support multiple templates
		// For now, use first template if provided
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

func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
