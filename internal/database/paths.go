package database

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// DefaultDBFileName is the standard SQLite database file name for DevGit metadata.
	DefaultDBFileName = "devgit.db"
)

// DataDir returns the platform-specific directory where DevGit stores application metadata.
func DataDir() (string, error) {
	if custom := os.Getenv("DEVGIT_DATA_DIR"); custom != "" {
		return custom, nil
	}

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "devgit"), nil

	case "windows":
		appData := os.Getenv("AppData")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("unable to determine user home directory: %w", err)
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "devgit"), nil

	default: // Linux and other Unix systems
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "devgit"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "devgit"), nil
	}
}

// DBPath returns the resolved path to the DevGit SQLite database file.
// If DEVGIT_DB_PATH is set in the environment, it takes precedence.
func DBPath() (string, error) {
	if custom := os.Getenv("DEVGIT_DB_PATH"); custom != "" {
		return custom, nil
	}

	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultDBFileName), nil
}
