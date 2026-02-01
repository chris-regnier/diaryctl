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
	Date    *time.Time // filter by date (local timezone)
	OrderBy string     // "created_at" (default: desc)
	Limit   int        // 0 = no limit
	Offset  int        // pagination offset
}

// Storage defines the interface for diary entry persistence.
type Storage interface {
	Create(e entry.Entry) error
	Get(id string) (entry.Entry, error)
	List(opts ListOptions) ([]entry.Entry, error)
	Update(id string, content string) (entry.Entry, error)
	Delete(id string) error
	Close() error
}
