package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

const notRepoMessage = "❌ Not a Git repository.\n\nRun devgit inside a Git repository."

// NewStatusCmd creates and returns the 'status' command.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the working tree status",
		Long: `Show the working tree status, including repository root, active branch,
and staged, unstaged, or untracked file changes.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStatus(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	return cmd
}

// RunStatus executes the status workflow against the specified working directory.
// If workDir is empty, the current working directory is used.
func RunStatus(ctx context.Context, out, errOut io.Writer, workDir string) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	status, err := client.Status(ctx)
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		fmt.Fprintf(errOut, "❌ Git operation failed: %v\n", err)
		return err
	}

	RenderStatus(out, status)
	return nil
}

// RenderStatus formats and writes the repository status to the provided writer.
func RenderStatus(w io.Writer, status *git.RepositoryStatus) {
	fmt.Fprintf(w, "Repository: %s\n", status.Root)
	fmt.Fprintf(w, "Branch: %s\n", status.Branch)

	if status.IsClean() {
		fmt.Fprintln(w, "\nNo changes (working tree clean)")
		return
	}

	fmt.Fprintf(w, "\nChanges:\n\n")
	for _, f := range status.Files {
		fmt.Fprintf(w, "%s %s\n", f.Status, f.DisplayPath())
	}
}
