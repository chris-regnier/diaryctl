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
	Date      *time.Time // filter by single date (local timezone)
	StartDate *time.Time // inclusive lower bound (nil = no lower bound)
	EndDate   *time.Time // inclusive upper bound (nil = no upper bound)
	OrderBy   string     // "created_at" (default: desc)
	Limit     int        // 0 = no limit
	Offset    int        // pagination offset
}

// DaySummary represents an aggregated view of entries for a single calendar day.
type DaySummary struct {
	Date    time.Time // Calendar date (time part zeroed, local timezone)
	Count   int       // Number of entries on this day
	Preview string    // Content preview of the most recent entry (≤80 chars, single line)
}

// ListDaysOptions controls filtering for ListDays operations.
type ListDaysOptions struct {
	StartDate *time.Time // inclusive lower bound (nil = no lower bound)
	EndDate   *time.Time // inclusive upper bound (nil = no upper bound)
}

// Storage defines the interface for diary entry persistence.
type Storage interface {
	Create(e entry.Entry) error
	Get(id string) (entry.Entry, error)
	List(opts ListOptions) ([]entry.Entry, error)
	ListDays(opts ListDaysOptions) ([]DaySummary, error)
	Update(id string, content string) (entry.Entry, error)
	Delete(id string) error
	Close() error
}
