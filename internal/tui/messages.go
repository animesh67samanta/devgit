package tui

import "github.com/animesh67samanta/devgit/internal/git"

// repoDataMsg contains all fetched Git service data.
type repoDataMsg struct {
	Root          string
	CurrentBranch string
	Status        *git.RepositoryStatus
	Branches      *git.BranchListResult
	Tracking      *git.TrackingInfo
	LogEntries    []git.Commit
	DiffSummary   *git.DiffSummary
	Stashes       []git.StashEntry
	RepoState     git.RepoState
	Err           error
}

// actionResultMsg notifies the UI of the outcome of a Git action.
type actionResultMsg struct {
	Title   string
	Message string
	IsError bool
}

// commitDetailMsg contains detailed message for a selected commit.
type commitDetailMsg struct {
	Commit git.Commit
}
