package storage

import (
	"testing"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
)

// mockStorageV2 is a test implementation of StorageV2 that verifies
// the interface compiles correctly with all required methods.
type mockStorageV2 struct{}

// GetDay returns or creates a day for the given date.
func (m *mockStorageV2) GetDay(date time.Time) (day.Day, error) {
	return day.Day{}, nil
}

// ListDays returns a list of day summaries matching the filter criteria.
func (m *mockStorageV2) ListDays(opts ListDaysOptions) ([]DaySummary, error) {
	return nil, nil
}

// DeleteDay deletes a day and all its blocks.
func (m *mockStorageV2) DeleteDay(date time.Time) error {
	return nil
}

// CreateBlock creates a new block for the given date.
func (m *mockStorageV2) CreateBlock(date time.Time, block block.Block) error {
	return nil
}

// GetBlock returns a block by ID along with the date it belongs to.
func (m *mockStorageV2) GetBlock(blockID string) (block.Block, time.Time, error) {
	return block.Block{}, time.Time{}, nil
}

// UpdateBlock updates the content and attributes of a block.
func (m *mockStorageV2) UpdateBlock(blockID string, content string, attributes map[string]string) error {
	return nil
}

// DeleteBlock deletes a block by ID.
func (m *mockStorageV2) DeleteBlock(blockID string) error {
	return nil
}

// ListBlocks returns all blocks for the given date.
func (m *mockStorageV2) ListBlocks(date time.Time) ([]block.Block, error) {
	return nil, nil
}

// SearchBlocks searches for blocks matching the given criteria.
func (m *mockStorageV2) SearchBlocks(opts SearchOptions) ([]BlockResult, error) {
	return nil, nil
}

// CreateTemplate creates a new template.
func (m *mockStorageV2) CreateTemplate(t Template) error {
	return nil
}

// GetTemplate returns a template by ID.
func (m *mockStorageV2) GetTemplate(id string) (Template, error) {
	return Template{}, nil
}

// GetTemplateByName returns a template by name.
func (m *mockStorageV2) GetTemplateByName(name string) (Template, error) {
	return Template{}, nil
}

// ListTemplates returns all templates.
func (m *mockStorageV2) ListTemplates() ([]Template, error) {
	return nil, nil
}

// UpdateTemplate updates a template's name, content, and attributes.
func (m *mockStorageV2) UpdateTemplate(id string, name string, content string, attributes map[string]string) (Template, error) {
	return Template{}, nil
}

// DeleteTemplate deletes a template by ID.
func (m *mockStorageV2) DeleteTemplate(id string) error {
	return nil
}

// Close closes the storage connection.
func (m *mockStorageV2) Close() error {
	return nil
}

// TestStorageV2Interface verifies that mockStorageV2 implements StorageV2.
// This test ensures the interface compiles and all methods are present.
func TestStorageV2Interface(t *testing.T) {
	var _ StorageV2 = (*mockStorageV2)(nil)
}
