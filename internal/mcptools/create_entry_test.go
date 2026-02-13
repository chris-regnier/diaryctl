package mcptools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/mcptools"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServer_CreateEntry(t *testing.T) {
	dir := t.TempDir()
	store, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	_, clientTransport := mcptools.NewDiaryMCPServer(store)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	t.Run("creates entry with content only", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "create_entry",
			Arguments: mcptools.CreateEntryInput{Content: "Test entry content"},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		var output mcptools.CreateEntryOutput
		outputJSON, _ := json.Marshal(result.StructuredContent)
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			t.Fatalf("failed to unmarshal output: %v", err)
		}

		if output.ID == "" {
			t.Error("expected non-empty ID")
		}
		if output.Date != time.Now().UTC().Format("2006-01-02") {
			t.Errorf("date = %q, want today", output.Date)
		}
		if output.Preview == "" {
			t.Error("expected non-empty preview")
		}

		// Verify entry was persisted
		e, err := store.Get(output.ID)
		if err != nil {
			t.Fatalf("entry not found in storage: %v", err)
		}
		if e.Content != "Test entry content" {
			t.Errorf("stored content = %q, want %q", e.Content, "Test entry content")
		}
	})

	t.Run("creates entry with template", func(t *testing.T) {
		tmpl := storage.Template{
			ID:        "tmpl0001",
			Name:      "standup",
			Content:   "## Yesterday\n{{.yesterday}}\n\n## Today\n{{.today}}",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateTemplate(tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}

		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "create_entry",
			Arguments: mcptools.CreateEntryInput{
				Content:       "Extra notes here",
				TemplateNames: []string{"standup"},
				TemplateVariables: map[string]string{
					"yesterday": "Did stuff",
					"today":     "More stuff",
				},
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		var output mcptools.CreateEntryOutput
		outputJSON, _ := json.Marshal(result.StructuredContent)
		_ = json.Unmarshal(outputJSON, &output)

		e, err := store.Get(output.ID)
		if err != nil {
			t.Fatalf("entry not found: %v", err)
		}
		if !strings.Contains(e.Content, "Did stuff") {
			t.Error("expected rendered template variable 'yesterday' in content")
		}
		if !strings.Contains(e.Content, "Extra notes here") {
			t.Error("expected appended content")
		}
		if len(e.Templates) != 1 || e.Templates[0].TemplateName != "standup" {
			t.Errorf("expected standup template ref, got %v", e.Templates)
		}
	})

	t.Run("rejects empty content without templates", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "create_entry",
			Arguments: mcptools.CreateEntryInput{Content: ""},
		})
		if err != nil {
			t.Fatalf("CallTool returned unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected IsError for empty content")
		}
	})

	t.Run("rejects unknown template", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "create_entry",
			Arguments: mcptools.CreateEntryInput{
				Content:       "content",
				TemplateNames: []string{"nonexistent"},
			},
		})
		if err != nil {
			t.Fatalf("CallTool returned unexpected error: %v", err)
		}
		if !result.IsError {
			t.Error("expected IsError for unknown template")
		}
	})
}
