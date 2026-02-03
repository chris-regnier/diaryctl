package storage

import (
	"errors"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
)

// Sentinel errors for storage operations.
var (
	ErrNotFound   = errors.New("entry not found")
	ErrConflict   = errors.New("concurrent write conflict")
	ErrStorage    = errors.New("storage error")
	ErrValidation = errors.New("validation error")
)

// ListOptions controls filtering and ordering for List operations.
type ListOptions struct {
	Date         *time.Time // filter by single date (local timezone)
	StartDate    *time.Time // inclusive lower bound (nil = no lower bound)
	EndDate      *time.Time // inclusive upper bound (nil = no upper bound)
	TemplateName string     // filter entries by template attribution
	ContextName  string     // filter entries by context name
	OrderBy      string     // "created_at" (default: desc)
	Limit        int        // 0 = no limit
	Offset       int        // pagination offset
}

// DaySummary represents an aggregated view of entries for a single calendar day.
type DaySummary struct {
	Date    time.Time // Calendar date (time part zeroed, local timezone)
	Count   int       // Number of entries on this day
	Preview string    // Content preview of the most recent entry (≤80 chars, single line)
}

// ListDaysOptions controls filtering for ListDays operations.
type ListDaysOptions struct {
	StartDate    *time.Time // inclusive lower bound (nil = no lower bound)
	EndDate      *time.Time // inclusive upper bound (nil = no upper bound)
	TemplateName string     // filter days to those with entries matching this template
}

// Template represents a reusable content template.
type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Context represents a semantic grouping for diary entries.
type Context struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"` // "manual", "git", etc.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Storage defines the interface for diary entry persistence.
type Storage interface {
	// Entry methods
	Create(e entry.Entry) error
	Get(id string) (entry.Entry, error)
	List(opts ListOptions) ([]entry.Entry, error)
	ListDays(opts ListDaysOptions) ([]DaySummary, error)
	Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error)
	Delete(id string) error
	Close() error

	// Template methods
	CreateTemplate(t Template) error
	GetTemplate(id string) (Template, error)
	GetTemplateByName(name string) (Template, error)
	ListTemplates() ([]Template, error)
	UpdateTemplate(id string, name string, content string) (Template, error)
	DeleteTemplate(id string) error

	// Context methods
	CreateContext(c Context) error
	GetContext(id string) (Context, error)
	GetContextByName(name string) (Context, error)
	ListContexts() ([]Context, error)
	DeleteContext(id string) error
	AttachContext(entryID string, contextID string) error
	DetachContext(entryID string, contextID string) error
}
