// Package store persists saved queries in a local SQLite database.
//
// SQLite is used pure-Go (modernc.org/sqlite) to keep the "single self-
// contained binary" story intact — no cgo, no libsqlite install on the host.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned by Get/Update/Delete when no saved_query row matches.
var ErrNotFound = errors.New("saved query not found")

// ErrDuplicateName is returned by Insert/Update when the unique(name) index rejects a write.
var ErrDuplicateName = errors.New("saved query with this name already exists")

// Param is one element of a saved query's params_json array. `Type` is a
// ClickHouse type string (e.g. "String", "DateTime", "UInt32") — it is echoed
// verbatim into the `{name:Type}` placeholder the SQL template uses.
type Param struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Label   string `json:"label,omitempty"`
	Widget  string `json:"widget,omitempty"`
	Default any    `json:"default,omitempty"`
}

// SavedQuery is the persisted template.
type SavedQuery struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SQLTemplate string    `json:"sql_template"`
	Params      []Param   `json:"params"`
	DefaultViz  string    `json:"default_viz"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store is a thin CRUD layer over SQLite.
type Store struct {
	db *sql.DB
}

// Open opens the SQLite database at path and runs the schema migration.
// Pass ":memory:" for an ephemeral test database.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc.org/sqlite ignores this on :memory: but is important on-disk.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS saved_query (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  name         TEXT NOT NULL UNIQUE,
  description  TEXT NOT NULL DEFAULT '',
  sql_template TEXT NOT NULL,
  params_json  TEXT NOT NULL DEFAULT '[]',
  default_viz  TEXT NOT NULL DEFAULT 'auto',
  created_by   TEXT NOT NULL DEFAULT '',
  created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`)
	return err
}

// Insert persists a new saved query and populates q.ID / q.CreatedAt / q.UpdatedAt.
func (s *Store) Insert(ctx context.Context, q *SavedQuery) error {
	params, err := marshalParams(q.Params)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO saved_query (name, description, sql_template, params_json, default_viz, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, q.Name, q.Description, q.SQLTemplate, params, defaultViz(q.DefaultViz), q.CreatedBy, now, now)
	if err != nil {
		return classify(err)
	}
	q.ID, _ = res.LastInsertId()
	q.CreatedAt = now
	q.UpdatedAt = now
	return nil
}

// Update writes a saved query in place. Returns ErrNotFound if id is unknown.
func (s *Store) Update(ctx context.Context, q *SavedQuery) error {
	params, err := marshalParams(q.Params)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
UPDATE saved_query
   SET name = ?, description = ?, sql_template = ?, params_json = ?, default_viz = ?, updated_at = ?
 WHERE id = ?
`, q.Name, q.Description, q.SQLTemplate, params, defaultViz(q.DefaultViz), now, q.ID)
	if err != nil {
		return classify(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	q.UpdatedAt = now
	return nil
}

// Delete removes the saved query by id.
func (s *Store) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM saved_query WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Get fetches a saved query by id.
func (s *Store) Get(ctx context.Context, id int64) (*SavedQuery, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, description, sql_template, params_json, default_viz, created_by, created_at, updated_at
  FROM saved_query WHERE id = ?
`, id)
	return scanOne(row)
}

// GetByName fetches a saved query by name.
func (s *Store) GetByName(ctx context.Context, name string) (*SavedQuery, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, description, sql_template, params_json, default_viz, created_by, created_at, updated_at
  FROM saved_query WHERE name = ?
`, name)
	return scanOne(row)
}

// List returns all saved queries, most-recently-updated first.
func (s *Store) List(ctx context.Context) ([]SavedQuery, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, description, sql_template, params_json, default_viz, created_by, created_at, updated_at
  FROM saved_query
 ORDER BY updated_at DESC, id DESC
`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SavedQuery
	for rows.Next() {
		q, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

// Ping verifies the SQLite connection is usable.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOne(row rowScanner) (*SavedQuery, error) {
	q, err := scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return q, nil
}

func scanRow(row rowScanner) (*SavedQuery, error) {
	var (
		q      SavedQuery
		params string
	)
	if err := row.Scan(&q.ID, &q.Name, &q.Description, &q.SQLTemplate,
		&params, &q.DefaultViz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return nil, err
	}
	if params != "" {
		if err := json.Unmarshal([]byte(params), &q.Params); err != nil {
			return nil, fmt.Errorf("params_json: %w", err)
		}
	}
	return &q, nil
}

func marshalParams(p []Param) (string, error) {
	if p == nil {
		return "[]", nil
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal params: %w", err)
	}
	return string(b), nil
}

func defaultViz(v string) string {
	if v == "" {
		return "auto"
	}
	return v
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	// modernc.org/sqlite returns errors with the message
	// "constraint failed: UNIQUE constraint failed: saved_query.name (2067)".
	// A substring match keeps us free of importing the driver's error package.
	if containsUniqueViolation(err.Error()) {
		return ErrDuplicateName
	}
	return err
}

func containsUniqueViolation(msg string) bool {
	return strings.Contains(msg, "UNIQUE constraint failed") &&
		strings.Contains(msg, "saved_query.name")
}
