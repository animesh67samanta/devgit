package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

// LogCmdFlags holds command-line flags for devgit log.
type LogCmdFlags struct {
	Limit  int
	All    bool
	Branch string
}

// NewLogCmd creates and returns the 'log' command.
func NewLogCmd() *cobra.Command {
	var flags LogCmdFlags

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commit history",
		Long: `Show readable commit history with formatted hashes, author details,
and relative commit timestamps.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunLog(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}

	cmd.Flags().IntVarP(&flags.Limit, "limit", "n", 20, "Number of commits to show (0 for unlimited)")
	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Show commits from all branches")
	cmd.Flags().StringVarP(&flags.Branch, "branch", "b", "", "Show commits from a specific branch")

	return cmd
}

// RunLog executes the log workflow and outputs formatted commits.
func RunLog(ctx context.Context, out, errOut io.Writer, workDir string, flags LogCmdFlags) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	commits, err := client.Log(ctx, git.LogOptions{
		Limit:  flags.Limit,
		All:    flags.All,
		Branch: flags.Branch,
	})
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		fmt.Fprintf(errOut, "❌ Git operation failed: %v\n", err)
		return err
	}

	RenderLog(out, commits)
	return nil
}

// RenderLog formats and displays the commit history.
func RenderLog(w io.Writer, commits []git.Commit) {
	fmt.Fprintln(w, "Commit History")
	fmt.Fprintln(w)

	if len(commits) == 0 {
		fmt.Fprintln(w, "No commits yet.")
		return
	}

	for i, c := range commits {
		fmt.Fprintf(w, "%-7s  %s\n", c.ShortHash, c.Subject)
		fmt.Fprintf(w, "         %s · %s\n", c.AuthorName, c.RelativeDate)
		if i < len(commits)-1 {
			fmt.Fprintln(w)
		}
	}
}
