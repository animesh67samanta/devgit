package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTuiCmdOutsideRepo(t *testing.T) {
	emptyDir := t.TempDir()
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunTUI(ctx, strings.NewReader("q"), outBuf, errBuf, emptyDir)
	if err == nil {
		t.Fatal("expected error outside git repository, got nil")
	}

	if !strings.Contains(errBuf.String(), notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errBuf.String())
	}
}

func TestTuiSubcommandWiring(t *testing.T) {
	rootCmd := NewRootCmd()
	tuiCmd, _, err := rootCmd.Find([]string{"tui"})
	if err != nil || tuiCmd == nil {
		t.Fatalf("expected 'tui' command to be registered on root, err: %v", err)
	}

	if tuiCmd.Name() != "tui" {
		t.Errorf("expected command name 'tui', got: %s", tuiCmd.Name())
	}

	if !strings.Contains(tuiCmd.Short, "terminal UI") {
		t.Errorf("expected Short description to mention terminal UI, got: %s", tuiCmd.Short)
	}
}

func TestTuiCmdInsideRepoWithImmediateQuit(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Send 'q' to immediately quit TUI
	err := RunTUI(ctx, strings.NewReader("q\n"), outBuf, errBuf, dir)
	if err != nil {
		t.Fatalf("expected clean run and exit on 'q', got: %v\nstderr: %s", err, errBuf.String())
	}
}
