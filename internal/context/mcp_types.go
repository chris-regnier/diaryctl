package context

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
