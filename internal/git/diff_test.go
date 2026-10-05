package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffModifiedFiles(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	// Commit initial file
	file1 := filepath.Join(dir, "UserController.php")
	file2 := filepath.Join(dir, "UserService.php")
	if err := os.WriteFile(file1, []byte("class UserController {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("class UserService {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")

	// Modify both files without staging
	if err := os.WriteFile(file1, []byte("class UserController { /* modified */ }\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("class UserService { /* modified */ }\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	summary, err := client.DiffSummary(ctx, false)
	if err != nil {
		t.Fatalf("expected no error from DiffSummary, got: %v", err)
	}

	if len(summary.Modified) != 2 {
		t.Fatalf("expected 2 modified files, got: %d", len(summary.Modified))
	}
	if summary.Modified[0] != "UserController.php" || summary.Modified[1] != "UserService.php" {
		t.Errorf("unexpected modified files: %v", summary.Modified)
	}

	raw, err := client.RawDiff(ctx, false)
	if err != nil {
		t.Fatalf("expected no error from RawDiff, got: %v", err)
	}
	if !strings.Contains(raw, "UserController.php") || !strings.Contains(raw, "UserService.php") {
		t.Errorf("expected raw diff to mention changed files, got: %s", raw)
	}
}

func TestDiffStagedFiles(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	// Initial commit
	baseFile := filepath.Join(dir, "base.txt")
	if err := os.WriteFile(baseFile, []byte("base content\n"), 0644); err != nil {
		t.Fatalf("failed to write base file: %v", err)
	}
	runGit(t, dir, "add", "base.txt")
	runGit(t, dir, "commit", "-m", "base commit")

	// Stage a newly added file
	newFile := filepath.Join(dir, "UserRequest.php")
	if err := os.WriteFile(newFile, []byte("class UserRequest {}\n"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}
	runGit(t, dir, "add", "UserRequest.php")

	// Unstaged diff should NOT show staged file
	unstagedSummary, err := client.DiffSummary(ctx, false)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if !unstagedSummary.IsEmpty() {
		t.Errorf("expected unstaged diff to be empty, got: %+v", unstagedSummary)
	}

	// Staged diff SHOULD show the added file
	stagedSummary, err := client.DiffSummary(ctx, true)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}
	if len(stagedSummary.Added) != 1 {
		t.Fatalf("expected 1 added file in staged summary, got: %d", len(stagedSummary.Added))
	}
	if stagedSummary.Added[0] != "UserRequest.php" {
		t.Errorf("expected added file 'UserRequest.php', got: %q", stagedSummary.Added[0])
	}
}

func TestDiffDeletedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	delFile := filepath.Join(dir, "obsolete.txt")
	if err := os.WriteFile(delFile, []byte("obsolete\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "obsolete.txt")
	runGit(t, dir, "commit", "-m", "commit obsolete file")

	// Delete without staging
	if err := os.Remove(delFile); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	summary, err := client.DiffSummary(ctx, false)
	if err != nil {
		t.Fatalf("expected no error: %v", err)
	}

	if len(summary.Deleted) != 1 {
		t.Fatalf("expected 1 deleted file, got: %d", len(summary.Deleted))
	}
	if summary.Deleted[0] != "obsolete.txt" {
		t.Errorf("expected deleted file 'obsolete.txt', got: %q", summary.Deleted[0])
	}
}

func TestParseDiffSummary(t *testing.T) {
	raw := `M	UserController.php
M	UserService.php
A	UserRequest.php
D	OldService.php
R100	old.go	new.go
`
	summary := ParseDiffSummary(raw)
	if len(summary.Modified) != 2 {
		t.Errorf("expected 2 modified, got: %d", len(summary.Modified))
	}
	if len(summary.Added) != 1 {
		t.Errorf("expected 1 added, got: %d", len(summary.Added))
	}
	if len(summary.Deleted) != 1 {
		t.Errorf("expected 1 deleted, got: %d", len(summary.Deleted))
	}
	if len(summary.Renamed) != 1 {
		t.Errorf("expected 1 renamed, got: %d", len(summary.Renamed))
	}
	if summary.Renamed[0] != "old.go -> new.go" {
		t.Errorf("expected rename 'old.go -> new.go', got: %q", summary.Renamed[0])
	}
}
