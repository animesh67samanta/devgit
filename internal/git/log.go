package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
)

// Unit separator byte used as delimiter in git log --format to avoid collisions.
const logDelimiter = "\x1f"

// Commit represents parsed commit metadata.
type Commit struct {
	Hash         string // Full SHA-1/SHA-256 commit hash
	ShortHash    string // Abbreviated commit hash (typically 7 characters)
	AuthorName   string // Author name
	AuthorEmail  string // Author email address
	RelativeDate string // Relative commit date (e.g. "2 hours ago", "yesterday")
	Date         string // Formatted author date
	Subject      string // Commit message subject line
}

// LogOptions configures commit log retrieval.
type LogOptions struct {
	Limit  int    // Maximum number of commits to retrieve (0 for all)
	Branch string // Specific branch, tag, or revision range
	All    bool   // Include commits from all branches
}

// ParseLog parses output formatted with logDelimiter into a slice of Commit structs.
func ParseLog(rawOutput string) ([]Commit, error) {
	var commits []Commit
	scanner := bufio.NewScanner(strings.NewReader(rawOutput))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, logDelimiter)
		if len(parts) < 7 {
			continue
		}

		commits = append(commits, Commit{
			Hash:         parts[0],
			ShortHash:    parts[1],
			AuthorName:   parts[2],
			AuthorEmail:  parts[3],
			RelativeDate: parts[4],
			Date:         parts[5],
			Subject:      parts[6],
		})
	}

	return commits, scanner.Err()
}

// Log retrieves the commit history according to the specified options.
func (c *Client) Log(ctx context.Context, opts LogOptions) ([]Commit, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	formatArg := fmt.Sprintf("--format=%%H%s%%h%s%%an%s%%ae%s%%ar%s%%ad%s%%s",
		logDelimiter, logDelimiter, logDelimiter, logDelimiter, logDelimiter, logDelimiter)

	args := []string{"log", formatArg}

	if opts.Limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", opts.Limit))
	}
	if opts.All {
		args = append(args, "--all")
	}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		if errors.Is(err, ErrNotRepository) {
			return nil, ErrNotRepository
		}

		errStr := err.Error()
		// Handle repository with no commits yet gracefully
		if strings.Contains(errStr, "does not have any commits yet") ||
			strings.Contains(errStr, "unknown revision or path not in the working tree") ||
			strings.Contains(errStr, "ambiguous argument 'HEAD'") {
			return []Commit{}, nil
		}

		return nil, err
	}

	return ParseLog(out)
}
