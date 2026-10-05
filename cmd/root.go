package cmd

import (
	"os"
	"time"

	"github.com/animesh67samanta/devgit/internal/version"
	"github.com/spf13/cobra"
)

// Version represents the current version of devgit.
var Version = version.Version

// NewRootCmd returns a new instance of the root cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devgit",
		Short: "A safer, developer-friendly Git assistant",
		Long: `devgit is a developer-friendly Git assistant that provides both
a powerful CLI and an interactive terminal UI on top of the Git CLI.

The goal is not to replace Git, but to provide a safer, easier,
and more productive interface for daily development workflows.`,
		Version: Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			// In Phase 9, this will launch the interactive Bubble Tea terminal UI.
			// For Phase 1, display usage and guidance.
			return cmd.Help()
		},
	}

	var colorFlag, noColorFlag bool
	cmd.PersistentFlags().BoolVar(&colorFlag, "color", false, "Force color output")
	cmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable color output")

	var startTime time.Time
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		startTime = time.Now()
		if noColorFlag {
			_ = os.Setenv("NO_COLOR", "1")
		} else if colorFlag {
			_ = os.Unsetenv("NO_COLOR")
		}
		return nil
	}

	cmd.PersistentPostRun = func(c *cobra.Command, args []string) {
		duration := time.Since(startTime)
		cmdName := c.Name()
		if c.Parent() != nil && c.Parent().Name() != "devgit" {
			cmdName = c.Parent().Name() + " " + cmdName
		}
		if cmdName != "" && cmdName != "devgit" && cmdName != "help" {
			RecordCommandExecutionSafe(cmdName, duration, true, "")
		}
	}

	cmd.SetVersionTemplate("devgit version {{.Version}}\n")
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewDiffCmd())
	cmd.AddCommand(NewLogCmd())
	cmd.AddCommand(NewCommitCmd())
	cmd.AddCommand(NewPushCmd())
	cmd.AddCommand(NewPullCmd())
	cmd.AddCommand(NewBranchCmd())
	cmd.AddCommand(NewStashCmd())
	cmd.AddCommand(NewTuiCmd())
	cmd.AddCommand(NewConfigCmd())
	cmd.AddCommand(NewHistoryCmd())
	cmd.AddCommand(NewDBCmd())
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewCompletionCmd())
	return cmd
}

// GetColorOverride returns a boolean pointer if --color or --no-color was explicitly specified.
func GetColorOverride(cmd *cobra.Command) *bool {
	if cmd != nil {
		if cmd.Flags().Changed("no-color") {
			f := false
			return &f
		}
		if cmd.Flags().Changed("color") {
			t := true
			return &t
		}
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
