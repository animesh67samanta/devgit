package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBranchList(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// Initial commit on main and push with -u
	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("data"), 0644)
	runGit(t, localDir, "add", "test.txt")
	runGit(t, localDir, "commit", "-m", "init")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Create a second branch
	runGit(t, localDir, "branch", "feature/user")

	res, err := client.ListBranches(ctx)
	if err != nil {
		t.Fatalf("unexpected error listing branches: %v", err)
	}

	if len(res.Local) != 2 {
		t.Fatalf("expected 2 local branches, got %d", len(res.Local))
	}
	if len(res.Remote) != 1 {
		t.Fatalf("expected 1 remote branch, got %d", len(res.Remote))
	}

	var foundMain, foundFeature bool
	for _, b := range res.Local {
		if b.Name == "main" {
			foundMain = true
			if !b.IsCurrent {
				t.Error("expected main to be current branch")
			}
			if b.Upstream != "origin/main" {
				t.Errorf("expected upstream origin/main, got %q", b.Upstream)
			}
		}
		if b.Name == "feature/user" {
			foundFeature = true
			if b.IsCurrent {
				t.Error("expected feature/user not to be current")
			}
		}
	}

	if !foundMain || !foundFeature {
		t.Errorf("missing branches in list: foundMain=%v, foundFeature=%v", foundMain, foundFeature)
	}
}

func TestBranchCreateAndExists(t *testing.T) {
	localDir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	runGit(t, localDir, "add", "init.txt")
	runGit(t, localDir, "commit", "-m", "init")

	// 1. Create branch successfully
	err := client.CreateBranch(ctx, "feature/payment", "")
	if err != nil {
		t.Fatalf("unexpected error creating branch: %v", err)
	}

	exists, err := client.BranchExists(ctx, "feature/payment")
	if err != nil || !exists {
		t.Errorf("expected branch feature/payment to exist, err: %v", err)
	}

	// 2. Duplicate branch should fail
	err = client.CreateBranch(ctx, "feature/payment", "")
	if !errors.Is(err, ErrBranchAlreadyExists) {
		t.Errorf("expected ErrBranchAlreadyExists, got: %v", err)
	}

	// 3. Empty branch name should fail
	err = client.CreateBranch(ctx, "", "")
	if !errors.Is(err, ErrEmptyBranchName) {
		t.Errorf("expected ErrEmptyBranchName, got: %v", err)
	}
}

func TestBranchSwitch(t *testing.T) {
	localDir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	runGit(t, localDir, "add", "init.txt")
	runGit(t, localDir, "commit", "-m", "init")

	_ = client.CreateBranch(ctx, "dev", "")

	// Switch to dev
	err := client.SwitchBranch(ctx, "dev")
	if err != nil {
		t.Fatalf("unexpected error switching to dev: %v", err)
	}

	current, err := client.CurrentBranch(ctx)
	if err != nil || current != "dev" {
		t.Errorf("expected current branch 'dev', got %q, err: %v", current, err)
	}

	// Switch to non-existent branch
	err = client.SwitchBranch(ctx, "does-not-exist")
	if !errors.Is(err, ErrBranchNotFound) {
		t.Errorf("expected ErrBranchNotFound, got: %v", err)
	}
}

func TestBranchDeleteSafe(t *testing.T) {
	localDir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	runGit(t, localDir, "add", "init.txt")
	runGit(t, localDir, "commit", "-m", "init")

	_ = client.CreateBranch(ctx, "to-delete", "")

	// 1. Cannot delete current branch
	err := client.DeleteBranch(ctx, "main", DeleteBranchOptions{})
	if !errors.Is(err, ErrCannotDeleteCurrent) {
		t.Errorf("expected ErrCannotDeleteCurrent, got: %v", err)
	}

	// 2. Safe deletion of merged branch
	err = client.DeleteBranch(ctx, "to-delete", DeleteBranchOptions{})
	if err != nil {
		t.Fatalf("unexpected error deleting merged branch: %v", err)
	}

	exists, _ := client.BranchExists(ctx, "to-delete")
	if exists {
		t.Error("expected branch to-delete to be gone")
	}

	// 3. Delete unmerged branch without force should fail with ErrBranchNotMerged
	_ = client.CreateBranch(ctx, "unmerged", "")
	_ = client.SwitchBranch(ctx, "unmerged")
	f2 := filepath.Join(localDir, "unmerged.txt")
	_ = os.WriteFile(f2, []byte("unmerged"), 0644)
	runGit(t, localDir, "add", "unmerged.txt")
	runGit(t, localDir, "commit", "-m", "unmerged commit")
	_ = client.SwitchBranch(ctx, "main")

	err = client.DeleteBranch(ctx, "unmerged", DeleteBranchOptions{Force: false})
	if !errors.Is(err, ErrBranchNotMerged) {
		t.Errorf("expected ErrBranchNotMerged, got: %v", err)
	}

	// 4. Force delete unmerged branch with Force=true
	err = client.DeleteBranch(ctx, "unmerged", DeleteBranchOptions{Force: true})
	if err != nil {
		t.Fatalf("unexpected error force-deleting unmerged branch: %v", err)
	}

	exists, _ = client.BranchExists(ctx, "unmerged")
	if exists {
		t.Error("expected unmerged branch to be deleted with force")
	}
}

func TestBranchRename(t *testing.T) {
	localDir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "init.txt")
	_ = os.WriteFile(f, []byte("init"), 0644)
	runGit(t, localDir, "add", "init.txt")
	runGit(t, localDir, "commit", "-m", "init")

	_ = client.CreateBranch(ctx, "old-name", "")

	// Rename specific branch
	err := client.RenameBranch(ctx, "old-name", "new-name")
	if err != nil {
		t.Fatalf("unexpected error renaming branch: %v", err)
	}

	existsOld, _ := client.BranchExists(ctx, "old-name")
	existsNew, _ := client.BranchExists(ctx, "new-name")
	if existsOld || !existsNew {
		t.Errorf("expected old-name gone (%v) and new-name present (%v)", existsOld, existsNew)
	}
}
