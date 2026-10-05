package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a single versioned database migration step.
type Migration struct {
	Version     int
	Description string
	Up          func(ctx context.Context, tx *sql.Tx) error
}

// migrations contains all ordered database schema migrations.
var migrations = []Migration{
	{
		Version:     1,
		Description: "Initialize repositories, command_history, and preferences tables",
		Up:          migration001InitialSchema,
	},
}

// migration001InitialSchema creates the initial DevGit metadata schema.
func migration001InitialSchema(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		// Repositories table
		`CREATE TABLE IF NOT EXISTS repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			last_used_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_last_used ON repositories(last_used_at DESC);`,

		// Command history table
		`CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repository_id INTEGER,
			command TEXT NOT NULL,
			success BOOLEAN NOT NULL,
			executed_at TIMESTAMP NOT NULL,
			duration_ms INTEGER NOT NULL,
			FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE SET NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_command_history_executed ON command_history(executed_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_command_history_repo ON command_history(repository_id);`,

		// Preferences table
		`CREATE TABLE IF NOT EXISTS preferences (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed executing migration statement: %w", err)
		}
	}

	return nil
}

// ensureMigrationTable creates the schema_migrations table if it does not already exist.
func ensureMigrationTable(ctx context.Context, db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL
	);`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// CurrentVersion returns the highest schema migration version applied to the database.
func CurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	if err := ensureMigrationTable(ctx, db); err != nil {
		return 0, err
	}

	var version sql.NullInt64
	query := `SELECT MAX(version) FROM schema_migrations;`
	err := db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}

	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

// RunMigrations applies any pending migrations in sequential order inside transactions.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	if err := ensureMigrationTable(ctx, db); err != nil {
		return err
	}

	current, err := CurrentVersion(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if m.Version <= current {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin migration transaction for v%d: %w", m.Version, err)
		}

		if err := m.Up(ctx, tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration v%d (%s) failed: %w", m.Version, m.Description, err)
		}

		recordQuery := `INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?);`
		if _, err := tx.ExecContext(ctx, recordQuery, m.Version, m.Description, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration v%d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration v%d: %w", m.Version, err)
		}
	}

	return nil
}
