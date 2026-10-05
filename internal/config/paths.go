package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// GlobalConfigFileName is the standard configuration file name.
	GlobalConfigFileName = "config.yaml"

	// LocalConfigFileName is the repo-specific configuration file inside .git.
	LocalConfigFileName = "devgit.yaml"
)

// GlobalConfigDir returns the platform-specific directory where global DevGit config is stored.
func GlobalConfigDir() (string, error) {
	if custom := os.Getenv("DEVGIT_CONFIG_DIR"); custom != "" {
		return custom, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		// Fallback to ~/.config/devgit if UserConfigDir fails
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("unable to determine user configuration directory: %w", err)
		}
		return filepath.Join(home, ".config", "devgit"), nil
	}

	return filepath.Join(base, "devgit"), nil
}

// GlobalConfigPath returns the full path to the global configuration file.
func GlobalConfigPath() (string, error) {
	dir, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, GlobalConfigFileName), nil
}

// FindGitDir searches for a .git directory starting from startDir and walking upwards.
func FindGitDir(startDir string) (string, error) {
	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		startDir = wd
	}

	curr := filepath.Clean(startDir)
	for {
		candidate := filepath.Join(curr, ".git")
		if fi, err := os.Stat(candidate); err == nil {
			if fi.IsDir() {
				return candidate, nil
			}
			// If .git is a file (e.g. worktree or submodule), return directory
			return candidate, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return "", ErrNotGitRepository
}

// LocalConfigPath returns the path to the repository-local configuration file (.git/devgit.yaml).
func LocalConfigPath(workDir string) (string, error) {
	gitDir, err := FindGitDir(workDir)
	if err != nil {
		return "", ErrNotGitRepository
	}

	// If gitDir is a file (git worktree), use the directory containing the file or its parent
	fi, err := os.Stat(gitDir)
	if err == nil && !fi.IsDir() {
		return filepath.Join(filepath.Dir(gitDir), LocalConfigFileName), nil
	}

	return filepath.Join(gitDir, LocalConfigFileName), nil
}
