package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigListCmd(t *testing.T) {
	ctx := context.Background()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Isolate global config
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	err := RunConfigList(ctx, outBuf, errBuf, "", ConfigCmdFlags{})
	if err != nil {
		t.Fatalf("expected no error from RunConfigList, got: %v", err)
	}

	out := outBuf.String()
	if !strings.Contains(out, "DevGit Configuration") {
		t.Errorf("expected header 'DevGit Configuration', got: %s", out)
	}
	if !strings.Contains(out, "theme: default") {
		t.Errorf("expected 'theme: default', got: %s", out)
	}
	if !strings.Contains(out, "default_remote: origin") {
		t.Errorf("expected 'default_remote: origin', got: %s", out)
	}
	if !strings.Contains(out, "color: true") {
		t.Errorf("expected 'color: true', got: %s", out)
	}
}

func TestConfigGetCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	// 1. Get existing keys
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	err := RunConfigGet(ctx, outBuf, errBuf, "", "ui.theme")
	if err != nil || strings.TrimSpace(outBuf.String()) != "default" {
		t.Errorf("expected 'default', got: %q, err: %v", outBuf.String(), err)
	}

	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigGet(ctx, outBuf, errBuf, "", "git.default_remote")
	if err != nil || strings.TrimSpace(outBuf.String()) != "origin" {
		t.Errorf("expected 'origin', got: %q, err: %v", outBuf.String(), err)
	}

	// 2. Get unknown key
	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigGet(ctx, outBuf, errBuf, "", "unknown.key")
	if err == nil {
		t.Fatal("expected error getting unknown key, got nil")
	}
	if !strings.Contains(errBuf.String(), "Unknown configuration key: unknown.key") {
		t.Errorf("expected unknown key error message, got: %s", errBuf.String())
	}
}

func TestConfigSetAndResetGlobalCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// 1. Set global key
	err := RunConfigSet(ctx, outBuf, errBuf, "", "ui.theme", "dracula", false)
	if err != nil {
		t.Fatalf("expected Set success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "ui.theme = dracula") {
		t.Errorf("expected confirmation message, got: %s", outBuf.String())
	}

	// Verify get returns new value
	outBuf.Reset()
	errBuf.Reset()
	_ = RunConfigGet(ctx, outBuf, errBuf, "", "ui.theme")
	if strings.TrimSpace(outBuf.String()) != "dracula" {
		t.Errorf("expected 'dracula', got: %q", outBuf.String())
	}

	// 2. Reset key
	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigReset(ctx, outBuf, errBuf, "", "ui.theme", false)
	if err != nil {
		t.Fatalf("expected Reset success, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Reset ui.theme to default") {
		t.Errorf("expected reset message, got: %s", outBuf.String())
	}

	// Verify get returns default
	outBuf.Reset()
	errBuf.Reset()
	_ = RunConfigGet(ctx, outBuf, errBuf, "", "ui.theme")
	if strings.TrimSpace(outBuf.String()) != "default" {
		t.Errorf("expected 'default' after reset, got: %q", outBuf.String())
	}
}

func TestConfigSetAndResetLocalCmd(t *testing.T) {
	dir := setupTestGitRepo(t)
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// 1. Set local key
	err := RunConfigSet(ctx, outBuf, errBuf, dir, "git.default_remote", "upstream", true)
	if err != nil {
		t.Fatalf("expected local Set success, got: %v\nstderr: %s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "git.default_remote = upstream") {
		t.Errorf("expected confirmation message, got: %s", outBuf.String())
	}

	// Verify get inside repo returns local override
	outBuf.Reset()
	errBuf.Reset()
	_ = RunConfigGet(ctx, outBuf, errBuf, dir, "git.default_remote")
	if strings.TrimSpace(outBuf.String()) != "upstream" {
		t.Errorf("expected 'upstream', got: %q", outBuf.String())
	}

	// Verify get outside repo returns global default
	outBuf.Reset()
	errBuf.Reset()
	_ = RunConfigGet(ctx, outBuf, errBuf, tmpDir, "git.default_remote")
	if strings.TrimSpace(outBuf.String()) != "origin" {
		t.Errorf("expected 'origin' outside repo, got: %q", outBuf.String())
	}

	// 2. Reset local key
	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigReset(ctx, outBuf, errBuf, dir, "git.default_remote", true)
	if err != nil {
		t.Fatalf("expected local Reset success, got: %v", err)
	}
	if !strings.Contains(outBuf.String(), "Removed local override for git.default_remote") {
		t.Errorf("expected removed local override message, got: %s", outBuf.String())
	}
}

func TestConfigLocalOutsideRepo(t *testing.T) {
	ctx := context.Background()
	emptyDir := t.TempDir()

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// List --local outside repo
	err := RunConfigList(ctx, outBuf, errBuf, emptyDir, ConfigCmdFlags{Local: true})
	if err == nil {
		t.Fatal("expected error for list --local outside repo, got nil")
	}
	if !strings.Contains(errBuf.String(), notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errBuf.String())
	}

	// Set --local outside repo
	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigSet(ctx, outBuf, errBuf, emptyDir, "ui.theme", "dark", true)
	if err == nil {
		t.Fatal("expected error for set --local outside repo, got nil")
	}
	if !strings.Contains(errBuf.String(), notRepoMessage) {
		t.Errorf("expected not repo message, got: %s", errBuf.String())
	}
}

func TestConfigSetInvalidValue(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", filepath.Join(tmpDir, "global"))

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Invalid boolean
	err := RunConfigSet(ctx, outBuf, errBuf, "", "output.color", "not-a-boolean", false)
	if err == nil {
		t.Fatal("expected error setting invalid boolean, got nil")
	}
	if !strings.Contains(errBuf.String(), "expected true or false") {
		t.Errorf("expected 'expected true or false' error, got: %s", errBuf.String())
	}

	// Unknown key
	outBuf.Reset()
	errBuf.Reset()
	err = RunConfigSet(ctx, outBuf, errBuf, "", "custom.unknown", "value", false)
	if err == nil {
		t.Fatal("expected error setting unknown key, got nil")
	}
	if !strings.Contains(errBuf.String(), "Unknown configuration key") {
		t.Errorf("expected unknown key error, got: %s", errBuf.String())
	}
}

func TestConfigInvalidYAMLCli(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	badFile := filepath.Join(tmpDir, "config.yaml")
	_ = os.WriteFile(badFile, []byte("ui: [broken-yaml"), 0644)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunConfigList(ctx, outBuf, errBuf, "", ConfigCmdFlags{})
	if err == nil {
		t.Fatal("expected error reading invalid YAML, got nil")
	}

	errOut := errBuf.String()
	if !strings.Contains(errOut, "Invalid DevGit configuration") {
		t.Errorf("expected invalid config error, got: %s", errOut)
	}
	if !strings.Contains(errOut, badFile) {
		t.Errorf("expected error to contain bad file path, got: %s", errOut)
	}

	// Verify file is preserved
	content, _ := os.ReadFile(badFile)
	if string(content) != "ui: [broken-yaml" {
		t.Errorf("expected invalid file not to be overwritten, got: %s", string(content))
	}
}

func TestConfigResetUnknownKey(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	err := RunConfigReset(ctx, outBuf, errBuf, "", "invalid.unknown", false)
	if err == nil {
		t.Fatal("expected error resetting unknown key, got nil")
	}
	if !strings.Contains(errBuf.String(), "Unknown configuration key") {
		t.Errorf("expected unknown key error, got: %s", errBuf.String())
	}
}

func TestConfigCobraWiring(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	// Test config get via root command
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"config", "get", "ui.theme"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful execution of 'config get ui.theme', got: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "default" {
		t.Errorf("expected 'default', got: %q", buf.String())
	}

	// Test config set via root command
	cmd = NewRootCmd()
	buf.Reset()
	errBuf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"config", "set", "ui.theme", "dracula"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful execution of 'config set ui.theme dracula', got: %v", err)
	}
	if !strings.Contains(buf.String(), "ui.theme = dracula") {
		t.Errorf("expected confirmation output, got: %s", buf.String())
	}

	// Test config reset via root command
	cmd = NewRootCmd()
	buf.Reset()
	errBuf.Reset()
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"config", "reset", "ui.theme"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("expected successful execution of 'config reset ui.theme', got: %v", err)
	}
	if !strings.Contains(buf.String(), "Reset ui.theme to default") {
		t.Errorf("expected reset output, got: %s", buf.String())
	}
}
