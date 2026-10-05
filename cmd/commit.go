package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/spf13/cobra"
)

// CommitCmdFlags holds the command-line flags for devgit commit.
type CommitCmdFlags struct {
	Message string
	All     bool
	Yes     bool
}

// NewCommitCmd creates and returns the 'commit' command.
func NewCommitCmd() *cobra.Command {
	var flags CommitCmdFlags

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Record changes to the repository safely",
		Long: `Record changes to the repository with a guided, safe workflow.

Allows selective file staging, commit message validation, and commit previews
to prevent accidental or unintended commits.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCommit(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "", flags)
		},
	}

	cmd.Flags().StringVarP(&flags.Message, "message", "m", "", "Commit message")
	cmd.Flags().BoolVarP(&flags.All, "all", "a", false, "Select all changed files")
	cmd.Flags().BoolVarP(&flags.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

// RunCommit executes the complete commit workflow.
func RunCommit(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string, flags CommitCmdFlags) error {
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

	// 2. Detect working-tree and staged changes
	status, err := client.Status(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Git operation failed: %v\n", err)
		return err
	}

	if status.IsClean() {
		fmt.Fprintln(errOut, "❌ Nothing to commit.\n\nThe working tree is clean.")
		return git.ErrNoChangesToCommit
	}

	// 3. Check repository state warnings (detached HEAD, merge, rebase)
	if strings.Contains(status.Branch, "HEAD detached") {
		fmt.Fprintln(out, "⚠ You are currently in detached HEAD state.")
		fmt.Fprintln(out)
	}

	repoState, err := client.RepositoryState(ctx)
	if err == nil {
		if repoState.IsMerge {
			fmt.Fprintln(out, "⚠ Merge in progress.")
			fmt.Fprintln(out)
		} else if repoState.IsRebase {
			fmt.Fprintln(out, "⚠ Rebase in progress.")
			fmt.Fprintln(out)
		}
	}

	scanner := bufio.NewScanner(in)

	// 4. File selection
	var selectedFiles []git.FileStatus
	if flags.All {
		selectedFiles = status.Files
	} else {
		selectedFiles, err = promptFileSelection(scanner, out, errOut, status)
		if err != nil {
			return err
		}
		if len(selectedFiles) == 0 {
			fmt.Fprintln(out, "Operation cancelled.")
			return nil
		}
	}

	// Print selected files summary
	fmt.Fprintln(out, "\nSelected:")
	fmt.Fprintln(out)
	for _, f := range selectedFiles {
		fmt.Fprintf(out, "✓ %s\n", f.DisplayPath())
	}
	fmt.Fprintln(out)

	// 5. Commit message
	message := flags.Message
	if strings.TrimSpace(message) == "" {
		fmt.Fprintln(out, "Commit message:")
		fmt.Fprint(out, "> ")
		if !scanner.Scan() {
			fmt.Fprintln(errOut, "❌ Commit message cannot be empty.")
			return git.ErrEmptyCommitMessage
		}
		message = scanner.Text()
	}

	trimmedMsg := strings.TrimSpace(message)
	if trimmedMsg == "" {
		fmt.Fprintln(errOut, "❌ Commit message cannot be empty.")
		return git.ErrEmptyCommitMessage
	}

	// 6. Commit preview
	fmt.Fprintln(out, "\nCommit Preview")
	fmt.Fprintln(out, "────────────────────────────")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Branch:\n%s\n\n", status.Branch)
	fmt.Fprintln(out, "Files:")
	for _, f := range selectedFiles {
		fmt.Fprintf(out, "  %s\n", f.DisplayPath())
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Message:\n  %s\n\n", trimmedMsg)

	// 7. Explicit confirmation
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

	// 8. Prepare file paths for staging and commit
	// In case of renamed files, stage both OrigPath (old file removal) and Path (new file addition)
	var filePaths []string
	for _, f := range selectedFiles {
		if f.OrigPath != "" {
			filePaths = append(filePaths, f.OrigPath, f.Path)
		} else {
			filePaths = append(filePaths, f.Path)
		}
	}

	// 9 & 10. Stage only selected files and execute commit
	result, err := client.Commit(ctx, git.CommitOptions{
		Files:   filePaths,
		Message: trimmedMsg,
	})
	if err != nil {
		fmt.Fprintf(errOut, "❌ Commit failed.\n\nGit reported:\n%v\n", err)
		return err
	}

	// 11. Display result
	fmt.Fprintln(out, "\n✓ Files staged")
	fmt.Fprintln(out, "✓ Commit created")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n", result.ShortHash, result.Subject)

	return nil
}

func promptFileSelection(scanner *bufio.Scanner, out, errOut io.Writer, status *git.RepositoryStatus) ([]git.FileStatus, error) {
	fmt.Fprintln(out, "Changes:")
	fmt.Fprintln(out)

	hasStaged := len(status.StagedFiles()) > 0
	hasUnstaged := len(status.UnstagedFiles()) > 0 || len(status.UntrackedFiles()) > 0

	if hasStaged && hasUnstaged {
		fmt.Fprintln(out, "Staged Changes:")
		for i, f := range status.Files {
			if f.IsStaged() {
				fmt.Fprintf(out, "  %d. [✓] %s\n", i+1, f.DisplayPath())
			}
		}
		fmt.Fprintln(out, "\nUnstaged Changes:")
		for i, f := range status.Files {
			if !f.IsStaged() {
				fmt.Fprintf(out, "  %d. [ ] %s\n", i+1, f.DisplayPath())
			}
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Select files to commit:")
		fmt.Fprintln(out, "Enter numbers separated by comma, press Enter to commit staged files, 'a' for all, or 'c' to cancel:")
	} else if hasStaged {
		fmt.Fprintln(out, "Staged Changes:")
		for i, f := range status.Files {
			fmt.Fprintf(out, "  %d. [✓] %s\n", i+1, f.DisplayPath())
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Select files to commit:")
		fmt.Fprintln(out, "Press Enter to commit staged files, enter numbers separated by comma, or 'c' to cancel:")
	} else {
		for i, f := range status.Files {
			fmt.Fprintf(out, "%d. [ ] %s\n", i+1, f.DisplayPath())
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Select files to commit:")
		fmt.Fprintln(out, "Enter numbers separated by comma (or 'a' for all, 'c' to cancel):")
	}

	fmt.Fprint(out, "> ")
	if !scanner.Scan() {
		return nil, errors.New("input stream ended unexpectedly")
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "c" || input == "cancel" || input == "q" {
		return nil, nil
	}

	// If empty input and staged files exist, commit currently staged files
	if input == "" {
		if hasStaged {
			return status.StagedFiles(), nil
		}
		fmt.Fprintln(errOut, "❌ No files selected.")
		return nil, git.ErrNoFilesSelected
	}

	// Select all files
	if input == "a" || input == "all" {
		return status.Files, nil
	}

	// Parse comma-separated numbers
	parts := strings.Split(input, ",")
	seen := make(map[int]bool)
	var selected []git.FileStatus

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		num, err := strconv.Atoi(p)
		if err != nil || num < 1 || num > len(status.Files) {
			fmt.Fprintf(errOut, "❌ Invalid file selection: %s\n", p)
			return nil, fmt.Errorf("invalid file selection: %s", p)
		}

		if !seen[num] {
			seen[num] = true
			selected = append(selected, status.Files[num-1])
		}
	}

	if len(selected) == 0 {
		fmt.Fprintln(errOut, "❌ No files selected.")
		return nil, git.ErrNoFilesSelected
	}

	return selected, nil
}
