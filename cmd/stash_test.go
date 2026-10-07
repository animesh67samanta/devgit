package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStashListCmdEmpty(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStashList(ctx, outBuf, errBuf, dir)
	if err != nil {
		t.Fatalf("expected no error from RunStashList, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "✓ No stashes found.") {
		t.Errorf("expected '✓ No stashes found.', got: %s", out)
	}
}

func TestStashListCmdWithEntries(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("line 1\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("stash 1\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "first stash")

	_ = os.WriteFile(f, []byte("stash 2\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "second stash")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStashList(ctx, outBuf, errBuf, dir)
	if err != nil {
		t.Fatalf("expected no error from RunStashList, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Stashes") {
		t.Errorf("expected 'Stashes' header, got: %s", out)
	}
	if !strings.Contains(out, "stash@{0}") || !strings.Contains(out, "second stash") {
		t.Errorf("expected stash@{0} second stash, got: %s", out)
	}
	if !strings.Contains(out, "stash@{1}") || !strings.Contains(out, "first stash") {
		t.Errorf("expected stash@{1} first stash, got: %s", out)
	}
	if !strings.Contains(out, "2 stashes") {
		t.Errorf("expected '2 stashes', got: %s", out)
	}
}

func TestStashSaveCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	// 1. Clean working tree
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunStashSave(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashSaveCmdFlags{})
	if err != nil {
		t.Fatalf("expected clean tree to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Working tree is clean.") {
		t.Errorf("expected clean tree message, got: %s", outBuf.String())
	}

	// 2. Untracked file only, without -u
	untracked := filepath.Join(dir, "untracked.txt")
	_ = os.WriteFile(untracked, []byte("untracked\n"), 0644)

	outBuf.Reset()
	errBuf.Reset()
	err = RunStashSave(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashSaveCmdFlags{IncludeUntracked: false})
	if err != nil {
		t.Fatalf("expected untracked warning to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "⚠ Untracked files are present.") {
		t.Errorf("expected untracked warning, got: %s", outBuf.String())
	}

	// 3. User cancels prompt ('n')
	_ = os.WriteFile(f, []byte("modified\n"), 0644)

	outBuf.Reset()
	errBuf.Reset()
	err = RunStashSave(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, StashSaveCmdFlags{})
	if err != nil {
		t.Fatalf("expected cancel to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected operation cancelled message, got: %s", outBuf.String())
	}

	// Verify working tree still modified
	content, _ := os.ReadFile(f)
	if strings.TrimRight(string(content), "\r\n") != "modified" {
		t.Errorf("expected file to remain modified, got: %s", string(content))
	}

	// 4. User confirms prompt ('y')
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashSave(ctx, strings.NewReader("y\n"), outBuf, errBuf, dir, StashSaveCmdFlags{Message: "saved work"})
	if err != nil {
		t.Fatalf("expected save with prompt to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Changes stashed successfully.") {
		t.Errorf("expected success message, got: %s", outBuf.String())
	}

	// 5. Save with -y flag and -u flag
	_ = os.WriteFile(untracked, []byte("untracked 2\n"), 0644)
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashSave(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashSaveCmdFlags{
		IncludeUntracked: true,
		Yes:              true,
		Message:          "untracked stash",
	})
	if err != nil {
		t.Fatalf("expected save with -y and -u to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Changes stashed successfully.") {
		t.Errorf("expected success message, got: %s", outBuf.String())
	}
}

func TestStashPopCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("stashed work\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "work to pop")

	// 1. User cancels pop prompt ('n')
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunStashPop(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, StashActionCmdFlags{}, "0")
	if err != nil {
		t.Fatalf("expected cancel to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected operation cancelled message, got: %s", outBuf.String())
	}

	// 2. User confirms pop prompt ('y')
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashPop(ctx, strings.NewReader("y\n"), outBuf, errBuf, dir, StashActionCmdFlags{}, "0")
	if err != nil {
		t.Fatalf("expected pop to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Stash applied and removed.") {
		t.Errorf("expected stash applied and removed message, got: %s", outBuf.String())
	}

	// Verify content restored and stash removed
	content, _ := os.ReadFile(f)
	if strings.TrimRight(string(content), "\r\n") != "stashed work" {
		t.Errorf("expected restored content, got: %s", string(content))
	}

	// 3. Pop non-existent stash index
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashPop(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashActionCmdFlags{Yes: true}, "99")
	if err == nil {
		t.Fatal("expected error popping nonexistent stash, got nil")
	}
	if !strings.Contains(errBuf.String(), "No stash exists at index 99.") {
		t.Errorf("expected no stash at index 99 message, got: %s", errBuf.String())
	}

	// 4. Pop invalid index
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashPop(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashActionCmdFlags{Yes: true}, "invalid")
	if err == nil {
		t.Fatal("expected error with invalid index, got nil")
	}
	if !strings.Contains(errBuf.String(), "Invalid stash index.") {
		t.Errorf("expected invalid index message, got: %s", errBuf.String())
	}

	// 5. Pop conflict handling
	_ = os.WriteFile(f, []byte("stash version\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "conflict test")
	_ = os.WriteFile(f, []byte("head version\n"), 0644)
	execGit(t, dir, "commit", "-am", "head commit")

	outBuf.Reset()
	errBuf.Reset()
	err = RunStashPop(ctx, strings.NewReader("y\n"), outBuf, errBuf, dir, StashActionCmdFlags{}, "0")
	if err == nil {
		t.Fatal("expected error on conflict, got nil")
	}
	if !strings.Contains(errBuf.String(), "Stash application encountered conflicts.") {
		t.Errorf("expected conflict message, got: %s", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "The stash has been kept by Git.") {
		t.Errorf("expected stash kept message, got: %s", errBuf.String())
	}
}

func TestStashApplyCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("applied work\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "work to apply")

	// 1. Apply with -y flag
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunStashApply(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashActionCmdFlags{Yes: true}, "0")
	if err != nil {
		t.Fatalf("expected apply to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Stash applied.") {
		t.Errorf("expected stash applied message, got: %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "stash@{0} remains available.") {
		t.Errorf("expected stash preserved message, got: %s", outBuf.String())
	}

	// 2. Dirty tree check: prompt warning
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashApply(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, StashActionCmdFlags{}, "0")
	if err != nil {
		t.Fatalf("expected cancel on dirty tree to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "⚠ Your working tree contains local changes.") {
		t.Errorf("expected dirty tree warning, got: %s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected cancelled message, got: %s", outBuf.String())
	}
}

func TestStashDropCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("to drop\n"), 0644)
	execGit(t, dir, "stash", "push", "-m", "stash to drop")

	// 1. User cancels drop ('n')
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunStashDrop(ctx, strings.NewReader("n\n"), outBuf, errBuf, dir, StashActionCmdFlags{}, "0")
	if err != nil {
		t.Fatalf("expected cancel to return nil, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Operation cancelled.") {
		t.Errorf("expected cancelled message, got: %s", outBuf.String())
	}

	// 2. Drop with -y flag
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashDrop(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashActionCmdFlags{Yes: true}, "0")
	if err != nil {
		t.Fatalf("expected drop with -y to succeed, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "✓ Stash dropped.") {
		t.Errorf("expected stash dropped message, got: %s", outBuf.String())
	}

	// 3. Drop nonexistent index
	outBuf.Reset()
	errBuf.Reset()
	err = RunStashDrop(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashActionCmdFlags{Yes: true}, "0")
	if err == nil {
		t.Fatal("expected error dropping nonexistent stash, got nil")
	}
	if !strings.Contains(errBuf.String(), "No stash exists at index 0.") {
		t.Errorf("expected no stash message, got: %s", errBuf.String())
	}
}

func TestStashCmdOutsideRepo(t *testing.T) {
	emptyDir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStashList(ctx, outBuf, errBuf, emptyDir)
	if err == nil {
		t.Fatal("expected error outside git repo, got nil")
	}
	if !strings.Contains(errBuf.String(), notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errBuf.String())
	}
}

func TestStashCmdOperationInProgress(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	execGit(t, dir, "add", "file.txt")
	execGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("dirty\n"), 0644)

	// Simulate merge in progress
	gitDir := filepath.Join(dir, ".git")
	mergeHeadFile := filepath.Join(gitDir, "MERGE_HEAD")
	_ = os.WriteFile(mergeHeadFile, []byte("dummyhash\n"), 0644)
	defer os.Remove(mergeHeadFile)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunStashSave(ctx, strings.NewReader(""), outBuf, errBuf, dir, StashSaveCmdFlags{Yes: true})
	if err == nil {
		t.Fatal("expected error when merge in progress, got nil")
	}
	if !strings.Contains(errBuf.String(), "Merge in progress") {
		t.Errorf("expected merge in progress error message, got: %s", errBuf.String())
	}
}
