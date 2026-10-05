package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.UI.Theme != "default" {
		t.Errorf("expected default theme 'default', got %q", cfg.UI.Theme)
	}
	if cfg.Git.DefaultRemote != "origin" {
		t.Errorf("expected default remote 'origin', got %q", cfg.Git.DefaultRemote)
	}
	if len(cfg.Git.ProtectedBranches) != 4 {
		t.Errorf("expected 4 default protected branches, got %d", len(cfg.Git.ProtectedBranches))
	}
	if !cfg.Output.Color {
		t.Errorf("expected default color to be true")
	}
}

func TestConfigGetAndSet(t *testing.T) {
	cfg := DefaultConfig()

	// Get existing keys
	val, err := cfg.Get("ui.theme")
	if err != nil || val != "default" {
		t.Errorf("expected 'default', got %q, err: %v", val, err)
	}

	val, err = cfg.Get("output.color")
	if err != nil || val != "true" {
		t.Errorf("expected 'true', got %q, err: %v", val, err)
	}

	// Unknown key
	_, err = cfg.Get("invalid.key")
	if !errors.Is(err, ErrUnknownKey) {
		t.Errorf("expected ErrUnknownKey, got: %v", err)
	}

	// Set valid keys
	err = cfg.Set("ui.theme", "nord")
	if err != nil || cfg.UI.Theme != "nord" {
		t.Errorf("failed to set ui.theme: %v", err)
	}

	err = cfg.Set("output.color", "false")
	if err != nil || cfg.Output.Color != false {
		t.Errorf("failed to set output.color: %v", err)
	}

	err = cfg.Set("git.protected_branches", "main, dev, release")
	if err != nil || len(cfg.Git.ProtectedBranches) != 3 {
		t.Errorf("failed to set protected_branches: %v", err)
	}

	// Set invalid value
	err = cfg.Set("output.color", "not_a_bool")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue for bool, got: %v", err)
	}

	// Set unknown key
	err = cfg.Set("foo.bar", "val")
	if !errors.Is(err, ErrUnknownKey) {
		t.Errorf("expected ErrUnknownKey, got: %v", err)
	}
}

func TestIsProtectedBranch(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.IsProtectedBranch("main") {
		t.Error("expected 'main' to be protected")
	}
	if !cfg.IsProtectedBranch("MASTER") {
		t.Error("expected 'MASTER' (case-insensitive) to be protected")
	}
	if !cfg.IsProtectedBranch("production") {
		t.Error("expected 'production' to be protected")
	}
	if !cfg.IsProtectedBranch("prod") {
		t.Error("expected 'prod' to be protected")
	}
	if cfg.IsProtectedBranch("feature/test") {
		t.Error("expected 'feature/test' not to be protected")
	}
}

func TestPaths(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DEVGIT_CONFIG_DIR", tmpDir)

	globalPath, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("unexpected error getting global path: %v", err)
	}
	if !strings.HasPrefix(globalPath, tmpDir) {
		t.Errorf("expected global path inside temp dir %s, got %s", tmpDir, globalPath)
	}

	// Outside git repo
	_, err = LocalConfigPath(tmpDir)
	if !errors.Is(err, ErrNotGitRepository) {
		t.Errorf("expected ErrNotGitRepository, got: %v", err)
	}

	// Inside simulated git repo
	gitDir := filepath.Join(tmpDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)
	localPath, err := LocalConfigPath(tmpDir)
	if err != nil {
		t.Fatalf("expected local path success, got: %v", err)
	}
	if localPath != filepath.Join(gitDir, "devgit.yaml") {
		t.Errorf("expected %s, got %s", filepath.Join(gitDir, "devgit.yaml"), localPath)
	}
}

func TestLoadPrecedence(t *testing.T) {
	tempBase := t.TempDir()
	globalDir := filepath.Join(tempBase, "global")
	repoDir := filepath.Join(tempBase, "repo")
	gitDir := filepath.Join(repoDir, ".git")
	_ = os.MkdirAll(globalDir, 0755)
	_ = os.MkdirAll(gitDir, 0755)

	globalFile := filepath.Join(globalDir, "config.yaml")
	localFile := filepath.Join(gitDir, "devgit.yaml")

	// 1. Default only
	cfg, err := Load(LoadOptions{
		WorkDir:          repoDir,
		GlobalConfigPath: globalFile,
		SkipEnv:          true,
	})
	if err != nil {
		t.Fatalf("failed to load default config: %v", err)
	}
	if cfg.Git.DefaultRemote != "origin" {
		t.Errorf("expected default 'origin', got %q", cfg.Git.DefaultRemote)
	}

	// 2. Global overrides Default
	_ = os.WriteFile(globalFile, []byte("git:\n  default_remote: global-remote\n"), 0644)
	cfg, err = Load(LoadOptions{
		WorkDir:          repoDir,
		GlobalConfigPath: globalFile,
		SkipEnv:          true,
	})
	if err != nil {
		t.Fatalf("failed to load with global: %v", err)
	}
	if cfg.Git.DefaultRemote != "global-remote" {
		t.Errorf("expected global override 'global-remote', got %q", cfg.Git.DefaultRemote)
	}

	// 3. Local overrides Global
	_ = os.WriteFile(localFile, []byte("git:\n  default_remote: local-remote\n"), 0644)
	cfg, err = Load(LoadOptions{
		WorkDir:          repoDir,
		GlobalConfigPath: globalFile,
		SkipEnv:          true,
	})
	if err != nil {
		t.Fatalf("failed to load with local: %v", err)
	}
	if cfg.Git.DefaultRemote != "local-remote" {
		t.Errorf("expected local override 'local-remote', got %q", cfg.Git.DefaultRemote)
	}

	// 4. Environment overrides Local
	t.Setenv("DEVGIT_GIT_DEFAULT_REMOTE", "env-remote")
	cfg, err = Load(LoadOptions{
		WorkDir:          repoDir,
		GlobalConfigPath: globalFile,
		SkipEnv:          false,
	})
	if err != nil {
		t.Fatalf("failed to load with env: %v", err)
	}
	if cfg.Git.DefaultRemote != "env-remote" {
		t.Errorf("expected env override 'env-remote', got %q", cfg.Git.DefaultRemote)
	}

	// 5. CLI overrides Environment (ColorOverride)
	t.Setenv("DEVGIT_OUTPUT_COLOR", "true")
	cliFalse := false
	cfg, err = Load(LoadOptions{
		WorkDir:          repoDir,
		GlobalConfigPath: globalFile,
		ColorOverride:    &cliFalse,
	})
	if err != nil {
		t.Fatalf("failed to load with CLI override: %v", err)
	}
	if cfg.Output.Color != false {
		t.Errorf("expected CLI override false, got true")
	}
}

func TestSetAndResetKeyAtomic(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "subdir", "config.yaml")

	// Set creates parent directories and file atomically
	err := SetKey(configFile, "ui.theme", "dracula")
	if err != nil {
		t.Fatalf("expected SetKey success, got: %v", err)
	}

	// Verify file content
	raw, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if !strings.Contains(string(raw), "theme: dracula") {
		t.Errorf("expected YAML to contain theme: dracula, got: %s", string(raw))
	}

	// Add second key
	err = SetKey(configFile, "output.color", "false")
	if err != nil {
		t.Fatalf("failed to set second key: %v", err)
	}

	// Reset first key
	err = ResetKey(configFile, "ui.theme")
	if err != nil {
		t.Fatalf("expected ResetKey success, got: %v", err)
	}

	raw, _ = os.ReadFile(configFile)
	if strings.Contains(string(raw), "theme:") {
		t.Errorf("expected theme to be removed, got: %s", string(raw))
	}
	if !strings.Contains(string(raw), "color: false") {
		t.Errorf("expected color to remain, got: %s", string(raw))
	}

	// Reset final key -> removes file
	err = ResetKey(configFile, "output.color")
	if err != nil {
		t.Fatalf("expected ResetKey final success, got: %v", err)
	}
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Errorf("expected config file to be cleaned up after last key removed")
	}
}

func TestInvalidYAMLHandling(t *testing.T) {
	tempDir := t.TempDir()
	badFile := filepath.Join(tempDir, "bad.yaml")
	_ = os.WriteFile(badFile, []byte("git: [invalid-yaml"), 0644)

	_, err := Load(LoadOptions{
		GlobalConfigPath: badFile,
		SkipEnv:          true,
	})
	if err == nil {
		t.Fatal("expected error loading invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid DevGit configuration") {
		t.Errorf("expected friendly invalid config message, got: %v", err)
	}
	if !strings.Contains(err.Error(), badFile) {
		t.Errorf("expected error to cite file path, got: %v", err)
	}

	// Verify file was NOT deleted or overwritten
	content, _ := os.ReadFile(badFile)
	if string(content) != "git: [invalid-yaml" {
		t.Errorf("expected invalid file to remain intact, got: %s", string(content))
	}
}

func TestUnknownEnvironmentVariablesIgnored(t *testing.T) {
	t.Setenv("DEVGIT_UNKNOWN_FIELD", "should_be_ignored")
	t.Setenv("DEVGIT_ANOTHER_RANDOM", "random_value")

	cfg, err := Load(LoadOptions{SkipGlobal: true, SkipLocal: true})
	if err != nil {
		t.Fatalf("unexpected error loading with unknown env vars: %v", err)
	}

	// Defaults remain intact
	if cfg.UI.Theme != "default" || cfg.Git.DefaultRemote != "origin" {
		t.Errorf("expected default values to be unaffected by unknown env vars")
	}
}

func TestValidationErrorsOnEmptyValues(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	// Empty theme
	err := SetKey(configFile, "ui.theme", "   ")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue for empty theme, got: %v", err)
	}

	// Empty remote
	err = SetKey(configFile, "git.default_remote", "")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue for empty remote, got: %v", err)
	}

	// Empty protected branches
	err = SetKey(configFile, "git.protected_branches", "  ,  ")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue for empty protected branches, got: %v", err)
	}

	// Secrets or arbitrary keys rejected
	err = SetKey(configFile, "auth.token", "secret123")
	if !errors.Is(err, ErrUnknownKey) {
		t.Errorf("expected ErrUnknownKey for secret keys, got: %v", err)
	}
}
