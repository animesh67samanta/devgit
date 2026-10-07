package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Repository represents a tracked Git repository workspace in DevGit metadata.
type Repository struct {
	ID         int64
	Path       string
	LastUsedAt time.Time
	CreatedAt  time.Time
}

// GetOrCreateRepository retrieves or registers a repository by its filesystem path.
// It normalizes the path using filepath.Clean and updates last_used_at.
func (d *DB) GetOrCreateRepository(ctx context.Context, path string) (*Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("repository path cannot be empty")
	}

	cleanPath := filepath.Clean(path)
	now := time.Now().UTC()

	// Try atomic upsert with RETURNING
	query := `INSERT INTO repositories (path, last_used_at, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET last_used_at = excluded.last_used_at
		RETURNING id, path, last_used_at, created_at;`

	var repo Repository
	err := d.sqlDB.QueryRowContext(ctx, query, cleanPath, now, now).Scan(
		&repo.ID,
		&repo.Path,
		&repo.LastUsedAt,
		&repo.CreatedAt,
	)
	if err == nil {
		return &repo, nil
	}

	// Fallback for drivers or configurations that do not support RETURNING
	selectQuery := `SELECT id, path, last_used_at, created_at FROM repositories WHERE path = ?;`
	err = d.sqlDB.QueryRowContext(ctx, selectQuery, cleanPath).Scan(
		&repo.ID,
		&repo.Path,
		&repo.LastUsedAt,
		&repo.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register repository %s: %w", cleanPath, err)
	}

	return &repo, nil
}

// GetRepositoryByPath retrieves a repository record by exact path match.
func (d *DB) GetRepositoryByPath(ctx context.Context, path string) (*Repository, error) {
	cleanPath := filepath.Clean(path)
	query := `SELECT id, path, last_used_at, created_at FROM repositories WHERE path = ?;`

	var repo Repository
	err := d.sqlDB.QueryRowContext(ctx, query, cleanPath).Scan(
		&repo.ID,
		&repo.Path,
		&repo.LastUsedAt,
		&repo.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query repository %s: %w", cleanPath, err)
	}

	return &repo, nil
}

// TouchRepository updates the last_used_at timestamp of a repository.
func (d *DB) TouchRepository(ctx context.Context, path string) error {
	cleanPath := filepath.Clean(path)
	now := time.Now().UTC()

	query := `UPDATE repositories SET last_used_at = ? WHERE path = ?;`
	res, err := d.sqlDB.ExecContext(ctx, query, now, cleanPath)
	if err != nil {
		return fmt.Errorf("failed to touch repository %s: %w", cleanPath, err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		_, err := d.GetOrCreateRepository(ctx, cleanPath)
		return err
	}

	return nil
}

// ListRecentRepositories returns the most recently accessed repositories up to limit.
func (d *DB) ListRecentRepositories(ctx context.Context, limit int) ([]Repository, error) {
	if limit <= 0 {
		limit = 20
	}

	query := `SELECT id, path, last_used_at, created_at
		FROM repositories
		ORDER BY last_used_at DESC, id DESC
		LIMIT ?;`

	rows, err := d.sqlDB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent repositories: %w", err)
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.Path, &r.LastUsedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed scanning repository row: %w", err)
		}
		repos = append(repos, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading repositories: %w", err)
	}

	return repos, nil
}

// DeleteRepository removes a repository record from metadata.
func (d *DB) DeleteRepository(ctx context.Context, id int64) error {
	query := `DELETE FROM repositories WHERE id = ?;`
	_, err := d.sqlDB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete repository id %d: %w", id, err)
	}
	return nil
}

// CleanMissingRepositories inspects tracked repositories on disk and removes those whose paths no longer exist.
func (d *DB) CleanMissingRepositories(ctx context.Context) (int, error) {
	repos, err := d.ListRecentRepositories(ctx, 1000)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, r := range repos {
		if _, err := os.Stat(r.Path); os.IsNotExist(err) {
			if err := d.DeleteRepository(ctx, r.ID); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}
