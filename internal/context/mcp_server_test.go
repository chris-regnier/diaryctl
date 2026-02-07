package context_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
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
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	// Call search_entries tool
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_entries",
		Arguments: icontext.SearchInput{Query: "Go interfaces", Limit: 10},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	// Verify result contains matching entry
	var output icontext.SearchOutput
	if result.StructuredContent != nil {
		// Use structured content if available
		outputJSON, _ := json.Marshal(result.StructuredContent)
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			t.Fatalf("failed to unmarshal structured content: %v", err)
		}
	} else if len(result.Content) > 0 {
		// Fall back to text content
		contentJSON, _ := json.Marshal(result.Content[0])
		var textContent struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(contentJSON, &textContent); err != nil {
			t.Fatalf("failed to unmarshal content: %v", err)
		}
		if err := json.Unmarshal([]byte(textContent.Text), &output); err != nil {
			t.Fatalf("failed to unmarshal output: %v", err)
		}
	} else {
		t.Fatal("expected content in result")
	}

	if len(output.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(output.Entries))
	}
	if len(output.Entries) > 0 && output.Entries[0].ID != "testid01" {
		t.Errorf("expected entry testid01, got %s", output.Entries[0].ID)
	}
}

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
	session, _ := client.Connect(context.Background(), clientTransport, nil)

	// Filter for today only
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "filter_entries",
		Arguments: icontext.FilterInput{
			StartDate: now.Format("2006-01-02"),
			EndDate:   now.Format("2006-01-02"),
			Limit:     10,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	var output icontext.FilterOutput
	if result.StructuredContent != nil {
		outputJSON, _ := json.Marshal(result.StructuredContent)
		_ = json.Unmarshal(outputJSON, &output)
	}

	if len(output.Entries) != 1 {
		t.Errorf("expected 1 entry for today, got %d", len(output.Entries))
	}
}
