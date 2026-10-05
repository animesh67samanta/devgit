package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPushSuccessful(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("content"), 0644)
	runGit(t, localDir, "add", "test.txt")
	runGit(t, localDir, "commit", "-m", "init")

	// Push with SetUpstream
	res, err := client.Push(ctx, PushOptions{
		Remote:      "origin",
		Branch:      "main",
		SetUpstream: true,
	})
	if err != nil {
		t.Fatalf("expected no error from Push, got: %v", err)
	}

	if res.Remote != "origin" || res.Branch != "main" {
		t.Errorf("unexpected PushResult: %+v", res)
	}

	// Verify tracking info shows up-to-date
	info, err := client.TrackingInfo(ctx, "main")
	if err != nil || !info.IsUpToDate() {
		t.Errorf("expected branch to be up to date after push, got info: %+v, err: %v", info, err)
	}
}

func TestPushNoRemote(t *testing.T) {
	localDir, client := setupTestRepo(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("data"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "init")

	_, err := client.Push(ctx, PushOptions{
		Remote: "origin",
		Branch: "main",
	})
	if err == nil {
		t.Fatal("expected error pushing without remote, got nil")
	}
}

func TestPushDetachedHEAD(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("data"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "init")

	// Detach HEAD
	runGit(t, localDir, "checkout", "--detach")

	_, err := client.Push(ctx, PushOptions{
		Remote: "origin",
	})
	if !errors.Is(err, ErrDetachedHEAD) {
		t.Errorf("expected ErrDetachedHEAD, got: %v", err)
	}
}

func TestPushRejectedNonFastForward(t *testing.T) {
	localDir, remoteDir, client := setupLocalAndRemote(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "base")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Create a second clone and push a commit
	secondDir := filepath.Join(t.TempDir(), "second")
	runGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	runGit(t, secondDir, "config", "user.name", "Second User")
	runGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "second.txt")
	_ = os.WriteFile(sFile, []byte("second"), 0644)
	runGit(t, secondDir, "add", "second.txt")
	runGit(t, secondDir, "commit", "-m", "second commit")
	runGit(t, secondDir, "push", "origin", "main")

	// Local makes a different commit without pulling
	_ = os.WriteFile(f, []byte("local diff"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "local commit")

	_, err := client.Push(ctx, PushOptions{
		Remote: "origin",
		Branch: "main",
	})
	if !errors.Is(err, ErrPushRejected) {
		t.Errorf("expected ErrPushRejected on non-fast-forward, got: %v", err)
	}
}

func TestPushSpecialBranchName(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// Initial commit on main
	f := filepath.Join(localDir, "base.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	runGit(t, localDir, "add", "base.txt")
	runGit(t, localDir, "commit", "-m", "base")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Create and switch to special branch
	specialBranch := "feature/issue-123_auth"
	runGit(t, localDir, "checkout", "-b", specialBranch)
	f2 := filepath.Join(localDir, "feature.txt")
	_ = os.WriteFile(f2, []byte("feature"), 0644)
	runGit(t, localDir, "add", "feature.txt")
	runGit(t, localDir, "commit", "-m", "feature commit")

	res, err := client.Push(ctx, PushOptions{
		Remote:      "origin",
		Branch:      specialBranch,
		SetUpstream: true,
	})
	if err != nil {
		t.Fatalf("expected no error pushing special branch, got: %v", err)
	}
	if res.Branch != specialBranch {
		t.Errorf("expected branch %q, got %q", specialBranch, res.Branch)
	}
}

func TestIsProtectedBranch(t *testing.T) {
	protected := []string{"main", "master", "production", "prod", "MAIN", "Master"}
	for _, b := range protected {
		if !IsProtectedBranch(b) {
			t.Errorf("expected %q to be identified as protected", b)
		}
	}

	nonProtected := []string{"feature/login", "bugfix/123", "dev", "staging"}
	for _, b := range nonProtected {
		if IsProtectedBranch(b) {
			t.Errorf("expected %q to NOT be protected", b)
		}
	}
}
