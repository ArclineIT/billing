package db

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database and runs schema migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := runSchema(db); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	slog.Info("database ready", "path", path)
	return db, nil
}

func runSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS customers (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			email             TEXT    NOT NULL UNIQUE,
			password_hash     TEXT    NOT NULL,
			first_name        TEXT    NOT NULL DEFAULT '',
			last_name         TEXT    NOT NULL DEFAULT '',
			stripe_customer_id TEXT   NOT NULL DEFAULT '',
			created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			token       TEXT PRIMARY KEY,
			customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			expires_at  TEXT    NOT NULL,
			created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		)`,

		`CREATE TABLE IF NOT EXISTS subscriptions (
			id                     INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id            INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			stripe_subscription_id TEXT    NOT NULL UNIQUE,
			stripe_price_id        TEXT    NOT NULL DEFAULT '',
			plan_name              TEXT    NOT NULL DEFAULT '',
			status                 TEXT    NOT NULL DEFAULT 'active',
			current_period_end     TEXT    NOT NULL DEFAULT '',
			created_at             TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
			updated_at             TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		)`,

		`CREATE TABLE IF NOT EXISTS invoices (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_id      INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			stripe_invoice_id TEXT   NOT NULL UNIQUE,
			amount_cents     INTEGER NOT NULL DEFAULT 0,
			currency         TEXT    NOT NULL DEFAULT 'usd',
			status           TEXT    NOT NULL DEFAULT 'open',
			invoice_pdf_url  TEXT    NOT NULL DEFAULT '',
			period_start     TEXT    NOT NULL DEFAULT '',
			period_end       TEXT    NOT NULL DEFAULT '',
			created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		)`,

		`CREATE TABLE IF NOT EXISTS password_resets (
			token       TEXT PRIMARY KEY,
			customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
			expires_at  TEXT    NOT NULL,
			used        INTEGER NOT NULL DEFAULT 0
		)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}
	return nil
}
