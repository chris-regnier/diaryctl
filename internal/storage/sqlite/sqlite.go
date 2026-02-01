package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

	// Enable WAL mode
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
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
	schema := `
		CREATE TABLE IF NOT EXISTS entries (
			id         TEXT PRIMARY KEY,
			content    TEXT NOT NULL CHECK(length(trim(content)) > 0),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			CHECK(created_at <= updated_at)
		);
		CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_entries_date ON entries(date(created_at));
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("%w: creating schema: %v", storage.ErrStorage, err)
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

	_, err := s.db.Exec(
		"INSERT INTO entries (id, content, created_at, updated_at) VALUES (?, ?, ?, ?)",
		e.ID,
		e.Content,
		e.CreatedAt.UTC().Format(time.RFC3339),
		e.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("%w: inserting entry: %v", storage.ErrStorage, err)
	}
	return nil
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

	return e, nil
}

// List returns entries matching the given options.
func (s *Store) List(opts storage.ListOptions) ([]entry.Entry, error) {
	query := "SELECT id, content, created_at, updated_at FROM entries"
	var args []interface{}

	if opts.Date != nil {
		query += " WHERE date(created_at) = date(?)"
		args = append(args, opts.Date.Format("2006-01-02"))
	}

	query += " ORDER BY created_at DESC"

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
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []entry.Entry{}
	}

	return entries, rows.Err()
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
