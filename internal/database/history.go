package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HistoryEntry represents a single command execution record.
type HistoryEntry struct {
	ID           int64
	RepositoryID *int64
	Command      string
	Success      bool
	ExecutedAt   time.Time
	DurationMS   int64
}

// HistoryRecord combines HistoryEntry with optional RepositoryPath for user display.
type HistoryRecord struct {
	HistoryEntry
	RepositoryPath string
}

// HistoryListOptions configures filtering and pagination for history retrieval.
type HistoryListOptions struct {
	Limit        int
	RepositoryID *int64
	OnlySuccess  bool
	OnlyFailed   bool
}

// RecordCommand persists a command execution event into the command_history table.
func (d *DB) RecordCommand(ctx context.Context, entry HistoryEntry) (*HistoryEntry, error) {
	if entry.Command == "" {
		return nil, fmt.Errorf("command string cannot be empty")
	}

	if entry.ExecutedAt.IsZero() {
		entry.ExecutedAt = nowUTC()
	}

	query := `INSERT INTO command_history (repository_id, command, success, executed_at, duration_ms)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, repository_id, command, success, executed_at, duration_ms;`

	var recorded HistoryEntry
	err := d.sqlDB.QueryRowContext(
		ctx,
		query,
		entry.RepositoryID,
		entry.Command,
		entry.Success,
		entry.ExecutedAt,
		entry.DurationMS,
	).Scan(
		&recorded.ID,
		&recorded.RepositoryID,
		&recorded.Command,
		&recorded.Success,
		&recorded.ExecutedAt,
		&recorded.DurationMS,
	)
	if err == nil {
		return &recorded, nil
	}

	// Fallback for drivers that do not support RETURNING
	fallbackQuery := `INSERT INTO command_history (repository_id, command, success, executed_at, duration_ms)
		VALUES (?, ?, ?, ?, ?);`
	res, err := d.sqlDB.ExecContext(
		ctx,
		fallbackQuery,
		entry.RepositoryID,
		entry.Command,
		entry.Success,
		entry.ExecutedAt,
		entry.DurationMS,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record command history: %w", err)
	}

	lastID, err := res.LastInsertId()
	if err == nil {
		entry.ID = lastID
	}

	return &entry, nil
}

// ListHistory queries command history matching the specified filter criteria.
func (d *DB) ListHistory(ctx context.Context, opts HistoryListOptions) ([]HistoryRecord, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	query := `SELECT h.id, h.repository_id, h.command, h.success, h.executed_at, h.duration_ms, COALESCE(r.path, '')
		FROM command_history h
		LEFT JOIN repositories r ON h.repository_id = r.id`

	var conditions []string
	var args []interface{}

	if opts.RepositoryID != nil {
		conditions = append(conditions, "h.repository_id = ?")
		args = append(args, *opts.RepositoryID)
	}

	if opts.OnlySuccess {
		conditions = append(conditions, "h.success = 1")
	} else if opts.OnlyFailed {
		conditions = append(conditions, "h.success = 0")
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY h.executed_at DESC LIMIT ?;"
	args = append(args, opts.Limit)

	rows, err := d.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query command history: %w", err)
	}
	defer rows.Close()

	var records []HistoryRecord
	for rows.Next() {
		var rec HistoryRecord
		var repoID sql.NullInt64
		err := rows.Scan(
			&rec.ID,
			&repoID,
			&rec.Command,
			&rec.Success,
			&rec.ExecutedAt,
			&rec.DurationMS,
			&rec.RepositoryPath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning history row: %w", err)
		}
		if repoID.Valid {
			v := repoID.Int64
			rec.RepositoryID = &v
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading history rows: %w", err)
	}

	return records, nil
}

// ClearHistory removes all records from the command_history table.
func (d *DB) ClearHistory(ctx context.Context) (int64, error) {
	res, err := d.sqlDB.ExecContext(ctx, "DELETE FROM command_history;")
	if err != nil {
		return 0, fmt.Errorf("failed to clear command history: %w", err)
	}
	return res.RowsAffected()
}

// PruneHistory cleans up old history entries keeping only the latest keepMax records
// or those newer than the given duration.
func (d *DB) PruneHistory(ctx context.Context, keepMax int, olderThan time.Duration) (int64, error) {
	var totalDeleted int64

	// 1. Delete records older than cutoff if olderThan > 0
	if olderThan > 0 {
		cutoff := time.Now().UTC().Add(-olderThan)
		res, err := d.sqlDB.ExecContext(ctx, "DELETE FROM command_history WHERE executed_at < ?;", cutoff)
		if err != nil {
			return 0, fmt.Errorf("failed pruning by age: %w", err)
		}
		aff, _ := res.RowsAffected()
		totalDeleted += aff
	}

	// 2. Keep at most keepMax latest records if keepMax > 0
	if keepMax > 0 {
		query := `DELETE FROM command_history WHERE id NOT IN (
			SELECT id FROM command_history ORDER BY executed_at DESC LIMIT ?
		);`
		res, err := d.sqlDB.ExecContext(ctx, query, keepMax)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed pruning by count: %w", err)
		}
		aff, _ := res.RowsAffected()
		totalDeleted += aff
	}

	return totalDeleted, nil
}
