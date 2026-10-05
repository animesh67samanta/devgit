package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitCmdSuccessInteractive(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file1 := filepath.Join(dir, "UserController.php")
	file2 := filepath.Join(dir, "UserService.php")
	_ = os.WriteFile(file1, []byte("UserController"), 0644)
	_ = os.WriteFile(file2, []byte("UserService"), 0644)

	// User input:
	// 1) Select files "1,2"
	// 2) Commit message "Add user validation"
	// 3) Confirmation "y"
	input := strings.NewReader("1,2\nAdd user validation\ny\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error running interactive commit, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Selected:") {
		t.Errorf("expected 'Selected:' in output, got: %s", out)
	}
	if !strings.Contains(out, "✓ UserController.php") || !strings.Contains(out, "✓ UserService.php") {
		t.Errorf("expected selected files with checkmarks, got: %s", out)
	}
	if !strings.Contains(out, "Commit Preview") {
		t.Errorf("expected 'Commit Preview', got: %s", out)
	}
	if !strings.Contains(out, "✓ Files staged") {
		t.Errorf("expected '✓ Files staged', got: %s", out)
	}
	if !strings.Contains(out, "✓ Commit created") {
		t.Errorf("expected '✓ Commit created', got: %s", out)
	}
	if !strings.Contains(out, "Add user validation") {
		t.Errorf("expected commit message in output, got: %s", out)
	}
}

func TestCommitCmdCancellationAtConfirmation(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "temp.txt")
	_ = os.WriteFile(file, []byte("temp"), 0644)

	// User input: select 1, message, then reject confirmation 'n'
	input := strings.NewReader("1\nWill cancel\nn\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error on cancellation, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.', got: %s", out)
	}
	if strings.Contains(out, "✓ Commit created") {
		t.Errorf("did not expect commit created on cancel, got: %s", out)
	}
}

func TestCommitCmdCancellationAtFileSelection(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "temp.txt")
	_ = os.WriteFile(file, []byte("temp"), 0644)

	// User input: 'c' to cancel at selection
	input := strings.NewReader("c\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error on cancel, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.', got: %s", out)
	}
}

func TestCommitCmdEmptyMessage(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	file := filepath.Join(dir, "data.txt")
	_ = os.WriteFile(file, []byte("data"), 0644)

	// User selects file 1, then enters blank message
	input := strings.NewReader("1\n   \n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err == nil {
		t.Fatal("expected error on empty commit message, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Commit message cannot be empty.") {
		t.Errorf("expected empty message error in stderr, got: %s", errOut)
	}
}

func TestCommitCmdNoChanges(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	// Initial commit so working tree is clean
	f := filepath.Join(dir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	execGit(t, dir, "add", "init.txt")
	execGit(t, dir, "commit", "-m", "init")

	input := strings.NewReader("")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err == nil {
		t.Fatal("expected error when nothing to commit, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Nothing to commit.") || !strings.Contains(errOut, "The working tree is clean.") {
		t.Errorf("expected 'Nothing to commit' message, got: %s", errOut)
	}
}

func TestCommitCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	input := strings.NewReader("")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err == nil {
		t.Fatal("expected error outside repo, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errOut)
	}
}

func TestCommitCmdWithStagedFiles(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f1 := filepath.Join(dir, "README.md")
	f2 := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(f1, []byte("readme"), 0644)
	_ = os.WriteFile(f2, []byte("test"), 0644)

	// Stage only f1
	execGit(t, dir, "add", "README.md")

	// User input: press Enter at file selection to commit staged files, message, confirm 'y'
	input := strings.NewReader("\nCommit staged readme\ny\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, input, outBuf, errBuf, dir, CommitCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Staged Changes:") {
		t.Errorf("expected 'Staged Changes:' section, got: %s", out)
	}
	if !strings.Contains(out, "Unstaged Changes:") {
		t.Errorf("expected 'Unstaged Changes:' section, got: %s", out)
	}
	if !strings.Contains(out, "✓ README.md") {
		t.Errorf("expected README.md selected, got: %s", out)
	}
	if strings.Contains(out, "✓ test.txt") {
		t.Errorf("did not expect test.txt to be selected, got: %s", out)
	}
}

func TestCommitCmdFlagsDirect(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "flag_test.go")
	_ = os.WriteFile(f, []byte("package main"), 0644)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunCommit(ctx, nil, outBuf, errBuf, dir, CommitCmdFlags{
		Message: "Add flag_test.go via flags",
		All:     true,
		Yes:     true,
	})
	if err != nil {
		t.Fatalf("expected no error with flags, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "✓ Commit created") {
		t.Errorf("expected commit created, got: %s", out)
	}
	if !strings.Contains(out, "Add flag_test.go via flags") {
		t.Errorf("expected commit subject in output, got: %s", out)
	}
}
