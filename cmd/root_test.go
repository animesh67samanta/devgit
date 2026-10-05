package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running --help, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "devgit is a developer-friendly Git assistant") {
		t.Errorf("expected help output to describe devgit, got: %s", output)
	}

	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected help output to contain Usage, got: %s", output)
	}
}

func TestRootCmdVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running --version, got: %v", err)
	}

	output := buf.String()
	expected := "devgit version " + Version
	if !strings.Contains(output, expected) {
		t.Errorf("expected version output to contain %q, got: %s", expected, output)
	}
}

func TestRootCmdDefaultRun(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected no error running root command, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected root command without args to display help, got: %s", output)
	}
}
