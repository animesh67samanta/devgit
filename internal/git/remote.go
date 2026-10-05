package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNoRemote indicates that no Git remote is configured for the repository.
	ErrNoRemote = errors.New("no git remote is configured")

	// ErrNoUpstream indicates that the current branch has no upstream tracking branch.
	ErrNoUpstream = errors.New("no upstream tracking branch configured")

	// ErrDetachedHEAD indicates that HEAD is detached from any local branch.
	ErrDetachedHEAD = errors.New("cannot perform operation in detached HEAD state")
)

// Remote represents a configured Git remote repository.
type Remote struct {
	Name string // Remote alias (e.g. "origin")
	URL  string // Remote fetch or push URL
}

// TrackingInfo describes the upstream tracking relationship and ahead/behind counts for a branch.
type TrackingInfo struct {
	Branch   string // Local branch name
	Remote   string // Remote name (e.g. "origin")
	Upstream string // Full upstream tracking ref (e.g. "origin/main")
	Ahead    int    // Number of local commits not present on remote
	Behind   int    // Number of remote commits not present locally
}

// HasUpstream returns true if an upstream tracking branch is configured.
func (t *TrackingInfo) HasUpstream() bool {
	return t.Upstream != ""
}

// IsUpToDate returns true if local and remote branches are synchronized.
func (t *TrackingInfo) IsUpToDate() bool {
	return t.HasUpstream() && t.Ahead == 0 && t.Behind == 0
}

// IsDiverged returns true if both local and remote have unique commits.
func (t *TrackingInfo) IsDiverged() bool {
	return t.HasUpstream() && t.Ahead > 0 && t.Behind > 0
}

// Remotes returns the list of all configured Git remotes.
func (c *Client) Remotes(ctx context.Context) ([]Remote, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	out, err := c.Run(ctx, "remote")
	if err != nil {
		return nil, err
	}

	var remotes []Remote
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			continue
		}

		url, _ := c.GetRemoteURL(ctx, name)
		remotes = append(remotes, Remote{
			Name: name,
			URL:  url,
		})
	}

	return remotes, scanner.Err()
}

// GetRemoteURL returns the URL configured for the given remote name.
func (c *Client) GetRemoteURL(ctx context.Context, name string) (string, error) {
	out, err := c.RunTrimmed(ctx, "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return out, nil
}

// Fetch downloads objects and refs from the given remote without merging.
// If remote is empty, the default remote is used.
func (c *Client) Fetch(ctx context.Context, remote string) error {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return err
	}
	if !inside {
		return ErrNotRepository
	}

	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
	}

	_, err = c.Run(ctx, args...)
	return err
}

// TrackingInfo resolves the tracking status for a branch, including upstream ref and ahead/behind counts.
// If branch is empty, the active branch is used.
func (c *Client) TrackingInfo(ctx context.Context, branch string) (*TrackingInfo, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	if branch == "" {
		current, err := c.CurrentBranch(ctx)
		if err != nil {
			return nil, err
		}
		branch = current
	}

	// Detached HEAD cannot have upstream tracking
	if strings.Contains(branch, "HEAD detached") {
		return &TrackingInfo{Branch: branch}, nil
	}

	info := &TrackingInfo{Branch: branch}

	// Attempt to resolve upstream branch ref (e.g. "origin/main")
	upstream, err := c.RunTrimmed(ctx, "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	if err != nil || upstream == "" {
		// No upstream configured for this branch
		return info, nil
	}

	info.Upstream = upstream
	// Extract remote name from upstream (e.g. "origin" from "origin/main")
	if slashIdx := strings.Index(upstream, "/"); slashIdx != -1 {
		info.Remote = upstream[:slashIdx]
	}

	// Calculate ahead and behind counts using symmetric difference: branch...upstream
	counts, err := c.RunTrimmed(ctx, "rev-list", "--left-right", "--count", branch+"..."+upstream)
	if err == nil {
		var ahead, behind int
		if _, scanErr := fmt.Sscanf(counts, "%d\t%d", &ahead, &behind); scanErr == nil {
			info.Ahead = ahead
			info.Behind = behind
		}
	}

	return info, nil
}
