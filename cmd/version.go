package cmd

import (
	"fmt"
	"io"

	"github.com/animesh67samanta/devgit/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates and returns the 'version' subcommand.
func NewVersionCmd() *cobra.Command {
	var shortFlag bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display application version and build information",
		Long: `Print detailed application version information, including the release
semantic version, git commit SHA, build date, and target platform.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			RunVersion(cmd.OutOrStdout(), shortFlag)
		},
	}

	cmd.Flags().BoolVarP(&shortFlag, "short", "s", false, "Display short version format")
	return cmd
}

// RunVersion writes formatted version information to out.
func RunVersion(out io.Writer, short bool) {
	if short {
		fmt.Fprintln(out, version.Short())
	} else {
		fmt.Fprintln(out, version.Info())
	}
}
