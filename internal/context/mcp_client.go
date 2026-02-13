package context

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chris-regnier/diaryctl/internal/mcptools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient wraps an MCP client session for calling tools.
type MCPClient struct {
	session *mcp.ClientSession
}

// NewMCPClient creates a client connected to the given transport.
func NewMCPClient(transport mcp.Transport) (*MCPClient, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "diaryctl-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to MCP server: %w", err)
	}

	return &MCPClient{session: session}, nil
}

// CallTool invokes a tool by name with the given arguments.
func (c *MCPClient) CallTool(ctx context.Context, name string, args any) (any, error) {
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, fmt.Errorf("call tool %s: %w", name, err)
	}

	// Return structured content if available
	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	// Fall back to parsing text content
	if len(result.Content) > 0 {
		contentJSON, err := json.Marshal(result.Content[0])
		if err != nil {
			return nil, fmt.Errorf("marshal content: %w", err)
		}

		var textContent struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(contentJSON, &textContent); err != nil {
			return nil, fmt.Errorf("unmarshal content: %w", err)
		}

		var rawOutput any
		if err := json.Unmarshal([]byte(textContent.Text), &rawOutput); err != nil {
			return nil, fmt.Errorf("unmarshal text: %w", err)
		}
		return rawOutput, nil
	}

	return nil, nil
}

// Search calls the search_entries tool.
func (c *MCPClient) Search(ctx context.Context, query string, limit int) ([]EntryResult, error) {
	result, err := c.CallTool(ctx, "search_entries", mcptools.SearchInput{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	// Convert result to SearchOutput
	outputJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}

	var output mcptools.SearchOutput
	if err := json.Unmarshal(outputJSON, &output); err != nil {
		return nil, fmt.Errorf("unmarshal search output: %w", err)
	}
	return convertEntryResults(output.Entries), nil
}

// Filter calls the filter_entries tool.
func (c *MCPClient) Filter(ctx context.Context, input mcptools.FilterInput) ([]EntryResult, error) {
	result, err := c.CallTool(ctx, "filter_entries", input)
	if err != nil {
		return nil, err
	}

	// Convert result to FilterOutput
	outputJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}

	var output mcptools.FilterOutput
	if err := json.Unmarshal(outputJSON, &output); err != nil {
		return nil, fmt.Errorf("unmarshal filter output: %w", err)
	}
	return convertEntryResults(output.Entries), nil
}

// convertEntryResults converts mcptools.EntryResult to context.EntryResult.
func convertEntryResults(results []mcptools.EntryResult) []EntryResult {
	out := make([]EntryResult, len(results))
	for i, r := range results {
		out[i] = EntryResult{
			ID:      r.ID,
			Preview: r.Preview,
			Date:    r.Date,
			Score:   r.Score,
		}
	}
	return out
}
