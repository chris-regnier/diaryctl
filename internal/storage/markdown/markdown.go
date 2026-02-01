package markdown

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
)

// Store implements storage.Storage using Markdown files with YAML front-matter.
type Store struct {
	baseDir string // e.g. ~/.diaryctl/entries/
}

// New creates a new Markdown file storage backend.
func New(dataDir string) (*Store, error) {
	entriesDir := filepath.Join(dataDir, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: creating entries directory: %v", storage.ErrStorage, err)
	}
	return &Store{baseDir: entriesDir}, nil
}

// Close is a no-op for the Markdown backend.
func (s *Store) Close() error {
	return nil
}

func (s *Store) entryPath(e entry.Entry) string {
	t := e.CreatedAt
	return filepath.Join(s.baseDir, t.Format("2006"), t.Format("01"), t.Format("02"), e.ID+".md")
}

func (s *Store) marshal(e entry.Entry) []byte {
	fm := fmt.Sprintf("---\nid: %s\ncreated_at: %s\nupdated_at: %s\n---\n\n%s",
		e.ID,
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.UpdatedAt.UTC().Format(time.RFC3339),
		e.Content,
	)
	return []byte(fm)
}

type frontMatter struct {
	ID        string `yaml:"id"`
	CreatedAt string `yaml:"created_at"`
	UpdatedAt string `yaml:"updated_at"`
}

func (s *Store) unmarshal(data []byte) (entry.Entry, error) {
	var fm frontMatter
	content, err := frontmatter.Parse(strings.NewReader(string(data)), &fm)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: parsing front-matter: %v", storage.ErrStorage, err)
	}

	createdAt, err := time.Parse(time.RFC3339, fm.CreatedAt)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	updatedAt, err := time.Parse(time.RFC3339, fm.UpdatedAt)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}

	return entry.Entry{
		ID:        fm.ID,
		Content:   strings.TrimSpace(string(content)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

// atomicWrite writes data to a temp file then renames it to the target path.
func (s *Store) atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("%w: creating directory: %v", storage.ErrStorage, err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("%w: creating temp file: %v", storage.ErrStorage, err)
	}
	tmpName := tmp.Name()

	// Lock the temp file during write
	if err := syscall.Flock(int(tmp.Fd()), syscall.LOCK_EX); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("%w: acquiring lock: %v", storage.ErrStorage, err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("%w: writing temp file: %v", storage.ErrStorage, err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("%w: closing temp file: %v", storage.ErrStorage, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("%w: renaming file: %v", storage.ErrStorage, err)
	}

	return nil
}

// Create persists a new diary entry as a Markdown file.
func (s *Store) Create(e entry.Entry) error {
	if err := entry.ValidateContent(e.Content); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}

	path := s.entryPath(e)

	// Check if file already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: entry %s already exists", storage.ErrConflict, e.ID)
	}

	return s.atomicWrite(path, s.marshal(e))
}

// Get retrieves an entry by ID by scanning the directory tree.
func (s *Store) Get(id string) (entry.Entry, error) {
	path, err := s.findEntryPath(id)
	if err != nil {
		return entry.Entry{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: reading file: %v", storage.ErrStorage, err)
	}

	return s.unmarshal(data)
}

// findEntryPath locates the file for a given entry ID.
func (s *Store) findEntryPath(id string) (string, error) {
	var found string
	err := filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == id+".md" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("%w: scanning entries: %v", storage.ErrStorage, err)
	}
	if found == "" {
		return "", storage.ErrNotFound
	}
	return found, nil
}

// List returns entries matching the given options.
func (s *Store) List(opts storage.ListOptions) ([]entry.Entry, error) {
	var entries []entry.Entry

	err := filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable files
		}

		e, err := s.unmarshal(data)
		if err != nil {
			return nil // skip malformed files
		}

		// Date filter
		if opts.Date != nil {
			entryDate := e.CreatedAt.Local().Truncate(24 * time.Hour)
			filterDate := opts.Date.Truncate(24 * time.Hour)
			if !entryDate.Equal(filterDate) {
				return nil
			}
		}

		entries = append(entries, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: listing entries: %v", storage.ErrStorage, err)
	}

	// Sort by created_at descending (reverse chronological)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	// Apply offset
	if opts.Offset > 0 && opts.Offset < len(entries) {
		entries = entries[opts.Offset:]
	} else if opts.Offset >= len(entries) {
		return []entry.Entry{}, nil
	}

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(entries) {
		entries = entries[:opts.Limit]
	}

	return entries, nil
}

// Update modifies an existing entry's content.
func (s *Store) Update(id string, content string) (entry.Entry, error) {
	if err := entry.ValidateContent(content); err != nil {
		return entry.Entry{}, fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}

	path, err := s.findEntryPath(id)
	if err != nil {
		return entry.Entry{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: reading file: %v", storage.ErrStorage, err)
	}

	e, err := s.unmarshal(data)
	if err != nil {
		return entry.Entry{}, err
	}

	e.Content = content
	e.UpdatedAt = time.Now().UTC()

	if err := s.atomicWrite(path, s.marshal(e)); err != nil {
		return entry.Entry{}, err
	}

	return e, nil
}

// Delete removes an entry permanently.
func (s *Store) Delete(id string) error {
	path, err := s.findEntryPath(id)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("%w: deleting file: %v", storage.ErrStorage, err)
	}

	return nil
}
