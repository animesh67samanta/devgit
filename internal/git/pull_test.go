package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPullSuccessfulFastForward(t *testing.T) {
	localDir, remoteDir, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// Initial commit & push
	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "v1")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Simulate commit on remote by second user
	secondDir := filepath.Join(t.TempDir(), "second")
	runGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	runGit(t, secondDir, "config", "user.name", "Second User")
	runGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "f.txt")
	_ = os.WriteFile(sFile, []byte("v2 from remote"), 0644)
	runGit(t, secondDir, "add", "f.txt")
	runGit(t, secondDir, "commit", "-m", "v2 remote commit")
	runGit(t, secondDir, "push", "origin", "main")

	// Fetch remote in local
	if err := client.Fetch(ctx, "origin"); err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	info, err := client.TrackingInfo(ctx, "main")
	if err != nil || info.Behind != 1 {
		t.Fatalf("expected 1 commit behind, got info: %+v, err: %v", info, err)
	}

	// Pull with fast-forward
	res, err := client.Pull(ctx, PullOptions{
		Remote: "origin",
		Branch: "main",
		FFOnly: true,
	})
	if err != nil {
		t.Fatalf("expected fast-forward pull to succeed, got: %v", err)
	}

	if res.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", res.Branch)
	}

	// Verify local file content updated
	updatedContent, _ := os.ReadFile(f)
	if string(updatedContent) != "v2 from remote" {
		t.Errorf("expected file content 'v2 from remote', got %q", string(updatedContent))
	}
}

func TestPullDetachedHEAD(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "v1")

	runGit(t, localDir, "checkout", "--detach")

	_, err := client.Pull(ctx, PullOptions{
		Remote: "origin",
		Branch: "main",
	})
	if !errors.Is(err, ErrDetachedHEAD) {
		t.Errorf("expected ErrDetachedHEAD, got: %v", err)
	}
}

func TestPullFastForwardNotPossibleWhenDiverged(t *testing.T) {
	localDir, remoteDir, client := setupLocalAndRemote(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "base")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Remote commit from second clone
	secondDir := filepath.Join(t.TempDir(), "second")
	runGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	runGit(t, secondDir, "config", "user.name", "Second User")
	runGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "remote.txt")
	_ = os.WriteFile(sFile, []byte("remote commit"), 0644)
	runGit(t, secondDir, "add", "remote.txt")
	runGit(t, secondDir, "commit", "-m", "remote commit")
	runGit(t, secondDir, "push", "origin", "main")

	// Local commit diverging from remote
	_ = os.WriteFile(f, []byte("local modified"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "local commit")

	// Fetch
	_ = client.Fetch(ctx, "origin")

	// Attempt pull with FFOnly
	_, err := client.Pull(ctx, PullOptions{
		Remote: "origin",
		Branch: "main",
		FFOnly: true,
	})
	if !errors.Is(err, ErrFastForwardNotPossible) {
		t.Errorf("expected ErrFastForwardNotPossible, got: %v", err)
	}
}
