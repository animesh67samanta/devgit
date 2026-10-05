package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	// ErrNotRepository indicates that the command was run outside a Git work tree.
	ErrNotRepository = errors.New("not a git repository")

	// ErrGitNotFound indicates that the git executable is not available on PATH.
	ErrGitNotFound = errors.New("git executable not found in PATH")
)

// GitError represents an error returned by executing a Git command.
type GitError struct {
	Args     []string
	Stderr   string
	Stdout   string
	ExitCode int
	Err      error
}

func (e *GitError) Error() string {
	if e.Stderr != "" {
		return strings.TrimSpace(e.Stderr)
	}
	if e.Stdout != "" {
		return strings.TrimSpace(e.Stdout)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("git command failed with exit code %d", e.ExitCode)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// Client provides an interface for executing Git operations.
type Client struct {
	gitPath string
	workDir string
}

// NewClient initializes a new Git client for the given working directory.
// If workDir is empty, the current working directory is used.
func NewClient(workDir string) (*Client, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrGitNotFound
	}

	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		workDir = wd
	}

	return &Client{
		gitPath: gitPath,
		workDir: workDir,
	}, nil
}

// WorkDir returns the working directory configured for this client.
func (c *Client) WorkDir() string {
	return c.workDir
}

// Run executes a git command with the given arguments and returns its standard output.
func (c *Client) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.gitPath, args...)
	cmd.Dir = c.workDir
	// LC_ALL=C ensures consistent output formatting across different system locales.
	// GIT_TERMINAL_PROMPT=0 prevents hanging if Git prompts for credentials.
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		stderrLower := strings.ToLower(stderrStr)
		if strings.Contains(stderrLower, "not a git repository") {
			return "", ErrNotRepository
		}

		exitCode := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}

		return "", &GitError{
			Args:     args,
			Stderr:   stderrStr,
			Stdout:   stdout.String(),
			ExitCode: exitCode,
			Err:      err,
		}
	}

	return stdout.String(), nil
}

// RunTrimmed executes a git command and returns standard output with leading/trailing whitespace removed.
func (c *Client) RunTrimmed(ctx context.Context, args ...string) (string, error) {
	out, err := c.Run(ctx, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
