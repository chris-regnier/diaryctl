package mcptools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/mcptools"
	"github.com/chris-regnier/diaryctl/internal/storage"
	"github.com/chris-regnier/diaryctl/internal/storage/markdown"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServer_ListTemplates(t *testing.T) {
	dir := t.TempDir()
	store, err := markdown.New(dir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create test templates
	for _, name := range []string{"alpha", "beta", "gamma"} {
		tmpl := storage.Template{
			ID:        "tmpl-" + name,
			Name:      name,
			Content:   "Content for " + name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateTemplate(tmpl); err != nil {
			t.Fatalf("failed to create template %s: %v", name, err)
		}
	}

	_, clientTransport := mcptools.NewDiaryMCPServer(store)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	t.Run("lists all templates", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "list_templates",
			Arguments: mcptools.ListTemplatesInput{Limit: 20},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		var output mcptools.ListTemplatesOutput
		outputJSON, _ := json.Marshal(result.StructuredContent)
		if err := json.Unmarshal(outputJSON, &output); err != nil {
			t.Fatalf("failed to unmarshal output: %v", err)
		}

		if len(output.Templates) != 3 {
			t.Errorf("expected 3 templates, got %d", len(output.Templates))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "list_templates",
			Arguments: mcptools.ListTemplatesInput{Limit: 1},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		var output mcptools.ListTemplatesOutput
		outputJSON, _ := json.Marshal(result.StructuredContent)
		_ = json.Unmarshal(outputJSON, &output)

		if len(output.Templates) != 1 {
			t.Errorf("expected 1 template with limit=1, got %d", len(output.Templates))
		}
	})

	t.Run("defaults limit when zero", func(t *testing.T) {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "list_templates",
			Arguments: mcptools.ListTemplatesInput{Limit: 0},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}

		var output mcptools.ListTemplatesOutput
		outputJSON, _ := json.Marshal(result.StructuredContent)
		_ = json.Unmarshal(outputJSON, &output)

		if len(output.Templates) != 3 {
			t.Errorf("expected all 3 templates with default limit, got %d", len(output.Templates))
		}
	})
}
