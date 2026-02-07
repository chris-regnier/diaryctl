# MCP Server

diaryctl includes a built-in Model Context Protocol (MCP) server that exposes diary search and filtering capabilities to AI assistants and other MCP clients.

## Overview

The MCP server provides two tools:
- **search_entries**: Fuzzy text search over diary entry content
- **filter_entries**: Filter entries by date range and template

## Running the Server

Start the MCP server with:

```bash
diaryctl mcp-serve
```

The server uses stdio transport for communication, making it compatible with MCP clients like Claude Desktop.

### Configuration Options

You can configure the storage backend and data directory:

```bash
# Use SQLite backend
diaryctl mcp-serve --storage sqlite

# Custom config file
diaryctl mcp-serve --config /path/to/config.toml
```

## Claude Desktop Integration

To use diaryctl with Claude Desktop, add it to your MCP servers configuration:

### macOS/Linux

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "diaryctl": {
      "command": "/path/to/diaryctl",
      "args": ["mcp-serve"]
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "diaryctl": {
      "command": "C:\\path\\to\\diaryctl.exe",
      "args": ["mcp-serve"]
    }
  }
}
```

## Available Tools

### search_entries

Performs fuzzy text search over diary entry content.

**Input:**
```json
{
  "query": "text to search for",
  "limit": 10
}
```

**Output:**
```json
{
  "entries": [
    {
      "id": "abc12345",
      "preview": "First 100 characters of entry...",
      "date": "2026-01-15",
      "score": 1.0
    }
  ]
}
```

### filter_entries

Filters diary entries by date range and/or template.

**Input:**
```json
{
  "start_date": "2026-01-01",
  "end_date": "2026-01-31",
  "template_names": ["daily", "work"],
  "limit": 10
}
```

**Output:**
```json
{
  "entries": [
    {
      "id": "xyz67890",
      "preview": "First 100 characters of entry...",
      "date": "2026-01-15",
      "score": 1.0
    }
  ]
}
```

## Example Queries in Claude

Once configured, you can ask Claude questions like:

- "Search my diary for entries about 'project launch'"
- "Show me diary entries from last week"
- "Find entries tagged with the 'work' template"
- "What did I write about on January 15th?"

## Logging

The server logs to stderr (since stdout is reserved for MCP protocol messages). When running, you'll see:

```
2026/02/06 19:39:53 Starting diaryctl MCP server (stdio transport)
2026/02/06 19:39:53 Storage backend: markdown
2026/02/06 19:39:53 Data directory: /Users/you/.diaryctl
```

## Architecture

The MCP server implementation consists of:

- `internal/context/mcp_server.go` - Core server with tool handlers
- `internal/context/mcp_client.go` - Client wrapper (for testing)
- `internal/context/composite.go` - Provider interface implementation
- `cmd/mcp_serve.go` - CLI command

The server uses the official Go SDK: `github.com/modelcontextprotocol/go-sdk`

## Development

To test the MCP server locally:

```bash
# Build
go build -o diaryctl .

# Run with test data
diaryctl seed  # Generate test entries
diaryctl mcp-serve
```

For programmatic testing, see `internal/context/mcp_server_test.go`.
