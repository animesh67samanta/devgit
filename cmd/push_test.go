package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupLocalRemotePair(t *testing.T) (string, string) {
	t.Helper()
	baseDir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(baseDir); err == nil {
		baseDir = realDir
	}

	remoteDir := filepath.Join(baseDir, "remote.git")
	localDir := filepath.Join(baseDir, "local")

	execGit(t, baseDir, "init", "--bare", "-b", "main", remoteDir)
	execGit(t, baseDir, "clone", remoteDir, localDir)
	execGit(t, localDir, "config", "user.name", "DevGit Test")
	execGit(t, localDir, "config", "user.email", "devgit@example.com")

	return localDir, remoteDir
}

func TestPushCmdForceRejected(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(context.Background(), strings.NewReader(""), outBuf, errBuf, t.TempDir(), PushCmdFlags{Force: true})
	if err == nil {
		t.Fatal("expected error on force push, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Force push is not supported in Phase 5.") {
		t.Errorf("expected force push rejection message, got: %s", errOut)
	}
}

func TestPushCmdNoRemote(t *testing.T) {
	localDir := setupTestGitRepo(t)
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(context.Background(), strings.NewReader(""), outBuf, errBuf, localDir, PushCmdFlags{})
	if err == nil {
		t.Fatal("expected error with no remote, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ No Git remote is configured.") {
		t.Errorf("expected no remote message, got: %s", errOut)
	}
}

func TestPushCmdNoUpstreamPrompt(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("test"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "init")

	// Branch has no upstream yet; user enters 'y' to set upstream and push
	input := strings.NewReader("y\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, input, outBuf, errBuf, localDir, PushCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error pushing and setting upstream, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "⚠ Current branch has no upstream.") {
		t.Errorf("expected no upstream warning, got: %s", out)
	}
	if !strings.Contains(out, "✓ Push completed") {
		t.Errorf("expected push completed, got: %s", out)
	}
}

func TestPushCmdAlreadySynchronized(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("test"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "init")
	execGit(t, localDir, "push", "-u", "origin", "main")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PushCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error when already synchronized, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "✓ Nothing to push.") {
		t.Errorf("expected '✓ Nothing to push.', got: %s", out)
	}
}

func TestPushCmdInteractiveSuccess(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Create a new local commit
	_ = os.WriteFile(f, []byte("v2"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v2")

	// User confirms push with 'y'
	input := strings.NewReader("y\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, input, outBuf, errBuf, localDir, PushCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Status:") || !strings.Contains(out, "1 commits to push") {
		t.Errorf("expected status with commits to push, got: %s", out)
	}
	if !strings.Contains(out, "✓ Push completed") {
		t.Errorf("expected '✓ Push completed', got: %s", out)
	}
}

func TestPushCmdCancellation(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	_ = os.WriteFile(f, []byte("v2"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v2")

	// User rejects with 'n'
	input := strings.NewReader("n\n")
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, input, outBuf, errBuf, localDir, PushCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error on cancel, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "Operation cancelled.") {
		t.Errorf("expected 'Operation cancelled.', got: %s", out)
	}
	if strings.Contains(out, "✓ Push completed") {
		t.Errorf("did not expect push to complete on cancel, got: %s", out)
	}
}

func TestPushCmdDirectYesFlag(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")
	execGit(t, localDir, "push", "-u", "origin", "main")

	_ = os.WriteFile(f, []byte("v2"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v2")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, nil, outBuf, errBuf, localDir, PushCmdFlags{Yes: true})
	if err != nil {
		t.Fatalf("expected no error with -y, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "✓ Push completed") {
		t.Errorf("expected push completed, got: %s", out)
	}
}

func TestPushCmdDetachedHEAD(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("v1"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "v1")

	execGit(t, localDir, "checkout", "--detach")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PushCmdFlags{})
	if err == nil {
		t.Fatal("expected error pushing from detached HEAD, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "❌ Cannot push from detached HEAD.") {
		t.Errorf("expected detached HEAD error, got: %s", errOut)
	}
}

func TestPushCmdOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunPush(context.Background(), strings.NewReader(""), outBuf, errBuf, dir, PushCmdFlags{})
	if err == nil {
		t.Fatal("expected error outside repo, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errOut)
	}
}

func TestPushCmdConfiguredProtectedBranch(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	// Initial commit on main and push
	f := filepath.Join(localDir, "test.txt")
	_ = os.WriteFile(f, []byte("test"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "init")
	execGit(t, localDir, "push", "-u", "origin", "main")

	// Create and switch to staging branch
	execGit(t, localDir, "checkout", "-b", "staging")
	execGit(t, localDir, "push", "-u", "origin", "staging")

	// Add commit to staging so ahead count > 0
	_ = os.WriteFile(f, []byte("staging content"), 0644)
	execGit(t, localDir, "add", "test.txt")
	execGit(t, localDir, "commit", "-m", "staging update")

	// Configure staging as protected branch in .git/devgit.yaml
	devgitYaml := filepath.Join(localDir, ".git", "devgit.yaml")
	_ = os.WriteFile(devgitYaml, []byte("git:\n  protected_branches:\n    - main\n    - staging\n"), 0644)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunPush(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PushCmdFlags{Yes: true})
	if err != nil {
		t.Fatalf("expected push success, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "Protected/default branch detected.") {
		t.Errorf("expected protected branch warning for configured 'staging' branch, got: %s", out)
	}
}

func TestPushCmdConfiguredDefaultRemote(t *testing.T) {
	localDir, _ := setupLocalRemotePair(t)
	ctx := context.Background()

	// Create an upstream bare remote directory
	upstreamBare := t.TempDir()
	execGit(t, upstreamBare, "init", "--bare")
	execGit(t, localDir, "remote", "add", "upstream", upstreamBare)

	// Configure upstream as default remote in .git/devgit.yaml
	devgitYaml := filepath.Join(localDir, ".git", "devgit.yaml")
	_ = os.WriteFile(devgitYaml, []byte("git:\n  default_remote: upstream\n"), 0644)

	// Create new branch
	execGit(t, localDir, "checkout", "-b", "feature-x")
	f := filepath.Join(localDir, "feature.txt")
	_ = os.WriteFile(f, []byte("feature"), 0644)
	execGit(t, localDir, "add", "feature.txt")
	execGit(t, localDir, "commit", "-m", "feature commit")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunPush(ctx, strings.NewReader(""), outBuf, errBuf, localDir, PushCmdFlags{Yes: true})
	if err != nil {
		t.Fatalf("expected push success, got: %v\nstderr: %s", err, errBuf.String())
	}

	out := outBuf.String()
	if !strings.Contains(out, "upstream/feature-x") {
		t.Errorf("expected push target to use configured default remote 'upstream', got: %s", out)
	}
}
