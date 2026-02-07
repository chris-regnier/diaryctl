package cmd

import (
	"context"
	"log"
	"os"

	icontext "github.com/chris-regnier/diaryctl/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpServeCmd = &cobra.Command{
	Use:   "mcp-serve",
	Short: "Run MCP server on stdio",
	Long: `Starts a Model Context Protocol (MCP) server that exposes diary search
and filter tools over stdio transport. This allows MCP clients like Claude Desktop
to query your diary entries.

Available tools:
  - search_entries: Fuzzy text search over diary content
  - filter_entries: Filter entries by date range and template

Example usage in Claude Desktop config:
  {
    "mcpServers": {
      "diaryctl": {
        "command": "/path/to/diaryctl",
        "args": ["mcp-serve"]
      }
    }
  }`,
	RunE: runMCPServe,
}

func init() {
	rootCmd.AddCommand(mcpServeCmd)
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	// Storage is already initialized in PersistentPreRunE
	if store == nil {
		return cmd.Help()
	}

	// Create MCP server with registered tools
	server := icontext.CreateMCPServer(store)

	// Log to stderr (stdout is reserved for MCP protocol)
	log.SetOutput(os.Stderr)
	log.Printf("Starting diaryctl MCP server (stdio transport)")
	log.Printf("Storage backend: %s", appConfig.Storage)
	log.Printf("Data directory: %s", appConfig.DataDir)

	// Run server with stdio transport
	// This blocks until the transport is closed
	return server.Run(context.Background(), &mcp.StdioTransport{})
}
