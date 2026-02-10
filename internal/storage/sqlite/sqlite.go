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
		`CREATE TABLE IF NOT EXISTS contexts (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL UNIQUE,
			source     TEXT NOT NULL DEFAULT 'manual',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS entry_contexts (
			entry_id    TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
			context_id  TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
			PRIMARY KEY (entry_id, context_id)
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

	for _, ref := range e.Contexts {
		_, err = tx.Exec(
			"INSERT OR IGNORE INTO entry_contexts (entry_id, context_id) VALUES (?, ?)",
			e.ID, ref.ContextID,
		)
		if err != nil {
			return fmt.Errorf("%w: inserting context ref: %v", storage.ErrStorage, err)
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

	// Load context refs
	ctxRefs, err := s.loadContextRefs(id)
	if err != nil {
		return entry.Entry{}, err
	}
	e.Contexts = ctxRefs

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

	if opts.ContextName != "" {
		query += " JOIN entry_contexts ec ON ec.entry_id = entries.id JOIN contexts ctx ON ctx.id = ec.context_id"
		conditions = append(conditions, "ctx.name = ?")
		args = append(args, opts.ContextName)
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

		// Load context refs
		ctxRefs, err := s.loadContextRefs(e.ID)
		if err != nil {
			return nil, err
		}
		e.Contexts = ctxRefs

		entries = append(entries, e)
	}

	if entries == nil {
		entries = []entry.Entry{}
	}

	return entries, rows.Err()
}

// ListDays returns aggregated day summaries grouped by date.
func (s *Store) ListDays(opts storage.ListDaysOptions) ([]storage.DaySummary, error) {
	query := `SELECT date(entries.created_at, 'localtime') as day, COUNT(*) as cnt,
		(SELECT content FROM entries e2 WHERE date(e2.created_at, 'localtime') = date(entries.created_at, 'localtime') ORDER BY e2.created_at DESC LIMIT 1) as preview
		FROM entries`
	var args []any
	var conditions []string

	if opts.TemplateName != "" {
		query += ` JOIN entry_templates et ON et.entry_id = entries.id`
		conditions = append(conditions, "et.template_name = ?")
		args = append(args, opts.TemplateName)
	}

	if opts.StartDate != nil {
		conditions = append(conditions, "date(entries.created_at, 'localtime') >= ?")
		args = append(args, opts.StartDate.Format("2006-01-02"))
	}
	if opts.EndDate != nil {
		conditions = append(conditions, "date(entries.created_at, 'localtime') <= ?")
		args = append(args, opts.EndDate.Format("2006-01-02"))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " GROUP BY date(entries.created_at, 'localtime') ORDER BY day DESC"

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

// Update modifies an existing entry's content and optionally its template refs.
// Pass nil for templates to preserve existing refs.
func (s *Store) Update(id string, content string, templates []entry.TemplateRef) (entry.Entry, error) {
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

	if templates != nil {
		// Replace template refs
		if _, err := tx.Exec("DELETE FROM entry_templates WHERE entry_id = ?", id); err != nil {
			return entry.Entry{}, fmt.Errorf("%w: clearing template refs: %v", storage.ErrStorage, err)
		}
		for _, ref := range templates {
			if _, err := tx.Exec(
				"INSERT INTO entry_templates (entry_id, template_id, template_name) VALUES (?, ?, ?)",
				id, ref.TemplateID, ref.TemplateName,
			); err != nil {
				return entry.Entry{}, fmt.Errorf("%w: inserting template ref: %v", storage.ErrStorage, err)
			}
		}
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

// UpdateTemplate modifies an existing template's name, content, and attributes.
// Note: V1 sqlite schema does not support attributes, so the attributes parameter is ignored.
func (s *Store) UpdateTemplate(id string, name string, content string, attributes map[string]string) (storage.Template, error) {
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

// --- Context methods ---

// CreateContext persists a new context.
func (s *Store) CreateContext(c storage.Context) error {
	if err := entry.ValidateContextName(c.Name); err != nil {
		return fmt.Errorf("%w: %v", storage.ErrValidation, err)
	}
	_, err := s.db.Exec(
		"INSERT INTO contexts (id, name, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		c.ID, c.Name, c.Source,
		c.CreatedAt.UTC().Format(time.RFC3339),
		c.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: context %q already exists", storage.ErrConflict, c.Name)
		}
		return fmt.Errorf("%w: inserting context: %v", storage.ErrStorage, err)
	}
	return nil
}

// GetContext retrieves a context by ID.
func (s *Store) GetContext(id string) (storage.Context, error) {
	row := s.db.QueryRow(
		"SELECT id, name, source, created_at, updated_at FROM contexts WHERE id = ?", id,
	)
	return s.scanContext(row)
}

// GetContextByName retrieves a context by name.
func (s *Store) GetContextByName(name string) (storage.Context, error) {
	row := s.db.QueryRow(
		"SELECT id, name, source, created_at, updated_at FROM contexts WHERE name = ?", name,
	)
	return s.scanContext(row)
}

func (s *Store) scanContext(row *sql.Row) (storage.Context, error) {
	var c storage.Context
	var createdStr, updatedStr string
	if err := row.Scan(&c.ID, &c.Name, &c.Source, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return storage.Context{}, storage.ErrNotFound
		}
		return storage.Context{}, fmt.Errorf("%w: scanning context: %v", storage.ErrStorage, err)
	}
	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return storage.Context{}, fmt.Errorf("%w: parsing created_at: %v", storage.ErrStorage, err)
	}
	c.UpdatedAt, err = time.Parse(time.RFC3339, updatedStr)
	if err != nil {
		return storage.Context{}, fmt.Errorf("%w: parsing updated_at: %v", storage.ErrStorage, err)
	}
	return c, nil
}

// ListContexts returns all contexts sorted by name.
func (s *Store) ListContexts() ([]storage.Context, error) {
	rows, err := s.db.Query(
		"SELECT id, name, source, created_at, updated_at FROM contexts ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("%w: listing contexts: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var contexts []storage.Context
	for rows.Next() {
		var c storage.Context
		var createdStr, updatedStr string
		if err := rows.Scan(&c.ID, &c.Name, &c.Source, &createdStr, &updatedStr); err != nil {
			return nil, fmt.Errorf("%w: scanning context row: %v", storage.ErrStorage, err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		contexts = append(contexts, c)
	}
	return contexts, rows.Err()
}

// DeleteContext removes a context by ID.
func (s *Store) DeleteContext(id string) error {
	result, err := s.db.Exec("DELETE FROM contexts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("%w: deleting context: %v", storage.ErrStorage, err)
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

// AttachContext links an entry to a context.
func (s *Store) AttachContext(entryID string, contextID string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO entry_contexts (entry_id, context_id) VALUES (?, ?)",
		entryID, contextID,
	)
	if err != nil {
		return fmt.Errorf("%w: attaching context: %v", storage.ErrStorage, err)
	}
	return nil
}

// DetachContext removes the link between an entry and a context.
func (s *Store) DetachContext(entryID string, contextID string) error {
	_, err := s.db.Exec(
		"DELETE FROM entry_contexts WHERE entry_id = ? AND context_id = ?",
		entryID, contextID,
	)
	if err != nil {
		return fmt.Errorf("%w: detaching context: %v", storage.ErrStorage, err)
	}
	return nil
}

// loadContextRefs loads context references for an entry.
func (s *Store) loadContextRefs(entryID string) ([]entry.ContextRef, error) {
	rows, err := s.db.Query(
		"SELECT c.id, c.name FROM entry_contexts ec JOIN contexts c ON c.id = ec.context_id WHERE ec.entry_id = ?",
		entryID,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: querying context refs: %v", storage.ErrStorage, err)
	}
	defer rows.Close()

	var refs []entry.ContextRef
	for rows.Next() {
		var ref entry.ContextRef
		if err := rows.Scan(&ref.ContextID, &ref.ContextName); err != nil {
			return nil, fmt.Errorf("%w: scanning context ref: %v", storage.ErrStorage, err)
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
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
