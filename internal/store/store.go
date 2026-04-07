package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

// Counter is a named integer counter scoped to a namespace. The
// (name, namespace) pair is unique. Counters auto-create on first
// Increment/Set so callers don't have to declare them up front.
type Counter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Value       int64  `json:"value"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "tally.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS counters(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		namespace TEXT DEFAULT 'default',
		value INTEGER DEFAULT 0,
		description TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now')),
		updated_at TEXT DEFAULT(datetime('now')),
		UNIQUE(name, namespace)
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_counters_namespace ON counters(namespace)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(
		resource TEXT NOT NULL,
		record_id TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(resource, record_id)
	)`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string   { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) Create(c *Counter) error {
	c.ID = genID()
	c.CreatedAt = now()
	c.UpdatedAt = c.CreatedAt
	if c.Namespace == "" {
		c.Namespace = "default"
	}
	_, err := d.db.Exec(
		`INSERT INTO counters(id, name, namespace, value, description, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Namespace, c.Value, c.Description, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (d *DB) Get(name, ns string) *Counter {
	if ns == "" {
		ns = "default"
	}
	var c Counter
	err := d.db.QueryRow(
		`SELECT id, name, namespace, value, description, created_at, updated_at
		 FROM counters WHERE name=? AND namespace=?`,
		name, ns,
	).Scan(&c.ID, &c.Name, &c.Namespace, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil
	}
	return &c
}

func (d *DB) GetByID(id string) *Counter {
	var c Counter
	err := d.db.QueryRow(
		`SELECT id, name, namespace, value, description, created_at, updated_at
		 FROM counters WHERE id=?`,
		id,
	).Scan(&c.ID, &c.Name, &c.Namespace, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil
	}
	return &c
}

func (d *DB) List(ns string) []Counter {
	q := `SELECT id, name, namespace, value, description, created_at, updated_at FROM counters`
	args := []any{}
	if ns != "" && ns != "all" {
		q += ` WHERE namespace=?`
		args = append(args, ns)
	}
	q += ` ORDER BY namespace ASC, name ASC`
	rows, _ := d.db.Query(q, args...)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []Counter
	for rows.Next() {
		var c Counter
		rows.Scan(&c.ID, &c.Name, &c.Namespace, &c.Value, &c.Description, &c.CreatedAt, &c.UpdatedAt)
		o = append(o, c)
	}
	return o
}

// Increment atomically adds 'by' to the counter's value, auto-creating
// the counter if it doesn't exist. 'by' can be negative for decrement.
func (d *DB) Increment(name, ns string, by int64) *Counter {
	if ns == "" {
		ns = "default"
	}
	t := now()
	c := d.Get(name, ns)
	if c == nil {
		id := genID()
		d.db.Exec(
			`INSERT INTO counters(id, name, namespace, value, description, created_at, updated_at)
			 VALUES(?, ?, ?, ?, ?, ?, ?)`,
			id, name, ns, by, "", t, t,
		)
		return d.GetByID(id)
	}
	d.db.Exec(`UPDATE counters SET value=value+?, updated_at=? WHERE id=?`, by, t, c.ID)
	return d.Get(name, ns)
}

// Set assigns an absolute value to the counter, auto-creating it if
// it doesn't exist.
func (d *DB) Set(name, ns string, val int64) *Counter {
	if ns == "" {
		ns = "default"
	}
	t := now()
	c := d.Get(name, ns)
	if c == nil {
		id := genID()
		d.db.Exec(
			`INSERT INTO counters(id, name, namespace, value, description, created_at, updated_at)
			 VALUES(?, ?, ?, ?, ?, ?, ?)`,
			id, name, ns, val, "", t, t,
		)
		return d.GetByID(id)
	}
	d.db.Exec(`UPDATE counters SET value=?, updated_at=? WHERE id=?`, val, t, c.ID)
	return d.Get(name, ns)
}

// Reset zeroes a counter without deleting it. Returns the counter.
func (d *DB) Reset(id string) *Counter {
	d.db.Exec(`UPDATE counters SET value=0, updated_at=? WHERE id=?`, now(), id)
	return d.GetByID(id)
}

// Update edits the metadata fields of a counter (name, namespace,
// description). Value is managed by Increment/Set/Reset, never by
// Update. The original implementation had no Update method at all.
func (d *DB) Update(id string, c *Counter) error {
	_, err := d.db.Exec(
		`UPDATE counters SET name=?, namespace=?, description=?, updated_at=? WHERE id=?`,
		c.Name, c.Namespace, c.Description, now(), id,
	)
	return err
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM counters WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM counters`).Scan(&n)
	return n
}

// Namespaces returns the distinct namespaces present, useful for
// populating UI filter dropdowns.
func (d *DB) Namespaces() []string {
	rows, _ := d.db.Query(`SELECT DISTINCT namespace FROM counters ORDER BY namespace ASC`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		out = append(out, s)
	}
	return out
}

// Stats returns total counters, the sum of all values across all
// counters, the namespace count, and a per-namespace breakdown of
// counter counts and value sums.
func (d *DB) Stats() map[string]any {
	m := map[string]any{
		"total":        d.Count(),
		"total_value":  int64(0),
		"namespaces":   0,
		"by_namespace": map[string]map[string]any{},
	}

	var totalValue int64
	d.db.QueryRow(`SELECT COALESCE(SUM(value), 0) FROM counters`).Scan(&totalValue)
	m["total_value"] = totalValue

	var nsCount int
	d.db.QueryRow(`SELECT COUNT(DISTINCT namespace) FROM counters`).Scan(&nsCount)
	m["namespaces"] = nsCount

	if rows, _ := d.db.Query(`SELECT namespace, COUNT(*), COALESCE(SUM(value), 0) FROM counters GROUP BY namespace`); rows != nil {
		defer rows.Close()
		by := map[string]map[string]any{}
		for rows.Next() {
			var ns string
			var cnt int
			var sum int64
			rows.Scan(&ns, &cnt, &sum)
			by[ns] = map[string]any{"count": cnt, "sum": sum}
		}
		m["by_namespace"] = by
	}

	return m
}

// ─── Extras ───────────────────────────────────────────────────────

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
