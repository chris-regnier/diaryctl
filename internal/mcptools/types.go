package mcptools

// SearchInput is the input schema for the search_entries MCP tool.
type SearchInput struct {
	Query string `json:"query" jsonschema-description:"Text to search for in entry content"`
	Limit int    `json:"limit" jsonschema-description:"Maximum number of results to return"`
}

// SearchOutput is the output schema for the search_entries MCP tool.
type SearchOutput struct {
	Entries []EntryResult `json:"entries"`
}

// FilterInput is the input schema for the filter_entries MCP tool.
type FilterInput struct {
	StartDate     string   `json:"start_date,omitempty" jsonschema-description:"ISO date lower bound (inclusive)"`
	EndDate       string   `json:"end_date,omitempty" jsonschema-description:"ISO date upper bound (inclusive)"`
	TemplateNames []string `json:"template_names,omitempty" jsonschema-description:"Filter to entries using these templates"`
	Limit         int      `json:"limit" jsonschema-description:"Maximum number of results"`
}

// FilterOutput is the output schema for the filter_entries MCP tool.
type FilterOutput struct {
	Entries []EntryResult `json:"entries"`
}

// EntryResult is the common output format for entry-related MCP tools.
type EntryResult struct {
	ID      string  `json:"id"`
	Preview string  `json:"preview"`
	Date    string  `json:"date"`
	Score   float64 `json:"score"`
}

// CreateEntryInput is the input schema for the create_entry MCP tool.
type CreateEntryInput struct {
	Content           string            `json:"content" jsonschema-description:"Entry content"`
	TemplateNames     []string          `json:"template_names,omitempty" jsonschema-description:"Template names to compose"`
	TemplateVariables map[string]string `json:"template_variables,omitempty" jsonschema-description:"Variables for template substitution"`
}

// CreateEntryOutput is the output schema for the create_entry MCP tool.
type CreateEntryOutput struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Preview string `json:"preview"`
}

// ListTemplatesInput is the input schema for the list_templates MCP tool.
type ListTemplatesInput struct {
	Limit int `json:"limit" jsonschema-description:"Maximum number of templates to return"`
}

// ListTemplatesOutput is the output schema for the list_templates MCP tool.
type ListTemplatesOutput struct {
	Templates []TemplateResult `json:"templates"`
}

// TemplateResult represents a template in list_templates output.
type TemplateResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Preview string `json:"preview"`
}
