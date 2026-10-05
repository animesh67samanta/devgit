package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrBranchAlreadyExists is returned when trying to create a branch that already exists.
	ErrBranchAlreadyExists = errors.New("branch already exists")

	// ErrBranchNotFound is returned when the target branch cannot be found.
	ErrBranchNotFound = errors.New("branch not found")

	// ErrCannotDeleteCurrent is returned when attempting to delete the currently checked-out branch.
	ErrCannotDeleteCurrent = errors.New("cannot delete the currently checked-out branch")

	// ErrBranchNotMerged is returned when deleting an unmerged branch without force.
	ErrBranchNotMerged = errors.New("branch is not fully merged")

	// ErrEmptyBranchName is returned when a blank branch name is provided.
	ErrEmptyBranchName = errors.New("branch name cannot be empty")
)

// Branch represents metadata for a local or remote Git branch.
type Branch struct {
	Name       string // Branch name (e.g. "main", "feature/login", "origin/main")
	IsCurrent  bool   // True if this branch is currently checked out
	IsRemote   bool   // True if this is a remote tracking branch
	Upstream   string // Upstream tracking branch if configured (e.g. "origin/main")
	CommitHash string // Abbreviated commit hash at HEAD of branch
	Subject    string // Commit message subject at HEAD of branch
}

// BranchListResult contains segregated local and remote branch listings.
type BranchListResult struct {
	Local   []Branch
	Remote  []Branch
	Current string
}

// DeleteBranchOptions configures deletion parameters.
type DeleteBranchOptions struct {
	Force bool // If true, uses `git branch -D` instead of `-d`
}

// ListBranches returns all local and remote branches with current branch and upstream tracking.
func (c *Client) ListBranches(ctx context.Context) (*BranchListResult, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	currentBranch, _ := c.CurrentBranch(ctx)

	// Format: refname | refname:short | HEAD | upstream:short | objectname:short | subject
	fmtArg := fmt.Sprintf("--format=%%(refname)%s%%(refname:short)%s%%(HEAD)%s%%(upstream:short)%s%%(objectname:short)%s%%(contents:subject)",
		logDelimiter, logDelimiter, logDelimiter, logDelimiter, logDelimiter)

	out, err := c.Run(ctx, "for-each-ref", fmtArg, "refs/heads", "refs/remotes")
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	result := &BranchListResult{
		Local:   make([]Branch, 0),
		Remote:  make([]Branch, 0),
		Current: currentBranch,
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, logDelimiter)
		if len(parts) < 6 {
			continue
		}

		refName := parts[0]
		shortName := parts[1]
		headMark := parts[2]
		upstream := parts[3]
		commitHash := parts[4]
		subject := parts[5]

		isCurrent := headMark == "*" || shortName == currentBranch

		// Ignore symbolic HEAD pointers on remotes (e.g. origin/HEAD)
		if strings.HasPrefix(refName, "refs/remotes/") && strings.HasSuffix(shortName, "/HEAD") {
			continue
		}

		branch := Branch{
			Name:       shortName,
			IsCurrent:  isCurrent,
			Upstream:   upstream,
			CommitHash: commitHash,
			Subject:    subject,
		}

		if strings.HasPrefix(refName, "refs/heads/") {
			branch.IsRemote = false
			result.Local = append(result.Local, branch)
		} else if strings.HasPrefix(refName, "refs/remotes/") {
			branch.IsRemote = true
			result.Remote = append(result.Remote, branch)
		}
	}

	return result, scanner.Err()
}

// BranchExists checks whether a local branch exists.
func (c *Client) BranchExists(ctx context.Context, name string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, ErrEmptyBranchName
	}

	_, err := c.Run(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// CreateBranch creates a new local branch pointing at startPoint (or HEAD if startPoint is empty).
func (c *Client) CreateBranch(ctx context.Context, name string, startPoint string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyBranchName
	}

	exists, err := c.BranchExists(ctx, name)
	if err == nil && exists {
		return ErrBranchAlreadyExists
	}

	args := []string{"branch", "--", name}
	if startPoint != "" {
		args = append(args, startPoint)
	}

	_, err = c.Run(ctx, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return ErrBranchAlreadyExists
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

// SwitchBranch switches to the specified branch.
func (c *Client) SwitchBranch(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyBranchName
	}

	// Try `git switch` first, then fallback to `git checkout`
	_, err := c.Run(ctx, "switch", "--", name)
	if err != nil {
		_, checkoutErr := c.Run(ctx, "checkout", "--", name)
		if checkoutErr != nil {
			errStr := strings.ToLower(checkoutErr.Error())
			if strings.Contains(errStr, "pathspec") || strings.Contains(errStr, "did not match") {
				return ErrBranchNotFound
			}
			return checkoutErr
		}
	}

	return nil
}

// DeleteBranch deletes a local branch safely (-d by default, -D if force requested).
func (c *Client) DeleteBranch(ctx context.Context, name string, opts DeleteBranchOptions) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyBranchName
	}

	current, err := c.CurrentBranch(ctx)
	if err == nil && current == name {
		return ErrCannotDeleteCurrent
	}

	delFlag := "-d"
	if opts.Force {
		delFlag = "-D"
	}

	_, err = c.Run(ctx, "branch", delFlag, "--", name)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "not fully merged") {
			return ErrBranchNotMerged
		}
		if strings.Contains(errStr, "branch") && strings.Contains(errStr, "not found") {
			return ErrBranchNotFound
		}
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	return nil
}

// RenameBranch renames oldName to newName. If oldName is empty, the current branch is renamed.
func (c *Client) RenameBranch(ctx context.Context, oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return ErrEmptyBranchName
	}

	oldName = strings.TrimSpace(oldName)

	args := []string{"branch", "-m"}
	if oldName != "" {
		args = append(args, oldName, newName)
	} else {
		args = append(args, newName)
	}

	_, err := c.Run(ctx, args...)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "already exists") {
			return ErrBranchAlreadyExists
		}
		return fmt.Errorf("failed to rename branch: %w", err)
	}

	return nil
}
