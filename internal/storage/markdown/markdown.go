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

// localDate returns the local midnight for the given time.
func localDate(t time.Time) time.Time {
	y, m, d := t.Local().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// Store implements storage.Storage using Markdown files with YAML front-matter.
type Store struct {
	baseDir      string // e.g. ~/.diaryctl/entries/
	templatesDir string // e.g. ~/.diaryctl/templates/
}

// New creates a new Markdown file storage backend.
func New(dataDir string) (*Store, error) {
	entriesDir := filepath.Join(dataDir, "entries")
	if err := os.MkdirAll(entriesDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: creating entries directory: %v", storage.ErrStorage, err)
	}
	templatesDir := filepath.Join(dataDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: creating templates directory: %v", storage.ErrStorage, err)
	}
	return &Store{baseDir: entriesDir, templatesDir: templatesDir}, nil
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
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", e.ID)
	fmt.Fprintf(&b, "created_at: %s\n", e.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "updated_at: %s\n", e.UpdatedAt.UTC().Format(time.RFC3339))
	if len(e.Templates) > 0 {
		b.WriteString("templates:\n")
		for _, ref := range e.Templates {
			fmt.Fprintf(&b, "  - template_id: %s\n", ref.TemplateID)
			fmt.Fprintf(&b, "    template_name: %s\n", ref.TemplateName)
		}
	}
	b.WriteString("---\n\n")
	b.WriteString(e.Content)
	return []byte(b.String())
}

type fmTemplateRef struct {
	TemplateID   string `yaml:"template_id"`
	TemplateName string `yaml:"template_name"`
}

type frontMatter struct {
	ID        string          `yaml:"id"`
	CreatedAt string          `yaml:"created_at"`
	UpdatedAt string          `yaml:"updated_at"`
	Templates []fmTemplateRef `yaml:"templates"`
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

	var templates []entry.TemplateRef
	for _, ref := range fm.Templates {
		templates = append(templates, entry.TemplateRef{
			TemplateID:   ref.TemplateID,
			TemplateName: ref.TemplateName,
		})
	}

	return entry.Entry{
		ID:        fm.ID,
		Content:   strings.TrimSpace(string(content)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Templates: templates,
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

		entryDate := localDate(e.CreatedAt)

		// Date filter (takes precedence over range)
		if opts.Date != nil {
			filterDate := localDate(*opts.Date)
			if !entryDate.Equal(filterDate) {
				return nil
			}
		} else {
			// Date range filters
			if opts.StartDate != nil {
				if entryDate.Before(localDate(*opts.StartDate)) {
					return nil
				}
			}
			if opts.EndDate != nil {
				if entryDate.After(localDate(*opts.EndDate)) {
					return nil
				}
			}
		}

		// Template name filter
		if opts.TemplateName != "" {
			found := false
			for _, ref := range e.Templates {
				if ref.TemplateName == opts.TemplateName {
					found = true
					break
				}
			}
			if !found {
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

// ListDays returns aggregated day summaries by scanning the directory tree.
func (s *Store) ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error) {
	type dayData struct {
		date   time.Time
		count  int
		newest entry.Entry
	}
	days := make(map[string]*dayData)

	err := filepath.WalkDir(s.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		e, err := s.unmarshal(data)
		if err != nil {
			return nil
		}

		entryDate := localDate(e.CreatedAt)

		// Apply date range filters
		if opts.StartDate != nil {
			if entryDate.Before(localDate(*opts.StartDate)) {
				return nil
			}
		}
		if opts.EndDate != nil {
			if entryDate.After(localDate(*opts.EndDate)) {
				return nil
			}
		}

		// Template name filter
		if opts.TemplateName != "" {
			found := false
			for _, ref := range e.Templates {
				if ref.TemplateName == opts.TemplateName {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		key := entryDate.Format("2006-01-02")
		dd, exists := days[key]
		if !exists {
			dd = &dayData{date: entryDate}
			days[key] = dd
		}
		dd.count++
		if dd.newest.ID == "" || e.CreatedAt.After(dd.newest.CreatedAt) {
			dd.newest = e
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: scanning entries: %v", storage.ErrStorage, err)
	}

	summaries := make([]storage.DaySummary, 0, len(days))
	for _, dd := range days {
		preview := strings.ReplaceAll(dd.newest.Content, "\n", " ")
		if len(preview) > 80 {
			preview = preview[:80]
		}
		summaries = append(summaries, storage.DaySummary{
			Date:    dd.date,
			Count:   dd.count,
			Preview: preview,
		})
	}

	// Sort reverse chronological
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Date.After(summaries[j].Date)
	})

	return summaries, nil
}

// Update modifies an existing entry's content and optionally its template refs.
// Pass nil for templates to preserve existing refs.
func (s *Store) Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error) {
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
	if templates != nil {
		e.Templates = templates
	}

	if err := s.atomicWrite(path, s.marshal(e)); err != nil {
		return entry.Entry{}, err
	}

	return e, nil
}

// --- Template methods ---

type templateFrontMatter struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	CreatedAt string `yaml:"created_at"`
	UpdatedAt string `yaml:"updated_at"`
}

func (s *Store) marshalTemplate(t storage.Template) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", t.ID)
	fmt.Fprintf(&b, "name: %s\n", t.Name)
	fmt.Fprintf(&b, "created_at: %s\n", t.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "updated_at: %s\n", t.UpdatedAt.UTC().Format(time.RFC3339))
	b.WriteString("---\n\n")
	b.WriteString(t.Content)
	return []byte(b.String())
}

func (s *Store) unmarshalTemplate(data []byte) (storage.Template, error) {
	var fm templateFrontMatter
	content, err := frontmatter.Parse(strings.NewReader(string(data)), &fm)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: parsing template front-matter: %v", storage.ErrStorage, err)
	}
	createdAt, err := time.Parse(time.RFC3339, fm.CreatedAt)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	updatedAt, err := time.Parse(time.RFC3339, fm.UpdatedAt)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}
	return storage.Template{
		ID:        fm.ID,
		Name:      fm.Name,
		Content:   strings.TrimSpace(string(content)),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (s *Store) templatePath(name string) string {
	return filepath.Join(s.templatesDir, name+".md")
}

// CreateTemplate persists a new template as a Markdown file.
func (s *Store) CreateTemplate(t storage.Template) error {
	if err := entry.ValidateTemplateName(t.Name); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}
	path := s.templatePath(t.Name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: template %q already exists", storage.ErrConflict, t.Name)
	}
	return s.atomicWrite(path, s.marshalTemplate(t))
}

// GetTemplate retrieves a template by ID by scanning the templates directory.
func (s *Store) GetTemplate(id string) (storage.Template, error) {
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: reading templates dir: %v", storage.ErrStorage, err)
	}
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.templatesDir, de.Name()))
		if err != nil {
			continue
		}
		tmpl, err := s.unmarshalTemplate(data)
		if err != nil {
			continue
		}
		if tmpl.ID == id {
			return tmpl, nil
		}
	}
	return storage.Template{}, storage.ErrNotFound
}

// GetTemplateByName retrieves a template by name.
func (s *Store) GetTemplateByName(name string) (storage.Template, error) {
	path := s.templatePath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.Template{}, storage.ErrNotFound
		}
		return storage.Template{}, fmt.Errorf("%w: reading template file: %v", storage.ErrStorage, err)
	}
	return s.unmarshalTemplate(data)
}

// ListTemplates returns all templates sorted by name.
func (s *Store) ListTemplates() ([]storage.Template, error) {
	entries, err := os.ReadDir(s.templatesDir)
	if err != nil {
		return nil, fmt.Errorf("%w: reading templates dir: %v", storage.ErrStorage, err)
	}
	var templates []storage.Template
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.templatesDir, de.Name()))
		if err != nil {
			continue
		}
		tmpl, err := s.unmarshalTemplate(data)
		if err != nil {
			continue
		}
		templates = append(templates, tmpl)
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates, nil
}

// UpdateTemplate modifies an existing template's name and content.
func (s *Store) UpdateTemplate(id string, name string, content string) (storage.Template, error) {
	// Find existing template by ID
	existing, err := s.GetTemplate(id)
	if err != nil {
		return storage.Template{}, err
	}

	updated := existing
	updated.Name = name
	updated.Content = content
	updated.UpdatedAt = time.Now().UTC()

	// If name changed, remove old file
	if existing.Name != name {
		if err := entry.ValidateTemplateName(name); err != nil {
			return storage.Template{}, fmt.Errorf("%w: %v", storage.ErrValidation, err)
		}
		// Check new name doesn't conflict
		newPath := s.templatePath(name)
		if _, err := os.Stat(newPath); err == nil {
			return storage.Template{}, fmt.Errorf("%w: template %q already exists", storage.ErrConflict, name)
		}
		os.Remove(s.templatePath(existing.Name))
	}

	if err := s.atomicWrite(s.templatePath(name), s.marshalTemplate(updated)); err != nil {
		return storage.Template{}, err
	}
	return updated, nil
}

// DeleteTemplate removes a template by ID.
func (s *Store) DeleteTemplate(id string) error {
	tmpl, err := s.GetTemplate(id)
	if err != nil {
		return err
	}
	if err := os.Remove(s.templatePath(tmpl.Name)); err != nil {
		return fmt.Errorf("%w: deleting template file: %v", storage.ErrStorage, err)
	}
	return nil
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
