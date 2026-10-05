package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	// ErrNothingToStash is returned when attempting to stash with a clean working tree.
	ErrNothingToStash = errors.New("nothing to stash")

	// ErrStashNotFound is returned when the requested stash index or ref does not exist.
	ErrStashNotFound = errors.New("stash not found")

	// ErrStashConflict is returned when applying or popping a stash results in merge conflicts.
	ErrStashConflict = errors.New("stash application conflict")

	// ErrInvalidStashIndex is returned when a stash reference or index cannot be parsed.
	ErrInvalidStashIndex = errors.New("invalid stash index")
)

// StashEntry represents an individual entry in the Git stash stack.
type StashEntry struct {
	Index   int    // 0-based stash index
	Ref     string // Standard Git reference, e.g. "stash@{0}"
	Message string // Stash description or commit subject
}

// StashSaveOptions configures stash creation options.
type StashSaveOptions struct {
	Message          string
	IncludeUntracked bool
}

// StashApplyOptions configures stash application.
type StashApplyOptions struct {
	Index int
}

// StashPopOptions configures stash popping.
type StashPopOptions struct {
	Index int
}

// StashDropOptions configures stash dropping.
type StashDropOptions struct {
	Index int
}

// ParseStashIndex parses a stash selector (e.g. "0", "1", "stash@{0}") into an integer index.
// An empty string defaults to index 0.
func ParseStashIndex(ref string) (int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, nil
	}
	if strings.HasPrefix(ref, "stash@{") && strings.HasSuffix(ref, "}") {
		inner := ref[len("stash@{") : len(ref)-1]
		idx, err := strconv.Atoi(inner)
		if err != nil || idx < 0 {
			return 0, fmt.Errorf("%w: %q", ErrInvalidStashIndex, ref)
		}
		return idx, nil
	}
	idx, err := strconv.Atoi(ref)
	if err != nil || idx < 0 {
		return 0, fmt.Errorf("%w: %q", ErrInvalidStashIndex, ref)
	}
	return idx, nil
}

// ListStashes returns all stash entries ordered from newest (index 0) to oldest.
func (c *Client) ListStashes(ctx context.Context) ([]StashEntry, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	// Use machine-readable format: %gd = stash ref (stash@{0}), \x1f = ASCII Unit Separator, %gs = subject
	raw, err := c.Run(ctx, "stash", "list", "--format=%gd\x1f%gs")
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %w", err)
	}

	var entries []StashEntry
	scanner := bufio.NewScanner(strings.NewReader(raw))
	lineIdx := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var ref, msg string
		if strings.Contains(line, "\x1f") {
			parts := strings.SplitN(line, "\x1f", 2)
			ref = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				msg = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			ref = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				msg = strings.TrimSpace(parts[1])
			}
		} else {
			ref = line
		}

		idx, err := ParseStashIndex(ref)
		if err != nil {
			idx = lineIdx
		}

		entries = append(entries, StashEntry{
			Index:   idx,
			Ref:     fmt.Sprintf("stash@{%d}", idx),
			Message: msg,
		})
		lineIdx++
	}

	return entries, nil
}

// HasStashes returns true if at least one stash entry exists.
func (c *Client) HasStashes(ctx context.Context) (bool, error) {
	stashes, err := c.ListStashes(ctx)
	if err != nil {
		return false, err
	}
	return len(stashes) > 0, nil
}

// GetStash returns the stash entry with the given index, or ErrStashNotFound.
func (c *Client) GetStash(ctx context.Context, index int) (*StashEntry, error) {
	stashes, err := c.ListStashes(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range stashes {
		if s.Index == index {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("%w: no stash exists at index %d", ErrStashNotFound, index)
}

// SaveStash creates a new stash with the specified options.
func (c *Client) SaveStash(ctx context.Context, opts StashSaveOptions) (*StashEntry, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	// Verify no active merge/rebase
	repoState, err := c.RepositoryState(ctx)
	if err == nil && (repoState.IsMerge || repoState.IsRebase) {
		return nil, ErrOperationInProgress
	}

	// Inspect working tree changes
	status, err := c.Status(ctx)
	if err != nil {
		return nil, err
	}

	hasTrackedChanges := len(status.StagedFiles()) > 0 || len(status.UnstagedFiles()) > 0
	hasUntrackedFiles := len(status.UntrackedFiles()) > 0

	if !hasTrackedChanges && (!hasUntrackedFiles || !opts.IncludeUntracked) {
		return nil, ErrNothingToStash
	}

	args := []string{"stash", "push"}
	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}
	if strings.TrimSpace(opts.Message) != "" {
		args = append(args, "-m", opts.Message)
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(out), "no local changes to save") ||
			strings.Contains(strings.ToLower(err.Error()), "no local changes to save") {
			return nil, ErrNothingToStash
		}
		return nil, fmt.Errorf("failed to save stash: %w", err)
	}

	if strings.Contains(strings.ToLower(out), "no local changes to save") {
		return nil, ErrNothingToStash
	}

	// Retrieve the created stash@{0}
	created, err := c.GetStash(ctx, 0)
	if err == nil {
		return created, nil
	}

	return &StashEntry{
		Index:   0,
		Ref:     "stash@{0}",
		Message: opts.Message,
	}, nil
}

// ApplyStash applies a stash without removing it from the stash stack.
func (c *Client) ApplyStash(ctx context.Context, opts StashApplyOptions) error {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return err
	}
	if !inside {
		return ErrNotRepository
	}

	repoState, err := c.RepositoryState(ctx)
	if err == nil && (repoState.IsMerge || repoState.IsRebase) {
		return ErrOperationInProgress
	}

	if _, err := c.GetStash(ctx, opts.Index); err != nil {
		return err
	}

	ref := fmt.Sprintf("stash@{%d}", opts.Index)
	out, err := c.Run(ctx, "stash", "apply", ref)
	if err != nil {
		errStr := strings.ToLower(out + " " + err.Error())
		if strings.Contains(errStr, "conflict") {
			return ErrStashConflict
		}
		return fmt.Errorf("failed to apply stash %s: %w", ref, err)
	}

	return nil
}

// PopStash applies a stash and removes it from the stash stack if clean.
func (c *Client) PopStash(ctx context.Context, opts StashPopOptions) error {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return err
	}
	if !inside {
		return ErrNotRepository
	}

	repoState, err := c.RepositoryState(ctx)
	if err == nil && (repoState.IsMerge || repoState.IsRebase) {
		return ErrOperationInProgress
	}

	if _, err := c.GetStash(ctx, opts.Index); err != nil {
		return err
	}

	ref := fmt.Sprintf("stash@{%d}", opts.Index)
	out, err := c.Run(ctx, "stash", "pop", ref)
	if err != nil {
		errStr := strings.ToLower(out + " " + err.Error())
		if strings.Contains(errStr, "conflict") {
			return ErrStashConflict
		}
		return fmt.Errorf("failed to pop stash %s: %w", ref, err)
	}

	return nil
}

// DropStash permanently removes a stash from the stash stack.
func (c *Client) DropStash(ctx context.Context, opts StashDropOptions) error {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return err
	}
	if !inside {
		return ErrNotRepository
	}

	if _, err := c.GetStash(ctx, opts.Index); err != nil {
		return err
	}

	ref := fmt.Sprintf("stash@{%d}", opts.Index)
	_, err = c.Run(ctx, "stash", "drop", ref)
	if err != nil {
		return fmt.Errorf("failed to drop stash %s: %w", ref, err)
	}

	return nil
}
