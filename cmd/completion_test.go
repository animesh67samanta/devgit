package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionBash(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "bash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful bash completion generation, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "bash completion") && !strings.Contains(out, "__devgit") {
		t.Errorf("expected bash completion script, got length: %d", len(out))
	}
}

func TestCompletionZsh(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "zsh"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful zsh completion generation, got: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Errorf("expected non-empty zsh completion script")
	}
}

func TestCompletionFish(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "fish"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful fish completion generation, got: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Errorf("expected non-empty fish completion script")
	}
}

func TestCompletionPowerShell(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"completion", "powershell"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful powershell completion generation, got: %v", err)
	}

	out := buf.String()
	if len(out) == 0 {
		t.Errorf("expected non-empty powershell completion script")
	}
}

func TestCompletionInvalidShell(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"completion", "invalid_shell"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid shell, got nil")
	}
}
