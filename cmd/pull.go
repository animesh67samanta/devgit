package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

// PullCmdFlags holds command-line flags for devgit pull.
type PullCmdFlags struct {
	Remote string
	Branch string
	Yes    bool
}

// NewPullCmd creates and returns the 'pull' command.
func NewPullCmd() *cobra.Command {
	var flags PullCmdFlags

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch from and integrate with another repository safely",
		Long: `Fetch from and update the local branch from its upstream tracking branch.

Enforces fast-forward only (--ff-only) by default to prevent unexpected merges
or rebases, detects dirty working trees, and requires explicit confirmation.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPull(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}

	cmd.Flags().StringVar(&flags.Remote, "remote", "", "Target remote name")
	cmd.Flags().StringVar(&flags.Branch, "branch", "", "Branch to pull")
	cmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Confirm pull without prompting")

	return cmd
}

// RunPull executes the pull workflow with safety checks.
func RunPull(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags PullCmdFlags) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	// 1. Verify Git repository
	inside, err := client.IsInsideWorkTree(ctx)
	if err != nil || !inside {
		fmt.Fprintln(errOut, notRepoMessage)
		return git.ErrNotRepository
	}

	// 2. Detect current branch
	branch := flags.Branch
	if branch == "" {
		current, err := client.CurrentBranch(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "❌ Failed to detect current branch: %v\n", err)
			return err
		}
		branch = current
	}

	if strings.Contains(branch, "HEAD detached") {
		fmt.Fprintln(errOut, "❌ Cannot pull to detached HEAD.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Create or switch to a branch first.")
		return git.ErrDetachedHEAD
	}

	// 3. Check merge / rebase in progress
	repoState, err := client.RepositoryState(ctx)
	if err == nil {
		if repoState.IsMerge {
			fmt.Fprintln(errOut, "❌ A Git operation is already in progress.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Current state:")
			fmt.Fprintln(errOut, "  Merge in progress")
			return git.ErrOperationInProgress
		} else if repoState.IsRebase {
			fmt.Fprintln(errOut, "❌ A Git operation is already in progress.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Current state:")
			fmt.Fprintln(errOut, "  Rebase in progress")
			return git.ErrOperationInProgress
		}
	}

	// 4. Detect remotes
	remotes, err := client.Remotes(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to check remotes: %v\n", err)
		return err
	}

	if len(remotes) == 0 {
		fmt.Fprintln(errOut, "❌ No Git remote is configured.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Add a remote first:")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "git remote add origin <url>")
		return git.ErrNoRemote
	}

	scanner := bufio.NewScanner(in)

	// 5. Detect upstream tracking
	info, err := client.TrackingInfo(ctx, branch)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to check tracking info: %v\n", err)
		return err
	}

	if info == nil || !info.HasUpstream() {
		fmt.Fprintln(errOut, "❌ Current branch has no upstream tracking branch.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Set upstream first:")
		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "git branch --set-upstream-to=origin/%s\n", branch)
		return git.ErrNoUpstream
	}

	targetRemote := flags.Remote
	if targetRemote == "" {
		targetRemote = info.Remote
		if targetRemote == "" {
			targetRemote = remotes[0].Name
		}
	}

	// 6. Check dirty working tree
	status, err := client.Status(ctx)
	if err == nil && !status.IsClean() {
		fmt.Fprintln(out, "⚠ Your working tree contains local changes.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Modified:")
		for _, f := range status.Files {
			fmt.Fprintf(out, "  %s %s\n", f.Status, f.DisplayPath())
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Pulling may cause conflicts.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "DevGit will not automatically stash your changes.")
		fmt.Fprintln(out)

		if !flags.Yes {
			fmt.Fprint(out, "Continue? [y/N] ")
			if !scanner.Scan() {
				fmt.Fprintln(out, "\nOperation cancelled.")
				return nil
			}
			ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if ans != "y" && ans != "yes" {
				fmt.Fprintln(out, "Operation cancelled.")
				return nil
			}
		}
	}

	// 7. Fetch remote objects to ensure tracking info is fresh
	_ = client.Fetch(ctx, targetRemote)
	info, _ = client.TrackingInfo(ctx, branch)

	// 8. Check already up to date
	if info != nil && info.IsUpToDate() {
		fmt.Fprintln(out, "✓ Already up to date.")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s matches %s.\n", branch, info.Upstream)
		return nil
	}

	// 9. Check diverged branches (fast-forward impossible)
	if info != nil && info.IsDiverged() {
		fmt.Fprintln(errOut, "❌ Fast-forward pull is not possible.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "The local and remote branches have diverged.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "DevGit will not automatically merge or rebase.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Inspect the changes and choose an appropriate strategy manually.")
		return git.ErrFastForwardNotPossible
	}

	// 10. Pull confirmation
	fmt.Fprintf(out, "Branch: %s\n", branch)
	fmt.Fprintf(out, "Upstream: %s\n", info.Upstream)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Status:")
	fmt.Fprintf(out, "  ↑ %d commits ahead\n", info.Ahead)
	fmt.Fprintf(out, "  ↓ %d commits behind\n", info.Behind)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Pull will update your local branch.")
	fmt.Fprintln(out)

	if !flags.Yes {
		fmt.Fprint(out, "Continue? [y/N] ")
		if !scanner.Scan() {
			fmt.Fprintln(out, "\nOperation cancelled.")
			return nil
		}
		ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if ans != "y" && ans != "yes" {
			fmt.Fprintln(out, "Operation cancelled.")
			return nil
		}
	}

	// 11. Execute pull
	_, err = client.Pull(ctx, git.PullOptions{
		Remote: targetRemote,
		Branch: branch,
		FFOnly: true,
	})
	if err != nil {
		if errors.Is(err, git.ErrFastForwardNotPossible) {
			fmt.Fprintln(errOut, "❌ Fast-forward pull is not possible.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "The local and remote branches have diverged.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "DevGit will not automatically merge or rebase.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Inspect the changes and choose an appropriate strategy manually.")
			return err
		}
		fmt.Fprintf(errOut, "❌ Pull failed:\n%v\n", err)
		return err
	}

	// 12. Display result
	fmt.Fprintln(out, "\n✓ Pull completed")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Local branch %s updated to match %s.\n", branch, info.Upstream)
	return nil
}
