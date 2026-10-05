package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitOneFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "app.go")
	if err := os.WriteFile(file, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"app.go"},
		Message: "Add app.go",
	})
	if err != nil {
		t.Fatalf("expected no error from Commit, got: %v", err)
	}

	if result.Subject != "Add app.go" {
		t.Errorf("expected subject 'Add app.go', got %q", result.Subject)
	}
	if len(result.ShortHash) == 0 {
		t.Error("expected non-empty ShortHash")
	}
	if len(result.Hash) == 0 {
		t.Error("expected non-empty Hash")
	}
	if len(result.Files) != 1 || result.Files[0] != "app.go" {
		t.Errorf("unexpected files in result: %v", result.Files)
	}
}

func TestCommitMultipleFiles(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f1 := filepath.Join(dir, "file1.txt")
	f2 := filepath.Join(dir, "file2.txt")
	_ = os.WriteFile(f1, []byte("1"), 0644)
	_ = os.WriteFile(f2, []byte("2"), 0644)

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"file1.txt", "file2.txt"},
		Message: "Add two files",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(result.Files) != 2 {
		t.Errorf("expected 2 files, got: %d", len(result.Files))
	}
}

func TestCommitModifiedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "main.go")
	_ = os.WriteFile(f, []byte("package main\n"), 0644)
	_, _ = client.Commit(ctx, CommitOptions{Files: []string{"main.go"}, Message: "initial"})

	_ = os.WriteFile(f, []byte("package main\n// modified\n"), 0644)

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"main.go"},
		Message: "Update main.go",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Subject != "Update main.go" {
		t.Errorf("expected 'Update main.go', got %q", result.Subject)
	}
}

func TestCommitNewFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "newfile.txt")
	_ = os.WriteFile(f, []byte("new"), 0644)

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"newfile.txt"},
		Message: "Add newfile.txt",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Subject != "Add newfile.txt" {
		t.Errorf("expected subject 'Add newfile.txt', got %q", result.Subject)
	}
}

func TestCommitDeletedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "obsolete.txt")
	_ = os.WriteFile(f, []byte("obsolete"), 0644)
	_, _ = client.Commit(ctx, CommitOptions{Files: []string{"obsolete.txt"}, Message: "initial"})

	_ = os.Remove(f)

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"obsolete.txt"},
		Message: "Remove obsolete.txt",
	})
	if err != nil {
		t.Fatalf("expected no error committing deleted file, got: %v", err)
	}
	if result.Subject != "Remove obsolete.txt" {
		t.Errorf("expected 'Remove obsolete.txt', got %q", result.Subject)
	}
}

func TestCommitRenamedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	_ = os.WriteFile(oldPath, []byte("content"), 0644)
	_, _ = client.Commit(ctx, CommitOptions{Files: []string{"old.txt"}, Message: "initial"})

	_ = os.Rename(oldPath, newPath)

	result, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"old.txt", "new.txt"},
		Message: "Rename old.txt to new.txt",
	})
	if err != nil {
		t.Fatalf("expected no error committing rename, got: %v", err)
	}
	if result.Subject != "Rename old.txt to new.txt" {
		t.Errorf("expected subject 'Rename old.txt to new.txt', got %q", result.Subject)
	}
}

func TestCommitEmptyMessage(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	// Empty string
	_, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"any.txt"},
		Message: "",
	})
	if !errors.Is(err, ErrEmptyCommitMessage) {
		t.Errorf("expected ErrEmptyCommitMessage, got: %v", err)
	}

	// Whitespace only
	_, err = client.Commit(ctx, CommitOptions{
		Files:   []string{"any.txt"},
		Message: "   \t\n  ",
	})
	if !errors.Is(err, ErrEmptyCommitMessage) {
		t.Errorf("expected ErrEmptyCommitMessage for whitespace message, got: %v", err)
	}
}

func TestCommitNoChanges(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	// Commit with no files and no staged changes
	_, err := client.Commit(ctx, CommitOptions{
		Message: "Nothing changed",
	})
	if err == nil {
		t.Fatal("expected error when committing with nothing to commit, got nil")
	}
}

func TestCommitResultHash(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(f, []byte("hash test"), 0644)

	res, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"test.txt"},
		Message: "Test hash verification",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Hash) != 40 && len(res.Hash) != 64 { // SHA-1 is 40, SHA-256 is 64
		t.Errorf("unexpected hash length %d: %s", len(res.Hash), res.Hash)
	}
	if len(res.ShortHash) < 7 {
		t.Errorf("short hash too short: %s", res.ShortHash)
	}
	if !strings.HasPrefix(res.Hash, res.ShortHash) {
		t.Errorf("expected full hash %s to start with short hash %s", res.Hash, res.ShortHash)
	}
}

func TestCommitFailure(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	// Staging a non-existent file should fail
	_, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"non_existent_file_12345.xyz"},
		Message: "Will fail",
	})
	if err == nil {
		t.Fatal("expected error when staging non-existent file, got nil")
	}
}

func TestCommitStagedChanges(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "staged.txt")
	_ = os.WriteFile(f, []byte("pre-staged content"), 0644)
	runGit(t, dir, "add", "staged.txt")

	// Commit without passing files explicitly (commits what's staged)
	res, err := client.Commit(ctx, CommitOptions{
		Message: "Commit pre-staged file",
	})
	if err != nil {
		t.Fatalf("expected no error committing staged file: %v", err)
	}
	if res.Subject != "Commit pre-staged file" {
		t.Errorf("expected 'Commit pre-staged file', got %q", res.Subject)
	}
}

func TestCommitMixedStagedAndUnstagedChanges(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f1 := filepath.Join(dir, "staged_file.txt")
	f2 := filepath.Join(dir, "unstaged_file.txt")
	_ = os.WriteFile(f1, []byte("staged"), 0644)
	_ = os.WriteFile(f2, []byte("unstaged"), 0644)

	// Stage f1 manually
	runGit(t, dir, "add", "staged_file.txt")

	// Now commit selecting f1 and f2
	res, err := client.Commit(ctx, CommitOptions{
		Files:   []string{"staged_file.txt", "unstaged_file.txt"},
		Message: "Commit both files",
	})
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if res.Subject != "Commit both files" {
		t.Errorf("unexpected subject: %q", res.Subject)
	}

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("status error: %v", err)
	}
	if !status.IsClean() {
		t.Errorf("expected repository to be clean after committing both files, got %d files", len(status.Files))
	}
}

func TestCommitFilenameWithSpaces(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	filename := "file with spaces and (parentheses).txt"
	f := filepath.Join(dir, filename)
	_ = os.WriteFile(f, []byte("special filename"), 0644)

	res, err := client.Commit(ctx, CommitOptions{
		Files:   []string{filename},
		Message: "Add file with spaces",
	})
	if err != nil {
		t.Fatalf("expected no error committing file with spaces: %v", err)
	}
	if res.Subject != "Add file with spaces" {
		t.Errorf("unexpected subject: %q", res.Subject)
	}
}

func TestCommitFilenameBeginningWithDash(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	filename := "-dashed-file.txt"
	f := filepath.Join(dir, filename)
	_ = os.WriteFile(f, []byte("leading dash"), 0644)

	res, err := client.Commit(ctx, CommitOptions{
		Files:   []string{filename},
		Message: "Add leading dash file",
	})
	if err != nil {
		t.Fatalf("expected no error committing file beginning with dash: %v", err)
	}
	if res.Subject != "Add leading dash file" {
		t.Errorf("unexpected subject: %q", res.Subject)
	}
}

func TestCommitNonGitDirectory(t *testing.T) {
	dir := t.TempDir()
	client, err := NewClient(dir)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()

	_, err = client.Commit(ctx, CommitOptions{
		Files:   []string{"app.go"},
		Message: "test",
	})
	if !errors.Is(err, ErrNotRepository) {
		t.Errorf("expected ErrNotRepository, got: %v", err)
	}
}
