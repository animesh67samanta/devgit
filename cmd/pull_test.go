package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPullCmdSuccess(t *testing.T) {
	localDir, remoteDir := setupLocalRemotePair(t)
	ctx := context.Background()

	// Initial commit & push
	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Remote commit from second clone
	secondDir := filepath.Join(t.TempDir(), "second")
	execGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	execGit(t, secondDir, "config", "user.name", "Second User")
	execGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "test.txt")
	_ = os.WriteFile(sFile, []byte("v2 from remote"), 0644)
	execGit(t, secondDir, "add", "test.txt")
	execGit(t, secondDir, "commit", "-m", "v2 remote commit")
	execGit(t, secondDir, "push", "origin", "main")

	// User enters 'y' to pull
	input := strings.NewReader("y\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(ctx, input, outBuf, errBuf, localDir, PullCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error pulling, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Status:") || !strings.Contains(out, "1 commits behind") {
		t.Errorf("expected status behind in output, got: %s", out)
	}
	if !strings.Contains(out, "✓ Pull completed") {
		t.Errorf("expected '✓ Pull completed', got: %s", out)
	}

	// Verify local file updated
	content, _ := os.ReadFile(f)
	if string(content) != "v2 from remote" {
		t.Errorf("expected updated content 'v2 from remote', got: %q", string(content))
	}
}

func TestPullCmdAlreadyUpToDate(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PullCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error when up to date, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "✓ Already up to date.") {
		t.Errorf("expected '✓ Already up to date.', got: %s", out)
	}
}

func TestPullCmdDirtyWorkingTreeConfirmation(t *testing.T) {
	localDir, remoteDir := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "base.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	execGit(t, localDir, "add", "base.txt")
	execGit(t, localDir, "commit", "-m", "base")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Remote commit on another file
	secondDir := filepath.Join(t.TempDir(), "second")
	execGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	execGit(t, secondDir, "config", "user.name", "Second User")
	execGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "remote.txt")
	_ = os.WriteFile(sFile, []byte("remote"), 0644)
	execGit(t, secondDir, "add", "remote.txt")
	execGit(t, secondDir, "commit", "-m", "remote commit")
	execGit(t, secondDir, "push", "origin", "main")

	// Local makes an uncommitted modification to a local file
	uncommittedFile := filepath.Join(localDir, "uncommitted.txt")
	_ = os.WriteFile(uncommittedFile, []byte("local uncommitted changes"), 0644)

	// User confirms dirty working tree warning ('y') and then confirms pull ('y')
	input := strings.NewReader("y\ny\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(ctx, input, outBuf, errBuf, localDir, PullCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error pulling with non-conflicting dirty tree: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "⚠ Your working tree contains local changes.") {
		t.Errorf("expected dirty working tree warning, got: %s", out)
	}
	if !strings.Contains(out, "✓ Pull completed") {
		t.Errorf("expected pull completed, got: %s", out)
	}
}

func TestPullCmdDivergedRejected(t *testing.T) {
	localDir, remoteDir := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("base"), 0644)
	execGit(t, localDir, "add", "f.txt")
	execGit(t, localDir, "commit", "-m", "base")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Remote commit from second clone
	secondDir := filepath.Join(t.TempDir(), "second")
	execGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	execGit(t, secondDir, "config", "user.name", "Second User")
	execGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "remote.txt")
	_ = os.WriteFile(sFile, []byte("remote"), 0644)
	execGit(t, secondDir, "add", "remote.txt")
	execGit(t, secondDir, "commit", "-m", "remote commit")
	execGit(t, secondDir, "push", "origin", "main")

	// Local commit diverging from remote
	_ = os.WriteFile(f, []byte("local diff"), 0644)
	execGit(t, localDir, "add", "f.txt")
	execGit(t, localDir, "commit", "-m", "local commit")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PullCmdFlags{})
	if err == nil {
		t.Fatal("expected error on diverged pull, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Fast-forward pull is not possible.") {
		t.Errorf("expected fast-forward pull rejection message, got: %s", errOut)
	}
}

func TestPullCmdCancellation(t *testing.T) {
	localDir, remoteDir := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "f.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "f.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Remote commit
	secondDir := filepath.Join(t.TempDir(), "second")
	execGit(t, t.TempDir(), "clone", remoteDir, secondDir)
	execGit(t, secondDir, "config", "user.name", "Second User")
	execGit(t, secondDir, "config", "user.email", "second@test.com")
	sFile := filepath.Join(secondDir, "f.txt")
	_ = os.WriteFile(sFile, []byte("v2"), 0644)
	execGit(t, secondDir, "add", "f.txt")
	execGit(t, secondDir, "commit", "-m", "v2")
	execGit(t, secondDir, "push", "origin", "main")

	// User enters 'n' at confirmation
	input := strings.NewReader("n\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(ctx, input, outBuf, errBuf, localDir, PullCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error on cancel, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.', got: %s", out)
	}
	if strings.Contains(out, "✓ Pull completed") {
		t.Errorf("did not expect pull completed on cancel, got: %s", out)
	}
}

func TestPullCmdNoRemote(t *testing.T) {
	localDir := setupTestGitRepo(t)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(context.Background(), strings.NewReader(""), outBuf, errBuf, localDir, PullCmdFlags{})
	if err == nil {
		t.Fatal("expected error with no remote, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ No Git remote is configured.") {
		t.Errorf("expected no remote message, got: %s", errOut)
	}
}

func TestPullCmdNoUpstream(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("test"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "init")

	// Branch has not been pushed, so no upstream exists
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(context.Background(), strings.NewReader(""), outBuf, errBuf, localDir, PullCmdFlags{})
	if err == nil {
		t.Fatal("expected error with no upstream, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Current branch has no upstream tracking branch.") {
		t.Errorf("expected no upstream error message, got: %s", errOut)
	}
}

func TestPullCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPull(context.Background(), strings.NewReader(""), outBuf, errBuf, dir, PullCmdFlags{})
	if err == nil {
		t.Fatal("expected error outside repo, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errOut)
	}
}
