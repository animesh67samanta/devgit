package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"devgit/internal/config"
	"github.com/spf13/cobra"
)

// ConfigCmdFlags holds flags for config operations.
type ConfigCmdFlags struct {
	Local  bool
	Global bool
}

// NewConfigCmd creates and returns the 'config' command and its subcommands.
func NewConfigCmd() *cobra.Command {
	var listFlags ConfigCmdFlags

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage DevGit configuration",
		Long: `View, get, set, and reset DevGit configuration settings across
global user preferences and repository-local overrides.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunConfigList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", listFlags)
		},
	}
	configCmd.Flags().BoolVar(&listFlags.Local, "local", false, "Display repository-local configuration only")
	configCmd.Flags().BoolVar(&listFlags.Global, "global", false, "Display global configuration only")

	// Subcommand: devgit config list
	listCmd := &cobra.Command{
		Use:           "list",
		Short:         "List effective configuration settings",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunConfigList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", listFlags)
		},
	}
	listCmd.Flags().BoolVar(&listFlags.Local, "local", false, "Display repository-local configuration only")
	listCmd.Flags().BoolVar(&listFlags.Global, "global", false, "Display global configuration only")

	// Subcommand: devgit config get <key>
	getCmd := &cobra.Command{
		Use:           "get <key>",
		Short:         "Get the value of a configuration key",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Please specify a configuration key (e.g. ui.theme, git.default_remote).")
				return errors.New("missing configuration key")
			}
			return RunConfigGet(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", args[0])
		},
	}

	// Subcommand: devgit config set [--local] <key> <value>
	var setFlags ConfigCmdFlags
	setCmd := &cobra.Command{
		Use:           "set [--local] <key> <value>",
		Short:         "Set a configuration key to a specific value",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Usage: devgit config set [--local] <key> <value>")
				return errors.New("missing arguments for config set")
			}
			key := args[0]
			val := strings.Join(args[1:], " ")
			return RunConfigSet(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", key, val, setFlags.Local)
		},
	}
	setCmd.Flags().BoolVar(&setFlags.Local, "local", false, "Set configuration in repository-local scope (.git/devgit.yaml)")

	// Subcommand: devgit config reset [--local] <key>
	var resetFlags ConfigCmdFlags
	resetCmd := &cobra.Command{
		Use:           "reset [--local] <key>",
		Short:         "Reset a configuration key back to its default value",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Please specify a configuration key to reset.")
				return errors.New("missing key for config reset")
			}
			return RunConfigReset(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", args[0], resetFlags.Local)
		},
	}
	resetCmd.Flags().BoolVar(&resetFlags.Local, "local", false, "Reset configuration key in repository-local scope")

	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(resetCmd)

	return configCmd
}

// RunConfigList formats and displays configuration settings.
func RunConfigList(ctx context.Context, out, errOut io.Writer, workDir string, flags ConfigCmdFlags) error {
	if flags.Local {
		_, err := config.LocalConfigPath(workDir)
		if err != nil {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
	}

	loadOpts := config.LoadOptions{
		WorkDir:    workDir,
		SkipLocal:  flags.Global,
		SkipGlobal: flags.Local,
	}

	cfg, err := config.Load(loadOpts)
	if err != nil {
		fmt.Fprintf(errOut, "❌ %v\n", err)
		return err
	}

	fmt.Fprintln(out, "DevGit Configuration")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "UI")
	fmt.Fprintf(out, "  theme: %s\n", cfg.UI.Theme)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Git")
	fmt.Fprintf(out, "  default_remote: %s\n", cfg.Git.DefaultRemote)
	fmt.Fprintln(out, "  protected_branches:")
	for _, b := range cfg.Git.ProtectedBranches {
		fmt.Fprintf(out, "    %s\n", b)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Output")
	fmt.Fprintf(out, "  color: %t\n", cfg.Output.Color)

	return nil
}

// RunConfigGet retrieves and outputs the value of a specific configuration key.
func RunConfigGet(ctx context.Context, out, errOut io.Writer, workDir, key string) error {
	cfg, err := config.Load(config.LoadOptions{WorkDir: workDir})
	if err != nil {
		fmt.Fprintf(errOut, "❌ %v\n", err)
		return err
	}

	val, err := cfg.Get(key)
	if err != nil {
		if errors.Is(err, config.ErrUnknownKey) {
			fmt.Fprintf(errOut, "❌ Unknown configuration key: %s\n", key)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to get %s: %v\n", key, err)
		return err
	}

	fmt.Fprintln(out, val)
	return nil
}

// RunConfigSet updates a configuration setting in either global or local scope.
func RunConfigSet(ctx context.Context, out, errOut io.Writer, workDir, key, value string, local bool) error {
	var targetFile string
	if local {
		path, err := config.LocalConfigPath(workDir)
		if err != nil {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		targetFile = path
	} else {
		path, err := config.GlobalConfigPath()
		if err != nil {
			fmt.Fprintf(errOut, "❌ Unable to determine configuration directory: %v\n", err)
			return err
		}
		targetFile = path
	}

	err := config.SetKey(targetFile, key, value)
	if err != nil {
		if errors.Is(err, config.ErrUnknownKey) {
			fmt.Fprintf(errOut, "❌ Unknown configuration key: %s\n", key)
			return err
		}
		if errors.Is(err, config.ErrInvalidValue) {
			msg := err.Error()
			if strings.HasPrefix(strings.ToLower(msg), "invalid configuration value for ") {
				msg = "Invalid value for " + msg[len("invalid configuration value for "):]
			}
			if !strings.HasSuffix(msg, ".") {
				msg += "."
			}
			fmt.Fprintf(errOut, "❌ %s\n", msg)
			return err
		}
		fmt.Fprintf(errOut, "❌ Unable to write configuration file.\n\nReason: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "✓ Configuration updated.")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s = %s\n", key, value)
	return nil
}

// RunConfigReset removes a configuration override in either global or local scope.
func RunConfigReset(ctx context.Context, out, errOut io.Writer, workDir, key string, local bool) error {
	var targetFile string
	if local {
		path, err := config.LocalConfigPath(workDir)
		if err != nil {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		targetFile = path
	} else {
		path, err := config.GlobalConfigPath()
		if err != nil {
			fmt.Fprintf(errOut, "❌ Unable to determine configuration directory: %v\n", err)
			return err
		}
		targetFile = path
	}

	err := config.ResetKey(targetFile, key)
	if err != nil {
		if errors.Is(err, config.ErrUnknownKey) {
			fmt.Fprintf(errOut, "❌ Unknown configuration key: %s\n", key)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to reset %s: %v\n", key, err)
		return err
	}

	if local {
		fmt.Fprintf(out, "✓ Removed local override for %s.\n", key)
	} else {
		fmt.Fprintf(out, "✓ Reset %s to default.\n", key)
	}
	return nil
}
