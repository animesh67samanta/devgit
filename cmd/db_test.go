package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/animesh67samanta/devgit/internal/database"
)

func TestHistoryCmdEmpty(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_DB_PATH", filepath.Join(tmpDir, "devgit.db"))

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunHistoryList(ctx, outBuf, errBuf, HistoryFlags{Limit: 20})
	if err != nil {
		t.Fatalf("expected no error on empty history, got: %v", err)
	}

	if !strings.Contains(outBuf.String(), "No command history found") {
		t.Errorf("expected 'No command history found', got: %s", outBuf.String())
	}
}

func TestHistoryCmdListAndClear(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")
	t.Setenv("DEVGIT_DB_PATH", dbPath)

	// Seed records
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	repo, _ := db.GetOrCreateRepository(ctx, "/path/to/myrepo")
	_, _ = db.RecordCommand(ctx, database.HistoryEntry{
		RepositoryID: &repo.ID,
		Command:      "devgit status",
		Success:      true,
		DurationMS:   12,
		ExecutedAt:   time.Now().UTC(),
	})
	_, _ = db.RecordCommand(ctx, database.HistoryEntry{
		RepositoryID: &repo.ID,
		Command:      "devgit commit",
		Success:      true,
		DurationMS:   55,
		ExecutedAt:   time.Now().UTC(),
	})
	db.Close()

	// 1. List history
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err = RunHistoryList(ctx, outBuf, errBuf, HistoryFlags{Limit: 10})
	if err != nil {
		t.Fatalf("failed to run history list: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "DevGit Command History") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, "devgit status") || !strings.Contains(out, "devgit commit") {
		t.Errorf("expected commands in output, got: %s", out)
	}

	// 2. Clear history with keep 1
	outBuf.Reset()
	errBuf.Reset()
	err = RunHistoryClear(ctx, outBuf, errBuf, HistoryFlags{Keep: 1})
	if err != nil {
		t.Fatalf("failed clearing history with keep: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Kept 1 most recent") {
		t.Errorf("expected keep confirmation, got: %s", outBuf.String())
	}

	// 3. Clear all remaining history
	outBuf.Reset()
	errBuf.Reset()
	err = RunHistoryClear(ctx, outBuf, errBuf, HistoryFlags{})
	if err != nil {
		t.Fatalf("failed clearing history: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Command history cleared") {
		t.Errorf("expected cleared confirmation, got: %s", outBuf.String())
	}
}

func TestDBStatusCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")
	t.Setenv("DEVGIT_DB_PATH", dbPath)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDBStatus(ctx, outBuf, errBuf)
	if err != nil {
		t.Fatalf("expected RunDBStatus success, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "DevGit Database Diagnostics") {
		t.Errorf("expected diagnostics header, got: %s", out)
	}
	if !strings.Contains(out, "Status:        Ready") {
		t.Errorf("expected Status: Ready, got: %s", out)
	}
	if !strings.Contains(out, "Schema:        Version 1") {
		t.Errorf("expected Schema Version 1, got: %s", out)
	}
}

func TestDBReposCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")
	t.Setenv("DEVGIT_DB_PATH", dbPath)

	// Seed repo
	db, _ := database.Open(dbPath)
	repoPath := filepath.FromSlash("/path/repo-alpha")
	_, _ = db.GetOrCreateRepository(ctx, repoPath)
	db.Close()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDBRepos(ctx, outBuf, errBuf)
	if err != nil {
		t.Fatalf("expected RunDBRepos success, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Recent Repositories") {
		t.Errorf("expected header, got: %s", out)
	}
	if !strings.Contains(out, repoPath) {
		t.Errorf("expected repo path %q, got: %s", repoPath, out)
	}
}

func TestDBPruneCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")
	t.Setenv("DEVGIT_DB_PATH", dbPath)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDBPrune(ctx, outBuf, errBuf, 10)
	if err != nil {
		t.Fatalf("expected RunDBPrune success, got: %v", err)
	}

	if !strings.Contains(outBuf.String(), "Database maintenance complete") {
		t.Errorf("expected maintenance complete message, got: %s", outBuf.String())
	}
}

func TestDBPreferencesCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devgit.db")
	t.Setenv("DEVGIT_DB_PATH", dbPath)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// 1. Set preference
	err := RunDBPrefSet(ctx, outBuf, errBuf, "tui.theme", "nord")
	if err != nil {
		t.Fatalf("failed setting pref: %v", err)
	}
	if !strings.Contains(outBuf.String(), "tui.theme = nord") {
		t.Errorf("expected confirmation output, got: %s", outBuf.String())
	}

	// 2. Get preference
	outBuf.Reset()
	errBuf.Reset()
	err = RunDBPrefGet(ctx, outBuf, errBuf, "tui.theme")
	if err != nil {
		t.Fatalf("failed getting pref: %v", err)
	}
	if strings.TrimSpace(outBuf.String()) != "nord" {
		t.Errorf("expected 'nord', got: %q", outBuf.String())
	}

	// 3. List preferences
	outBuf.Reset()
	errBuf.Reset()
	err = RunDBPrefList(ctx, outBuf, errBuf)
	if err != nil {
		t.Fatalf("failed listing prefs: %v", err)
	}
	if !strings.Contains(outBuf.String(), "tui.theme = nord") {
		t.Errorf("expected list to contain tui.theme = nord, got: %s", outBuf.String())
	}

	// 4. Get unknown preference
	outBuf.Reset()
	errBuf.Reset()
	err = RunDBPrefGet(ctx, outBuf, errBuf, "nonexistent.key")
	if err == nil {
		t.Fatal("expected error getting nonexistent preference, got nil")
	}
	if !strings.Contains(errBuf.String(), "Preference not found") {
		t.Errorf("expected not found message, got: %s", errBuf.String())
	}
}

func TestGitCommandsUnaffectedWhenDatabaseFails(t *testing.T) {
	// Point to an un-creatable or invalid DB path
	t.Setenv("DEVGIT_DB_PATH", "/dev/null/impossible/path/db.sqlite")

	// Verify RecordCommandExecutionSafe does not panic
	RecordCommandExecutionSafe("status", 10*time.Millisecond, true, "/some/path")

	// Verify root command execution still succeeds with unavailable DB
	dir := setupTestGitRepo(t)
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"status"})

	cwd, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(cwd) }()

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected git command to succeed even if database fails, got: %v\nstderr: %s", err, errBuf.String())
	}
}
