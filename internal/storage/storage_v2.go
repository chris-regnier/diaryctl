package storage

import (
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
)

// StorageV2 defines the interface for day/block-based diary persistence.
// This is the V2 storage interface that operates on days and blocks instead
// of entries, providing a more structured approach to diary management.
type StorageV2 interface {
	// Day methods

	// GetDay returns the day for the given date, creating an empty day if it doesn't exist.
	// The date parameter will be normalized to midnight local time.
	GetDay(date time.Time) (day.Day, error)

	// ListDays returns a list of day summaries matching the filter criteria.
	// Days are returned in descending order by date (most recent first).
	ListDays(opts ListDaysOptions) ([]DaySummary, error)

	// DeleteDay deletes a day and all its blocks.
	// Returns ErrNotFound if the day doesn't exist.
	DeleteDay(date time.Time) error

	// Block methods

	// CreateBlock creates a new block for the given date.
	// The date will be normalized to midnight local time.
	// The block's ID, CreatedAt, and UpdatedAt timestamps should be set by the caller.
	CreateBlock(date time.Time, block block.Block) error

	// GetBlock returns a block by ID along with the date it belongs to.
	// Returns ErrNotFound if the block doesn't exist.
	GetBlock(blockID string) (block.Block, time.Time, error)

	// UpdateBlock updates the content and attributes of a block.
	// The block's UpdatedAt timestamp will be updated automatically.
	// Returns ErrNotFound if the block doesn't exist.
	UpdateBlock(blockID string, content string, attributes map[string]string) error

	// DeleteBlock deletes a block by ID.
	// Returns ErrNotFound if the block doesn't exist.
	DeleteBlock(blockID string) error

	// ListBlocks returns all blocks for the given date, ordered by CreatedAt ascending.
	// Returns an empty slice if no blocks exist for the date.
	ListBlocks(date time.Time) ([]block.Block, error)

	// SearchBlocks searches for blocks matching the given criteria.
	// Results are ordered by date descending, then by CreatedAt descending.
	SearchBlocks(opts SearchOptions) ([]BlockResult, error)

	// Template methods

	// CreateTemplate creates a new template with attributes.
	// The template's ID, CreatedAt, and UpdatedAt timestamps should be set by the caller.
	CreateTemplate(t Template) error

	// GetTemplate returns a template by ID.
	// Returns ErrNotFound if the template doesn't exist.
	GetTemplate(id string) (Template, error)

	// GetTemplateByName returns a template by name.
	// Returns ErrNotFound if the template doesn't exist.
	GetTemplateByName(name string) (Template, error)

	// ListTemplates returns all templates ordered by name ascending.
	ListTemplates() ([]Template, error)

	// UpdateTemplate updates a template's name, content, and attributes.
	// The template's UpdatedAt timestamp will be updated automatically.
	// Returns the updated template or ErrNotFound if the template doesn't exist.
	UpdateTemplate(id string, name string, content string, attributes map[string]string) (Template, error)

	// DeleteTemplate deletes a template by ID.
	// Returns ErrNotFound if the template doesn't exist.
	DeleteTemplate(id string) error

	// Lifecycle methods

	// Close closes the storage connection and releases any resources.
	Close() error
}

// SearchOptions defines the criteria for searching blocks.
// All criteria are ANDed together (a block must match ALL specified criteria).
type SearchOptions struct {
	// StartDate is the inclusive lower bound for block dates (nil = no lower bound)
	StartDate *time.Time

	// EndDate is the inclusive upper bound for block dates (nil = no upper bound)
	EndDate *time.Time

	// Attributes filters blocks by key-value pairs (AND logic).
	// A block must have ALL specified attributes with matching values.
	// Example: {"type": "note", "mood": "happy"} matches only blocks with both attributes.
	Attributes map[string]string

	// ContentQuery performs full-text search on block content.
	// The exact search behavior (case-sensitivity, partial matches) is implementation-dependent.
	ContentQuery string

	// Limit restricts the maximum number of results (0 = no limit)
	Limit int

	// Offset skips the first N results for pagination
	Offset int
}

// BlockResult represents a single search result containing a block and its day.
type BlockResult struct {
	// Block is the matching block
	Block block.Block

	// Day is the normalized date (midnight local time) that this block belongs to
	Day time.Time
}
