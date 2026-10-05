package git

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
)

// FileStatus represents the Git status of an individual file.
type FileStatus struct {
	Path     string // Relative path of the file (or new path in case of rename)
	OrigPath string // Original path before renaming, if applicable
	Status   string // 2-character short status code (e.g. " M", "M ", "??", " D", "A ")
	Staged   rune   // Index status ('M', 'A', 'D', 'R', 'C', '?', ' ')
	Unstaged rune   // Working tree status ('M', 'D', '?', ' ')
}

// DisplayPath returns the formatted path, including the rename target if renamed.
func (f FileStatus) DisplayPath() string {
	if f.OrigPath != "" {
		return f.OrigPath + " -> " + f.Path
	}
	return f.Path
}

// IsStaged returns true if the file has changes in the staging area (index).
func (f FileStatus) IsStaged() bool {
	return f.Staged != ' ' && f.Staged != '?'
}

// IsUntracked returns true if the file is untracked by Git.
func (f FileStatus) IsUntracked() bool {
	return f.Status == "??"
}

// IsModified returns true if the file is modified in index or working tree.
func (f FileStatus) IsModified() bool {
	return f.Staged == 'M' || f.Unstaged == 'M'
}

// IsDeleted returns true if the file is deleted in index or working tree.
func (f FileStatus) IsDeleted() bool {
	return f.Staged == 'D' || f.Unstaged == 'D'
}

// IsAdded returns true if the file is newly added/staged in index.
func (f FileStatus) IsAdded() bool {
	return f.Staged == 'A'
}

// IsRenamed returns true if the file has been renamed.
func (f FileStatus) IsRenamed() bool {
	return f.Staged == 'R' || f.OrigPath != ""
}

// RepositoryStatus encapsulates the complete status of a repository.
type RepositoryStatus struct {
	Root     string
	Branch   string
	Upstream string
	Ahead    int
	Behind   int
	Files    []FileStatus
}

// IsClean returns true if there are no staged, unstaged, or untracked changes.
func (s *RepositoryStatus) IsClean() bool {
	return len(s.Files) == 0
}

// StagedFiles returns all files that have staged changes.
func (s *RepositoryStatus) StagedFiles() []FileStatus {
	var files []FileStatus
	for _, f := range s.Files {
		if f.IsStaged() {
			files = append(files, f)
		}
	}
	return files
}

// UnstagedFiles returns all files that have unstaged modifications or deletions.
func (s *RepositoryStatus) UnstagedFiles() []FileStatus {
	var files []FileStatus
	for _, f := range s.Files {
		if f.Unstaged != ' ' && f.Unstaged != '?' {
			files = append(files, f)
		}
	}
	return files
}

// UntrackedFiles returns all untracked files.
func (s *RepositoryStatus) UntrackedFiles() []FileStatus {
	var files []FileStatus
	for _, f := range s.Files {
		if f.IsUntracked() {
			files = append(files, f)
		}
	}
	return files
}

// ParseStatus parses the output of `git status --short --branch` into a RepositoryStatus.
func ParseStatus(root string, rawOutput string) (*RepositoryStatus, error) {
	status := &RepositoryStatus{
		Root:  root,
		Files: make([]FileStatus, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(rawOutput))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		// Parse branch header line: "## ..."
		if strings.HasPrefix(line, "## ") {
			parseBranchHeader(line[3:], status)
			continue
		}

		// Parse file status line: XY <path>
		if len(line) < 3 {
			continue
		}

		statusCode := line[0:2]
		stagedRune := rune(line[0])
		unstagedRune := rune(line[1])
		rawPath := strings.TrimSpace(line[3:])

		var origPath, finalPath string
		// Handle rename format: "old -> new"
		if strings.Contains(rawPath, " -> ") {
			parts := strings.SplitN(rawPath, " -> ", 2)
			origPath = unquotePath(parts[0])
			finalPath = unquotePath(parts[1])
		} else {
			finalPath = unquotePath(rawPath)
		}

		status.Files = append(status.Files, FileStatus{
			Path:     finalPath,
			OrigPath: origPath,
			Status:   statusCode,
			Staged:   stagedRune,
			Unstaged: unstagedRune,
		})
	}

	return status, scanner.Err()
}

func parseBranchHeader(header string, status *RepositoryStatus) {
	header = strings.TrimSpace(header)

	// Case: "Initial commit on <branch>"
	if strings.HasPrefix(header, "Initial commit on ") {
		status.Branch = strings.TrimPrefix(header, "Initial commit on ")
		return
	}

	// Case: "No commits yet on <branch>"
	if strings.HasPrefix(header, "No commits yet on ") {
		status.Branch = strings.TrimPrefix(header, "No commits yet on ")
		return
	}

	// Case: "HEAD (no branch)"
	if header == "HEAD (no branch)" {
		status.Branch = "(HEAD detached)"
		return
	}

	branchPart := header
	// Parse ahead/behind info if present: "[ahead 1, behind 2]"
	if idx := strings.Index(header, " ["); idx != -1 {
		branchPart = header[:idx]
		bracketInfo := strings.Trim(header[idx+2:], "[]")
		for _, part := range strings.Split(bracketInfo, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "ahead ") {
				fmt.Sscanf(part, "ahead %d", &status.Ahead)
			} else if strings.HasPrefix(part, "behind ") {
				fmt.Sscanf(part, "behind %d", &status.Behind)
			}
		}
	}

	// Check for upstream tracking: "<branch>...<upstream>"
	if strings.Contains(branchPart, "...") {
		parts := strings.SplitN(branchPart, "...", 2)
		status.Branch = strings.TrimSpace(parts[0])
		status.Upstream = strings.TrimSpace(parts[1])
	} else {
		status.Branch = strings.TrimSpace(branchPart)
	}
}

func unquotePath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "\"") && strings.HasSuffix(p, "\"") {
		if unquoted, err := strconv.Unquote(p); err == nil {
			return unquoted
		}
	}
	return p
}

// Status returns the complete parsed status of the repository.
func (c *Client) Status(ctx context.Context) (*RepositoryStatus, error) {
	root, err := c.Root(ctx)
	if err != nil {
		return nil, err
	}

	rawOutput, err := c.Run(ctx, "status", "--short", "--branch")
	if err != nil {
		return nil, err
	}

	status, err := ParseStatus(root, rawOutput)
	if err != nil {
		return nil, err
	}

	// If branch was not determined from status header, fallback to CurrentBranch
	if status.Branch == "" {
		branch, err := c.CurrentBranch(ctx)
		if err == nil {
			status.Branch = branch
		}
	}

	return status, nil
}
