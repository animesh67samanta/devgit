package cmd

import (
	"bytes"
	"strings"
	"testing"

	"devgit/internal/version"
)

func TestVersionCmdOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	RunVersion(buf, false)

	out := buf.String()
	if !strings.Contains(out, "devgit version "+version.Version) {
		t.Errorf("expected version %s in output, got: %s", version.Version, out)
	}
	if !strings.Contains(out, "commit:") {
		t.Errorf("expected commit in output, got: %s", out)
	}
	if !strings.Contains(out, "built:") {
		t.Errorf("expected built date in output, got: %s", out)
	}
}

func TestVersionCmdShort(t *testing.T) {
	buf := new(bytes.Buffer)
	RunVersion(buf, true)

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, "devgit version "+version.Version) {
		t.Errorf("expected short version output, got: %s", out)
	}
	if strings.Contains(out, "\n") {
		t.Errorf("expected single line for short version, got: %s", out)
	}
}

func TestVersionSubcommandExecution(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected 'devgit version' to execute without error, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "devgit version "+version.Version) {
		t.Errorf("expected version output, got: %s", out)
	}
}
