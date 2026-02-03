# BAML + MCP SDK Context Providers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add context enrichment capabilities using BAML for type-safe LLM calls and the MCP SDK for exposing diary tools as an MCP server.

**Architecture:** Three-layer design: (1) BAML-generated client for structured LLM enrichment with provider fallback, (2) local MCP server exposing search/filter tools over diary data, (3) unified `ContextProvider` interface composing both. The MCP server uses in-memory transports for internal use and can be exposed externally.

**Tech Stack:** BAML (codegen), `github.com/modelcontextprotocol/go-sdk/mcp`, existing `storage.Storage` interface.

---

## Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add MCP SDK dependency**

```bash
go get github.com/modelcontextprotocol/go-sdk/mcp@latest
```

**Step 2: Verify dependency added**

Run: `go mod tidy && grep modelcontextprotocol go.mod`
Expected: Line containing `github.com/modelcontextprotocol/go-sdk`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add MCP SDK dependency"
```

---

## Task 2: Create Context Package Structure

**Files:**
- Create: `internal/context/provider.go`

**Step 1: Write the provider interface**

```go
package context

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
)

// EntryResult represents a search result from the context provider.
type EntryResult struct {
	ID      string  `json:"id"`
	Preview string  `json:"preview"`
	Date    string  `json:"date"`
	Score   float64 `json:"score"`
}

// EnrichedContent holds LLM-enriched template output.
type EnrichedContent struct {
	Content          string   `json:"content"`
	SuggestedTags    []string `json:"suggested_tags"`
	Mood             string   `json:"mood,omitempty"`
	FollowUpPrompts  []string `json:"follow_up_prompts"`
}

// Provider defines the interface for context enrichment operations.
type Provider interface {
	// Search performs fuzzy search over diary entries.
	Search(ctx context.Context, query string, limit int) ([]EntryResult, error)

	// Filter retrieves entries matching date/template criteria.
	Filter(ctx context.Context, opts storage.ListOptions) ([]EntryResult, error)
}
```

**Step 2: Verify file compiles**

Run: `go build ./internal/context/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/context/provider.go
git commit -m "feat(context): add Provider interface and result types"
```

---

## Task 3: Create MCP Server - Tool Types

**Files:**
- Create: `internal/context/mcp_types.go`

**Step 1: Write MCP tool input/output types**

```go
package context

// SearchInput is the input schema for the search_entries MCP tool.
type SearchInput struct {
	Query string `json:"query" jsonschema:"description=Text to search for in entry content"`
	Limit int    `json:"limit" jsonschema:"description=Maximum number of results to return"`
}

// SearchOutput is the output schema for the search_entries MCP tool.
type SearchOutput struct {
	Entries []EntryResult `json:"entries"`
}

// FilterInput is the input schema for the filter_entries MCP tool.
type FilterInput struct {
	StartDate     string   `json:"start_date,omitempty" jsonschema:"description=ISO date lower bound (inclusive)"`
	EndDate       string   `json:"end_date,omitempty" jsonschema:"description=ISO date upper bound (inclusive)"`
	TemplateNames []string `json:"template_names,omitempty" jsonschema:"description=Filter to entries using these templates"`
	Limit         int      `json:"limit" jsonschema:"description=Maximum number of results"`
}

// FilterOutput is the output schema for the filter_entries MCP tool.
type FilterOutput struct {
	Entries []EntryResult `json:"entries"`
}
```

**Step 2: Verify file compiles**

Run: `go build ./internal/context/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/context/mcp_types.go
git commit -m "feat(context): add MCP tool input/output types"
```

---

## Task 4: Create MCP Server - Test First

**Files:**
- Create: `internal/context/mcp_server_test.go`

**Step 1: Write the failing test**

```go
package context_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	icontext "github.com/chris-regnier/diaryctl/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServer_SearchEntries(t *testing.T) {
	// Setup: create temp storage with test entries
	dir := t.TempDir()
	store, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create test entries
	e1 := entry.Entry{
		ID:        "testid01",
		Content:   "Today I learned about Go interfaces",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	e2 := entry.Entry{
		ID:        "testid02",
		Content:   "Meeting notes from standup",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Create(e1); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}
	if err := store.Create(e2); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Create MCP server and client
	_, clientTransport := icontext.NewDiaryMCPServer(store)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	_, err = client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	// Call search_entries tool
	args, _ := json.Marshal(icontext.SearchInput{Query: "Go interfaces", Limit: 10})
	result, err := client.CallTool(context.Background(), &mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "search_entries",
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// Verify result contains matching entry
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var output icontext.SearchOutput
	if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if len(output.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(output.Entries))
	}
	if output.Entries[0].ID != "testid01" {
		t.Errorf("expected entry testid01, got %s", output.Entries[0].ID)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/context/... -v -run TestMCPServer_SearchEntries`
Expected: FAIL - `NewDiaryMCPServer` undefined

**Step 3: Commit failing test**

```bash
git add internal/context/mcp_server_test.go
git commit -m "test(context): add MCP server search test (red)"
```

---

## Task 5: Implement MCP Server

**Files:**
- Create: `internal/context/mcp_server.go`

**Step 1: Write the MCP server implementation**

```go
package context

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewDiaryMCPServer creates an in-memory MCP server exposing diary tools.
// Returns the server and a client transport for connecting to it.
func NewDiaryMCPServer(store storage.Storage) (*mcp.Server, mcp.Transport) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "diaryctl",
		Version: "1.0.0",
	}, nil)

	// Register search_entries tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_entries",
		Description: "Fuzzy search diary entries by content",
	}, searchHandler(store))

	// Register filter_entries tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "filter_entries",
		Description: "Filter diary entries by date range and template",
	}, filterHandler(store))

	// Start server in background
	go func() {
		_, _ = server.Connect(context.Background(), serverTransport, nil)
	}()

	return server, clientTransport
}

func searchHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
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

func filterHandler(store storage.Storage) func(ctx context.Context, req *mcp.CallToolRequest, input FilterInput) (*mcp.CallToolResult, FilterOutput, error) {
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
	return time.Parse("2006-01-02", s)
}
```

**Step 2: Add missing import**

Add `"time"` to the imports in `mcp_server.go`.

**Step 3: Run test to verify it passes**

Run: `go test ./internal/context/... -v -run TestMCPServer_SearchEntries`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/context/mcp_server.go
git commit -m "feat(context): implement MCP server with search and filter tools"
```

---

## Task 6: Add Filter Tool Test

**Files:**
- Modify: `internal/context/mcp_server_test.go`

**Step 1: Write the filter test**

Add this test function to `mcp_server_test.go`:

```go
func TestMCPServer_FilterEntries(t *testing.T) {
	dir := t.TempDir()
	store, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create entries on different dates
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	e1 := entry.Entry{
		ID:        "today001",
		Content:   "Entry from today",
		CreatedAt: now,
		UpdatedAt: now,
	}
	e2 := entry.Entry{
		ID:        "yester01",
		Content:   "Entry from yesterday",
		CreatedAt: yesterday,
		UpdatedAt: yesterday,
	}
	_ = store.Create(e1)
	_ = store.Create(e2)

	_, clientTransport := icontext.NewDiaryMCPServer(store)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	_, _ = client.Connect(context.Background(), clientTransport, nil)

	// Filter for today only
	args, _ := json.Marshal(icontext.FilterInput{
		StartDate: now.Format("2006-01-02"),
		EndDate:   now.Format("2006-01-02"),
		Limit:     10,
	})
	result, err := client.CallTool(context.Background(), &mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "filter_entries",
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	textContent := result.Content[0].(mcp.TextContent)
	var output icontext.FilterOutput
	_ = json.Unmarshal([]byte(textContent.Text), &output)

	if len(output.Entries) != 1 {
		t.Errorf("expected 1 entry for today, got %d", len(output.Entries))
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/context/... -v`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add internal/context/mcp_server_test.go
git commit -m "test(context): add MCP filter tool test"
```

---

## Task 7: Create MCP Client Wrapper

**Files:**
- Create: `internal/context/mcp_client.go`

**Step 1: Write the client wrapper**

```go
package context

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient wraps an MCP client for calling tools.
type MCPClient struct {
	client *mcp.Client
}

// NewMCPClient creates a client connected to the given transport.
func NewMCPClient(transport mcp.Transport) (*MCPClient, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "diaryctl-client",
		Version: "1.0.0",
	}, nil)

	_, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to MCP server: %w", err)
	}

	return &MCPClient{client: client}, nil
}

// CallTool invokes a tool by name with the given arguments.
func (c *MCPClient) CallTool(ctx context.Context, name string, args any) (json.RawMessage, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshal args: %w", err)
	}

	result, err := c.client.CallTool(ctx, &mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: argsJSON,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("call tool %s: %w", name, err)
	}

	if len(result.Content) == 0 {
		return nil, nil
	}

	if text, ok := result.Content[0].(mcp.TextContent); ok {
		return json.RawMessage(text.Text), nil
	}

	return nil, fmt.Errorf("unexpected content type: %T", result.Content[0])
}

// Search calls the search_entries tool.
func (c *MCPClient) Search(ctx context.Context, query string, limit int) ([]EntryResult, error) {
	raw, err := c.CallTool(ctx, "search_entries", SearchInput{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	var output SearchOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf("unmarshal search output: %w", err)
	}
	return output.Entries, nil
}

// Filter calls the filter_entries tool.
func (c *MCPClient) Filter(ctx context.Context, input FilterInput) ([]EntryResult, error) {
	raw, err := c.CallTool(ctx, "filter_entries", input)
	if err != nil {
		return nil, err
	}

	var output FilterOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf("unmarshal filter output: %w", err)
	}
	return output.Entries, nil
}
```

**Step 2: Verify compilation**

Run: `go build ./internal/context/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/context/mcp_client.go
git commit -m "feat(context): add MCP client wrapper with Search and Filter methods"
```

---

## Task 8: Create Composite Provider

**Files:**
- Create: `internal/context/composite.go`

**Step 1: Write the composite provider**

```go
package context

import (
	"context"

	"github.com/chris-regnier/diaryctl/internal/storage"
)

// CompositeProvider implements Provider using the local MCP server.
type CompositeProvider struct {
	client *MCPClient
}

// NewCompositeProvider creates a provider backed by an in-memory MCP server.
func NewCompositeProvider(store storage.Storage) (*CompositeProvider, error) {
	_, transport := NewDiaryMCPServer(store)
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
	input := FilterInput{
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
```

**Step 2: Verify compilation**

Run: `go build ./internal/context/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/context/composite.go
git commit -m "feat(context): add CompositeProvider implementing Provider interface"
```

---

## Task 9: Add Integration Test

**Files:**
- Create: `internal/context/composite_test.go`

**Step 1: Write integration test**

```go
package context_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	icontext "github.com/chris-regnier/diaryctl/internal/context"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
)

func TestCompositeProvider_Search(t *testing.T) {
	dir := t.TempDir()
	store, _ := markdown.New(dir)
	defer store.Close()

	_ = store.Create(entry.Entry{
		ID:        "search01",
		Content:   "Learning about MCP protocols",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	provider, err := icontext.NewCompositeProvider(store)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	results, err := provider.Search(context.Background(), "MCP", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "search01" {
		t.Errorf("expected search01, got %s", results[0].ID)
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/context/... -v -run TestCompositeProvider`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/context/composite_test.go
git commit -m "test(context): add CompositeProvider integration test"
```

---

## Task 10: Add BAML Configuration Files

**Files:**
- Create: `baml_src/clients.baml`
- Create: `baml_src/enrichment.baml`

**Step 1: Create BAML clients configuration**

Create `baml_src/clients.baml`:

```baml
client<llm> Claude {
  provider anthropic
  options {
    model "claude-sonnet-4-20250514"
    api_key env.ANTHROPIC_API_KEY
  }
}

client<llm> GPT {
  provider openai
  options {
    model "gpt-4o"
    api_key env.OPENAI_API_KEY
  }
}

client<llm> Primary {
  provider fallback
  options {
    strategy [Claude, GPT]
  }
}
```

**Step 2: Create BAML enrichment definitions**

Create `baml_src/enrichment.baml`:

```baml
class TemplateContext {
  date string @description("ISO date for the entry")
  day_of_week string
  recent_entries string[] @description("Last 3 entry previews for context")
  custom map<string, string> @description("Additional key-value context")
}

class EnrichedContent {
  content string @description("The expanded template content")
  suggested_tags string[] @description("Relevant tags for categorization")
  mood string? @description("Detected emotional tone: reflective, energetic, anxious, calm, etc.")
  follow_up_prompts string[] @description("Questions to inspire further reflection")
}

function EnrichTemplate(
  template_content: string,
  template_name: string,
  context: TemplateContext
) -> EnrichedContent {
  client Primary
  prompt #"
    You are helping expand a diary template into personalized content.
    
    Template name: {{ template_name }}
    Template:
    {{ template_content }}
    
    Context:
    - Date: {{ context.date }} ({{ context.day_of_week }})
    - Recent entries: {{ context.recent_entries }}
    - Additional: {{ context.custom }}
    
    Expand this template with relevant prompts and context.
    Keep the original structure but add date-aware suggestions and personalized prompts.
  "#
}

function ExtractMetadata(entry_content: string) -> EnrichedContent {
  client Primary
  prompt #"
    Analyze this diary entry and extract metadata.
    
    Entry:
    {{ entry_content }}
    
    Extract:
    - Suggested tags for categorization
    - The overall mood/emotional tone
    - Follow-up questions for deeper reflection
    
    Return the original content unchanged in the content field.
  "#
}
```

**Step 3: Commit BAML files**

```bash
git add baml_src/
git commit -m "feat(baml): add client and enrichment configurations"
```

---

## Task 11: Document BAML Integration (Placeholder)

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Add BAML integration note**

Add to the "Active Technologies" section in CLAUDE.md:

```markdown
- BAML for type-safe LLM calls (context enrichment) - requires `baml-cli generate`
- MCP SDK for tool exposure (`github.com/modelcontextprotocol/go-sdk/mcp`)
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add BAML and MCP SDK to active technologies"
```

---

## Task 12: Run Full Test Suite

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests PASS

**Step 2: Verify build**

Run: `go build .`
Expected: No errors

---

## Summary

This plan implements:

1. **MCP Server** (Tasks 2-6): Exposes `search_entries` and `filter_entries` tools over diary data
2. **MCP Client** (Task 7): Typed wrapper for calling MCP tools
3. **CompositeProvider** (Tasks 8-9): Unified interface implementing `Provider`
4. **BAML Configuration** (Task 10): Schema for LLM enrichment (codegen step deferred)

**Not Implemented** (future work):
- BAML Go client generation (requires `baml-cli` toolchain setup)
- BAML enricher integration into CompositeProvider
- External MCP server connection (remote transports)
- Semantic/embedding-based search

---

Plan complete and saved to `docs/plans/2026-02-02-baml-mcp-context-providers.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
