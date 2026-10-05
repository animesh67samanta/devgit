package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/animesh67samanta/devgit/internal/config"
	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

// PushCmdFlags holds command-line flags for devgit push.
type PushCmdFlags struct {
	Remote string
	Branch string
	Yes    bool
	Force  bool
}

// NewPushCmd creates and returns the 'push' command.
func NewPushCmd() *cobra.Command {
	var flags PushCmdFlags

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local commits to a remote repository safely",
		Long: `Push local commits to a remote repository with safety verifications,
ahead/behind diagnostics, protected branch warnings, and explicit confirmation.

Force pushing is strictly disabled to prevent remote history loss.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPush(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}

	cmd.Flags().StringVar(&flags.Remote, "remote", "", "Target remote name")
	cmd.Flags().StringVar(&flags.Branch, "branch", "", "Branch to push")
	cmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Confirm push without prompting")
	cmd.Flags().BoolVarP(&flags.Force, "force", "f", false, "Force push (rejected in Phase 5)")

	return cmd
}

// RunPush executes the push workflow with all safety checks.
func RunPush(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags PushCmdFlags) error {
	// 1. Force push safety check
	if flags.Force {
		fmt.Fprintln(errOut, "❌ Force push is not supported in Phase 5.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Force pushing can overwrite remote history.")
		return git.ErrForcePushDisabled
	}

	// 2. Verify repository
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	inside, err := client.IsInsideWorkTree(ctx)
	if err != nil || !inside {
		fmt.Fprintln(errOut, notRepoMessage)
		return git.ErrNotRepository
	}

	// Load effective configuration
	cfg, err := config.Load(config.LoadOptions{WorkDir: workDir})
	if err != nil {
		fmt.Fprintf(errOut, "❌ %v\n", err)
		return err
	}

	// 3. Detect current branch
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
		fmt.Fprintln(errOut, "❌ Cannot push from detached HEAD.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Create or switch to a branch first.")
		return git.ErrDetachedHEAD
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

	root, _ := client.Root(ctx)
	scanner := bufio.NewScanner(in)

	// 5. Detect upstream and tracking info
	info, err := client.TrackingInfo(ctx, branch)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to check tracking info: %v\n", err)
		return err
	}

	targetRemote := flags.Remote
	if targetRemote == "" {
		if info != nil && info.Remote != "" {
			targetRemote = info.Remote
		} else {
			defaultRemote := cfg.Git.DefaultRemote
			if defaultRemote == "" {
				defaultRemote = "origin"
			}
			found := false
			for _, r := range remotes {
				if r.Name == defaultRemote {
					targetRemote = defaultRemote
					found = true
					break
				}
			}
			if !found {
				targetRemote = remotes[0].Name
			}
		}
	}

	// If no upstream is configured, offer to push and set upstream
	if info == nil || !info.HasUpstream() {
		fmt.Fprintln(out, "⚠ Current branch has no upstream.")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Branch: %s\n\n", branch)
		fmt.Fprintln(out, "No upstream tracking branch is configured.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Would you like to push this branch and set the upstream?")
		fmt.Fprintln(out)

		if !flags.Yes {
			fmt.Fprint(out, "Push and set upstream? [y/N] ")
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

		res, err := client.Push(ctx, git.PushOptions{
			Remote:      targetRemote,
			Branch:      branch,
			SetUpstream: true,
		})
		if err != nil {
			return handlePushError(errOut, err)
		}

		fmt.Fprintln(out, "\n✓ Push completed")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s is now up to date.\n", res.Target)
		return nil
	}

	// 6. If branch is already synchronized with remote, skip unnecessary push
	if info.IsUpToDate() {
		fmt.Fprintln(out, "✓ Nothing to push.")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "%s is already up to date with %s.\n", branch, info.Upstream)
		return nil
	}

	// 7. Display repository and tracking information
	fmt.Fprintf(out, "Repository: %s\n", root)
	fmt.Fprintf(out, "Branch: %s\n", branch)
	fmt.Fprintf(out, "Remote: %s\n", targetRemote)
	fmt.Fprintf(out, "Upstream: %s\n", info.Upstream)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Status:")
	if info.IsDiverged() {
		fmt.Fprintf(out, "  ↑ %d commits ahead\n", info.Ahead)
		fmt.Fprintf(out, "  ↓ %d commits behind\n", info.Behind)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "⚠ Local and remote branches have diverged.")
	} else {
		fmt.Fprintf(out, "  ↑ %d commits to push\n", info.Ahead)
		fmt.Fprintf(out, "  ↓ %d commits to pull\n", info.Behind)
	}
	fmt.Fprintln(out)

	// 8. Protected branch warning
	if cfg.IsProtectedBranch(branch) {
		fmt.Fprintln(out, "⚠ You are about to push to:")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Branch: %s\n", branch)
		fmt.Fprintf(out, "Remote: %s\n", info.Upstream)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Protected/default branch detected.")
		fmt.Fprintln(out)
	}

	// 9. Display push target and request confirmation
	fmt.Fprintln(out, "Push:")
	fmt.Fprintf(out, "  %s → %s\n\n", branch, info.Upstream)

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

	// 10. Execute push
	res, err := client.Push(ctx, git.PushOptions{
		Remote: targetRemote,
		Branch: branch,
	})
	if err != nil {
		return handlePushError(errOut, err)
	}

	// 11. Display result
	fmt.Fprintln(out, "\n✓ Push completed")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s is now up to date.\n", res.Target)
	return nil
}

func handlePushError(errOut io.Writer, err error) error {
	if errors.Is(err, git.ErrPushRejected) {
		fmt.Fprintln(errOut, "❌ Push rejected.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "The remote branch contains commits that are not present locally.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Your branch may be behind the remote.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Run:")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "devgit pull")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "or inspect the remote changes before pushing again.")
		return err
	}

	if errors.Is(err, git.ErrAuthenticationFailed) {
		fmt.Fprintln(errOut, "❌ Push failed: authentication was rejected.")
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Check your Git credentials, SSH key, or remote configuration.")
		return err
	}

	fmt.Fprintf(errOut, "❌ Push failed:\n%v\n", err)
	return err
}
