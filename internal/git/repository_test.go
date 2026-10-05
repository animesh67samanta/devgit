package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) (string, *Client) {
	t.Helper()
	dir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(dir); err == nil {
		dir = realDir
	}

	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")

	client, err := NewClient(dir)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return dir, client
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func TestValidRepository(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	inside, err := client.IsInsideWorkTree(ctx)
	if err != nil {
		t.Fatalf("expected no error from IsInsideWorkTree, got: %v", err)
	}
	if !inside {
		t.Error("expected IsInsideWorkTree to be true")
	}

	root, err := client.Root(ctx)
	if err != nil {
		t.Fatalf("expected no error from Root, got: %v", err)
	}
	if root != dir {
		t.Errorf("expected root %q, got %q", dir, root)
	}
}

func TestNonGitDirectory(t *testing.T) {
	dir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(dir); err == nil {
		dir = realDir
	}

	client, err := NewClient(dir)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()

	inside, err := client.IsInsideWorkTree(ctx)
	if !errors.Is(err, ErrNotRepository) {
		t.Errorf("expected ErrNotRepository, got error: %v", err)
	}
	if inside {
		t.Error("expected IsInsideWorkTree to be false for non-git directory")
	}

	_, err = client.Root(ctx)
	if !errors.Is(err, ErrNotRepository) {
		t.Errorf("expected ErrNotRepository from Root, got: %v", err)
	}

	_, err = client.CurrentBranch(ctx)
	if !errors.Is(err, ErrNotRepository) {
		t.Errorf("expected ErrNotRepository from CurrentBranch, got: %v", err)
	}

	_, err = client.Status(ctx)
	if !errors.Is(err, ErrNotRepository) {
		t.Errorf("expected ErrNotRepository from Status, got: %v", err)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	// Initial branch before any commits
	branch, err := client.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("expected no error getting initial branch, got: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}

	// Make an initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")

	// Branch after commit
	branch, err = client.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("expected no error getting branch after commit, got: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}

	// Switch to a new branch
	runGit(t, dir, "checkout", "-b", "feature/awesome-feature")
	branch, err = client.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("expected no error getting branch on feature branch, got: %v", err)
	}
	if branch != "feature/awesome-feature" {
		t.Errorf("expected branch 'feature/awesome-feature', got %q", branch)
	}
}
