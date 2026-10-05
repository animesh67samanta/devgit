package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenAndMigrations(t *testing.T) {
	ctx := context.Background()

	// 1. In-memory database
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	diag, err := db.Diagnostics(ctx)
	if err != nil {
		t.Fatalf("failed getting diagnostics: %v", err)
	}
	if diag.SchemaVersion < 1 {
		t.Errorf("expected schema version >= 1, got: %d", diag.SchemaVersion)
	}

	// 2. Re-running migrations is idempotent
	if err := RunMigrations(ctx, db.sqlDB); err != nil {
		t.Fatalf("expected re-running migrations to succeed idempotently: %v", err)
	}

	// 3. File-based database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sub", "test.db")
	fileDB, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open file database: %v", err)
	}
	defer fileDB.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("expected db file to exist at %s: %v", dbPath, err)
	}
}

func TestDatabasePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Test DEVGIT_DB_PATH override
	customDB := filepath.Join(tmpDir, "custom.db")
	t.Setenv("DEVGIT_DB_PATH", customDB)
	p, err := DBPath()
	if err != nil || p != customDB {
		t.Errorf("expected %s, got: %s (err: %v)", customDB, p, err)
	}

	// Test DEVGIT_DATA_DIR override
	t.Setenv("DEVGIT_DB_PATH", "")
	t.Setenv("DEVGIT_DATA_DIR", tmpDir)
	p, err = DBPath()
	expected := filepath.Join(tmpDir, DefaultDBFileName)
	if err != nil || p != expected {
		t.Errorf("expected %s, got: %s (err: %v)", expected, p, err)
	}
}

func TestRepositories(t *testing.T) {
	ctx := context.Background()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// 1. Create repo
	path1 := "/path/to/project1"
	repo1, err := db.GetOrCreateRepository(ctx, path1)
	if err != nil {
		t.Fatalf("failed to create repo1: %v", err)
	}
	if repo1.ID == 0 || repo1.Path != path1 {
		t.Errorf("unexpected repo1 result: %+v", repo1)
	}

	// 2. Duplicate repo path updates last_used_at and preserves ID
	time.Sleep(10 * time.Millisecond)
	repo1Dup, err := db.GetOrCreateRepository(ctx, path1)
	if err != nil {
		t.Fatalf("failed to get existing repo1: %v", err)
	}
	if repo1Dup.ID != repo1.ID {
		t.Errorf("expected same ID %d, got: %d", repo1.ID, repo1Dup.ID)
	}
	if !repo1Dup.LastUsedAt.After(repo1.LastUsedAt) && !repo1Dup.LastUsedAt.Equal(repo1.LastUsedAt) {
		t.Errorf("expected last_used_at to be updated")
	}

	// 3. Create second repo
	path2 := "/path/to/project2"
	repo2, err := db.GetOrCreateRepository(ctx, path2)
	if err != nil {
		t.Fatalf("failed to create repo2: %v", err)
	}

	// 4. List recent repositories
	repos, err := db.ListRecentRepositories(ctx, 10)
	if err != nil {
		t.Fatalf("failed to list recent repos: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got: %d", len(repos))
	}
	if repos[0].ID != repo2.ID {
		t.Errorf("expected repo2 to be most recent, got: %d", repos[0].ID)
	}

	// 5. Delete repository
	if err := db.DeleteRepository(ctx, repo1.ID); err != nil {
		t.Fatalf("failed to delete repo1: %v", err)
	}
	reposAfter, err := db.ListRecentRepositories(ctx, 10)
	if err != nil || len(reposAfter) != 1 {
		t.Fatalf("expected 1 repo after delete, got: %d (err: %v)", len(reposAfter), err)
	}
}

func TestCommandHistory(t *testing.T) {
	ctx := context.Background()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	repo, err := db.GetOrCreateRepository(ctx, "/my/repo")
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	// 1. Record commands
	rec1, err := db.RecordCommand(ctx, HistoryEntry{
		RepositoryID: &repo.ID,
		Command:      "devgit status",
		Success:      true,
		DurationMS:   25,
	})
	if err != nil {
		t.Fatalf("failed recording command 1: %v", err)
	}
	if rec1.ID == 0 {
		t.Errorf("expected non-zero ID for command 1")
	}

	_, err = db.RecordCommand(ctx, HistoryEntry{
		RepositoryID: &repo.ID,
		Command:      "devgit push --invalid",
		Success:      false,
		DurationMS:   150,
	})
	if err != nil {
		t.Fatalf("failed recording command 2: %v", err)
	}

	// Record a command without repository (e.g. global config)
	_, err = db.RecordCommand(ctx, HistoryEntry{
		RepositoryID: nil,
		Command:      "devgit config list",
		Success:      true,
		DurationMS:   5,
	})
	if err != nil {
		t.Fatalf("failed recording global command: %v", err)
	}

	// 2. List all history
	all, err := db.ListHistory(ctx, HistoryListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("failed listing history: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 history records, got: %d", len(all))
	}

	// 3. Filter by success
	onlySuccess, err := db.ListHistory(ctx, HistoryListOptions{OnlySuccess: true})
	if err != nil || len(onlySuccess) != 2 {
		t.Fatalf("expected 2 successful records, got: %d (err: %v)", len(onlySuccess), err)
	}

	// 4. Filter by failed
	onlyFailed, err := db.ListHistory(ctx, HistoryListOptions{OnlyFailed: true})
	if err != nil || len(onlyFailed) != 1 {
		t.Fatalf("expected 1 failed record, got: %d (err: %v)", len(onlyFailed), err)
	}
	if !strings.Contains(onlyFailed[0].Command, "push") {
		t.Errorf("expected failed command to be push, got: %s", onlyFailed[0].Command)
	}

	// 5. Prune by keepMax
	deleted, err := db.PruneHistory(ctx, 1, 0)
	if err != nil {
		t.Fatalf("failed pruning history: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 records deleted, got: %d", deleted)
	}

	remaining, _ := db.ListHistory(ctx, HistoryListOptions{})
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining record, got: %d", len(remaining))
	}

	// 6. Clear all history
	cleared, err := db.ClearHistory(ctx)
	if err != nil || cleared != 1 {
		t.Fatalf("expected 1 record cleared, got: %d (err: %v)", cleared, err)
	}

	empty, _ := db.ListHistory(ctx, HistoryListOptions{})
	if len(empty) != 0 {
		t.Errorf("expected 0 records after clear, got: %d", len(empty))
	}
}

func TestPreferences(t *testing.T) {
	ctx := context.Background()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// 1. Get non-existent
	val, found, err := db.GetPreference(ctx, "tui.last_view")
	if err != nil || found || val != "" {
		t.Errorf("expected not found for empty preference, got: %q, found: %t, err: %v", val, found, err)
	}

	// 2. Set preference
	err = db.SetPreference(ctx, "tui.last_view", "branches")
	if err != nil {
		t.Fatalf("failed setting preference: %v", err)
	}

	val, found, err = db.GetPreference(ctx, "tui.last_view")
	if err != nil || !found || val != "branches" {
		t.Errorf("expected 'branches', got: %q (found: %t, err: %v)", val, found, err)
	}

	// 3. Update existing preference
	err = db.SetPreference(ctx, "tui.last_view", "log")
	if err != nil {
		t.Fatalf("failed updating preference: %v", err)
	}
	val, _, _ = db.GetPreference(ctx, "tui.last_view")
	if val != "log" {
		t.Errorf("expected 'log', got: %q", val)
	}

	// 4. Set another preference and list
	_ = db.SetPreference(ctx, "editor.wrap", "true")
	all, err := db.ListPreferences(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 preferences, got: %d (err: %v)", len(all), err)
	}
	if all["editor.wrap"] != "true" || all["tui.last_view"] != "log" {
		t.Errorf("unexpected preferences map: %+v", all)
	}

	// 5. Delete preference
	if err := db.DeletePreference(ctx, "editor.wrap"); err != nil {
		t.Fatalf("failed deleting preference: %v", err)
	}
	_, found, _ = db.GetPreference(ctx, "editor.wrap")
	if found {
		t.Errorf("expected editor.wrap to be deleted")
	}
}

func TestDiagnosticsAndSafety(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Seed data
	repo, _ := db.GetOrCreateRepository(ctx, "/path/test")
	_, _ = db.RecordCommand(ctx, HistoryEntry{
		RepositoryID: &repo.ID,
		Command:      "devgit status",
		Success:      true,
		DurationMS:   10,
	})
	_ = db.SetPreference(ctx, "pref.key", "val")

	diag, err := db.Diagnostics(ctx)
	if err != nil {
		t.Fatalf("failed getting diagnostics: %v", err)
	}

	if diag.Path != dbPath {
		t.Errorf("expected path %s, got: %s", dbPath, diag.Path)
	}
	if diag.RepositoryCount != 1 {
		t.Errorf("expected 1 repo, got: %d", diag.RepositoryCount)
	}
	if diag.HistoryCount != 1 {
		t.Errorf("expected 1 history entry, got: %d", diag.HistoryCount)
	}
	if diag.PreferenceCount != 1 {
		t.Errorf("expected 1 preference, got: %d", diag.PreferenceCount)
	}
	if diag.FileSizeBytes <= 0 {
		t.Errorf("expected non-zero file size, got: %d", diag.FileSizeBytes)
	}
}
