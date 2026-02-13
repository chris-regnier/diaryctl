package mcptools

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewDiaryMCPServer creates an in-memory MCP server exposing diary tools.
// Returns the server and a client transport for connecting to it.
func NewDiaryMCPServer(store storage.Storage) (*mcp.Server, mcp.Transport) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := CreateMCPServer(store, "")

	go func() {
		_, _ = server.Connect(context.Background(), serverTransport, nil)
	}()

	return server, clientTransport
}

// CreateMCPServer creates an MCP server with registered diary tools.
// dataDir is used for cache invalidation after write operations; pass "" to skip.
func CreateMCPServer(store storage.Storage, dataDir string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "diaryctl",
		Version: "1.0.0",
	}, nil)

	// Read tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_entries",
		Description: "Fuzzy search diary entries by content",
	}, SearchHandler(store))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "filter_entries",
		Description: "Filter diary entries by date range and template",
	}, FilterHandler(store))

	return server
}
