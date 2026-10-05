package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyCommitMessage is returned when a commit message is blank.
	ErrEmptyCommitMessage = errors.New("commit message cannot be empty")

	// ErrNoChangesToCommit is returned when the repository has no staged, unstaged, or untracked changes.
	ErrNoChangesToCommit = errors.New("nothing to commit, working tree clean")

	// ErrNoFilesSelected is returned when no files are selected for the commit.
	ErrNoFilesSelected = errors.New("no files selected for commit")
)

// CommitOptions specifies the files and message for a commit operation.
type CommitOptions struct {
	Files   []string // List of files to stage and commit. If empty, commits whatever is currently staged in the index.
	Message string   // Commit message subject/body.
}

// CommitResult contains information about a successfully created commit.
type CommitResult struct {
	Hash      string   // Full commit SHA.
	ShortHash string   // Abbreviated commit SHA.
	Subject   string   // Commit subject line.
	Branch    string   // Branch where the commit was created.
	Files     []string // List of files included in the commit.
}

// Stage stages the specified files using `git add --`.
// It executes Git safely using separate arguments with the `--` separator
// to prevent filenames beginning with '-' or containing spaces/symbols from being misinterpreted as flags.
func (c *Client) Stage(ctx context.Context, files ...string) error {
	if len(files) == 0 {
		return nil
	}

	args := append([]string{"add", "--"}, files...)
	_, err := c.Run(ctx, args...)
	return err
}

// Commit creates a Git commit with the given options.
// If opts.Files is non-empty, it stages ONLY those files and commits them.
func (c *Client) Commit(ctx context.Context, opts CommitOptions) (*CommitResult, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	trimmedMsg := strings.TrimSpace(opts.Message)
	if trimmedMsg == "" {
		return nil, ErrEmptyCommitMessage
	}

	// If specific files were selected, stage them first
	if len(opts.Files) > 0 {
		if err := c.Stage(ctx, opts.Files...); err != nil {
			return nil, fmt.Errorf("failed to stage files: %w", err)
		}

		commitArgs := []string{"commit", "-m", trimmedMsg, "--"}
		commitArgs = append(commitArgs, opts.Files...)
		_, err = c.Run(ctx, commitArgs...)
		if err != nil {
			return nil, fmt.Errorf("git commit failed: %w", err)
		}
	} else {
		// Commit existing staged changes in the index
		_, err = c.Run(ctx, "commit", "-m", trimmedMsg)
		if err != nil {
			return nil, fmt.Errorf("git commit failed: %w", err)
		}
	}

	// Retrieve details of the newly created commit
	logOut, err := c.RunTrimmed(ctx, "log", "-1", "--format=%H\x1f%h\x1f%s")
	if err != nil {
		return nil, fmt.Errorf("commit succeeded but failed to inspect result: %w", err)
	}

	parts := strings.Split(logOut, "\x1f")
	if len(parts) < 3 {
		return nil, fmt.Errorf("unexpected git log output: %q", logOut)
	}

	branch, _ := c.CurrentBranch(ctx)

	return &CommitResult{
		Hash:      parts[0],
		ShortHash: parts[1],
		Subject:   parts[2],
		Branch:    branch,
		Files:     opts.Files,
	}, nil
}
