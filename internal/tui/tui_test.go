package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/animesh67samanta/devgit/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

func setupTestRepo(t *testing.T) (string, *git.Client) {
	t.Helper()
	dir := t.TempDir()
	if realDir, err := filepath.EvalSymlinks(dir); err == nil {
		dir = realDir
	}

	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "core.autocrlf", "false")

	f := filepath.Join(dir, "README.md")
	_ = os.WriteFile(f, []byte("# Hello\n"), 0644)
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")

	client, err := git.NewClient(dir)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return dir, client
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\noutput: %s", strings.Join(args, " "), err, string(out))
	}
}

func TestModelInit(t *testing.T) {
	_, client := setupTestRepo(t)
	m := NewModel(client)

	if m.currentView != ViewDashboard {
		t.Errorf("expected initial view to be ViewDashboard (0), got: %d", m.currentView)
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected Init to return non-nil cmd")
	}

	// Execute cmd
	msg := cmd()
	repoData, ok := msg.(repoDataMsg)
	if !ok {
		t.Fatalf("expected repoDataMsg, got: %T", msg)
	}
	if repoData.Err != nil {
		t.Fatalf("expected no error loading repo data, got: %v", repoData.Err)
	}
	if repoData.CurrentBranch != "main" {
		t.Errorf("expected branch 'main', got: %s", repoData.CurrentBranch)
	}
}

func TestModelWindowResize(t *testing.T) {
	_, client := setupTestRepo(t)
	m := NewModel(client)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := updated.(Model)

	if model.width != 100 || model.height != 30 {
		t.Errorf("expected dimensions 100x30, got %dx%d", model.width, model.height)
	}
}

func TestModelViewSwitching(t *testing.T) {
	_, client := setupTestRepo(t)
	m := NewModel(client)
	m.isLoading = false

	// Key '2' -> ViewStatus
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	if m.currentView != ViewStatus {
		t.Errorf("expected ViewStatus (1), got: %d", m.currentView)
	}

	// Key '3' -> ViewBranches
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(Model)
	if m.currentView != ViewBranches {
		t.Errorf("expected ViewBranches (2), got: %d", m.currentView)
	}

	// Key '4' -> ViewLog
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = updated.(Model)
	if m.currentView != ViewLog {
		t.Errorf("expected ViewLog (3), got: %d", m.currentView)
	}

	// Key '5' -> ViewDiff
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = updated.(Model)
	if m.currentView != ViewDiff {
		t.Errorf("expected ViewDiff (4), got: %d", m.currentView)
	}

	// Key '6' -> ViewStashes
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	m = updated.(Model)
	if m.currentView != ViewStashes {
		t.Errorf("expected ViewStashes (5), got: %d", m.currentView)
	}

	// Key '1' -> ViewDashboard
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected ViewDashboard (0), got: %d", m.currentView)
	}

	// Tab -> ViewStatus
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.currentView != ViewStatus {
		t.Errorf("expected tab to advance to ViewStatus (1), got: %d", m.currentView)
	}

	// Shift+Tab -> ViewDashboard
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.currentView != ViewDashboard {
		t.Errorf("expected shift+tab to retreat to ViewDashboard (0), got: %d", m.currentView)
	}
}

func TestModelHelpAndQuit(t *testing.T) {
	_, client := setupTestRepo(t)
	m := NewModel(client)
	m.isLoading = false

	// Toggle help '?'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if !m.showHelp {
		t.Error("expected showHelp to be true")
	}

	view := m.View()
	if !strings.Contains(view, "DevGit Keyboard Navigation") {
		t.Errorf("expected help view to be rendered, got: %s", view)
	}

	// Dismiss help with 'esc'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.showHelp {
		t.Error("expected showHelp to be false after Esc")
	}

	// Quit 'q'
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit cmd on 'q'")
	}
}

func TestModelConfirmationModal(t *testing.T) {
	_, client := setupTestRepo(t)
	m := NewModel(client)
	m.isLoading = false
	m.currentBranch = "main"

	// Trigger Push action 'p'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	if !m.showConfirm {
		t.Fatal("expected showConfirm to be true on push")
	}

	view := m.View()
	if !strings.Contains(view, "Confirm Push") {
		t.Errorf("expected Confirm Push modal in view, got: %s", view)
	}

	// Cancel with 'n'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if m.showConfirm {
		t.Error("expected showConfirm to be false after 'n'")
	}

	// Trigger Push again and confirm with 'y'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if m.showConfirm {
		t.Error("expected showConfirm to close on 'y'")
	}
	if cmd == nil {
		t.Error("expected confirmAction to dispatch cmd")
	}
}

func TestModelRenderViews(t *testing.T) {
	dir, client := setupTestRepo(t)
	m := NewModel(client)
	m.width = 120
	m.height = 30
	m.isLoading = false

	// Populate mock repo data
	m.root = dir
	m.currentBranch = "main"
	m.status = &git.RepositoryStatus{
		Root:   dir,
		Branch: "main",
		Files: []git.FileStatus{
			{Path: "test.go", Status: " M", Staged: ' ', Unstaged: 'M'},
		},
	}
	m.branches = &git.BranchListResult{
		Local: []git.Branch{
			{Name: "main", IsCurrent: true},
			{Name: "feature-ui", IsCurrent: false},
		},
	}
	m.logEntries = []git.Commit{
		{
			Hash:         "abc1234567890",
			ShortHash:    "abc1234",
			AuthorName:   "DevGit Test",
			AuthorEmail:  "devgit@example.com",
			Date:         "2026-10-04 12:00:00",
			RelativeDate: "just now",
			Subject:      "Add UI tests",
		},
	}
	m.diffSummary = &git.DiffSummary{
		Modified: []string{"test.go"},
	}
	m.stashes = []git.StashEntry{
		{Index: 0, Ref: "stash@{0}", Message: "WIP work"},
	}

	// 1. Dashboard View
	m.currentView = ViewDashboard
	v := m.View()
	if !strings.Contains(v, "Branch & Remote") || !strings.Contains(v, "Working Tree") {
		t.Errorf("Dashboard view missing expected cards: %s", v)
	}

	// 2. Status View
	m.currentView = ViewStatus
	v = m.View()
	if !strings.Contains(v, "test.go") {
		t.Errorf("Status view missing modified file: %s", v)
	}

	// 3. Branches View
	m.currentView = ViewBranches
	v = m.View()
	if !strings.Contains(v, "feature-ui") {
		t.Errorf("Branches view missing branch: %s", v)
	}

	// 4. Log View
	m.currentView = ViewLog
	v = m.View()
	if !strings.Contains(v, "Add UI tests") {
		t.Errorf("Log view missing commit subject: %s", v)
	}

	// 5. Diff View
	m.currentView = ViewDiff
	v = m.View()
	if !strings.Contains(v, "Modified:") {
		t.Errorf("Diff view missing modified section: %s", v)
	}

	// 6. Stashes View
	m.currentView = ViewStashes
	v = m.View()
	if !strings.Contains(v, "WIP work") {
		t.Errorf("Stashes view missing stash message: %s", v)
	}
}
