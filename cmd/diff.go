package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

// DiffCmdFlags holds the command-line flags for devgit diff.
type DiffCmdFlags struct {
	Staged bool
	Raw    bool
}

// NewDiffCmd creates and returns the 'diff' command.
func NewDiffCmd() *cobra.Command {
	var flags DiffCmdFlags

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show changes in the working tree or staging area",
		Long: `Show changes between the working tree and the index, or between the index
and HEAD when --staged is specified.

Displays a clean summary of modified, added, deleted, and renamed files,
with an optional --raw view for traditional patch output.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDiff(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}

	cmd.Flags().BoolVar(&flags.Staged, "staged", false, "View staged changes instead of unstaged")
	cmd.Flags().BoolVar(&flags.Staged, "cached", false, "Alias for --staged")
	cmd.Flags().BoolVarP(&flags.Raw, "raw", "r", false, "Show raw unified diff output")

	return cmd
}

// RunDiff executes the diff workflow and outputs the result.
func RunDiff(ctx context.Context, out, errOut io.Writer, workDir string, flags DiffCmdFlags) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	if flags.Raw {
		raw, err := client.RawDiff(ctx, flags.Staged)
		if err != nil {
			if errors.Is(err, git.ErrNotRepository) {
				fmt.Fprintln(errOut, notRepoMessage)
				return err
			}
			fmt.Fprintf(errOut, "❌ Git operation failed: %v\n", err)
			return err
		}

		if raw == "" {
			if flags.Staged {
				fmt.Fprintln(out, "No staged changes")
			} else {
				fmt.Fprintln(out, "No changes")
			}
			return nil
		}

		fmt.Fprintln(out, raw)
		return nil
	}

	summary, err := client.DiffSummary(ctx, flags.Staged)
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		fmt.Fprintf(errOut, "❌ Git operation failed: %v\n", err)
		return err
	}

	RenderDiffSummary(out, summary, flags.Staged)
	return nil
}

// RenderDiffSummary formats and writes the structured diff summary.
func RenderDiffSummary(w io.Writer, summary *git.DiffSummary, staged bool) {
	title := "Changes"
	emptyMessage := "No changes"
	if staged {
		title = "Staged Changes"
		emptyMessage = "No staged changes"
	}

	if summary.IsEmpty() {
		fmt.Fprintln(w, emptyMessage)
		return
	}

	fmt.Fprintln(w, title)
	fmt.Fprintln(w, "────────────────────────")
	fmt.Fprintln(w)

	hasPrintedSection := false

	if len(summary.Modified) > 0 {
		fmt.Fprintln(w, "Modified:")
		for _, f := range summary.Modified {
			fmt.Fprintf(w, "  %s\n", f)
		}
		hasPrintedSection = true
	}

	if len(summary.Added) > 0 {
		if hasPrintedSection {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Added:")
		for _, f := range summary.Added {
			fmt.Fprintf(w, "  %s\n", f)
		}
		hasPrintedSection = true
	}

	if len(summary.Deleted) > 0 {
		if hasPrintedSection {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Deleted:")
		for _, f := range summary.Deleted {
			fmt.Fprintf(w, "  %s\n", f)
		}
		hasPrintedSection = true
	}

	if len(summary.Renamed) > 0 {
		if hasPrintedSection {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Renamed:")
		for _, f := range summary.Renamed {
			fmt.Fprintf(w, "  %s\n", f)
		}
		hasPrintedSection = true
	}

	if len(summary.Other) > 0 {
		if hasPrintedSection {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, "Other:")
		for _, f := range summary.Other {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
}
