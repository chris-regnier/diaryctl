package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chris-regnier/diaryctl/internal/entry"
	"github.com/chris-regnier/diaryctl/internal/storage"
	_ "github.com/tursodatabase/go-libsql"
)

// Store implements storage.Storage using SQLite via Turso/libSQL.
type Store struct {
	db *sql.DB
}

// New creates a new SQLite storage backend.
func New(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("%w: creating data directory: %v", storage.ErrStorage, err)
	}

	dbPath := filepath.Join(dataDir, "diaryctl.db")
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return nil, fmt.Errorf("%w: opening database: %v", storage.ErrStorage, err)
	}

	// Enable WAL mode (use QueryRow since PRAGMA returns a result row)
	var walMode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&walMode); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: enabling WAL mode: %v", storage.ErrStorage, err)
	}

	// Create schema
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func createSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id         TEXT PRIMARY KEY,
			content    TEXT NOT NULL CHECK(length(trim(content)) > 0),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			CHECK(created_at <= updated_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_entries_date ON entries(date(created_at))`,
		`CREATE TABLE IF NOT EXISTS templates (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL UNIQUE,
			content    TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS entry_templates (
			entry_id      TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			template_id   TEXT NOT NULL,
			template_name TEXT NOT NULL,
			PRIMARY KEY (entry_id, template_id)
		)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("%w: creating schema: %v", storage.ErrStorage, err)
		}
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Create persists a new diary entry.
func (s *Store) Create(e entry.Entry) error {
	if err := entry.ValidateContent(e.Content); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%w: beginning transaction: %v", storage.ErrStorage, err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO entries (id, content, created_at, updated_at) VALUES (?, ?, ?, ?)",
		e.ID,
		e.Content,
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: inserting entry: %v", storage.ErrStorage, err)
	}

	for _, ref := range e.Templates {
		_, err = tx.Exec(
			"INSERT INTO entry_templates (entry_id, template_id, template_name) VALUES (?, ?, ?)",
			e.ID, ref.TemplateID, ref.TemplateName,
		)
		if err != nil {
			return fmt.Errorf("%w: inserting template ref: %v", storage.ErrStorage, err)
		}
	}

	return tx.Commit()
}

// Get retrieves an entry by ID.
func (s *Store) Get(id string) (entry.Entry, error) {
	row := s.db.QueryRow(
		"SELECT id, content, created_at, updated_at FROM entries WHERE id = ?", id,
	)

	var e entry.Entry
	var createdStr, updatedStr string
	if err := row.Scan(&e.ID, &e.Content, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return entry.Entry{}, storage.ErrNotFound
		}
		return entry.Entry{}, fmt.Errorf("%w: querying entry: %v", storage.ErrStorage, err)
	}

	var err error
	e.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	e.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}

	// Load template refs
	refs, err := s.loadTemplateRefs(id)
	if err != nil {
		return entry.Entry{}, err
	}
	e.Templates = refs

	return e, nil
}

// loadTemplateRefs loads template references for an entry.
func (s *Store) loadTemplateRefs(entryID string) ([]entry.TemplateRef, error) {
	rows, err := s.db.Query(
		"SELECT template_id, template_name FROM entry_templates WHERE entry_id = ?", entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: querying template refs: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var refs []entry.TemplateRef
	for rows.Next() {
		var ref entry.TemplateRef
		if err := rows.Scan(&ref.TemplateID, &ref.TemplateName); err != nil {
			return nil, fmt.Errorf("%w: scanning template ref: %v", storage.ErrStorage, err)
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// List returns entries matching the given options.
func (s *Store) List(opts storage.ListOptions) ([]entry.Entry, error) {
	query := "SELECT DISTINCT entries.id, entries.content, entries.created_at, entries.updated_at FROM entries"
	var args []any
	var conditions []string

	if opts.TemplateName != "" {
		query += " JOIN entry_templates et ON et.entry_id = entries.id"
		conditions = append(conditions, "et.template_name = ?")
		args = append(args, opts.TemplateName)
	}

	if opts.Date != nil {
		// Date takes precedence over range
		conditions = append(conditions, "date(entries.created_at, 'localtime') = ?")
		args = append(args, opts.Date.Format("2006-01-02"))
	} else {
		if opts.StartDate != nil {
			conditions = append(conditions, "date(entries.created_at, 'localtime') >= ?")
			args = append(args, opts.StartDate.Format("2006-01-02"))
		}
		if opts.EndDate != nil {
			conditions = append(conditions, "date(entries.created_at, 'localtime') <= ?")
			args = append(args, opts.EndDate.Format("2006-01-02"))
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY entries.created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing entries: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var entries []entry.Entry
	for rows.Next() {
		var e entry.Entry
		var createdStr, updatedStr string
		if err := rows.Scan(&e.ID, &e.Content, &createdStr, &updatedStr); err != nil {
			return nil, fmt.Errorf("%w: scanning row: %v", storage.ErrStorage, err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		e.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

		// Load template refs
		refs, err := s.loadTemplateRefs(e.ID)
		if err != nil {
			return nil, err
		}
		e.Templates = refs

		entries = append(entries, e)
	}

	if entries == nil {
		entries = []entry.Entry{}
	}

	return entries, rows.Err()
}

// ListDays returns aggregated day summaries grouped by date.
func (s *Store) ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error) {
	query := `SELECT date(created_at, 'localtime') as day, COUNT(*) as cnt,
		(SELECT content FROM entries e2 WHERE date(e2.created_at, 'localtime') = date(entries.created_at, 'localtime') ORDER BY e2.created_at DESC LIMIT 1) as preview
		FROM entries`
	var args []interface{}
	var conditions []string

	if opts.StartDate != nil {
		conditions = append(conditions, "date(created_at, 'localtime') >= ?")
		args = append(args, opts.StartDate.Format("2006-01-02"))
	}
	if opts.EndDate != nil {
		conditions = append(conditions, "date(created_at, 'localtime') <= ?")
		args = append(args, opts.EndDate.Format("2006-01-02"))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY date(created_at, 'localtime') ORDER BY day DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing days: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var summaries []storage.DaySummary
	for rows.Next() {
		var dayStr, preview string
		var count int
		if err := rows.Scan(&dayStr, &count, &preview); err != nil {
			return nil, fmt.Errorf("%w: scanning day row: %v", storage.ErrStorage, err)
		}
		// libSQL's date() may return "YYYY-MM-DD" or "YYYY-MM-DDT00:00:00Z"
		dateStr := dayStr
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			return nil, fmt.Errorf("%w: parsing date: %v", storage.ErrStorage, err)
		}
		// Truncate preview to 80 chars, single line
		preview = strings.ReplaceAll(preview, "\n", " ")
		if len(preview) > 80 {
			preview = preview[:80]
		}
		summaries = append(summaries, storage.DaySummary{
			Date:    date,
			Count:   count,
			Preview: preview,
		})
	}

	if summaries == nil {
		summaries = []storage.DaySummary{}
	}

	return summaries, rows.Err()
}

// Update modifies an existing entry's content.
func (s *Store) Update(id string, content string) (entry.Entry, error) {
	if err := entry.ValidateContent(content); err != nil {
		return entry.Entry{}, fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return entry.Entry{}, fmt.Errorf("%w: beginning transaction: %v", storage.ErrStorage, err)
	}
	defer tx.Rollback()

	// Check existence
	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM entries WHERE id = ?", id).Scan(&exists); err != nil {
		return entry.Entry{}, fmt.Errorf("%w: checking entry: %v", storage.ErrStorage, err)
	}
	if exists == 0 {
		return entry.Entry{}, storage.ErrNotFound
	}

	if _, err := tx.Exec(
		"UPDATE entries SET content = ?, updated_at = ? WHERE id = ?",
		content, now, id,
	); err != nil {
		return entry.Entry{}, fmt.Errorf("%w: updating entry: %v", storage.ErrStorage, err)
	}

	if err := tx.Commit(); err != nil {
		return entry.Entry{}, fmt.Errorf("%w: committing: %v", storage.ErrStorage, err)
	}

	return s.Get(id)
}

// --- Template methods ---

// CreateTemplate persists a new template.
func (s *Store) CreateTemplate(t storage.Template) error {
	if err := entry.ValidateTemplateName(t.Name); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}
	_, err := s.db.Exec(
		"INSERT INTO templates (id, name, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		t.ID, t.Name, t.Content,
		t.CreatedAt.UTC().Format(time.RFC3339),
		t.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: template %q already exists", storage.ErrConflict, t.Name)
		}
		return fmt.Errorf("%w: inserting template: %v", storage.ErrStorage, err)
	}
	return nil
}

// GetTemplate retrieves a template by ID.
func (s *Store) GetTemplate(id string) (storage.Template, error) {
	row := s.db.QueryRow(
		"SELECT id, name, content, created_at, updated_at FROM templates WHERE id = ?", id,
	)
	return s.scanTemplate(row)
}

// GetTemplateByName retrieves a template by name.
func (s *Store) GetTemplateByName(name string) (storage.Template, error) {
	row := s.db.QueryRow(
		"SELECT id, name, content, created_at, updated_at FROM templates WHERE name = ?", name,
	)
	return s.scanTemplate(row)
}

func (s *Store) scanTemplate(row *sql.Row) (storage.Template, error) {
	var t storage.Template
	var createdStr, updatedStr string
	if err := row.Scan(&t.ID, &t.Name, &t.Content, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return storage.Template{}, storage.ErrNotFound
		}
		return storage.Template{}, fmt.Errorf("%w: querying template: %v", storage.ErrStorage, err)
	}
	var err error
	t.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}
	return t, nil
}

// ListTemplates returns all templates sorted by name.
func (s *Store) ListTemplates() ([]storage.Template, error) {
	rows, err := s.db.Query(
		"SELECT id, name, content, created_at, updated_at FROM templates ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("%w: listing templates: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var templates []storage.Template
	for rows.Next() {
		var t storage.Template
		var createdStr, updatedStr string
		if err := rows.Scan(&t.ID, &t.Name, &t.Content, &createdStr, &updatedStr); err != nil {
			return nil, fmt.Errorf("%w: scanning template row: %v", storage.ErrStorage, err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// UpdateTemplate modifies an existing template's name and content.
func (s *Store) UpdateTemplate(id string, name string, content string) (storage.Template, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return storage.Template{}, fmt.Errorf("%w: beginning transaction: %v", storage.ErrStorage, err)
	}
	defer tx.Rollback()

	// Check existence
	var exists int
	if err := tx.QueryRow("SELECT COUNT(*) FROM templates WHERE id = ?", id).Scan(&exists); err != nil {
		return storage.Template{}, fmt.Errorf("%w: checking template: %v", storage.ErrStorage, err)
	}
	if exists == 0 {
		return storage.Template{}, storage.ErrNotFound
	}

	_, err = tx.Exec(
		"UPDATE templates SET name = ?, content = ?, updated_at = ? WHERE id = ?",
		name, content, now, id,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return storage.Template{}, fmt.Errorf("%w: template %q already exists", storage.ErrConflict, name)
		}
		return storage.Template{}, fmt.Errorf("%w: updating template: %v", storage.ErrStorage, err)
	}

	if err := tx.Commit(); err != nil {
		return storage.Template{}, fmt.Errorf("%w: committing: %v", storage.ErrStorage, err)
	}

	return s.GetTemplate(id)
}

// DeleteTemplate removes a template by ID.
func (s *Store) DeleteTemplate(id string) error {
	result, err := s.db.Exec("DELETE FROM templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("%w: deleting template: %v", storage.ErrStorage, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: checking rows affected: %v", storage.ErrStorage, err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// Delete removes an entry permanently.
func (s *Store) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("%w: deleting entry: %v", storage.ErrStorage, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: checking rows affected: %v", storage.ErrStorage, err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}
