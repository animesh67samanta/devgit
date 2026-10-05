package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffCmdWithChanges(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file1 := filepath.Join(dir, "UserController.php")
	file2 := filepath.Join(dir, "UserService.php")
	if err := os.WriteFile(file1, []byte("class UserController {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("class UserService {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", ".")
	execGit(t, dir, "commit", "-m", "init")

	// Modify files
	if err := os.WriteFile(file1, []byte("class UserController { /* edited */ }\n"), 0644); err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("class UserService { /* edited */ }\n"), 0644); err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDiff(ctx, outBuf, errBuf, dir, DiffCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error running diff, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Changes") {
		t.Errorf("expected 'Changes' header, got: %s", out)
	}
	if !strings.Contains(out, "Modified:") {
		t.Errorf("expected 'Modified:' section, got: %s", out)
	}
	if !strings.Contains(out, "UserController.php") || !strings.Contains(out, "UserService.php") {
		t.Errorf("expected modified files listed, got: %s", out)
	}
}

func TestDiffCmdStaged(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	baseFile := filepath.Join(dir, "base.txt")
	if err := os.WriteFile(baseFile, []byte("base\n"), 0644); err != nil {
		t.Fatalf("failed to write base file: %v", err)
	}
	execGit(t, dir, "add", "base.txt")
	execGit(t, dir, "commit", "-m", "init")

	// Stage an added file
	newFile := filepath.Join(dir, "UserRequest.php")
	if err := os.WriteFile(newFile, []byte("class UserRequest {}\n"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}
	execGit(t, dir, "add", "UserRequest.php")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDiff(ctx, outBuf, errBuf, dir, DiffCmdFlags{Staged: true})
	if err != nil {
		t.Fatalf("expected no error running staged diff, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Staged Changes") {
		t.Errorf("expected 'Staged Changes' header, got: %s", out)
	}
	if !strings.Contains(out, "Added:") {
		t.Errorf("expected 'Added:' section, got: %s", out)
	}
	if !strings.Contains(out, "UserRequest.php") {
		t.Errorf("expected 'UserRequest.php' listed, got: %s", out)
	}
}

func TestDiffCmdClean(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "clean.txt")
	if err := os.WriteFile(file, []byte("clean\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", "clean.txt")
	execGit(t, dir, "commit", "-m", "init")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDiff(ctx, outBuf, errBuf, dir, DiffCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := strings.TrimSpace(outBuf.String())
	if out != "No changes" {
		t.Errorf("expected 'No changes', got: %q", out)
	}
}

func TestDiffCmdRaw(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("line1\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	if err := os.WriteFile(file, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatalf("failed to edit file: %v", err)
	}

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDiff(ctx, outBuf, errBuf, dir, DiffCmdFlags{Raw: true})
	if err != nil {
		t.Fatalf("expected no error from raw diff, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "+line2") {
		t.Errorf("expected unified diff patch with '+line2', got: %s", out)
	}
}

func TestDiffCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunDiff(ctx, outBuf, errBuf, dir, DiffCmdFlags{})
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, notRepoMessage) {
		t.Errorf("expected %q, got: %q", notRepoMessage, errOutput)
	}
}
