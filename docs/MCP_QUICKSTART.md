# MCP Server Quick Start

Get your diaryctl MCP server running in 3 steps:

## 1. Build diaryctl

```bash
go build -o diaryctl .
```

## 2. Run the server

```bash
./diaryctl mcp-serve
```

The server will start and log to stderr:
```
2026/02/06 19:39:53 Starting diaryctl MCP server (stdio transport)
2026/02/06 19:39:53 Storage backend: markdown
2026/02/06 19:39:53 Data directory: /Users/you/.diaryctl
```

## 3. Configure Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "diaryctl": {
      "command": "/absolute/path/to/diaryctl",
      "args": ["mcp-serve"]
    }
  }
}
```

Replace `/absolute/path/to/diaryctl` with your actual path. Get it with:
```bash
pwd
# Output: /path/to/diaryctl
# Use: /path/to/diaryctl/diaryctl
```

## 4. Restart Claude Desktop

Quit and reopen Claude Desktop. Your diary entries are now queryable!

## Test it

Ask Claude:
- "Search my diary for entries about coding"
- "Show me what I wrote last week"
- "Find all entries from January 2026"

## Need Help?

See [docs/mcp-server.md](./mcp-server.md) for detailed documentation.
