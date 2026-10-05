package config

// DefaultConfig returns a new Config instance initialized with standard DevGit defaults.
func DefaultConfig() Config {
	return Config{
		UI: UIConfig{
			Theme: "default",
		},
		Git: GitConfig{
			DefaultRemote: "origin",
			ProtectedBranches: []string{
				"main",
				"master",
				"production",
				"prod",
			},
		},
		Output: OutputConfig{
			Color: true,
		},
	}
}

// DefaultConfigPtr returns a pointer to a new default Config instance.
func DefaultConfigPtr() *Config {
	c := DefaultConfig()
	return &c
}
