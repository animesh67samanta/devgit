package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrFastForwardNotPossible indicates that local and remote have diverged and fast-forward pull cannot be performed.
	ErrFastForwardNotPossible = errors.New("fast-forward pull is not possible: branches have diverged")

	// ErrOperationInProgress indicates that a merge or rebase is currently in progress.
	ErrOperationInProgress = errors.New("a git operation is already in progress")
)

// PullOptions configures a pull operation.
type PullOptions struct {
	Remote string // Remote alias (e.g. "origin")
	Branch string // Branch name to pull
	FFOnly bool   // Enforce fast-forward only (--ff-only)
}

// PullResult contains information about a completed pull operation.
type PullResult struct {
	Remote string // Remote pulled from
	Branch string // Branch pulled into
	Output string // Raw standard output from git pull
}

// Pull updates the local branch from the remote repository.
func (c *Client) Pull(ctx context.Context, opts PullOptions) (*PullResult, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	currentBranch, err := c.CurrentBranch(ctx)
	if err != nil {
		return nil, err
	}
	if strings.Contains(currentBranch, "HEAD detached") {
		return nil, ErrDetachedHEAD
	}

	repoState, err := c.RepositoryState(ctx)
	if err == nil {
		if repoState.IsMerge || repoState.IsRebase {
			return nil, ErrOperationInProgress
		}
	}

	branch := opts.Branch
	if branch == "" {
		branch = currentBranch
	}

	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	args := []string{"pull"}
	if opts.FFOnly {
		args = append(args, "--ff-only")
	}

	if remote != "" {
		args = append(args, remote)
		if branch != "" {
			args = append(args, branch)
		}
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "not possible to fast-forward") ||
			strings.Contains(errStr, "diverged") ||
			strings.Contains(errStr, "non-fast-forward") {
			return nil, ErrFastForwardNotPossible
		}

		if strings.Contains(errStr, "authentication failed") ||
			strings.Contains(errStr, "permission denied") {
			return nil, ErrAuthenticationFailed
		}

		return nil, fmt.Errorf("git pull failed: %w", err)
	}

	return &PullResult{
		Remote: remote,
		Branch: branch,
		Output: out,
	}, nil
}
