package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PreferenceEntry represents a stored preference key-value pair.
type PreferenceEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

// GetPreference retrieves the value of a persisted preference.
// Returns (value, found, error).
func (d *DB) GetPreference(ctx context.Context, key string) (string, bool, error) {
	normKey := strings.TrimSpace(key)
	if normKey == "" {
		return "", false, fmt.Errorf("preference key cannot be empty")
	}

	query := `SELECT value FROM preferences WHERE key = ?;`
	var val string
	err := d.sqlDB.QueryRowContext(ctx, query, normKey).Scan(&val)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get preference %s: %w", normKey, err)
	}

	return val, true, nil
}

// SetPreference persists or updates a preference setting.
func (d *DB) SetPreference(ctx context.Context, key, value string) error {
	normKey := strings.TrimSpace(key)
	if normKey == "" {
		return fmt.Errorf("preference key cannot be empty")
	}

	now := time.Now().UTC()
	query := `INSERT INTO preferences (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at;`

	_, err := d.sqlDB.ExecContext(ctx, query, normKey, value, now)
	if err != nil {
		return fmt.Errorf("failed to set preference %s: %w", normKey, err)
	}

	return nil
}

// DeletePreference removes a preference entry.
func (d *DB) DeletePreference(ctx context.Context, key string) error {
	normKey := strings.TrimSpace(key)
	if normKey == "" {
		return fmt.Errorf("preference key cannot be empty")
	}

	query := `DELETE FROM preferences WHERE key = ?;`
	_, err := d.sqlDB.ExecContext(ctx, query, normKey)
	if err != nil {
		return fmt.Errorf("failed to delete preference %s: %w", normKey, err)
	}

	return nil
}

// ListPreferences returns all stored preferences as a map.
func (d *DB) ListPreferences(ctx context.Context) (map[string]string, error) {
	query := `SELECT key, value FROM preferences ORDER BY key ASC;`
	rows, err := d.sqlDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list preferences: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("failed scanning preference row: %w", err)
		}
		result[k] = v
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading preferences: %w", err)
	}

	return result, nil
}
