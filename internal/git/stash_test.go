package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStashListEmpty(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	stashes, err := client.ListStashes(ctx)
	if err != nil {
		t.Fatalf("expected no error from ListStashes, got: %v", err)
	}
	if len(stashes) != 0 {
		t.Errorf("expected 0 stashes, got: %d", len(stashes))
	}

	has, err := client.HasStashes(ctx)
	if err != nil {
		t.Fatalf("expected no error from HasStashes, got: %v", err)
	}
	if has {
		t.Error("expected hasStashes to be false")
	}
}

func TestStashSaveAndList(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	// 1. Save modified file without message
	_ = os.WriteFile(f, []byte("modification 1\n"), 0644)
	saved1, err := client.SaveStash(ctx, StashSaveOptions{})
	if err != nil {
		t.Fatalf("expected SaveStash success, got: %v", err)
	}
	if saved1.Index != 0 || saved1.Ref != "stash@{0}" {
		t.Errorf("unexpected saved1 entry: %+v", saved1)
	}

	// 2. Save second modification with message
	_ = os.WriteFile(f, []byte("modification 2\n"), 0644)
	saved2, err := client.SaveStash(ctx, StashSaveOptions{Message: "custom message"})
	if err != nil {
		t.Fatalf("expected second SaveStash success, got: %v", err)
	}
	if saved2.Index != 0 {
		t.Errorf("expected index 0 for newest stash, got: %d", saved2.Index)
	}

	// 3. List stashes and verify ordering
	stashes, err := client.ListStashes(ctx)
	if err != nil {
		t.Fatalf("failed to list stashes: %v", err)
	}
	if len(stashes) != 2 {
		t.Fatalf("expected 2 stashes, got: %d", len(stashes))
	}

	if stashes[0].Index != 0 || !strings.Contains(stashes[0].Message, "custom message") {
		t.Errorf("expected stash@{0} to have custom message, got: %+v", stashes[0])
	}
	if stashes[1].Index != 1 {
		t.Errorf("expected stash@{1} index 1, got: %+v", stashes[1])
	}
}

func TestStashSaveCleanTree(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_, err := client.SaveStash(ctx, StashSaveOptions{})
	if !errors.Is(err, ErrNothingToStash) {
		t.Fatalf("expected ErrNothingToStash, got: %v", err)
	}
}

func TestStashSaveUntrackedFiles(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	// Untracked file
	untracked := filepath.Join(dir, "new.txt")
	_ = os.WriteFile(untracked, []byte("untracked content\n"), 0644)

	// Without IncludeUntracked -> ErrNothingToStash
	_, err := client.SaveStash(ctx, StashSaveOptions{IncludeUntracked: false})
	if !errors.Is(err, ErrNothingToStash) {
		t.Fatalf("expected ErrNothingToStash without IncludeUntracked, got: %v", err)
	}

	// With IncludeUntracked -> succeeds
	saved, err := client.SaveStash(ctx, StashSaveOptions{IncludeUntracked: true, Message: "stashed untracked"})
	if err != nil {
		t.Fatalf("expected SaveStash with IncludeUntracked to succeed, got: %v", err)
	}
	if saved.Index != 0 {
		t.Errorf("expected index 0, got: %d", saved.Index)
	}

	// Verify untracked file is removed from working tree
	if _, err := os.Stat(untracked); !os.IsNotExist(err) {
		t.Errorf("expected untracked file to be stashed away, but it still exists")
	}
}

func TestStashApply(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("stashed work\n"), 0644)
	_, err := client.SaveStash(ctx, StashSaveOptions{Message: "work to apply"})
	if err != nil {
		t.Fatalf("failed to save stash: %v", err)
	}

	// Apply stash
	err = client.ApplyStash(ctx, StashApplyOptions{Index: 0})
	if err != nil {
		t.Fatalf("expected ApplyStash to succeed, got: %v", err)
	}

	// Verify content restored
	content, _ := os.ReadFile(f)
	if string(content) != "stashed work\n" {
		t.Errorf("expected restored content, got: %s", string(content))
	}

	// Verify stash remains in list!
	stashes, err := client.ListStashes(ctx)
	if err != nil || len(stashes) != 1 {
		t.Errorf("expected stash to be preserved after apply, stashes count: %d", len(stashes))
	}
}

func TestStashPop(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("stashed work\n"), 0644)
	_, err := client.SaveStash(ctx, StashSaveOptions{Message: "work to pop"})
	if err != nil {
		t.Fatalf("failed to save stash: %v", err)
	}

	// Pop stash
	err = client.PopStash(ctx, StashPopOptions{Index: 0})
	if err != nil {
		t.Fatalf("expected PopStash to succeed, got: %v", err)
	}

	// Verify content restored
	content, _ := os.ReadFile(f)
	if string(content) != "stashed work\n" {
		t.Errorf("expected restored content, got: %s", string(content))
	}

	// Verify stash is removed from list
	stashes, err := client.ListStashes(ctx)
	if err != nil || len(stashes) != 0 {
		t.Errorf("expected stash to be removed after pop, stashes count: %d", len(stashes))
	}
}

func TestStashPopAndApplyConflict(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("line 1\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("conflicting stash line\n"), 0644)
	_, err := client.SaveStash(ctx, StashSaveOptions{Message: "conflict branch"})
	if err != nil {
		t.Fatalf("failed to save stash: %v", err)
	}

	// Commit conflicting change
	_ = os.WriteFile(f, []byte("conflicting commit line\n"), 0644)
	runGit(t, dir, "commit", "-am", "conflicting commit")

	// Pop stash should detect conflict
	err = client.PopStash(ctx, StashPopOptions{Index: 0})
	if !errors.Is(err, ErrStashConflict) {
		t.Fatalf("expected ErrStashConflict on pop, got: %v", err)
	}

	// CRITICAL: Verify stash was NOT dropped when pop encountered conflict!
	stashes, err := client.ListStashes(ctx)
	if err != nil || len(stashes) != 1 {
		t.Fatalf("expected stash to remain preserved after conflict, count: %d", len(stashes))
	}
}

func TestStashDrop(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("line 1\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("stash 1\n"), 0644)
	_, _ = client.SaveStash(ctx, StashSaveOptions{Message: "msg 1"})

	_ = os.WriteFile(f, []byte("stash 2\n"), 0644)
	_, _ = client.SaveStash(ctx, StashSaveOptions{Message: "msg 2"})

	// Drop stash@{0}
	err := client.DropStash(ctx, StashDropOptions{Index: 0})
	if err != nil {
		t.Fatalf("expected DropStash to succeed, got: %v", err)
	}

	stashes, err := client.ListStashes(ctx)
	if err != nil || len(stashes) != 1 {
		t.Fatalf("expected 1 stash after drop, got: %d", len(stashes))
	}

	// Drop nonexistent stash
	err = client.DropStash(ctx, StashDropOptions{Index: 99})
	if !errors.Is(err, ErrStashNotFound) {
		t.Fatalf("expected ErrStashNotFound for index 99, got: %v", err)
	}
}

func TestStashOperationInProgress(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(dir, "file.txt")
	_ = os.WriteFile(f, []byte("initial\n"), 0644)
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	_ = os.WriteFile(f, []byte("dirty\n"), 0644)

	// Simulate merge in progress
	gitDir := filepath.Join(dir, ".git")
	mergeHeadFile := filepath.Join(gitDir, "MERGE_HEAD")
	_ = os.WriteFile(mergeHeadFile, []byte("dummyhash\n"), 0644)
	defer os.Remove(mergeHeadFile)

	_, err := client.SaveStash(ctx, StashSaveOptions{})
	if !errors.Is(err, ErrOperationInProgress) {
		t.Fatalf("expected ErrOperationInProgress on save, got: %v", err)
	}
}

func TestParseStashIndex(t *testing.T) {
	tests := []struct {
		input       string
		expectedIdx int
		expectErr   bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"1", 1, false},
		{"12", 12, false},
		{"stash@{0}", 0, false},
		{"stash@{5}", 5, false},
		{"-1", 0, true},
		{"stash@{-1}", 0, true},
		{"invalid", 0, true},
		{"stash@{abc}", 0, true},
	}

	for _, tc := range tests {
		idx, err := ParseStashIndex(tc.input)
		if tc.expectErr && err == nil {
			t.Errorf("expected error for input %q, got nil", tc.input)
		}
		if !tc.expectErr && err != nil {
			t.Errorf("expected no error for input %q, got: %v", tc.input, err)
		}
		if !tc.expectErr && idx != tc.expectedIdx {
			t.Errorf("expected index %d for input %q, got: %d", tc.expectedIdx, tc.input, idx)
		}
	}
}
