package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrPushRejected indicates that the remote repository rejected the push (e.g. non-fast-forward).
	ErrPushRejected = errors.New("push rejected: remote contains work not present locally")

	// ErrAuthenticationFailed indicates that credentials or SSH key were rejected.
	ErrAuthenticationFailed = errors.New("push failed: authentication was rejected")

	// ErrForcePushDisabled is returned when force push options are requested.
	ErrForcePushDisabled = errors.New("force push is not supported")
)

// ProtectedBranches lists default branch names requiring extra user caution.
var ProtectedBranches = map[string]bool{
	"main":       true,
	"master":     true,
	"production": true,
	"prod":       true,
}

// IsProtectedBranch returns true if the specified branch name is commonly protected.
func IsProtectedBranch(branch string) bool {
	return ProtectedBranches[strings.ToLower(strings.TrimSpace(branch))]
}

// PushOptions configures a push operation.
type PushOptions struct {
	Remote      string // Target remote alias (e.g. "origin")
	Branch      string // Local branch name to push
	SetUpstream bool   // Whether to pass -u / --set-upstream to Git
}

// PushResult contains information about a completed push operation.
type PushResult struct {
	Remote string // Target remote name
	Branch string // Branch pushed
	Target string // Target ref (e.g. "origin/main")
	Output string // Raw standard output from git push
}

// Push pushes local branch commits to the remote repository.
func (c *Client) Push(ctx context.Context, opts PushOptions) (*PushResult, error) {
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

	branch := opts.Branch
	if branch == "" {
		branch = currentBranch
	}

	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	args := []string{"push"}
	if opts.SetUpstream {
		args = append(args, "-u", remote, branch)
	} else {
		args = append(args, remote, branch)
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "non-fast-forward") ||
			strings.Contains(errStr, "[rejected]") ||
			strings.Contains(errStr, "updates were rejected because the remote contains work") {
			return nil, ErrPushRejected
		}

		if strings.Contains(errStr, "authentication failed") ||
			strings.Contains(errStr, "permission denied") ||
			strings.Contains(errStr, "host key verification failed") {
			return nil, ErrAuthenticationFailed
		}

		return nil, fmt.Errorf("git push failed: %w", err)
	}

	target := fmt.Sprintf("%s/%s", remote, branch)
	return &PushResult{
		Remote: remote,
		Branch: branch,
		Target: target,
		Output: out,
	}, nil
}
