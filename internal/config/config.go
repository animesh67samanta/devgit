package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	// ErrUnknownKey is returned when referencing an unsupported configuration key.
	ErrUnknownKey = errors.New("unknown configuration key")

	// ErrInvalidValue is returned when a configuration value fails type or format validation.
	ErrInvalidValue = errors.New("invalid configuration value")

	// ErrNotGitRepository is returned when local configuration is requested outside a Git worktree.
	ErrNotGitRepository = errors.New("not a git repository")
)

// SupportedKeys lists all known configuration keys for validation.
var SupportedKeys = []string{
	"ui.theme",
	"git.default_remote",
	"git.protected_branches",
	"output.color",
}

// Config represents the complete effective DevGit configuration.
type Config struct {
	UI     UIConfig     `yaml:"ui"`
	Git    GitConfig    `yaml:"git"`
	Output OutputConfig `yaml:"output"`
}

// UIConfig holds styling and interface settings.
type UIConfig struct {
	Theme string `yaml:"theme"`
}

// GitConfig holds Git behavior and safety settings.
type GitConfig struct {
	DefaultRemote     string   `yaml:"default_remote"`
	ProtectedBranches []string `yaml:"protected_branches"`
}

// OutputConfig holds output rendering preferences.
type OutputConfig struct {
	Color bool `yaml:"color"`
}

// IsValidKey checks whether the provided dotted key is recognized.
func IsValidKey(key string) bool {
	norm := strings.ToLower(strings.TrimSpace(key))
	for _, k := range SupportedKeys {
		if k == norm {
			return true
		}
	}
	return false
}

// IsProtectedBranch returns true if the specified branch matches any configured protected branch.
func (c *Config) IsProtectedBranch(branch string) bool {
	cleanBranch := strings.TrimSpace(branch)
	if cleanBranch == "" {
		return false
	}
	for _, pb := range c.Git.ProtectedBranches {
		if strings.EqualFold(strings.TrimSpace(pb), cleanBranch) {
			return true
		}
	}
	return false
}

// Get returns the value of the given key as a formatted string.
func (c *Config) Get(key string) (string, error) {
	norm := strings.ToLower(strings.TrimSpace(key))
	switch norm {
	case "ui.theme":
		return c.UI.Theme, nil
	case "git.default_remote":
		return c.Git.DefaultRemote, nil
	case "git.protected_branches":
		return strings.Join(c.Git.ProtectedBranches, ", "), nil
	case "output.color":
		return strconv.FormatBool(c.Output.Color), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
}

// Set parses, validates, and sets the value for the given key in this Config instance.
func (c *Config) Set(key, value string) error {
	norm := strings.ToLower(strings.TrimSpace(key))
	val := strings.TrimSpace(value)

	switch norm {
	case "ui.theme":
		if val == "" {
			return fmt.Errorf("%w for %s: theme cannot be empty", ErrInvalidValue, key)
		}
		c.UI.Theme = val
		return nil

	case "git.default_remote":
		if val == "" {
			return fmt.Errorf("%w for %s: default_remote cannot be empty", ErrInvalidValue, key)
		}
		c.Git.DefaultRemote = val
		return nil

	case "git.protected_branches":
		var branches []string
		var rawList []string
		if strings.Contains(val, ",") {
			rawList = strings.Split(val, ",")
		} else {
			rawList = strings.Fields(val)
		}
		for _, b := range rawList {
			tb := strings.TrimSpace(b)
			if tb != "" {
				branches = append(branches, tb)
			}
		}
		if len(branches) == 0 {
			return fmt.Errorf("%w for %s: expected at least one branch name", ErrInvalidValue, key)
		}
		c.Git.ProtectedBranches = branches
		return nil

	case "output.color":
		b, err := parseBool(val)
		if err != nil {
			return fmt.Errorf("%w for %s: expected true or false", ErrInvalidValue, key)
		}
		c.Output.Color = b
		return nil

	default:
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
}

func parseBool(val string) (bool, error) {
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return false, ErrInvalidValue
	}
}
