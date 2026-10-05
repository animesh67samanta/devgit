package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// Version is the current semantic version of DevGit.
	// In release builds, this is injected via -ldflags "-X github.com/animesh67samanta/devgit/internal/version.Version=...".
	Version = "1.0.0"

	// Commit is the git commit SHA at build time.
	// In release builds, this is injected via -ldflags "-X github.com/animesh67samanta/devgit/internal/version.Commit=...".
	Commit = "unknown"

	// Date is the RFC3339 or ISO-8601 build timestamp.
	// In release builds, this is injected via -ldflags "-X github.com/animesh67samanta/devgit/internal/version.Date=...".
	Date = "unknown"
)

// Info returns formatted, multi-line build and version information.
func Info() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("devgit version %s\n", Version))
	sb.WriteString(fmt.Sprintf("commit:  %s\n", Commit))
	sb.WriteString(fmt.Sprintf("built:   %s\n", Date))
	sb.WriteString(fmt.Sprintf("os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("go:      %s", runtime.Version()))
	return sb.String()
}

// Short returns a single-line summary of the version.
func Short() string {
	if Commit != "" && Commit != "unknown" {
		return fmt.Sprintf("devgit version %s (%s)", Version, Commit)
	}
	return fmt.Sprintf("devgit version %s", Version)
}
