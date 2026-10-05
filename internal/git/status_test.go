package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStatusCleanRepository(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	testFile := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "add file.txt")

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("expected no error from Status, got: %v", err)
	}

	if !status.IsClean() {
		t.Errorf("expected clean status, got %d files", len(status.Files))
	}
	if status.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", status.Branch)
	}
}

func TestStatusModifiedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	testFile := filepath.Join(dir, "app.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "app.go")
	runGit(t, dir, "commit", "-m", "initial commit")

	// Modify file without staging
	if err := os.WriteFile(testFile, []byte("package main\n// modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("expected no error from Status, got: %v", err)
	}

	if status.IsClean() {
		t.Fatal("expected status to have changes, got clean")
	}

	if len(status.Files) != 1 {
		t.Fatalf("expected 1 file change, got %d", len(status.Files))
	}

	f := status.Files[0]
	if f.Path != "app.go" {
		t.Errorf("expected file path 'app.go', got %q", f.Path)
	}
	if f.Status != " M" {
		t.Errorf("expected status ' M', got %q", f.Status)
	}
	if !f.IsModified() {
		t.Error("expected IsModified() to be true")
	}
	if f.IsStaged() {
		t.Error("expected IsStaged() to be false")
	}
}

func TestStatusUntrackedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	// Create untracked file
	untrackedFile := filepath.Join(dir, "new_script.sh")
	if err := os.WriteFile(untrackedFile, []byte("#!/bin/bash\n"), 0644); err != nil {
		t.Fatalf("failed to create untracked file: %v", err)
	}

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("expected no error from Status, got: %v", err)
	}

	if len(status.Files) != 1 {
		t.Fatalf("expected 1 untracked file, got %d", len(status.Files))
	}

	f := status.Files[0]
	if f.Path != "new_script.sh" {
		t.Errorf("expected path 'new_script.sh', got %q", f.Path)
	}
	if f.Status != "??" {
		t.Errorf("expected status '??', got %q", f.Status)
	}
	if !f.IsUntracked() {
		t.Error("expected IsUntracked() to be true")
	}
}

func TestStatusDeletedFile(t *testing.T) {
	dir, client := setupTestRepo(t)
	ctx := context.Background()

	fileToDelete := filepath.Join(dir, "obsolete.txt")
	if err := os.WriteFile(fileToDelete, []byte("deprecated\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, dir, "add", "obsolete.txt")
	runGit(t, dir, "commit", "-m", "commit obsolete file")

	// Delete file from disk
	if err := os.Remove(fileToDelete); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}

	status, err := client.Status(ctx)
	if err != nil {
		t.Fatalf("expected no error from Status, got: %v", err)
	}

	if len(status.Files) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(status.Files))
	}

	f := status.Files[0]
	if f.Path != "obsolete.txt" {
		t.Errorf("expected path 'obsolete.txt', got %q", f.Path)
	}
	if f.Status != " D" {
		t.Errorf("expected status ' D', got %q", f.Status)
	}
	if !f.IsDeleted() {
		t.Error("expected IsDeleted() to be true")
	}
}

func TestParseStatus(t *testing.T) {
	raw := `## feature/payments...origin/feature/payments [ahead 2, behind 1]
 M app/UserController.php
M  app/UserService.php
?? new_test.go
 D deleted.txt
R  old_name.go -> new_name.go
`
	status, err := ParseStatus("/workspace/test", raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if status.Branch != "feature/payments" {
		t.Errorf("expected branch 'feature/payments', got %q", status.Branch)
	}
	if status.Upstream != "origin/feature/payments" {
		t.Errorf("expected upstream 'origin/feature/payments', got %q", status.Upstream)
	}
	if status.Ahead != 2 {
		t.Errorf("expected ahead 2, got %d", status.Ahead)
	}
	if status.Behind != 1 {
		t.Errorf("expected behind 1, got %d", status.Behind)
	}

	if len(status.Files) != 5 {
		t.Fatalf("expected 5 files, got %d", len(status.Files))
	}

	// 1. Unstaged modified
	f1 := status.Files[0]
	if f1.Path != "app/UserController.php" || f1.Status != " M" || !f1.IsModified() || f1.IsStaged() {
		t.Errorf("unexpected file 0: %+v", f1)
	}

	// 2. Staged modified
	f2 := status.Files[1]
	if f2.Path != "app/UserService.php" || f2.Status != "M " || !f2.IsStaged() {
		t.Errorf("unexpected file 1: %+v", f2)
	}

	// 3. Untracked
	f3 := status.Files[2]
	if f3.Path != "new_test.go" || f3.Status != "??" || !f3.IsUntracked() {
		t.Errorf("unexpected file 2: %+v", f3)
	}

	// 4. Deleted
	f4 := status.Files[3]
	if f4.Path != "deleted.txt" || f4.Status != " D" || !f4.IsDeleted() {
		t.Errorf("unexpected file 3: %+v", f4)
	}

	// 5. Renamed
	f5 := status.Files[4]
	if f5.Path != "new_name.go" || f5.OrigPath != "old_name.go" || !f5.IsRenamed() {
		t.Errorf("unexpected file 4: %+v", f5)
	}
	if f5.DisplayPath() != "old_name.go -> new_name.go" {
		t.Errorf("expected display path 'old_name.go -> new_name.go', got %q", f5.DisplayPath())
	}
}

func TestParseStatusInitialAndDetached(t *testing.T) {
	// Initial commit case
	rawInitial := `## Initial commit on main
?? README.md
`
	s1, err := ParseStatus("/root", rawInitial)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if s1.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", s1.Branch)
	}

	// Detached HEAD case
	rawDetached := `## HEAD (no branch)
 M main.go
`
	s2, err := ParseStatus("/root", rawDetached)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if s2.Branch != "(HEAD detached)" {
		t.Errorf("expected branch '(HEAD detached)', got %q", s2.Branch)
	}
}
