package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	dsn := filepath.Join(dataDir, "tally.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS projects (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        api_key TEXT NOT NULL UNIQUE,
        domain TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
     );
     CREATE TABLE IF NOT EXISTS events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        project_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        properties TEXT,
        session_id TEXT,
        ip_hash TEXT,
        user_agent TEXT,
        referrer TEXT,
        occurred_at DATETIME DEFAULT CURRENT_TIMESTAMP
     );
     CREATE TABLE IF NOT EXISTS funnels (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        project_id INTEGER NOT NULL,
        name TEXT NOT NULL,
        steps TEXT NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
     );`)
	return err
}
