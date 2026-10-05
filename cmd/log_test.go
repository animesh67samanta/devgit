package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogCmdEmptyRepo(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunLog(ctx, outBuf, errBuf, dir, LogCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error running log on empty repo, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Commit History") {
		t.Errorf("expected 'Commit History' header, got: %s", out)
	}
	if !strings.Contains(out, "No commits yet.") {
		t.Errorf("expected 'No commits yet.' message, got: %s", out)
	}
}

func TestLogCmdWithCommits(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(file, []byte("v1\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", "app.txt")
	execGit(t, dir, "commit", "-m", "Add payment validation")

	if err := os.WriteFile(file, []byte("v2\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", "app.txt")
	execGit(t, dir, "commit", "-m", "Update billing calculation")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunLog(ctx, outBuf, errBuf, dir, LogCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error running log, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Commit History") {
		t.Errorf("expected 'Commit History' header, got: %s", out)
	}
	if !strings.Contains(out, "Update billing calculation") {
		t.Errorf("expected commit message 'Update billing calculation', got: %s", out)
	}
	if !strings.Contains(out, "Add payment validation") {
		t.Errorf("expected commit message 'Add payment validation', got: %s", out)
	}
	if !strings.Contains(out, "Test User ·") {
		t.Errorf("expected author and relative timestamp separator, got: %s", out)
	}
}

func TestLogCmdLimit(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "f.txt")
	for i := 1; i <= 5; i++ {
		_ = os.WriteFile(file, []byte(strings.Repeat("a", i)), 0644)
		execGit(t, dir, "add", "f.txt")
		execGit(t, dir, "commit", "-m", "Commit number "+string(rune('0'+i)))
	}

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunLog(ctx, outBuf, errBuf, dir, LogCmdFlags{Limit: 2})
	if err != nil {
		t.Fatalf("expected no error with limit, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Commit number 5") {
		t.Errorf("expected newest commit in output, got: %s", out)
	}
	if strings.Contains(out, "Commit number 1") {
		t.Errorf("did not expect oldest commit with limit 2, got: %s", out)
	}
}

func TestLogCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunLog(ctx, outBuf, errBuf, dir, LogCmdFlags{})
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, notRepoMessage) {
		t.Errorf("expected %q, got: %q", notRepoMessage, errOutput)
	}
}
