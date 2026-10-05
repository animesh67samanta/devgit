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

func TestBranchListCmd(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	// Initial commit on main and push
	f := filepath.Join(localDir, "file.txt")
	_ = os.WriteFile(f, []byte("hello"), 0644)
	execGit(t, localDir, "add", "file.txt")
	execGit(t, localDir, "commit", "-m", "init")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Create another local branch
	execGit(t, localDir, "branch", "feature-x")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunBranchList(ctx, outBuf, errBuf, localDir)
	if err != nil {
		t.Fatalf("expected no error from RunBranchList, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Branches") {
		t.Errorf("expected 'Branches' header, got: %s", out)
	}
	if !strings.Contains(out, "* main") {
		t.Errorf("expected current branch '* main', got: %s", out)
	}
	if !strings.Contains(out, "(origin/main)") {
		t.Errorf("expected upstream tracking '(origin/main)', got: %s", out)
	}
	if !strings.Contains(out, "feature-x") {
		t.Errorf("expected 'feature-x' in listing, got: %s", out)
	}
	if !strings.Contains(out, "Remote:") {
		t.Errorf("expected 'Remote:' section, got: %s", out)
	}
}

func TestBranchCreateCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	execGit(t, dir, "add", "init.txt")
	execGit(t, dir, "commit", "-m", "init")

	// 1. Create branch without switch
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunBranchCreate(ctx, strings.NewReader(""), outBuf, errBuf, dir, "feat-one", false)
	if err != nil {
		t.Fatalf("expected create branch success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'feat-one' created.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}

	// Verify we are still on main
	outCurrent, errCurrent := execGitOutput(t, dir, "branch", "--show-current")
	if errCurrent != nil || strings.TrimSpace(outCurrent) != "main" {
		t.Errorf("expected to remain on main, current is: %s", outCurrent)
	}

	// 2. Create branch with switch (-s)
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchCreate(ctx, strings.NewReader(""), outBuf, errBuf, dir, "feat-two", true)
	if err != nil {
		t.Fatalf("expected create branch with switch success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Switched to branch 'feat-two'.") {
		t.Errorf("expected switched confirmation, got: %s", outBuf.String())
	}

	outCurrent, errCurrent = execGitOutput(t, dir, "branch", "--show-current")
	if errCurrent != nil || strings.TrimSpace(outCurrent) != "feat-two" {
		t.Errorf("expected to be on feat-two, current is: %s", outCurrent)
	}

	// 3. Duplicate branch error
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchCreate(ctx, strings.NewReader(""), outBuf, errBuf, dir, "feat-two", false)
	if err == nil {
		t.Fatal("expected error creating duplicate branch, got nil")
	}
	if !strings.Contains(errBuf.String(), "already exists") {
		t.Errorf("expected duplicate error message, got: %s", errBuf.String())
	}

	// 4. Prompt for branch name if empty
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchCreate(ctx, strings.NewReader("feat-three\n"), outBuf, errBuf, dir, "", false)
	if err != nil {
		t.Fatalf("expected prompt create to succeed, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'feat-three' created.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestBranchSwitchCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	execGit(t, dir, "add", "init.txt")
	execGit(t, dir, "commit", "-m", "init")

	execGit(t, dir, "branch", "target-branch")

	// 1. Clean switch
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunBranchSwitch(ctx, strings.NewReader(""), outBuf, errBuf, dir, "target-branch", false)
	if err != nil {
		t.Fatalf("expected clean switch success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Switched to branch 'target-branch'.") {
		t.Errorf("expected switched confirmation, got: %s", outBuf.String())
	}

	// 2. Switch with dirty working tree, prompt user says 'n' -> cancelled
	_ = os.WriteFile(f, []byte("dirty content"), 0644)

	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchSwitch(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, "main", false)
	if err != nil {
		t.Fatalf("expected no error on cancel, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected cancellation message, got: %s", outBuf.String())
	}
	// Verify still on target-branch
	outCurrent, _ := execGitOutput(t, dir, "branch", "--show-current")
	if strings.TrimSpace(outCurrent) != "target-branch" {
		t.Errorf("expected to still be on target-branch, current is: %s", outCurrent)
	}

	// 3. Switch with dirty working tree, user says 'y' -> switches
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchSwitch(ctx, strings.NewReader("y\n"), outBuf, errBuf, dir, "main", false)
	if err != nil {
		t.Fatalf("expected switch after confirmation, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Switched to branch 'main'.") {
		t.Errorf("expected switch success, got: %s", outBuf.String())
	}

	// 4. Switch to non-existent branch
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchSwitch(ctx, strings.NewReader(""), outBuf, errBuf, dir, "non-existent", true)
	if err == nil {
		t.Fatal("expected error switching to non-existent branch, got nil")
	}
	if !strings.Contains(errBuf.String(), "not found") {
		t.Errorf("expected branch not found error, got: %s", errBuf.String())
	}

	// 5. Operation in progress check (rebase / merge)
	gitDir := filepath.Join(dir, ".git")
	mergeHeadFile := filepath.Join(gitDir, "MERGE_HEAD")
	_ = os.WriteFile(mergeHeadFile, []byte("dummyhash\n"), 0644)
	defer os.Remove(mergeHeadFile)

	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchSwitch(ctx, strings.NewReader(""), outBuf, errBuf, dir, "target-branch", true)
	if err == nil {
		t.Fatal("expected error when merge is in progress, got nil")
	}
	if !strings.Contains(errBuf.String(), "Merge in progress") {
		t.Errorf("expected merge in progress warning, got: %s", errBuf.String())
	}
}

func TestBranchDeleteCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	execGit(t, dir, "add", "init.txt")
	execGit(t, dir, "commit", "-m", "init")

	// 1. Guard against deleting current checked-out branch
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunBranchDelete(ctx, strings.NewReader(""), outBuf, errBuf, dir, "main", false, false)
	if err == nil {
		t.Fatal("expected error trying to delete current branch, got nil")
	}
	if !strings.Contains(errBuf.String(), "Cannot delete the currently checked-out branch 'main'.") {
		t.Errorf("expected cannot delete current branch error, got: %s", errBuf.String())
	}

	// 2. Safe delete of merged branch
	execGit(t, dir, "branch", "merged-feat")
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchDelete(ctx, strings.NewReader(""), outBuf, errBuf, dir, "merged-feat", false, false)
	if err != nil {
		t.Fatalf("expected safe delete of merged branch to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'merged-feat' deleted.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}

	// 3. Unmerged branch deletion safety
	execGit(t, dir, "checkout", "-b", "unmerged-feat")
	f2 := filepath.Join(dir, "unmerged.txt")
	_ = os.WriteFile(f2, []byte("unmerged"), 0644)
	execGit(t, dir, "add", "unmerged.txt")
	execGit(t, dir, "commit", "-m", "unmerged work")
	execGit(t, dir, "checkout", "main")

	// 3a. User cancels unmerged deletion prompt ('n')
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchDelete(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, "unmerged-feat", false, false)
	if err != nil {
		t.Fatalf("expected cancel on unmerged prompt, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected operation cancelled message, got: %s", outBuf.String())
	}

	// 3b. User confirms unmerged deletion prompt ('y') -> force deletes
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchDelete(ctx, strings.NewReader("y\n"), outBuf, errBuf, dir, "unmerged-feat", false, false)
	if err != nil {
		t.Fatalf("expected force delete after confirmation, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'unmerged-feat' force-deleted.") {
		t.Errorf("expected force-deleted message, got: %s", outBuf.String())
	}

	// 4. Force delete flag directly
	execGit(t, dir, "checkout", "-b", "force-feat")
	f3 := filepath.Join(dir, "force.txt")
	_ = os.WriteFile(f3, []byte("force"), 0644)
	execGit(t, dir, "add", "force.txt")
	execGit(t, dir, "commit", "-m", "force work")
	execGit(t, dir, "checkout", "main")

	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchDelete(ctx, strings.NewReader(""), outBuf, errBuf, dir, "force-feat", true, false)
	if err != nil {
		t.Fatalf("expected force delete flag to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'force-feat' deleted.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestBranchRenameCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	execGit(t, dir, "add", "init.txt")
	execGit(t, dir, "commit", "-m", "init")

	// 1. Rename current branch
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunBranchRename(ctx, strings.NewReader(""), outBuf, errBuf, dir, "", "trunk")
	if err != nil {
		t.Fatalf("expected rename current branch success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Current branch renamed to 'trunk'.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}

	// 2. Rename specific branch
	execGit(t, dir, "branch", "old-name")
	outBuf.Reset()
	errBuf.Reset()
	err = RunBranchRename(ctx, strings.NewReader(""), outBuf, errBuf, dir, "old-name", "renamed-branch")
	if err != nil {
		t.Fatalf("expected rename specific branch success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Branch 'old-name' renamed to 'renamed-branch'.") {
		t.Errorf("unexpected output: %s", outBuf.String())
	}
}

func TestBranchCmdOutsideRepo(t *testing.T) {
	emptyDir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunBranchList(ctx, outBuf, errBuf, emptyDir)
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}
	if !strings.Contains(errBuf.String(), notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errBuf.String())
	}
}

// execGitOutput is a helper to run git commands and capture stdout.
func execGitOutput(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	return string(out), err
}
