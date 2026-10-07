package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	clockMu   sync.Mutex
	lastClock time.Time
)

// nowUTC returns current UTC time, guaranteed to be strictly monotonically increasing
// across rapid consecutive calls even on operating systems with low timer resolution (e.g. Windows).
func nowUTC() time.Time {
	clockMu.Lock()
	defer clockMu.Unlock()

	now := time.Now().UTC()
	if !now.After(lastClock) {
		now = lastClock.Add(time.Microsecond)
	}
	lastClock = now
	return now
}

// DB wraps an underlying SQLite database connection and provides high-level
// operations for DevGit metadata storage.
type DB struct {
	sqlDB *sql.DB
	path  string
}

// Diagnostics holds operational status and metadata metrics about the DevGit database.
type Diagnostics struct {
	Path            string
	SchemaVersion   int
	FileSizeBytes   int64
	RepositoryCount int
	HistoryCount    int
	PreferenceCount int
	WALEnabled      bool
}

// Open opens a SQLite database at the specified filesystem path, runs all pending migrations,
// and configures recommended SQLite pragmas (WAL mode, foreign keys, busy timeout).
func Open(path string) (*DB, error) {
	if path == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("unable to create database directory: %w", err)
		}
	}

	// Connect to SQLite
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool limits appropriate for SQLite
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure pragmas
	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
	}
	if path != ":memory:" {
		pragmas = append(pragmas, `PRAGMA journal_mode = WAL;`, `PRAGMA synchronous = NORMAL;`)
	}

	for _, p := range pragmas {
		if _, err := sqlDB.ExecContext(ctx, p); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("failed setting pragma %q: %w", p, err)
		}
	}

	// Run versioned migrations
	if err := RunMigrations(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed running database migrations: %w", err)
	}

	return &DB{
		sqlDB: sqlDB,
		path:  path,
	}, nil
}

// OpenDefault opens the SQLite database using the standard platform-specific location.
func OpenDefault() (*DB, error) {
	p, err := DBPath()
	if err != nil {
		return nil, err
	}
	return Open(p)
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	if d.sqlDB == nil {
		return nil
	}
	return d.sqlDB.Close()
}

// Ping checks if the database is reachable and active.
func (d *DB) Ping(ctx context.Context) error {
	if d.sqlDB == nil {
		return fmt.Errorf("database not open")
	}
	return d.sqlDB.PingContext(ctx)
}

// Path returns the filesystem path of the database.
func (d *DB) Path() string {
	return d.path
}

// Diagnostics gathers and returns operational metrics for diagnostics.
func (d *DB) Diagnostics(ctx context.Context) (*Diagnostics, error) {
	diag := &Diagnostics{
		Path: d.path,
	}

	// Schema version
	ver, err := CurrentVersion(ctx, d.sqlDB)
	if err != nil {
		return nil, err
	}
	diag.SchemaVersion = ver

	// File size
	if d.path != ":memory:" {
		if fi, err := os.Stat(d.path); err == nil {
			diag.FileSizeBytes = fi.Size()
		}
	}

	// WAL mode check
	var journalMode string
	_ = d.sqlDB.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode)
	diag.WALEnabled = strings.EqualFold(journalMode, "wal")

	// Table counts
	_ = d.sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM repositories;").Scan(&diag.RepositoryCount)
	_ = d.sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM command_history;").Scan(&diag.HistoryCount)
	_ = d.sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM preferences;").Scan(&diag.PreferenceCount)

	return diag, nil
}
