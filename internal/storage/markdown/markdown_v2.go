package markdown

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/block"
	"github.com/chris-regnier/diaryctl/internal/day"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// MarkdownV2 implements storage.StorageV2 interface using JSON files.
// Days are stored as JSON files in a days/ subdirectory.
// Templates are stored as JSON files in a templates/ subdirectory.
type MarkdownV2 struct {
	basePath string
}

// Compile-time check that MarkdownV2 implements storage.StorageV2
var _ storage.StorageV2 = (*MarkdownV2)(nil)

// templateIDPattern defines the valid format for template IDs (e.g., tmpl0001).
// This prevents directory traversal attacks and ensures consistent naming.
var templateIDPattern = regexp.MustCompile(`^tmpl\d{4}$`)

// NewV2 creates a new MarkdownV2 storage instance.
// It ensures the necessary directories exist (days/, templates/).
//
// Parameters:
//   - basePath: The base directory where storage files will be created
//
// Returns an error if directory creation fails.
func NewV2(basePath string) (*MarkdownV2, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create days subdirectory
	daysPath := filepath.Join(basePath, "days")
	if err := os.MkdirAll(daysPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create days directory: %w", err)
	}

	// Create templates subdirectory
	templatesPath := filepath.Join(basePath, "templates")
	if err := os.MkdirAll(templatesPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create templates directory: %w", err)
	}

	return &MarkdownV2{
		basePath: basePath,
	}, nil
}

// getDayPath returns the file path for a day's JSON file.
// Format: ~/.diaryctl/days/2026-02-09.json
func (m *MarkdownV2) getDayPath(date time.Time) string {
	// Normalize date to ensure consistent file naming
	normalizedDate := day.NormalizeDate(date)
	filename := normalizedDate.Format("2006-01-02") + ".json"
	return filepath.Join(m.basePath, "days", filename)
}

// loadDay reads a day from disk. Returns an empty day if the file doesn't exist.
func (m *MarkdownV2) loadDay(date time.Time) (day.Day, error) {
	normalizedDate := day.NormalizeDate(date)
	path := m.getDayPath(normalizedDate)

	// If file doesn't exist, return an empty day
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return day.Day{
			Date:   normalizedDate,
			Blocks: []block.Block{},
		}, nil
	}

	// Read and parse the JSON file
	data, err := os.ReadFile(path)
	if err != nil {
		return day.Day{}, fmt.Errorf("failed to read day file: %w", err)
	}

	var d day.Day
	if err := json.Unmarshal(data, &d); err != nil {
		return day.Day{}, fmt.Errorf("failed to parse day JSON: %w", err)
	}

	return d, nil
}

// saveDay writes a day to disk as JSON.
func (m *MarkdownV2) saveDay(d day.Day) error {
	// Ensure days directory exists
	daysPath := filepath.Join(m.basePath, "days")
	if err := os.MkdirAll(daysPath, 0755); err != nil {
		return fmt.Errorf("failed to create days directory: %w", err)
	}

	path := m.getDayPath(d.Date)

	// Marshal day to JSON with indentation for readability
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal day to JSON: %w", err)
	}

	// Write to file with atomic write (write to temp file, then rename)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write day file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure and wrap both errors
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			return fmt.Errorf("failed to rename day file: %w (cleanup error: %v)", err, removeErr)
		}
		return fmt.Errorf("failed to rename day file: %w", err)
	}

	return nil
}

// GetDay returns the day for the given date, creating an empty day if it doesn't exist.
// The date parameter will be normalized to midnight local time.
func (m *MarkdownV2) GetDay(date time.Time) (day.Day, error) {
	return m.loadDay(date)
}

// ListDays returns a list of day summaries matching the filter criteria.
// Days are returned in descending order by date (most recent first).
// Stub for now - to be implemented in a later task.
func (m *MarkdownV2) ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error) {
	// Stub for now
	return []storage.DaySummary{}, nil
}

// DeleteDay deletes a day and all its blocks.
// Returns ErrNotFound if the day doesn't exist.
func (m *MarkdownV2) DeleteDay(date time.Time) error {
	path := m.getDayPath(date)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return storage.ErrNotFound
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete day file: %w", err)
	}

	return nil
}

// CreateBlock creates a new block for the given date.
// The date will be normalized to midnight local time.
// The block's ID, CreatedAt, and UpdatedAt timestamps MUST be set by the caller.
func (m *MarkdownV2) CreateBlock(date time.Time, blk block.Block) error {
	// Validate block
	if err := block.ValidateID(blk.ID); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}
	if err := block.ValidateContent(blk.Content); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}
	if blk.CreatedAt.IsZero() {
		return fmt.Errorf("%w: CreatedAt must be set", storage.ErrValidation)
	}
	if blk.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: UpdatedAt must be set", storage.ErrValidation)
	}

	// Load the day
	d, err := m.loadDay(date)
	if err != nil {
		return err
	}

	// Check for duplicate block ID
	if d.FindBlock(blk.ID) != -1 {
		return fmt.Errorf("%w: block with ID %s already exists", storage.ErrValidation, blk.ID)
	}

	// Add block to day
	d.AddBlock(blk)

	// Save the day
	return m.saveDay(d)
}

// GetBlock returns a block by ID along with the date it belongs to.
// Returns ErrNotFound if the block doesn't exist.
// Returns ErrValidation if blockID is empty.
func (m *MarkdownV2) GetBlock(blockID string) (block.Block, time.Time, error) {
	// Validate block ID
	if blockID == "" {
		return block.Block{}, time.Time{}, fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}

	// Search through all day files to find the block
	daysPath := filepath.Join(m.basePath, "days")
	entries, err := os.ReadDir(daysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return block.Block{}, time.Time{}, storage.ErrNotFound
		}
		return block.Block{}, time.Time{}, fmt.Errorf("failed to read days directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Parse the date from filename
		dateStr := entry.Name()[:len(entry.Name())-5] // Remove .json
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			continue // Skip invalid filenames
		}

		// Load the day
		d, err := m.loadDay(date)
		if err != nil {
			continue // Skip files that can't be loaded
		}

		// Search for the block
		idx := d.FindBlock(blockID)
		if idx != -1 {
			return d.Blocks[idx], d.Date, nil
		}
	}

	return block.Block{}, time.Time{}, storage.ErrNotFound
}

// UpdateBlock updates the content and attributes of a block.
// The block's UpdatedAt timestamp will be updated automatically.
// Returns ErrNotFound if the block doesn't exist.
// Returns ErrValidation if blockID is empty or content is invalid.
func (m *MarkdownV2) UpdateBlock(blockID string, content string, attributes map[string]string) error {
	// Validate inputs
	if blockID == "" {
		return fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}
	if err := block.ValidateContent(content); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}

	// Find the block
	blk, date, err := m.GetBlock(blockID)
	if err != nil {
		return err
	}

	// Load the day
	d, err := m.loadDay(date)
	if err != nil {
		return err
	}

	// Find and update the block
	idx := d.FindBlock(blockID)
	if idx == -1 {
		return storage.ErrNotFound
	}

	blk.Content = content
	blk.Attributes = attributes
	blk.UpdatedAt = time.Now()
	d.Blocks[idx] = blk
	d.UpdatedAt = time.Now()

	// Save the day
	return m.saveDay(d)
}

// DeleteBlock deletes a block by ID.
// Returns ErrNotFound if the block doesn't exist.
// Returns ErrValidation if blockID is empty.
func (m *MarkdownV2) DeleteBlock(blockID string) error {
	// Validate block ID
	if blockID == "" {
		return fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}

	// Find the block
	_, date, err := m.GetBlock(blockID)
	if err != nil {
		return err
	}

	// Load the day
	d, err := m.loadDay(date)
	if err != nil {
		return err
	}

	// Remove the block
	if !d.RemoveBlock(blockID) {
		return storage.ErrNotFound
	}

	// Save the day
	return m.saveDay(d)
}

// ListBlocks returns all blocks for the given date, ordered by CreatedAt ascending.
// Returns an empty slice if no blocks exist for the date.
func (m *MarkdownV2) ListBlocks(date time.Time) ([]block.Block, error) {
	d, err := m.loadDay(date)
	if err != nil {
		return nil, err
	}

	return d.Blocks, nil
}

// SearchBlocks searches for blocks matching the given criteria.
// Results are ordered by date descending, then by CreatedAt descending.
// Stub for now - to be implemented in a later task.
func (m *MarkdownV2) SearchBlocks(opts storage.SearchOptions) ([]storage.BlockResult, error) {
	// Stub for now
	return []storage.BlockResult{}, nil
}

// Template methods

// getTemplatePath returns the file path for a template's JSON file.
// Format: ~/.diaryctl/templates/tmpl0001.json
func (m *MarkdownV2) getTemplatePath(id string) string {
	filename := id + ".json"
	return filepath.Join(m.basePath, "templates", filename)
}

// loadTemplate reads a template from disk.
// Returns ErrNotFound if the file doesn't exist.
func (m *MarkdownV2) loadTemplate(id string) (storage.Template, error) {
	path := m.getTemplatePath(id)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return storage.Template{}, storage.ErrNotFound
	}

	// Read and parse the JSON file
	data, err := os.ReadFile(path)
	if err != nil {
		return storage.Template{}, fmt.Errorf("failed to read template file: %w", err)
	}

	var tmpl storage.Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return storage.Template{}, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return tmpl, nil
}

// saveTemplate writes a template to disk as JSON.
func (m *MarkdownV2) saveTemplate(t storage.Template) error {
	// Ensure templates directory exists
	templatesPath := filepath.Join(m.basePath, "templates")
	if err := os.MkdirAll(templatesPath, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	path := m.getTemplatePath(t.ID)

	// Marshal template to JSON with indentation for readability
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template to JSON: %w", err)
	}

	// Write to file with atomic write (write to temp file, then rename)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure and wrap both errors
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			return fmt.Errorf("failed to rename template file: %w (cleanup error: %v)", err, removeErr)
		}
		return fmt.Errorf("failed to rename template file: %w", err)
	}

	return nil
}

// CreateTemplate creates a new template with attributes.
// The template's ID, CreatedAt, and UpdatedAt timestamps MUST be set by the caller.
// Implementations SHOULD validate these fields and return ErrValidation if they are missing or invalid.
func (m *MarkdownV2) CreateTemplate(t storage.Template) error {
	// Validate template
	if t.ID == "" {
		return fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}
	// Validate template ID format to prevent directory traversal
	if !templateIDPattern.MatchString(t.ID) {
		return fmt.Errorf("%w: ID must match format 'tmpl####' (e.g., tmpl0001)", storage.ErrValidation)
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("%w: Name cannot be empty or whitespace", storage.ErrValidation)
	}
	if t.CreatedAt.IsZero() {
		return fmt.Errorf("%w: CreatedAt must be set", storage.ErrValidation)
	}
	if t.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: UpdatedAt must be set", storage.ErrValidation)
	}

	// Check if template already exists
	if _, err := m.loadTemplate(t.ID); err == nil {
		return fmt.Errorf("%w: template with ID %s already exists", storage.ErrValidation, t.ID)
	}

	// Save the template
	return m.saveTemplate(t)
}

// GetTemplate returns a template by ID.
// Returns ErrNotFound if the template doesn't exist.
// Returns ErrValidation if id is empty.
func (m *MarkdownV2) GetTemplate(id string) (storage.Template, error) {
	// Validate ID
	if id == "" {
		return storage.Template{}, fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}

	return m.loadTemplate(id)
}

// GetTemplateByName returns a template by name.
// Returns ErrNotFound if the template doesn't exist.
// Returns ErrValidation if name is empty.
func (m *MarkdownV2) GetTemplateByName(name string) (storage.Template, error) {
	// Validate name
	if name == "" {
		return storage.Template{}, fmt.Errorf("%w: name cannot be empty", storage.ErrValidation)
	}

	// List all templates and search for matching name
	templates, err := m.ListTemplates()
	if err != nil {
		return storage.Template{}, err
	}

	for _, tmpl := range templates {
		if tmpl.Name == name {
			return tmpl, nil
		}
	}

	return storage.Template{}, storage.ErrNotFound
}

// ListTemplates returns all templates ordered by name ascending.
func (m *MarkdownV2) ListTemplates() ([]storage.Template, error) {
	templatesPath := filepath.Join(m.basePath, "templates")

	// Read directory entries
	entries, err := os.ReadDir(templatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []storage.Template{}, nil
		}
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	// Load all template files
	var templates []storage.Template
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Extract template ID from filename (remove .json extension)
		id := entry.Name()[:len(entry.Name())-5]

		// Load the template
		tmpl, err := m.loadTemplate(id)
		if err != nil {
			continue // Skip files that can't be loaded
		}

		templates = append(templates, tmpl)
	}

	// Sort templates by name ascending
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// UpdateTemplate updates a template's name, content, and attributes.
// The template's UpdatedAt timestamp will be updated automatically.
// Returns the updated template or ErrNotFound if the template doesn't exist.
// Returns ErrValidation if id or name is empty.
func (m *MarkdownV2) UpdateTemplate(id string, name string, content string, attributes map[string]string) (storage.Template, error) {
	// Validate inputs
	if id == "" {
		return storage.Template{}, fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}
	if strings.TrimSpace(name) == "" {
		return storage.Template{}, fmt.Errorf("%w: name cannot be empty or whitespace", storage.ErrValidation)
	}

	// Load existing template
	tmpl, err := m.loadTemplate(id)
	if err != nil {
		return storage.Template{}, err
	}

	// Update fields
	tmpl.Name = name
	tmpl.Content = content
	tmpl.Attributes = attributes
	tmpl.UpdatedAt = time.Now()

	// Save the updated template
	if err := m.saveTemplate(tmpl); err != nil {
		return storage.Template{}, err
	}

	return tmpl, nil
}

// DeleteTemplate deletes a template by ID.
// Returns ErrNotFound if the template doesn't exist.
// Returns ErrValidation if id is empty.
func (m *MarkdownV2) DeleteTemplate(id string) error {
	// Validate ID
	if id == "" {
		return fmt.Errorf("%w: ID cannot be empty", storage.ErrValidation)
	}

	path := m.getTemplatePath(id)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return storage.ErrNotFound
	}

	// Delete the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	return nil
}

// Close closes the storage connection and releases any resources.
// For file-based storage, this is a no-op.
func (m *MarkdownV2) Close() error {
	return nil
}
