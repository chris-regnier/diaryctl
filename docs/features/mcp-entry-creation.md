# MCP Entry Creation

**Status:** Designed

## Overview

Adds write operations to the MCP server, enabling AI assistants to create diary entries with optional template support. Also adds a `list_templates` tool for template discovery.

## Current State

The MCP server exposes two read-only tools: `search_entries` and `filter_entries`. There are no write operations, so AI assistants cannot create entries.

## New MCP Tools

### `create_entry`

Create a diary entry with optional template composition and variable substitution.

**Input:**

```json
{
  "content": "Finished the auth module refactor today...",
  "template_names": ["standup"],
  "template_variables": {
    "yesterday": "Auth module refactor",
    "today": "API endpoint testing",
    "blockers": "None"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `content` | string | yes | Entry content |
| `template_names` | string[] | no | Template names to compose |
| `template_variables` | map | no | Variables for `{{.var}}` substitution |

**Output:**

```json
{
  "id": "abc12345",
  "date": "2026-02-12",
  "preview": "Finished the auth module refactor today..."
}
```

**Behavior:**
1. If `template_names` provided, compose templates via `template.Compose()`
2. If `template_variables` provided, render via `template.Render()`
3. Append `content` after rendered template output
4. Create entry with template refs for attribution
5. Invalidate shell prompt cache

### `list_templates`

Discover available templates and their previews.

**Input:**

```json
{
  "limit": 20
}
```

**Output:**

```json
{
  "templates": [
    {
      "id": "tmpl-001",
      "name": "standup",
      "preview": "## Yesterday\n{{yesterday}}\n\n## Today..."
    },
    {
      "id": "tmpl-002",
      "name": "retrospective",
      "preview": "## What went well\n{{went_well}}..."
    }
  ]
}
```

## Implementation

### Data Structures

```go
type CreateEntryInput struct {
    Content           string            `json:"content"`
    TemplateNames     []string          `json:"template_names,omitempty"`
    TemplateVariables map[string]string `json:"template_variables,omitempty"`
}

type CreateEntryOutput struct {
    ID      string `json:"id"`
    Date    string `json:"date"`
    Preview string `json:"preview"`
}

type ListTemplatesInput struct {
    Limit int `json:"limit"`
}

type ListTemplatesOutput struct {
    Templates []TemplateResult `json:"templates"`
}

type TemplateResult struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Preview string `json:"preview"`
}
```

### Tool Registration

```go
func CreateMCPServer(store storage.Storage) *mcp.Server {
    // ... existing tools ...

    // Add create_entry tool
    srv.HandleTool("create_entry", createEntryHandler(store))

    // Add list_templates tool
    srv.HandleTool("list_templates", listTemplatesHandler(store))
}
```

### Create Entry Flow

```go
func createEntryHandler(store storage.Storage) mcp.ToolHandlerFunc {
    return func(ctx context.Context, input CreateEntryInput) (*CreateEntryOutput, error) {
        var content string
        var refs []template.TemplateRef

        // 1. Compose templates if specified
        if len(input.TemplateNames) > 0 {
            composed, templateRefs, err := template.Compose(store, input.TemplateNames)
            if err != nil {
                return nil, err
            }
            refs = templateRefs

            // 2. Render variables if specified
            if len(input.TemplateVariables) > 0 {
                composed, err = template.Render(composed, input.TemplateVariables)
                if err != nil {
                    return nil, err
                }
            }

            content = composed + "\n\n" + input.Content
        } else {
            content = input.Content
        }

        // 3. Create entry
        e := entry.New(content, refs)
        if err := store.Create(e); err != nil {
            return nil, err
        }

        return &CreateEntryOutput{
            ID:      e.ID,
            Date:    e.CreatedAt.Format("2006-01-02"),
            Preview: truncate(e.Content, 200),
        }, nil
    }
}
```

## AI Assistant Workflow

A typical AI assistant interaction:

```
User: "Log my standup for today"

AI calls list_templates → sees "standup" template with prompts
AI asks user about yesterday, today, blockers
AI calls create_entry with:
  content: ""
  template_names: ["standup"]
  template_variables: {
    "yesterday": "Finished auth refactor",
    "today": "API testing",
    "blockers": "Waiting on design review"
  }

AI responds: "Created your standup entry for 2026-02-12"
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Unknown template name | Error with available template names |
| Empty content + no templates | Error: content required |
| Template render failure | Error with variable details |
| Storage write failure | Error with storage details |

## Related Features

- [Guided Capture](guided-capture.md) — CLI equivalent with interactive prompts
- [Full-Text Search](search.md) — Search entries created via MCP

---

*Feature designed February 2026.*
