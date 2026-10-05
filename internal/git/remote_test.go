package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupLocalAndRemote(t *testing.T) (string, string, *Client) {
	t.Helper()
	baseDir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(baseDir); err == nil {
		baseDir = realDir
	}

	remoteDir := filepath.Join(baseDir, "remote.git")
	localDir := filepath.Join(baseDir, "local")

	runGit(t, baseDir, "init", "--bare", "-b", "main", remoteDir)
	runGit(t, baseDir, "clone", remoteDir, localDir)
	runGit(t, localDir, "config", "user.name", "DevGit Test")
	runGit(t, localDir, "config", "user.email", "devgit@example.com")

	client, err := NewClient(localDir)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return localDir, remoteDir, client
}

func TestRemotesNoRemote(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	remotes, err := client.Remotes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remotes) != 0 {
		t.Errorf("expected 0 remotes, got %d", len(remotes))
	}
}

func TestRemotesOneAndMultiple(t *testing.T) {
	localDir, remoteDir, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// Initial clone has 1 remote: origin
	remotes, err := client.Remotes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(remotes))
	}
	if remotes[0].Name != "origin" {
		t.Errorf("expected remote name 'origin', got %q", remotes[0].Name)
	}
	if remotes[0].URL != remoteDir {
		t.Errorf("expected remote URL %q, got %q", remoteDir, remotes[0].URL)
	}

	// Add a second remote
	runGit(t, localDir, "remote", "add", "upstream", "https://github.com/example/repo.git")

	remotes, err = client.Remotes(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(remotes))
	}
}

func TestTrackingInfo(t *testing.T) {
	localDir, _, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// 1. Initial state before any commits/upstream
	info, err := client.TrackingInfo(ctx, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.HasUpstream() {
		t.Error("expected no upstream before first push")
	}

	// Create commit and push with -u
	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("hello"), 0644)
	runGit(t, localDir, "add", "test.txt")
	runGit(t, localDir, "commit", "-m", "init")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// 2. Up to date state
	info, err = client.TrackingInfo(ctx, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.HasUpstream() {
		t.Fatal("expected upstream to exist")
	}
	if info.Upstream != "origin/main" {
		t.Errorf("expected upstream 'origin/main', got %q", info.Upstream)
	}
	if info.Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", info.Remote)
	}
	if !info.IsUpToDate() {
		t.Errorf("expected up-to-date, got ahead=%d behind=%d", info.Ahead, info.Behind)
	}

	// 3. Ahead state
	_ = os.WriteFile(f, []byte("hello v2"), 0644)
	runGit(t, localDir, "add", "test.txt")
	runGit(t, localDir, "commit", "-m", "ahead commit")

	info, err = client.TrackingInfo(ctx, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Ahead != 1 || info.Behind != 0 {
		t.Errorf("expected ahead=1 behind=0, got ahead=%d behind=%d", info.Ahead, info.Behind)
	}
	if info.IsUpToDate() {
		t.Error("expected not up to date")
	}
}

func TestTrackingInfoDiverged(t *testing.T) {
	localDir, remoteDir, client := setupLocalAndRemote(t)
	ctx := context.Background()

	// Initial commit & push
	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "base")
	runGit(t, localDir, "push", "-u", "origin", "main")

	// Clone a second repo to simulate remote changes
	secondDir := filepath.Join(t.TempDir(), "second")
	runGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	runGit(t, secondDir, "config", "user.name", "Second User")
	runGit(t, secondDir, "config", "user.email", "second@test.com")
	secondFile := filepath.Join(secondDir, "remote_only.txt")
	_ = os.WriteFile(secondFile, []byte("remote"), 0644)
	runGit(t, secondDir, "add", "remote_only.txt")
	runGit(t, secondDir, "commit", "-m", "remote commit")
	runGit(t, secondDir, "push", "origin", "main")

	// In localDir, make a local commit
	_ = os.WriteFile(f, []byte("local modified"), 0644)
	runGit(t, localDir, "add", "f.txt")
	runGit(t, localDir, "commit", "-m", "local commit")

	// Fetch in localDir
	if err := client.Fetch(ctx, "origin"); err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	info, err := client.TrackingInfo(ctx, "main")
	if err != nil {
		t.Fatalf("tracking info error: %v", err)
	}

	if info.Ahead != 1 {
		t.Errorf("expected ahead=1, got %d", info.Ahead)
	}
	if info.Behind != 1 {
		t.Errorf("expected behind=1, got %d", info.Behind)
	}
	if !info.IsDiverged() {
		t.Error("expected IsDiverged() to be true")
	}
}
