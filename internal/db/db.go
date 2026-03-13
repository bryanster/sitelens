package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type Site struct {
	ID        int64      `json:"id"`
	URL       string     `json:"url"`
	Title     string     `json:"title"`
	Snippet   string     `json:"snippet"`
	Category  string     `json:"category"`
	Model     string     `json:"model"`
	Status    string     `json:"status"`
	ErrorMsg  string     `json:"error_msg,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
		CREATE SEQUENCE IF NOT EXISTS sites_id_seq;

		CREATE TABLE IF NOT EXISTS sites (
			id         INTEGER  DEFAULT nextval('sites_id_seq') PRIMARY KEY,
			url        TEXT     NOT NULL,
			title      TEXT     DEFAULT '',
			snippet    TEXT     DEFAULT '',
			category   TEXT     DEFAULT '',
			model      TEXT     DEFAULT '',
			status     TEXT     DEFAULT 'pending',
			error_msg  TEXT     DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	return err
}

func (d *DB) InsertSite(url string) (int64, error) {
	var id int64
	err := d.conn.QueryRow(
		`INSERT INTO sites (url) VALUES (?) RETURNING id`, url,
	).Scan(&id)
	return id, err
}

func (d *DB) UpdateSite(id int64, title, snippet, category, model, status, errMsg string) error {
	_, err := d.conn.Exec(
		`UPDATE sites SET title=?, snippet=?, category=?, model=?, status=?, error_msg=?, updated_at=now() WHERE id=?`,
		title, snippet, category, model, status, errMsg, id,
	)
	return err
}

func (d *DB) SetStatus(id int64, status, errMsg string) error {
	_, err := d.conn.Exec(
		`UPDATE sites SET status=?, error_msg=?, updated_at=now() WHERE id=?`,
		status, errMsg, id,
	)
	return err
}

func (d *DB) ListSites(categoryFilter string) ([]Site, error) {
	query := `SELECT id, url, title, snippet, category, model, status, error_msg, created_at, updated_at FROM sites`
	args := []any{}
	if categoryFilter != "" {
		query += ` WHERE category = ?`
		args = append(args, categoryFilter)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.URL, &s.Title, &s.Snippet, &s.Category, &s.Model, &s.Status, &s.ErrorMsg, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

func (d *DB) GetSite(id int64) (*Site, error) {
	var s Site
	err := d.conn.QueryRow(
		`SELECT id, url, title, snippet, category, model, status, error_msg, created_at, updated_at FROM sites WHERE id = ?`, id,
	).Scan(&s.ID, &s.URL, &s.Title, &s.Snippet, &s.Category, &s.Model, &s.Status, &s.ErrorMsg, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) DeleteSite(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM sites WHERE id = ?`, id)
	return err
}

func (d *DB) Categories() ([]string, error) {
	rows, err := d.conn.Query(`SELECT DISTINCT category FROM sites WHERE category != '' ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (d *DB) ListSitesByIDs(ids []int64, categoryFilter string) ([]Site, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// Build IN clause
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT id, url, title, snippet, category, model, status, error_msg, created_at, updated_at
		FROM sites WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	if categoryFilter != "" {
		query += ` AND category = ?`
		args = append(args, categoryFilter)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.URL, &s.Title, &s.Snippet, &s.Category, &s.Model, &s.Status, &s.ErrorMsg, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

func (d *DB) HasPending() (bool, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM sites WHERE status IN ('pending', 'processing')`).Scan(&count)
	return count > 0, err
}

type Stats struct {
	Total      int
	Done       int
	Pending    int
	Categories int
}

func (d *DB) Stats() (Stats, error) {
	var s Stats
	err := d.conn.QueryRow(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'done'),
			COUNT(*) FILTER (WHERE status IN ('pending', 'processing')),
			COUNT(DISTINCT category) FILTER (WHERE category != '')
		FROM sites
	`).Scan(&s.Total, &s.Done, &s.Pending, &s.Categories)
	return s, err
}

func (d *DB) SearchSites(query, category string) ([]Site, error) {
	args := []any{}
	where := []string{"status = 'done'"}

	if query != "" {
		like := "%" + query + "%"
		where = append(where, "(url ILIKE ? OR title ILIKE ? OR snippet ILIKE ?)")
		args = append(args, like, like, like)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}

	q := `SELECT id, url, title, snippet, category, model, status, error_msg, created_at, updated_at FROM sites WHERE `
	for i, w := range where {
		if i > 0 {
			q += " AND "
		}
		q += w
	}
	q += ` ORDER BY created_at DESC LIMIT 200`

	rows, err := d.conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.URL, &s.Title, &s.Snippet, &s.Category, &s.Model, &s.Status, &s.ErrorMsg, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}
