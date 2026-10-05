package git

import (
	"bufio"
	"context"
	"strings"
)

// DiffFile represents an individual file with changes detected by git diff.
type DiffFile struct {
	Path    string // Path of the file (or new path for renames)
	OldPath string // Original path before rename or copy
	Status  string // Raw status code: M, A, D, R, C, etc.
}

// DiffSummary provides categorized lists of changed files.
type DiffSummary struct {
	Modified []string
	Added    []string
	Deleted  []string
	Renamed  []string
	Other    []string
	Files    []DiffFile
}

// IsEmpty returns true if there are no changed files in the summary.
func (s *DiffSummary) IsEmpty() bool {
	return len(s.Files) == 0
}

// ParseDiffSummary parses the output of `git diff --name-status` into a structured DiffSummary.
func ParseDiffSummary(rawOutput string) *DiffSummary {
	summary := &DiffSummary{
		Modified: make([]string, 0),
		Added:    make([]string, 0),
		Deleted:  make([]string, 0),
		Renamed:  make([]string, 0),
		Other:    make([]string, 0),
		Files:    make([]DiffFile, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(rawOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		statusToken := strings.TrimSpace(fields[0])
		filePath := unquotePath(fields[1])

		if strings.HasPrefix(statusToken, "R") && len(fields) >= 3 {
			oldPath := filePath
			newPath := unquotePath(fields[2])
			summary.Renamed = append(summary.Renamed, oldPath+" -> "+newPath)
			summary.Files = append(summary.Files, DiffFile{
				Path:    newPath,
				OldPath: oldPath,
				Status:  statusToken,
			})
			continue
		}

		diffFile := DiffFile{
			Path:   filePath,
			Status: statusToken,
		}

		switch {
		case strings.HasPrefix(statusToken, "M"):
			summary.Modified = append(summary.Modified, filePath)
		case strings.HasPrefix(statusToken, "A"):
			summary.Added = append(summary.Added, filePath)
		case strings.HasPrefix(statusToken, "D"):
			summary.Deleted = append(summary.Deleted, filePath)
		default:
			summary.Other = append(summary.Other, filePath)
		}

		summary.Files = append(summary.Files, diffFile)
	}

	return summary
}

// DiffSummary returns the structured diff summary for unstaged or staged changes.
func (c *Client) DiffSummary(ctx context.Context, staged bool) (*DiffSummary, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return nil, err
	}
	if !inside {
		return nil, ErrNotRepository
	}

	args := []string{"diff", "--name-status"}
	if staged {
		args = append(args, "--staged")
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		return nil, err
	}

	return ParseDiffSummary(out), nil
}

// RawDiff returns the standard unified diff text from Git.
func (c *Client) RawDiff(ctx context.Context, staged bool) (string, error) {
	inside, err := c.IsInsideWorkTree(ctx)
	if err != nil {
		return "", err
	}
	if !inside {
		return "", ErrNotRepository
	}

	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}

	out, err := c.Run(ctx, args...)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(out, "\r\n"), nil
}
