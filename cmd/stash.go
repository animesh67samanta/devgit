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

// StashSaveCmdFlags holds flags for stash save operations.
type StashSaveCmdFlags struct {
	Message          string
	Yes              bool
	IncludeUntracked bool
}

// StashActionCmdFlags holds flags for stash apply/pop/drop operations.
type StashActionCmdFlags struct {
	Yes bool
}

// NewStashCmd creates and returns the 'stash' command and its subcommands.
func NewStashCmd() *cobra.Command {
	var saveFlags StashSaveCmdFlags
	var popFlags StashActionCmdFlags
	var applyFlags StashActionCmdFlags
	var dropFlags StashActionCmdFlags

	stashCmd := &cobra.Command{
		Use:   "stash",
		Short: "Manage Git stashes safely",
		Long: `Inspect, save, apply, pop, and drop stashes safely with working-tree
conflict guards, confirmation checks, and explicit stash indexing.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStashList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	// Subcommand: devgit stash list
	listCmd := &cobra.Command{
		Use:           "list",
		Short:         "List stashed changes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStashList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	// Subcommand: devgit stash save [message]
	saveCmd := &cobra.Command{
		Use:           "save [message]",
		Short:         "Save changes to stash safely",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := saveFlags.Message
			if msg == "" && len(args) > 0 {
				msg = strings.Join(args, " ")
			}
			flags := saveFlags
			flags.Message = msg
			return RunStashSave(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}
	saveCmd.Flags().StringVarP(&saveFlags.Message, "message", "m", "", "Stash description message")
	saveCmd.Flags().BoolVarP(&saveFlags.Yes, "yes", "y", false, "Confirm stash creation without prompting")
	saveCmd.Flags().BoolVarP(&saveFlags.IncludeUntracked, "include-untracked", "u", false, "Include untracked files in stash")

	// Subcommand: devgit stash pop [index]
	popCmd := &cobra.Command{
		Use:           "pop [index]",
		Short:         "Apply a stash and remove it from the stash stack",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			idxArg := ""
			if len(args) > 0 {
				idxArg = args[0]
			}
			return RunStashPop(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", popFlags, idxArg)
		},
	}
	popCmd.Flags().BoolVarP(&popFlags.Yes, "yes", "y", false, "Confirm stash pop without prompting")

	// Subcommand: devgit stash apply [index]
	applyCmd := &cobra.Command{
		Use:           "apply [index]",
		Short:         "Apply a stash while preserving it in the stash stack",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			idxArg := ""
			if len(args) > 0 {
				idxArg = args[0]
			}
			return RunStashApply(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", applyFlags, idxArg)
		},
	}
	applyCmd.Flags().BoolVarP(&applyFlags.Yes, "yes", "y", false, "Confirm stash apply without prompting")

	// Subcommand: devgit stash drop [index]
	dropCmd := &cobra.Command{
		Use:           "drop [index]",
		Short:         "Permanently delete a stash entry",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			idxArg := ""
			if len(args) > 0 {
				idxArg = args[0]
			}
			return RunStashDrop(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", dropFlags, idxArg)
		},
	}
	dropCmd.Flags().BoolVarP(&dropFlags.Yes, "yes", "y", false, "Confirm stash drop without prompting")

	stashCmd.AddCommand(listCmd)
	stashCmd.AddCommand(saveCmd)
	stashCmd.AddCommand(popCmd)
	stashCmd.AddCommand(applyCmd)
	stashCmd.AddCommand(dropCmd)

	return stashCmd
}

// RunStashList displays all available stashes.
func RunStashList(ctx context.Context, out, errOut io.Writer, workDir string) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	stashes, err := client.ListStashes(ctx)
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) {
			fmt.Fprintln(errOut, notRepoMessage)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to list stashes: %v\n", err)
		return err
	}

	if len(stashes) == 0 {
		fmt.Fprintln(out, "✓ No stashes found.")
		return nil
	}

	fmt.Fprintln(out, "Stashes")
	fmt.Fprintln(out)

	for _, s := range stashes {
		msg := s.Message
		if msg == "" {
			msg = "(no message)"
		}
		fmt.Fprintf(out, "%-10s %s\n", s.Ref, msg)
	}

	fmt.Fprintln(out)
	if len(stashes) == 1 {
		fmt.Fprintln(out, "1 stash")
	} else {
		fmt.Fprintf(out, "%d stashes\n", len(stashes))
	}

	return nil
}

// RunStashSave creates a new stash with changes from the working tree.
func RunStashSave(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags StashSaveCmdFlags) error {
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
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Complete or abort the current Git operation before using DevGit stash.")
			return git.ErrOperationInProgress
		} else if repoState.IsRebase {
			fmt.Fprintln(errOut, "❌ A Git operation is already in progress.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Current state:")
			fmt.Fprintln(errOut, "  Rebase in progress")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Complete or abort the current Git operation before using DevGit stash.")
			return git.ErrOperationInProgress
		}
	}

	// Inspect working tree changes
	status, err := client.Status(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to read repository status: %v\n", err)
		return err
	}

	hasTrackedChanges := len(status.StagedFiles()) > 0 || len(status.UnstagedFiles()) > 0
	hasUntrackedFiles := len(status.UntrackedFiles()) > 0

	if !hasTrackedChanges && !hasUntrackedFiles {
		fmt.Fprintln(out, "✓ Nothing to stash.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Working tree is clean.")
		return nil
	}

	if !hasTrackedChanges && hasUntrackedFiles && !flags.IncludeUntracked {
		fmt.Fprintln(out, "⚠ Untracked files are present.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "DevGit will not include untracked files in the stash by default.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Working tree contains no tracked changes to stash.")
		fmt.Fprintln(out, "Use --include-untracked (-u) to stash untracked files.")
		return nil
	}

	// Display working tree changes
	fmt.Fprintln(out, "Working tree contains:")
	fmt.Fprintln(out)
	for _, f := range status.Files {
		if f.IsUntracked() && !flags.IncludeUntracked {
			continue
		}
		fmt.Fprintf(out, "  %s %s\n", f.Status, f.DisplayPath())
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "These changes will be stashed.")

	if hasUntrackedFiles && !flags.IncludeUntracked {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "⚠ Untracked files are present.")
		fmt.Fprintln(out, "DevGit will not include untracked files in the stash by default.")
	}

	if !flags.Yes {
		scanner := bufio.NewScanner(in)
		fmt.Fprintln(out)
		fmt.Fprint(out, "Save changes to stash? [y/N] ")
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

	saved, err := client.SaveStash(ctx, git.StashSaveOptions{
		Message:          flags.Message,
		IncludeUntracked: flags.IncludeUntracked,
	})
	if err != nil {
		if errors.Is(err, git.ErrNothingToStash) {
			fmt.Fprintln(out, "✓ Nothing to stash.")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Working tree is clean.")
			return nil
		}
		fmt.Fprintf(errOut, "❌ Failed to save stash: %v\n", err)
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "✓ Changes stashed successfully.")
	fmt.Fprintln(out)
	if saved.Message != "" {
		fmt.Fprintf(out, "%s: %s\n", saved.Ref, saved.Message)
	} else {
		fmt.Fprintln(out, saved.Ref)
	}

	return nil
}

// RunStashPop applies and removes a stash entry.
func RunStashPop(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags StashActionCmdFlags, indexArg string) error {
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

	idx, err := git.ParseStashIndex(indexArg)
	if err != nil {
		fmt.Fprintln(errOut, "❌ Invalid stash index.")
		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "Please specify a valid numeric index or stash reference (e.g. 0 or stash@{0}).\n")
		return err
	}

	stash, err := client.GetStash(ctx, idx)
	if err != nil {
		if errors.Is(err, git.ErrStashNotFound) {
			fmt.Fprintln(errOut, "❌ Stash not found.")
			fmt.Fprintln(errOut)
			fmt.Fprintf(errOut, "No stash exists at index %d.\n", idx)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to inspect stash: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(in)

	// Dirty working tree check
	status, err := client.Status(ctx)
	if err == nil && !status.IsClean() {
		fmt.Fprintln(out, "⚠ Your working tree contains local changes.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Applying this stash may cause conflicts or overwrite working-tree files.")
		fmt.Fprintln(out, "DevGit will not automatically stash your current changes.")
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
			fmt.Fprintln(out)
		}
	}

	// Show selected stash and confirmation
	fmt.Fprintln(out, "Selected stash:")
	fmt.Fprintln(out, stash.Ref)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Message:")
	if stash.Message != "" {
		fmt.Fprintln(out, stash.Message)
	} else {
		fmt.Fprintln(out, "(no message)")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "⚠ Applying this stash will modify your working tree and remove the stash.")
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

	err = client.PopStash(ctx, git.StashPopOptions{Index: idx})
	if err != nil {
		if errors.Is(err, git.ErrStashConflict) {
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "⚠ Stash application encountered conflicts.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "The stash has been kept by Git.")
			fmt.Fprintln(errOut, "Resolve the conflicts manually before continuing.")
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to pop stash: %v\n", err)
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "✓ Stash applied and removed.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, stash.Ref)
	return nil
}

// RunStashApply applies a stash while preserving it in the stash stack.
func RunStashApply(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags StashActionCmdFlags, indexArg string) error {
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

	idx, err := git.ParseStashIndex(indexArg)
	if err != nil {
		fmt.Fprintln(errOut, "❌ Invalid stash index.")
		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "Please specify a valid numeric index or stash reference (e.g. 0 or stash@{0}).\n")
		return err
	}

	stash, err := client.GetStash(ctx, idx)
	if err != nil {
		if errors.Is(err, git.ErrStashNotFound) {
			fmt.Fprintln(errOut, "❌ Stash not found.")
			fmt.Fprintln(errOut)
			fmt.Fprintf(errOut, "No stash exists at index %d.\n", idx)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to inspect stash: %v\n", err)
		return err
	}

	scanner := bufio.NewScanner(in)

	// Dirty working tree check
	status, err := client.Status(ctx)
	if err == nil && !status.IsClean() {
		fmt.Fprintln(out, "⚠ Your working tree contains local changes.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Applying this stash may cause conflicts or overwrite working-tree files.")
		fmt.Fprintln(out, "DevGit will not automatically stash your current changes.")
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
			fmt.Fprintln(out)
		}
	}

	// Show selected stash and confirmation
	fmt.Fprintln(out, "Selected stash:")
	fmt.Fprintln(out, stash.Ref)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Message:")
	if stash.Message != "" {
		fmt.Fprintln(out, stash.Message)
	} else {
		fmt.Fprintln(out, "(no message)")
	}
	fmt.Fprintln(out)

	if !flags.Yes {
		fmt.Fprint(out, "Apply stash? [y/N] ")
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

	err = client.ApplyStash(ctx, git.StashApplyOptions{Index: idx})
	if err != nil {
		if errors.Is(err, git.ErrStashConflict) {
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "❌ Stash could not be applied cleanly.")
			fmt.Fprintln(errOut)
			fmt.Fprintln(errOut, "Git reported conflicts.")
			fmt.Fprintln(errOut, "Your stash has NOT been removed.")
			fmt.Fprintln(errOut, "Resolve the conflicts manually and inspect the working tree.")
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to apply stash: %v\n", err)
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "✓ Stash applied.")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s remains available.\n", stash.Ref)
	return nil
}

// RunStashDrop permanently removes a stash entry.
func RunStashDrop(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags StashActionCmdFlags, indexArg string) error {
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

	idx, err := git.ParseStashIndex(indexArg)
	if err != nil {
		fmt.Fprintln(errOut, "❌ Invalid stash index.")
		fmt.Fprintln(errOut)
		fmt.Fprintf(errOut, "Please specify a valid numeric index or stash reference (e.g. 0 or stash@{0}).\n")
		return err
	}

	stash, err := client.GetStash(ctx, idx)
	if err != nil {
		if errors.Is(err, git.ErrStashNotFound) {
			fmt.Fprintln(errOut, "❌ Stash not found.")
			fmt.Fprintln(errOut)
			fmt.Fprintf(errOut, "No stash exists at index %d.\n", idx)
			return err
		}
		fmt.Fprintf(errOut, "❌ Failed to inspect stash: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "⚠ You are about to permanently delete:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, stash.Ref)
	if stash.Message != "" {
		fmt.Fprintln(out, stash.Message)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "This operation cannot be undone through DevGit.")
	fmt.Fprintln(out)

	if !flags.Yes {
		scanner := bufio.NewScanner(in)
		fmt.Fprint(out, "Drop this stash? [y/N] ")
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

	err = client.DropStash(ctx, git.StashDropOptions{Index: idx})
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to drop stash: %v\n", err)
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "✓ Stash dropped.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, stash.Ref)
	return nil
}
