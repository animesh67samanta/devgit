package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// RepoState represents merge, rebase, and detached HEAD state.
type RepoState struct {
	IsDetached bool
	IsMerge    bool
	IsRebase   bool
}

// IsInsideWorkTree checks whether the current client directory is inside a Git working tree.
func (c *Client) IsInsideWorkTree(ctx context.Context) (bool, error) {
	out, err := c.RunTrimmed(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if err == ErrNotRepository {
			return false, ErrNotRepository
		}
		return false, err
	}
	return out == "true", nil
}

// Root returns the top-level directory of the Git working tree.
func (c *Client) Root(ctx context.Context) (string, error) {
	root, err := c.RunTrimmed(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return root, nil
}

// CurrentBranch returns the name of the active branch, or describes detached HEAD state.
func (c *Client) CurrentBranch(ctx context.Context) (string, error) {
	// First attempt: standard git branch --show-current
	branch, err := c.RunTrimmed(ctx, "branch", "--show-current")
	if err == nil && branch != "" {
		return branch, nil
	}

	// Second attempt: symbolic-ref (works for newly initialized repos with no commits)
	branch, err = c.RunTrimmed(ctx, "symbolic-ref", "--short", "HEAD")
	if err == nil && branch != "" {
		return branch, nil
	}

	// Third attempt: detached HEAD commit hash
	head, err := c.RunTrimmed(ctx, "rev-parse", "--short", "HEAD")
	if err == nil && head != "" {
		return "(HEAD detached at " + head + ")", nil
	}

	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		return "", ErrNotRepository
	}

	return "(no branch)", nil
}

// RepositoryState detects whether a merge, rebase, or detached HEAD is in progress.
func (c *Client) RepositoryState(ctx context.Context) (RepoState, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return RepoState{}, err
	}
	if !inside {
		return RepoState{}, ErrNotRepository
	}

	var state RepoState

	// Check detached HEAD
	branch, err := c.CurrentBranch(ctx)
	if err == nil && strings.Contains(branch, "HEAD detached") {
		state.IsDetached = true
	}

	// Check git directory for merge/rebase heads
	gitDir, err := c.RunTrimmed(ctx, "rev-parse", "--git-dir")
	if err == nil {
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(c.workDir, gitDir)
		}
		if _, err := os.Stat(filepath.Join(gitDir, "MERGE_HEAD")); err == nil {
			state.IsMerge = true
		}
		if _, err := os.Stat(filepath.Join(gitDir, "rebase-merge")); err == nil {
			state.IsRebase = true
		} else if _, err := os.Stat(filepath.Join(gitDir, "rebase-apply")); err == nil {
			state.IsRebase = true
		}
	}

	return state, nil
}
