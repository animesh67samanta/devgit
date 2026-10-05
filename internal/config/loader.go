package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadOptions configures configuration discovery and layering.
type LoadOptions struct {
	WorkDir          string
	GlobalConfigPath string
	LocalConfigPath  string
	SkipGlobal       bool
	SkipLocal        bool
	SkipEnv          bool
	ColorOverride    *bool
}

// partialConfig uses pointers to differentiate unset fields from zero values.
type partialConfig struct {
	UI struct {
		Theme *string `yaml:"theme"`
	} `yaml:"ui"`
	Git struct {
		DefaultRemote     *string  `yaml:"default_remote"`
		ProtectedBranches []string `yaml:"protected_branches"`
	} `yaml:"git"`
	Output struct {
		Color *bool `yaml:"color"`
	} `yaml:"output"`
}

// Load loads the effective configuration by layering defaults, global file, local file,
// environment variables, and CLI overrides according to precedence rules.
func Load(opts LoadOptions) (*Config, error) {
	cfg := DefaultConfig()

	// 1. Global configuration file
	if !opts.SkipGlobal {
		globalPath := opts.GlobalConfigPath
		if globalPath == "" {
			var err error
			globalPath, err = GlobalConfigPath()
			if err != nil {
				return nil, err
			}
		}

		if err := mergeConfigFile(&cfg, globalPath); err != nil {
			return nil, err
		}
	}

	// 2. Local repository configuration file (.git/devgit.yaml)
	if !opts.SkipLocal {
		localPath := opts.LocalConfigPath
		if localPath == "" && opts.WorkDir != "" {
			lp, err := LocalConfigPath(opts.WorkDir)
			if err == nil {
				localPath = lp
			}
		} else if localPath == "" {
			lp, err := LocalConfigPath("")
			if err == nil {
				localPath = lp
			}
		}

		if localPath != "" {
			if err := mergeConfigFile(&cfg, localPath); err != nil {
				return nil, err
			}
		}
	}

	// 3. Environment variable overrides (DEVGIT_*)
	if !opts.SkipEnv {
		applyEnvironmentOverrides(&cfg)
	}

	// 4. CLI flag overrides
	if opts.ColorOverride != nil {
		cfg.Output.Color = *opts.ColorOverride
	}

	return &cfg, nil
}

// mergeConfigFile merges a YAML configuration file into the target Config if the file exists.
func mergeConfigFile(target *Config, filePath string) error {
	if filePath == "" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read configuration file %s: %w", filePath, err)
	}

	var p partialConfig
	if err := yaml.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("Invalid DevGit configuration.\n\nFile:\n%s\n\nReason:\n%w", filePath, err)
	}

	if p.UI.Theme != nil {
		target.UI.Theme = *p.UI.Theme
	}
	if p.Git.DefaultRemote != nil {
		target.Git.DefaultRemote = *p.Git.DefaultRemote
	}
	if p.Git.ProtectedBranches != nil {
		target.Git.ProtectedBranches = p.Git.ProtectedBranches
	}
	if p.Output.Color != nil {
		target.Output.Color = *p.Output.Color
	}

	return nil
}

// applyEnvironmentOverrides updates Config with explicitly recognized DEVGIT_* environment variables.
func applyEnvironmentOverrides(c *Config) {
	if theme := os.Getenv("DEVGIT_UI_THEME"); theme != "" {
		c.UI.Theme = theme
	}
	if remote := os.Getenv("DEVGIT_GIT_DEFAULT_REMOTE"); remote != "" {
		c.Git.DefaultRemote = remote
	}
	if branches := os.Getenv("DEVGIT_GIT_PROTECTED_BRANCHES"); branches != "" {
		var list []string
		for _, b := range strings.Split(branches, ",") {
			tb := strings.TrimSpace(b)
			if tb != "" {
				list = append(list, tb)
			}
		}
		if len(list) > 0 {
			c.Git.ProtectedBranches = list
		}
	}
	if color := os.Getenv("DEVGIT_OUTPUT_COLOR"); color != "" {
		if b, err := parseBool(color); err == nil {
			c.Output.Color = b
		}
	}
}

// SetKey updates a configuration key in the specified file, performing atomic write.
func SetKey(filePath, key, value string) error {
	if !IsValidKey(key) {
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}

	norm := strings.ToLower(strings.TrimSpace(key))
	var typedVal interface{}

	switch norm {
	case "ui.theme":
		v := strings.TrimSpace(value)
		if v == "" {
			return fmt.Errorf("%w for %s: theme cannot be empty", ErrInvalidValue, key)
		}
		typedVal = v

	case "git.default_remote":
		v := strings.TrimSpace(value)
		if v == "" {
			return fmt.Errorf("%w for %s: default_remote cannot be empty", ErrInvalidValue, key)
		}
		typedVal = v

	case "git.protected_branches":
		var list []string
		var rawList []string
		if strings.Contains(value, ",") {
			rawList = strings.Split(value, ",")
		} else {
			rawList = strings.Fields(value)
		}
		for _, b := range rawList {
			tb := strings.TrimSpace(b)
			if tb != "" {
				list = append(list, tb)
			}
		}
		if len(list) == 0 {
			return fmt.Errorf("%w for %s: expected at least one branch name", ErrInvalidValue, key)
		}
		typedVal = list

	case "output.color":
		b, err := parseBool(value)
		if err != nil {
			return fmt.Errorf("%w for %s: expected true or false", ErrInvalidValue, key)
		}
		typedVal = b
	}

	// Read existing YAML map if file exists
	dataMap := make(map[string]map[string]interface{})
	if raw, err := os.ReadFile(filePath); err == nil {
		if err := yaml.Unmarshal(raw, &dataMap); err != nil {
			return fmt.Errorf("Invalid DevGit configuration.\n\nFile:\n%s\n\nReason:\n%w", filePath, err)
		}
	}

	parts := strings.Split(norm, ".")
	section := parts[0]
	prop := parts[1]

	if dataMap[section] == nil {
		dataMap[section] = make(map[string]interface{})
	}
	dataMap[section][prop] = typedVal

	out, err := yaml.Marshal(dataMap)
	if err != nil {
		return fmt.Errorf("failed to encode configuration: %w", err)
	}

	return atomicWriteFile(filePath, out)
}

// ResetKey removes a key from the specified configuration file.
func ResetKey(filePath, key string) error {
	if !IsValidKey(key) {
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	dataMap := make(map[string]map[string]interface{})
	if err := yaml.Unmarshal(raw, &dataMap); err != nil {
		return fmt.Errorf("Invalid DevGit configuration.\n\nFile:\n%s\n\nReason:\n%w", filePath, err)
	}

	parts := strings.Split(strings.ToLower(strings.TrimSpace(key)), ".")
	section := parts[0]
	prop := parts[1]

	if dataMap[section] != nil {
		delete(dataMap[section], prop)
		if len(dataMap[section]) == 0 {
			delete(dataMap, section)
		}
	}

	if len(dataMap) == 0 {
		// Entire file is now empty of overrides; remove it cleanly
		_ = os.Remove(filePath)
		return nil
	}

	out, err := yaml.Marshal(dataMap)
	if err != nil {
		return fmt.Errorf("failed to encode configuration: %w", err)
	}

	return atomicWriteFile(filePath, out)
}

// atomicWriteFile safely writes data to a temporary file in the target directory and renames it.
func atomicWriteFile(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to write configuration directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("unable to write configuration file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("unable to write configuration file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("unable to write configuration file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("unable to write configuration file: %w", err)
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		return fmt.Errorf("unable to write configuration file: %w", err)
	}

	return nil
}
