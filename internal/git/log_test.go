package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLogEmptyRepository(t *testing.T) {
	_, client := setupTestRepo(t)
	ctx := context.Background()

	commits, err := client.Log(ctx, LogOptions{})
	if err != nil {
		t.Fatalf("expected no error for empty repo log, got: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits in empty repo, got: %d", len(commits))
	}
}

func TestLogMultipleCommits(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	// Commit 1
	file1 := filepath.Join(dir, "file1.txt")
	if err := os.WriteFile(file1, []byte("one\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "file1.txt")
	runGit(t, dir, "commit", "-m", "First commit")

	// Commit 2
	file2 := filepath.Join(dir, "file2.txt")
	if err := os.WriteFile(file2, []byte("two\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "file2.txt")
	runGit(t, dir, "commit", "-m", "Second commit")

	// Commit 3
	file3 := filepath.Join(dir, "file3.txt")
	if err := os.WriteFile(file3, []byte("three\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "file3.txt")
	runGit(t, dir, "commit", "-m", "Third commit")

	// Test full log
	commits, err := client.Log(ctx, LogOptions{})
	if err != nil {
		t.Fatalf("expected no error from Log, got: %v", err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got: %d", len(commits))
	}

	// Commits should be in reverse chronological order
	if commits[0].Subject != "Third commit" {
		t.Errorf("expected newest commit 'Third commit', got %q", commits[0].Subject)
	}
	if commits[1].Subject != "Second commit" {
		t.Errorf("expected middle commit 'Second commit', got %q", commits[1].Subject)
	}
	if commits[2].Subject != "First commit" {
		t.Errorf("expected oldest commit 'First commit', got %q", commits[2].Subject)
	}

	for _, c := range commits {
		if len(c.ShortHash) == 0 {
			t.Error("expected non-empty ShortHash")
		}
		if c.AuthorName != "Test User" {
			t.Errorf("expected AuthorName 'Test User', got %q", c.AuthorName)
		}
		if len(c.RelativeDate) == 0 {
			t.Error("expected non-empty RelativeDate")
		}
	}

	// Test limit
	limitedCommits, err := client.Log(ctx, LogOptions{Limit: 2})
	if err != nil {
		t.Fatalf("expected no error with limit, got: %v", err)
	}
	if len(limitedCommits) != 2 {
		t.Fatalf("expected 2 commits with limit 2, got: %d", len(limitedCommits))
	}
	if limitedCommits[0].Subject != "Third commit" {
		t.Errorf("expected 'Third commit', got %q", limitedCommits[0].Subject)
	}
}

func TestParseLog(t *testing.T) {
	raw := "a82f91cb18\x1fa82f91c\x1fAnimesh\x1fanimesh@example.com\x1f2 hours ago\x1fSun Oct 4\x1fAdd payment validation\n" +
		"c19d223f00\x1fc19d223\x1fAnimesh\x1fanimesh@example.com\x1fyesterday\x1fSat Oct 3\x1fUpdate billing calculation\n"

	commits, err := ParseLog(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got: %d", len(commits))
	}

	if commits[0].ShortHash != "a82f91c" || commits[0].Subject != "Add payment validation" || commits[0].RelativeDate != "2 hours ago" {
		t.Errorf("unexpected commit 0: %+v", commits[0])
	}
	if commits[1].ShortHash != "c19d223" || commits[1].Subject != "Update billing calculation" || commits[1].RelativeDate != "yesterday" {
		t.Errorf("unexpected commit 1: %+v", commits[1])
	}
}
