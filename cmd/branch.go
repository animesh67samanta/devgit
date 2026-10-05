package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"devgit/internal/git"
	"github.com/spf13/cobra"
)

// BranchCmdFlags holds flags for branch operations.
type BranchCmdFlags struct {
	Force  bool
	Yes    bool
	Switch bool
}

// NewBranchCmd creates and returns the 'branch' command and its subcommands.
func NewBranchCmd() *cobra.Command {
	var flags BranchCmdFlags

	branchCmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage repository branches safely",
		Long: `Inspect, create, switch, and delete branches with safety verifications,
dirty-tree protection, and guarded deletion.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBranchList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	// Subcommand: devgit branch list
	listCmd := &cobra.Command{
		Use:           "list",
		Short:         "List local and remote branches",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunBranchList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	// Subcommand: devgit branch create <name>
	createCmd := &cobra.Command{
		Use:           "create <name>",
		Short:         "Create a new branch",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return RunBranchCreate(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", name, flags.Switch)
		},
	}
	createCmd.Flags().BoolVarP(&flags.Switch, "switch", "s", false, "Switch to the branch after creating it")

	// Subcommand: devgit branch switch <name>
	switchCmd := &cobra.Command{
		Use:           "switch <name>",
		Short:         "Switch to an existing branch",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return RunBranchSwitch(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", name, flags.Yes)
		},
	}
	switchCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Confirm branch switch without prompting")

	// Subcommand: devgit branch delete <name>
	deleteCmd := &cobra.Command{
		Use:           "delete <name>",
		Short:         "Delete a branch safely",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return RunBranchDelete(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", name, flags.Force, flags.Yes)
		},
	}
	deleteCmd.Flags().BoolVarP(&flags.Force, "force", "f", false, "Force delete unmerged branch (-D)")
	deleteCmd.Flags().BoolVarP(&flags.Force, "delete-force", "D", false, "Alias for --force")
	deleteCmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Confirm unmerged deletion without prompting")

	// Subcommand: devgit branch rename [old] <new>
	renameCmd := &cobra.Command{
		Use:           "rename [old] <new>",
		Short:         "Rename a branch",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := "", ""
			if len(args) == 1 {
				newName = args[0]
			} else if len(args) >= 2 {
				oldName = args[0]
				newName = args[1]
			}
			return RunBranchRename(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", oldName, newName)
		},
	}

	branchCmd.AddCommand(listCmd)
	branchCmd.AddCommand(createCmd)
	branchCmd.AddCommand(switchCmd)
	branchCmd.AddCommand(deleteCmd)
	branchCmd.AddCommand(renameCmd)

	return branchCmd
}

// RunBranchList displays all local and remote branches.
func RunBranchList(ctx context.Context, out, errOut io.Writer, workDir string) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	res, err := client.ListBranches(ctx)
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to list branches: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "Branches")
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Local:")
	for _, b := range res.Local {
		prefix := "  "
		if b.IsCurrent {
			prefix = "* "
		}

		tracking := ""
		if b.Upstream != "" {
			tracking = fmt.Sprintf(" (%s)", b.Upstream)
		}

		fmt.Fprintf(out, "%s%s%s\n", prefix, b.Name, tracking)
	}

	if len(res.Remote) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Remote:")
		for _, b := range res.Remote {
			fmt.Fprintf(out, "    %s\n", b.Name)
		}
	}

	return nil
}

// RunBranchCreate creates a new branch.
func RunBranchCreate(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir, name string, andSwitch bool) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	if strings.TrimSpace(name) == "" {
		scanner := bufio.NewScanner(in)
		fmt.Fprint(out, "Branch name: ")
		if !scanner.Scan() {
			fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
			return git.ErrEmptyBranchName
		}
		name = strings.TrimSpace(scanner.Text())
	}

	if strings.TrimSpace(name) == "" {
		fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
		return git.ErrEmptyBranchName
	}

	err = client.CreateBranch(ctx, name, "")
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		if errors.Is(err, git.ErrBranchAlreadyExists) {
			fmt.Fprintf(errOut, "❌ Branch '%s' already exists.\n", name)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to create branch: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "✓ Branch '%s' created.\n", name)

	if andSwitch {
		err = client.SwitchBranch(ctx, name)
		if err != nil {
			fmt.Fprintf(errOut, "❌ Failed to switch to branch: %v\n", err)
			return err
		}
		fmt.Fprintf(out, "✓ Switched to branch '%s'.\n", name)
	}

	return nil
}

// RunBranchSwitch switches to an existing branch with dirty-tree and operation checks.
func RunBranchSwitch(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir, name string, yes bool) error {
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

	// Check operation in progress
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

	scanner := bufio.NewScanner(in)
	if strings.TrimSpace(name) == "" {
		fmt.Fprint(out, "Branch to switch to: ")
		if !scanner.Scan() {
			fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
			return git.ErrEmptyBranchName
		}
		name = strings.TrimSpace(scanner.Text())
	}

	if strings.TrimSpace(name) == "" {
		fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
		return git.ErrEmptyBranchName
	}

	// Check dirty working tree
	status, err := client.Status(ctx)
	if err == nil && !status.IsClean() {
		fmt.Fprintln(out, "⚠ Your working tree contains uncommitted changes.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Switching branches may carry over changes or cause conflicts.")
		fmt.Fprintln(out)

		if !yes {
			fmt.Fprint(out, "Switch anyway? [y/N] ")
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

	err = client.SwitchBranch(ctx, name)
	if err != nil {
		if errors.Is(err, git.ErrBranchNotFound) {
			fmt.Fprintf(errOut, "❌ Branch '%s' not found.\n", name)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to switch to branch: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "✓ Switched to branch '%s'.\n", name)
	return nil
}

// RunBranchDelete deletes a branch safely (-d by default, guarded -D if unmerged).
func RunBranchDelete(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir, name string, force, yes bool) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	if strings.TrimSpace(name) == "" {
		scanner := bufio.NewScanner(in)
		fmt.Fprint(out, "Branch to delete: ")
		if !scanner.Scan() {
			fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
			return git.ErrEmptyBranchName
		}
		name = strings.TrimSpace(scanner.Text())
	}

	if strings.TrimSpace(name) == "" {
		fmt.Fprintln(errOut, "❌ Branch name cannot be empty.")
		return git.ErrEmptyBranchName
	}

	// Guard: Never delete currently checked-out branch
	current, err := client.CurrentBranch(ctx)
	if err == nil && current == name {
		fmt.Fprintf(errOut, "❌ Cannot delete the currently checked-out branch '%s'.\n", name)
		fmt.Fprintln(errOut)
		fmt.Fprintln(errOut, "Switch to another branch first.")
		return git.ErrCannotDeleteCurrent
	}

	// Attempt standard safe deletion (-d)
	err = client.DeleteBranch(ctx, name, git.DeleteBranchOptions{Force: force})
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		if errors.Is(err, git.ErrBranchNotFound) {
			fmt.Fprintf(errOut, "❌ Branch '%s' not found.\n", name)
			return err
		}

		// If unmerged, warn and request explicit confirmation for force-delete
		if errors.Is(err, git.ErrBranchNotMerged) {
			fmt.Fprintf(out, "⚠ The branch '%s' is not fully merged.\n", name)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Deleting it may permanently delete unmerged commits.")
			fmt.Fprintln(out)

			if !yes {
				scanner := bufio.NewScanner(in)
				fmt.Fprint(out, "Force delete? [y/N] ")
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

			// Execute force delete after confirmation
			err = client.DeleteBranch(ctx, name, git.DeleteBranchOptions{Force: true})
			if err != nil {
				fmt.Fprintf(errOut, "❌ Force delete failed: %v\n", err)
				return err
			}

			fmt.Fprintf(out, "✓ Branch '%s' force-deleted.\n", name)
			return nil
		}

		fmt.Fprintf(errOut, "❌ Failed to delete branch: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "✓ Branch '%s' deleted.\n", name)
	return nil
}

// RunBranchRename renames a branch.
func RunBranchRename(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir, oldName, newName string) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	if strings.TrimSpace(newName) == "" {
		fmt.Fprintln(errOut, "❌ New branch name cannot be empty.")
		return git.ErrEmptyBranchName
	}

	err = client.RenameBranch(ctx, oldName, newName)
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		if errors.Is(err, git.ErrBranchAlreadyExists) {
			fmt.Fprintf(errOut, "❌ Branch '%s' already exists.\n", newName)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to rename branch: %v\n", err)
		return err
	}

	if oldName == "" {
		fmt.Fprintf(out, "✓ Current branch renamed to '%s'.\n", newName)
	} else {
		fmt.Fprintf(out, "✓ Branch '%s' renamed to '%s'.\n", oldName, newName)
	}

	return nil
}
