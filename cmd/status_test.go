package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(dir); err == nil {
		dir = realDir
	}

	execGit(t, dir, "init", "-b", "main")
	execGit(t, dir, "config", "user.name", "Test User")
	execGit(t, dir, "config", "user.email", "test@example.com")
	execGit(t, dir, "config", "core.autocrlf", "false")

	return dir
}

func execGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\noutput: %s", strings.Join(args, " "), err, string(out))
	}
}

func TestStatusCmdInsideRepoWithChanges(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	// Create an initial commit
	initialFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Initial\n"), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	execGit(t, dir, "add", "README.md")
	execGit(t, dir, "commit", "-m", "initial commit")

	// Create modified and untracked files
	if err := os.WriteFile(initialFile, []byte("# Initial\nModified content\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	untrackedFile := filepath.Join(dir, "new_feature.go")
	if err := os.WriteFile(untrackedFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write untracked file: %v", err)
	}

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStatus(ctx, outBuf, errBuf, dir)
	if err != nil {
		t.Fatalf("expected no error running status, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Repository: "+dir) && !strings.Contains(filepath.ToSlash(out), "Repository: "+filepath.ToSlash(dir)) {
		t.Errorf("expected repository root %q, got: %s", dir, out)
	}
	if !strings.Contains(out, "Branch: main") {
		t.Errorf("expected branch main, got: %s", out)
	}
	if !strings.Contains(out, "Changes:") {
		t.Errorf("expected Changes header, got: %s", out)
	}
	if !strings.Contains(out, " M README.md") {
		t.Errorf("expected ' M README.md', got: %s", out)
	}
	if !strings.Contains(out, "?? new_feature.go") {
		t.Errorf("expected '?? new_feature.go', got: %s", out)
	}
}

func TestStatusCmdCleanRepo(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "clean commit")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStatus(ctx, outBuf, errBuf, dir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "No changes (working tree clean)") {
		t.Errorf("expected clean working tree message, got: %s", out)
	}
}

func TestStatusCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStatus(ctx, outBuf, errBuf, dir)
	if err == nil {
		t.Fatal("expected error running status outside git repo, got nil")
	}

	errOutput := errBuf.String()
	expectedMsg := "❌ Not a Git repository.\n\nRun devgit inside a Git repository."
	if !strings.Contains(errOutput, expectedMsg) {
		t.Errorf("expected %q in stderr, got: %q", expectedMsg, errOutput)
	}
}

func TestStatusSubcommandWiring(t *testing.T) {
	rootCmd := NewRootCmd()
	cmd, _, err := rootCmd.Find([]string{"status"})
	if err != nil {
		t.Fatalf("failed to find status command: %v", err)
	}
	if cmd == nil || cmd.Name() != "status" {
		t.Fatal("expected status command to be registered")
	}
}
