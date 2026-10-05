package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/animesh67samanta/devgit/internal/config"
	"github.com/animesh67samanta/devgit/internal/database"
	"github.com/animesh67samanta/devgit/internal/git"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View identifier constants
const (
	ViewDashboard = 0
	ViewStatus    = 1
	ViewBranches  = 2
	ViewLog       = 3
	ViewDiff      = 4
	ViewStashes   = 5
	viewCount     = 6
)

var viewNames = []string{
	"Dashboard",
	"Status",
	"Branches",
	"Log",
	"Diff",
	"Stashes",
}

// Model represents the top-level application state for the Bubble Tea TUI.
type Model struct {
	client *git.Client
	cfg    *config.Config
	db     *database.DB
	styles Styles
	keys   KeyMap

	// Dimensions
	width  int
	height int

	// Active view & navigation
	currentView int

	// Cursors per view
	statusCursor int
	branchCursor int
	logCursor    int
	stashCursor  int

	// Cached repository data
	root          string
	currentBranch string
	status        *git.RepositoryStatus
	branches      *git.BranchListResult
	tracking      *git.TrackingInfo
	logEntries    []git.Commit
	diffSummary   *git.DiffSummary
	stashes       []git.StashEntry
	repoState     git.RepoState
	loadErr       error

	// Loading state
	isLoading  bool
	loadingMsg string

	// Confirmation modal state
	showConfirm    bool
	confirmTitle   string
	confirmWarning string
	confirmPrompt  string
	confirmAction  func() tea.Cmd

	// Result / error modal state
	showResult    bool
	resultTitle   string
	resultMessage string
	resultIsError bool

	// Text input modal state (e.g. create branch)
	showInput   bool
	inputTitle  string
	inputPrompt string
	inputValue  string
	inputAction func(string) tea.Cmd

	// Help overlay
	showHelp bool
}

// NewModelWithConfigAndDB creates an initialized TUI Model with effective configuration and optional SQLite database.
func NewModelWithConfigAndDB(client *git.Client, cfg *config.Config, db *database.DB) Model {
	if cfg == nil {
		cfg = config.DefaultConfigPtr()
	}
	m := Model{
		client:      client,
		cfg:         cfg,
		db:          db,
		styles:      DefaultStyles(),
		keys:        DefaultKeyMap(),
		currentView: ViewDashboard,
		width:       80,
		height:      24,
		isLoading:   true,
		loadingMsg:  "Loading repository...",
	}

	if db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if val, found, err := db.GetPreference(ctx, "tui.last_view"); err == nil && found {
			for i, name := range viewNames {
				if strings.EqualFold(name, val) {
					m.currentView = i
					break
				}
			}
		}
	}

	return m
}

// NewModelWithConfig creates an initialized TUI Model with effective configuration.
func NewModelWithConfig(client *git.Client, cfg *config.Config) Model {
	return NewModelWithConfigAndDB(client, cfg, nil)
}

// NewModel creates an initialized TUI Model with default configuration.
func NewModel(client *git.Client) Model {
	return NewModelWithConfig(client, config.DefaultConfigPtr())
}

// Init triggers initial repository data loading.
func (m Model) Init() tea.Cmd {
	return m.loadRepoDataCmd()
}

// loadRepoDataCmd fetches all repository data asynchronously.
func (m Model) loadRepoDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		root, err := m.client.Root(ctx)
		if err != nil {
			return repoDataMsg{Err: err}
		}

		branch, _ := m.client.CurrentBranch(ctx)
		status, _ := m.client.Status(ctx)
		branches, _ := m.client.ListBranches(ctx)
		tracking, _ := m.client.TrackingInfo(ctx, branch)
		logEntries, _ := m.client.Log(ctx, git.LogOptions{Limit: 50})
		diffSummary, _ := m.client.DiffSummary(ctx, false)
		stashes, _ := m.client.ListStashes(ctx)
		repoState, _ := m.client.RepositoryState(ctx)

		return repoDataMsg{
			Root:          root,
			CurrentBranch: branch,
			Status:        status,
			Branches:      branches,
			Tracking:      tracking,
			LogEntries:    logEntries,
			DiffSummary:   diffSummary,
			Stashes:       stashes,
			RepoState:     repoState,
		}
	}
}

// Update handles incoming Bubble Tea messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case repoDataMsg:
		m.isLoading = false
		if msg.Err != nil {
			m.loadErr = msg.Err
			m.showResult = true
			m.resultTitle = "Repository Error"
			m.resultMessage = msg.Err.Error()
			m.resultIsError = true
			return m, nil
		}
		m.root = msg.Root
		m.currentBranch = msg.CurrentBranch
		m.status = msg.Status
		m.branches = msg.Branches
		m.tracking = msg.Tracking
		m.logEntries = msg.LogEntries
		m.diffSummary = msg.DiffSummary
		m.stashes = msg.Stashes
		m.repoState = msg.RepoState
		m.loadErr = nil
		return m, nil

	case actionResultMsg:
		m.isLoading = false
		m.showResult = true
		m.resultTitle = msg.Title
		m.resultMessage = msg.Message
		m.resultIsError = msg.IsError
		return m, m.loadRepoDataCmd()

	case tea.KeyMsg:
		// Handle Result Modal Dismissal
		if m.showResult {
			if key.Matches(msg, m.keys.Enter, m.keys.Back) {
				m.showResult = false
				return m, nil
			}
			return m, nil
		}

		// Handle Confirmation Modal
		if m.showConfirm {
			if key.Matches(msg, m.keys.ConfirmYes) {
				m.showConfirm = false
				if m.confirmAction != nil {
					act := m.confirmAction
					m.confirmAction = nil
					m.isLoading = true
					m.loadingMsg = "Executing operation..."
					return m, act()
				}
				return m, nil
			}
			if key.Matches(msg, m.keys.ConfirmNo, m.keys.Back) {
				m.showConfirm = false
				m.confirmAction = nil
				return m, nil
			}
			return m, nil
		}

		// Handle Input Modal
		if m.showInput {
			switch msg.Type {
			case tea.KeyEnter:
				m.showInput = false
				val := strings.TrimSpace(m.inputValue)
				act := m.inputAction
				m.inputAction = nil
				m.inputValue = ""
				if act != nil && val != "" {
					m.isLoading = true
					m.loadingMsg = "Creating..."
					return m, act(val)
				}
				return m, nil
			case tea.KeyEsc:
				m.showInput = false
				m.inputAction = nil
				m.inputValue = ""
				return m, nil
			case tea.KeyBackspace:
				if len(m.inputValue) > 0 {
					m.inputValue = m.inputValue[:len(m.inputValue)-1]
				}
				return m, nil
			default:
				if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
					m.inputValue += msg.String()
				}
				return m, nil
			}
		}

		// Handle Help Overlay
		if m.showHelp {
			if key.Matches(msg, m.keys.Help, m.keys.Back, m.keys.Quit) {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		// Global Keybindings
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		if key.Matches(msg, m.keys.Help) {
			m.showHelp = true
			return m, nil
		}
		if key.Matches(msg, m.keys.Refresh) {
			m.isLoading = true
			m.loadingMsg = "Refreshing repository..."
			return m, m.loadRepoDataCmd()
		}

		// Direct View Jumps
		if key.Matches(msg, m.keys.ViewDashboard) {
			m.currentView = ViewDashboard
			return m, nil
		}
		if key.Matches(msg, m.keys.ViewStatus) {
			m.currentView = ViewStatus
			return m, nil
		}
		if key.Matches(msg, m.keys.ViewBranches) {
			m.currentView = ViewBranches
			return m, nil
		}
		if key.Matches(msg, m.keys.ViewLog) {
			m.currentView = ViewLog
			return m, nil
		}
		if key.Matches(msg, m.keys.ViewDiff) {
			m.currentView = ViewDiff
			return m, nil
		}
		if key.Matches(msg, m.keys.ViewStashes) {
			m.currentView = ViewStashes
			return m, nil
		}

		// Tab Switching
		if key.Matches(msg, m.keys.NextTab, m.keys.Right) {
			m.currentView = (m.currentView + 1) % viewCount
			return m, nil
		}
		if key.Matches(msg, m.keys.PrevTab, m.keys.Left) {
			m.currentView = (m.currentView - 1 + viewCount) % viewCount
			return m, nil
		}

		// Push / Pull Quick Actions (Available from Dashboard, Status, Branches)
		if key.Matches(msg, m.keys.ActionPush) {
			return m.handlePushAction()
		}
		if key.Matches(msg, m.keys.ActionPull) {
			return m.handlePullAction()
		}

		// View-Specific Actions & Navigation
		return m.handleViewKeys(msg)
	}

	return m, nil
}

// handleViewKeys processes navigation and actions for the currently active tab.
func (m Model) handleViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewStatus:
		if m.status != nil && len(m.status.Files) > 0 {
			if key.Matches(msg, m.keys.Up) && m.statusCursor > 0 {
				m.statusCursor--
			}
			if key.Matches(msg, m.keys.Down) && m.statusCursor < len(m.status.Files)-1 {
				m.statusCursor++
			}
		}

	case ViewBranches:
		if m.branches != nil && len(m.branches.Local) > 0 {
			if key.Matches(msg, m.keys.Up) && m.branchCursor > 0 {
				m.branchCursor--
			}
			if key.Matches(msg, m.keys.Down) && m.branchCursor < len(m.branches.Local)-1 {
				m.branchCursor++
			}
			if key.Matches(msg, m.keys.Enter) {
				return m.handleBranchSwitch()
			}
			if key.Matches(msg, m.keys.ActionDelete) {
				return m.handleBranchDelete()
			}
		}
		if key.Matches(msg, m.keys.ActionCreate) {
			m.showInput = true
			m.inputTitle = "Create New Branch"
			m.inputPrompt = "Enter branch name:"
			m.inputValue = ""
			m.inputAction = func(name string) tea.Cmd {
				return func() tea.Msg {
					ctx := context.Background()
					err := m.client.CreateBranch(ctx, name, "")
					if err != nil {
						return actionResultMsg{Title: "Create Branch Failed", Message: err.Error(), IsError: true}
					}
					return actionResultMsg{Title: "Branch Created", Message: fmt.Sprintf("Created branch %q", name)}
				}
			}
			return m, nil
		}

	case ViewLog:
		if len(m.logEntries) > 0 {
			if key.Matches(msg, m.keys.Up) && m.logCursor > 0 {
				m.logCursor--
			}
			if key.Matches(msg, m.keys.Down) && m.logCursor < len(m.logEntries)-1 {
				m.logCursor++
			}
			if key.Matches(msg, m.keys.Enter) {
				sel := m.logEntries[m.logCursor]
				m.showResult = true
				m.resultTitle = fmt.Sprintf("Commit %s", sel.ShortHash)
				m.resultMessage = fmt.Sprintf("Author: %s <%s>\nDate:   %s (%s)\n\n%s", sel.AuthorName, sel.AuthorEmail, sel.Date, sel.RelativeDate, sel.Subject)
				m.resultIsError = false
				return m, nil
			}
		}

	case ViewStashes:
		if len(m.stashes) > 0 {
			if key.Matches(msg, m.keys.Up) && m.stashCursor > 0 {
				m.stashCursor--
			}
			if key.Matches(msg, m.keys.Down) && m.stashCursor < len(m.stashes)-1 {
				m.stashCursor++
			}
			if key.Matches(msg, m.keys.ActionApply) || key.Matches(msg, m.keys.Enter) {
				return m.handleStashApply()
			}
			if key.Matches(msg, m.keys.ActionPop) {
				return m.handleStashPop()
			}
			if key.Matches(msg, m.keys.ActionDelete) {
				return m.handleStashDrop()
			}
		}
	}

	return m, nil
}

// handlePushAction confirms and triggers safe push.
func (m Model) handlePushAction() (tea.Model, tea.Cmd) {
	if m.currentBranch == "" {
		m.showResult = true
		m.resultTitle = "Push Error"
		m.resultMessage = "Detached HEAD or no branch checked out."
		m.resultIsError = true
		return m, nil
	}

	warn := ""
	isProtected := false
	if m.cfg != nil {
		isProtected = m.cfg.IsProtectedBranch(m.currentBranch)
	} else {
		isProtected = git.IsProtectedBranch(m.currentBranch)
	}
	if isProtected {
		warn = fmt.Sprintf("⚠ WARNING: You are pushing directly to protected branch %q.", m.currentBranch)
	}

	prompt := fmt.Sprintf("Push branch %q to remote?", m.currentBranch)
	if m.tracking != nil && m.tracking.Ahead > 0 {
		prompt = fmt.Sprintf("Push %d local commit(s) on %q to remote?", m.tracking.Ahead, m.currentBranch)
	}

	m.showConfirm = true
	m.confirmTitle = "Confirm Push"
	m.confirmWarning = warn
	m.confirmPrompt = prompt
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			res, err := m.client.Push(ctx, git.PushOptions{Branch: m.currentBranch})
			if err != nil {
				return actionResultMsg{Title: "Push Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Push Succeeded", Message: fmt.Sprintf("✓ Pushed to %s/%s", res.Remote, res.Branch)}
		}
	}
	return m, nil
}

// handlePullAction confirms and triggers safe fast-forward pull.
func (m Model) handlePullAction() (tea.Model, tea.Cmd) {
	if m.currentBranch == "" {
		m.showResult = true
		m.resultTitle = "Pull Error"
		m.resultMessage = "Detached HEAD or no branch checked out."
		m.resultIsError = true
		return m, nil
	}

	warn := ""
	if m.status != nil && !m.status.IsClean() {
		warn = "⚠ Working tree contains uncommitted changes. Fast-forward pull may conflict."
	}

	prompt := fmt.Sprintf("Pull fast-forward changes for %q?", m.currentBranch)
	if m.tracking != nil && m.tracking.Behind > 0 {
		prompt = fmt.Sprintf("Pull %d remote commit(s) into %q?", m.tracking.Behind, m.currentBranch)
	}

	m.showConfirm = true
	m.confirmTitle = "Confirm Pull"
	m.confirmWarning = warn
	m.confirmPrompt = prompt
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			res, err := m.client.Pull(ctx, git.PullOptions{})
			if err != nil {
				return actionResultMsg{Title: "Pull Failed", Message: err.Error(), IsError: true}
			}
			if strings.Contains(res.Output, "Already up to date") {
				return actionResultMsg{Title: "Pull Status", Message: "Already up to date."}
			}
			return actionResultMsg{Title: "Pull Succeeded", Message: fmt.Sprintf("✓ Fast-forwarded changes from %s/%s", res.Remote, res.Branch)}
		}
	}
	return m, nil
}

// handleBranchSwitch confirms and switches branches safely.
func (m Model) handleBranchSwitch() (tea.Model, tea.Cmd) {
	if m.branches == nil || m.branchCursor >= len(m.branches.Local) {
		return m, nil
	}
	target := m.branches.Local[m.branchCursor]
	if target.IsCurrent {
		m.showResult = true
		m.resultTitle = "Branch Notice"
		m.resultMessage = fmt.Sprintf("Already on branch %q.", target.Name)
		m.resultIsError = false
		return m, nil
	}

	warn := ""
	if m.status != nil && !m.status.IsClean() {
		warn = "⚠ Working tree contains uncommitted changes. Switching branches may carry changes over."
	}

	m.showConfirm = true
	m.confirmTitle = "Switch Branch"
	m.confirmWarning = warn
	m.confirmPrompt = fmt.Sprintf("Switch to branch %q?", target.Name)
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			err := m.client.SwitchBranch(ctx, target.Name)
			if err != nil {
				return actionResultMsg{Title: "Switch Branch Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Branch Switched", Message: fmt.Sprintf("✓ Switched to %q", target.Name)}
		}
	}
	return m, nil
}

// handleBranchDelete confirms and deletes a branch safely.
func (m Model) handleBranchDelete() (tea.Model, tea.Cmd) {
	if m.branches == nil || m.branchCursor >= len(m.branches.Local) {
		return m, nil
	}
	target := m.branches.Local[m.branchCursor]
	if target.IsCurrent {
		m.showResult = true
		m.resultTitle = "Cannot Delete Branch"
		m.resultMessage = fmt.Sprintf("Cannot delete the currently checked-out branch %q. Switch to another branch first.", target.Name)
		m.resultIsError = true
		return m, nil
	}

	m.showConfirm = true
	m.confirmTitle = "Delete Branch"
	m.confirmWarning = "⚠ Safe branch deletion: will not force-delete unmerged commits."
	m.confirmPrompt = fmt.Sprintf("Delete branch %q?", target.Name)
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			err := m.client.DeleteBranch(ctx, target.Name, git.DeleteBranchOptions{Force: false})
			if err != nil {
				if errors.Is(err, git.ErrBranchNotMerged) {
					return actionResultMsg{
						Title:   "Branch Not Merged",
						Message: fmt.Sprintf("Branch %q is not fully merged. Use CLI force-delete (-D) if intentional.", target.Name),
						IsError: true,
					}
				}
				return actionResultMsg{Title: "Delete Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Branch Deleted", Message: fmt.Sprintf("✓ Deleted branch %q", target.Name)}
		}
	}
	return m, nil
}

// handleStashApply applies selected stash preserving it.
func (m Model) handleStashApply() (tea.Model, tea.Cmd) {
	if len(m.stashes) == 0 || m.stashCursor >= len(m.stashes) {
		return m, nil
	}
	sel := m.stashes[m.stashCursor]

	warn := ""
	if m.status != nil && !m.status.IsClean() {
		warn = "⚠ Working tree contains local changes. Applying stash may cause conflicts."
	}

	m.showConfirm = true
	m.confirmTitle = "Apply Stash"
	m.confirmWarning = warn
	m.confirmPrompt = fmt.Sprintf("Apply %s (%s)? Stash will remain in stack.", sel.Ref, sel.Message)
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			err := m.client.ApplyStash(ctx, git.StashApplyOptions{Index: sel.Index})
			if err != nil {
				if errors.Is(err, git.ErrStashConflict) {
					return actionResultMsg{
						Title:   "Stash Conflict",
						Message: "Conflict detected while applying stash. The stash has been preserved.",
						IsError: true,
					}
				}
				return actionResultMsg{Title: "Apply Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Stash Applied", Message: fmt.Sprintf("✓ Applied %s cleanly", sel.Ref)}
		}
	}
	return m, nil
}

// handleStashPop applies and drops selected stash.
func (m Model) handleStashPop() (tea.Model, tea.Cmd) {
	if len(m.stashes) == 0 || m.stashCursor >= len(m.stashes) {
		return m, nil
	}
	sel := m.stashes[m.stashCursor]

	warn := ""
	if m.status != nil && !m.status.IsClean() {
		warn = "⚠ Working tree contains local changes. Popping stash may cause conflicts."
	}

	m.showConfirm = true
	m.confirmTitle = "Pop Stash"
	m.confirmWarning = warn
	m.confirmPrompt = fmt.Sprintf("Pop %s? Changes will apply and stash will be removed.", sel.Ref)
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			err := m.client.PopStash(ctx, git.StashPopOptions{Index: sel.Index})
			if err != nil {
				if errors.Is(err, git.ErrStashConflict) {
					return actionResultMsg{
						Title:   "Stash Pop Conflict",
						Message: "Conflict encountered. Git has KEPT the stash entry safely.",
						IsError: true,
					}
				}
				return actionResultMsg{Title: "Pop Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Stash Popped", Message: fmt.Sprintf("✓ Popped %s and removed from stash stack", sel.Ref)}
		}
	}
	return m, nil
}

// handleStashDrop permanently removes selected stash.
func (m Model) handleStashDrop() (tea.Model, tea.Cmd) {
	if len(m.stashes) == 0 || m.stashCursor >= len(m.stashes) {
		return m, nil
	}
	sel := m.stashes[m.stashCursor]

	m.showConfirm = true
	m.confirmTitle = "Drop Stash"
	m.confirmWarning = "⚠ Permanent deletion: this cannot be undone."
	m.confirmPrompt = fmt.Sprintf("Permanently drop %s (%s)?", sel.Ref, sel.Message)
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			ctx := context.Background()
			err := m.client.DropStash(ctx, git.StashDropOptions{Index: sel.Index})
			if err != nil {
				return actionResultMsg{Title: "Drop Failed", Message: err.Error(), IsError: true}
			}
			return actionResultMsg{Title: "Stash Dropped", Message: fmt.Sprintf("✓ Dropped %s", sel.Ref)}
		}
	}
	return m, nil
}

// View renders the TUI interface.
func (m Model) View() string {
	if m.width == 0 {
		return "Initializing DevGit..."
	}

	// Render Header
	header := m.renderHeader()

	// Render Tabs
	tabs := m.renderTabs()

	// Render Active Body View
	var body string
	if m.isLoading {
		body = m.styles.Info.Render(fmt.Sprintf("\n  ⠋ %s\n", m.loadingMsg))
	} else if m.showHelp {
		body = m.renderHelp()
	} else {
		switch m.currentView {
		case ViewDashboard:
			body = m.renderDashboard()
		case ViewStatus:
			body = m.renderStatus()
		case ViewBranches:
			body = m.renderBranches()
		case ViewLog:
			body = m.renderLog()
		case ViewDiff:
			body = m.renderDiff()
		case ViewStashes:
			body = m.renderStashes()
		}
	}

	// Render Footer
	footer := m.renderFooter()

	mainView := lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, footer)

	// Overlays (Confirmation / Result / Input)
	if m.showConfirm {
		return m.overlayModal(m.renderConfirmModal())
	}
	if m.showResult {
		return m.overlayModal(m.renderResultModal())
	}
	if m.showInput {
		return m.overlayModal(m.renderInputModal())
	}

	return m.styles.App.Render(mainView)
}

func (m Model) renderHeader() string {
	title := m.styles.HeaderTitle.Render("devgit")
	path := m.styles.HeaderPath.Render(m.root)

	stateStr := ""
	if m.repoState.IsMerge {
		stateStr = m.styles.Danger.Render(" [MERGE IN PROGRESS] ")
	} else if m.repoState.IsRebase {
		stateStr = m.styles.Danger.Render(" [REBASE IN PROGRESS] ")
	}

	branchStr := ""
	if m.currentBranch != "" {
		branchStr = m.styles.CurrentBranch.Render(fmt.Sprintf(" ⎇ %s", m.currentBranch))
		if m.tracking != nil && m.tracking.HasUpstream() {
			branchStr += m.styles.UpstreamBranch.Render(fmt.Sprintf(" → %s", m.tracking.Upstream))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, path, branchStr, stateStr)
	return m.styles.HeaderBox.Width(m.width - 2).Render(left)
}

func (m Model) renderTabs() string {
	var tabItems []string
	for i, name := range viewNames {
		label := fmt.Sprintf("[%d] %s", i+1, name)
		if i == m.currentView {
			tabItems = append(tabItems, m.styles.TabActive.Render(label))
		} else {
			tabItems = append(tabItems, m.styles.TabInactive.Render(label))
		}
	}
	return m.styles.TabsBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabItems...))
}

func (m Model) renderDashboard() string {
	cardW := (m.width - 8) / 2
	if cardW < 32 {
		cardW = 32
	}

	// Card 1: Branch & Sync
	var branchContent strings.Builder
	branchContent.WriteString(fmt.Sprintf("Current: %s\n", m.styles.CurrentBranch.Render(m.currentBranch)))
	if m.tracking != nil && m.tracking.HasUpstream() {
		branchContent.WriteString(fmt.Sprintf("Tracking: %s\n", m.tracking.Upstream))
		aheadStr := fmt.Sprintf("↑ %d ahead", m.tracking.Ahead)
		behindStr := fmt.Sprintf("↓ %d behind", m.tracking.Behind)
		if m.tracking.Ahead > 0 {
			aheadStr = m.styles.Warning.Render(aheadStr)
		}
		if m.tracking.Behind > 0 {
			behindStr = m.styles.Warning.Render(behindStr)
		}
		branchContent.WriteString(fmt.Sprintf("Status:   %s  %s\n", aheadStr, behindStr))
	} else {
		branchContent.WriteString(m.styles.Muted.Render("No upstream configured\n"))
	}
	card1 := m.styles.CardBox.Width(cardW).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.styles.CardTitle.Render("Branch & Remote"), branchContent.String()),
	)

	// Card 2: Working Tree
	var statusContent strings.Builder
	if m.status != nil {
		if m.status.IsClean() {
			statusContent.WriteString(m.styles.Success.Render("✓ Working tree is clean\n"))
		} else {
			stagedCount := len(m.status.StagedFiles())
			unstagedCount := len(m.status.UnstagedFiles())
			untrackedCount := len(m.status.UntrackedFiles())
			statusContent.WriteString(fmt.Sprintf("● %d staged\n● %d unstaged\n● %d untracked\n", stagedCount, unstagedCount, untrackedCount))
		}
	}
	card2 := m.styles.CardBox.Width(cardW).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.styles.CardTitle.Render("Working Tree"), statusContent.String()),
	)

	// Card 3: Recent Commit
	var commitContent strings.Builder
	if len(m.logEntries) > 0 {
		c := m.logEntries[0]
		commitContent.WriteString(fmt.Sprintf("%s %s\n", m.styles.Info.Render(c.ShortHash), c.Subject))
		commitContent.WriteString(m.styles.Muted.Render(fmt.Sprintf("%s · %s\n", c.AuthorName, c.RelativeDate)))
	} else {
		commitContent.WriteString(m.styles.Muted.Render("No commit history found\n"))
	}
	card3 := m.styles.CardBox.Width(cardW).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.styles.CardTitle.Render("Latest Commit"), commitContent.String()),
	)

	// Card 4: Stashes & Quick Actions
	var stashContent strings.Builder
	stashContent.WriteString(fmt.Sprintf("Total Stashes: %d\n\n", len(m.stashes)))
	stashContent.WriteString(m.styles.KeyHelp.Render("p"))
	stashContent.WriteString(" Push  ")
	stashContent.WriteString(m.styles.KeyHelp.Render("P"))
	stashContent.WriteString(" Pull  ")
	stashContent.WriteString(m.styles.KeyHelp.Render("r"))
	stashContent.WriteString(" Refresh\n")
	card4 := m.styles.CardBox.Width(cardW).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.styles.CardTitle.Render("Stashes & Actions"), stashContent.String()),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, card1, card2)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, card3, card4)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow)
}

func (m Model) renderStatus() string {
	if m.status == nil || m.status.IsClean() {
		return m.styles.PanelBox.Width(m.width - 4).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.styles.CardTitle.Render("Working Tree"),
				m.styles.Success.Render("\n  ✓ Working tree is clean. Nothing to commit.\n"),
			),
		)
	}

	var sb strings.Builder
	sb.WriteString(m.styles.CardTitle.Render("Working Tree Changes"))
	sb.WriteString("\n\n")

	for i, f := range m.status.Files {
		prefix := "  "
		if i == m.statusCursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%s  %s", prefix, f.Status, f.DisplayPath())
		if i == m.statusCursor {
			sb.WriteString(m.styles.SelectedItem.Render(line))
		} else {
			sb.WriteString(m.styles.NormalItem.Render(line))
		}
		sb.WriteString("\n")
	}

	return m.styles.PanelBox.Width(m.width - 4).Render(sb.String())
}

func (m Model) renderBranches() string {
	if m.branches == nil {
		return m.styles.PanelBox.Width(m.width - 4).Render("Loading branches...")
	}

	var sb strings.Builder
	sb.WriteString(m.styles.CardTitle.Render("Branches"))
	sb.WriteString("  ")
	sb.WriteString(m.styles.Muted.Render("(Enter: switch, n: new, d: delete)"))
	sb.WriteString("\n\n")

	sb.WriteString(m.styles.HeaderTitle.Render("LOCAL BRANCHES"))
	sb.WriteString("\n")
	for i, b := range m.branches.Local {
		cursor := "  "
		if i == m.branchCursor {
			cursor = "> "
		}

		currMark := " "
		if b.IsCurrent {
			currMark = "*"
		}

		tracking := ""
		if b.Upstream != "" {
			tracking = fmt.Sprintf(" → %s", b.Upstream)
		}

		line := fmt.Sprintf("%s%s %-20s%s", cursor, currMark, b.Name, tracking)
		if i == m.branchCursor {
			sb.WriteString(m.styles.SelectedItem.Render(line))
		} else if b.IsCurrent {
			sb.WriteString(m.styles.CurrentBranch.Render(line))
		} else {
			sb.WriteString(m.styles.NormalItem.Render(line))
		}
		sb.WriteString("\n")
	}

	if len(m.branches.Remote) > 0 {
		sb.WriteString("\n")
		sb.WriteString(m.styles.HeaderTitle.Render("REMOTE BRANCHES"))
		sb.WriteString("\n")
		for _, b := range m.branches.Remote {
			sb.WriteString(m.styles.Muted.Render(fmt.Sprintf("    %s\n", b.Name)))
		}
	}

	return m.styles.PanelBox.Width(m.width - 4).Render(sb.String())
}

func (m Model) renderLog() string {
	if len(m.logEntries) == 0 {
		return m.styles.PanelBox.Width(m.width - 4).Render("No commits found.")
	}

	var sb strings.Builder
	sb.WriteString(m.styles.CardTitle.Render("Commit History"))
	sb.WriteString("  ")
	sb.WriteString(m.styles.Muted.Render("(Enter: view commit details)"))
	sb.WriteString("\n\n")

	start := 0
	if m.logCursor > 8 {
		start = m.logCursor - 8
	}
	end := start + 12
	if end > len(m.logEntries) {
		end = len(m.logEntries)
	}

	for i := start; i < end; i++ {
		c := m.logEntries[i]
		cursor := "  "
		if i == m.logCursor {
			cursor = "> "
		}

		dateStr := c.Date
		line := fmt.Sprintf("%s%-8s %-12s %-20s %s", cursor, c.ShortHash, truncate(dateStr, 12), truncate(c.AuthorName, 18), truncate(c.Subject, 35))
		if i == m.logCursor {
			sb.WriteString(m.styles.SelectedItem.Render(line))
		} else {
			sb.WriteString(m.styles.NormalItem.Render(line))
		}
		sb.WriteString("\n")
	}

	return m.styles.PanelBox.Width(m.width - 4).Render(sb.String())
}

func (m Model) renderDiff() string {
	var sb strings.Builder
	sb.WriteString(m.styles.CardTitle.Render("Diff Summary"))
	sb.WriteString("\n\n")

	if m.diffSummary == nil || (len(m.diffSummary.Modified) == 0 && len(m.diffSummary.Added) == 0 && len(m.diffSummary.Deleted) == 0) {
		sb.WriteString(m.styles.Success.Render("  ✓ No modified, added, or deleted files.\n"))
	} else {
		if len(m.diffSummary.Modified) > 0 {
			sb.WriteString(m.styles.Warning.Render("Modified:\n"))
			for _, f := range m.diffSummary.Modified {
				sb.WriteString(fmt.Sprintf("  M  %s\n", f))
			}
			sb.WriteString("\n")
		}
		if len(m.diffSummary.Added) > 0 {
			sb.WriteString(m.styles.Success.Render("Added:\n"))
			for _, f := range m.diffSummary.Added {
				sb.WriteString(fmt.Sprintf("  A  %s\n", f))
			}
			sb.WriteString("\n")
		}
		if len(m.diffSummary.Deleted) > 0 {
			sb.WriteString(m.styles.Danger.Render("Deleted:\n"))
			for _, f := range m.diffSummary.Deleted {
				sb.WriteString(fmt.Sprintf("  D  %s\n", f))
			}
			sb.WriteString("\n")
		}
	}

	return m.styles.PanelBox.Width(m.width - 4).Render(sb.String())
}

func (m Model) renderStashes() string {
	var sb strings.Builder
	sb.WriteString(m.styles.CardTitle.Render("Stashes"))
	sb.WriteString("  ")
	sb.WriteString(m.styles.Muted.Render("(a: apply, o: pop, d: drop)"))
	sb.WriteString("\n\n")

	if len(m.stashes) == 0 {
		sb.WriteString(m.styles.Success.Render("  ✓ No stashes found.\n"))
	} else {
		for i, s := range m.stashes {
			cursor := "  "
			if i == m.stashCursor {
				cursor = "> "
			}

			line := fmt.Sprintf("%s%-10s %s", cursor, s.Ref, s.Message)
			if i == m.stashCursor {
				sb.WriteString(m.styles.SelectedItem.Render(line))
			} else {
				sb.WriteString(m.styles.NormalItem.Render(line))
			}
			sb.WriteString("\n")
		}
	}

	return m.styles.PanelBox.Width(m.width - 4).Render(sb.String())
}

func (m Model) renderHelp() string {
	helpText := `DevGit Keyboard Navigation & Shortcuts

Navigation
  1..6         Jump directly to view (1:Dashboard 2:Status 3:Branches 4:Log 5:Diff 6:Stashes)
  Tab / → / l  Next tab
  Shift+Tab / ← / h  Previous tab
  ↑ / k        Move up in list
  ↓ / j        Move down in list
  Enter        Select item / view details

Actions
  p            Safe remote push (with ahead check and confirmation)
  P / u        Safe fast-forward pull (with behind check)
  r            Refresh repository state
  n            Create new branch (on Branches view)
  d            Delete branch (Branches view) or drop stash (Stashes view)
  a            Apply stash (Stashes view)
  o            Pop stash (Stashes view)
  ?            Toggle this help overlay
  q / Ctrl+C   Quit DevGit TUI

Press ? or Esc to return.`

	return m.styles.PanelBox.Width(m.width - 4).Render(helpText)
}

func (m Model) renderFooter() string {
	keys := []string{
		m.styles.KeyHelp.Render("1..6") + " " + m.styles.KeyDesc.Render("views"),
		m.styles.KeyHelp.Render("↑↓") + " " + m.styles.KeyDesc.Render("nav"),
		m.styles.KeyHelp.Render("Enter") + " " + m.styles.KeyDesc.Render("select"),
		m.styles.KeyHelp.Render("p") + " " + m.styles.KeyDesc.Render("push"),
		m.styles.KeyHelp.Render("P") + " " + m.styles.KeyDesc.Render("pull"),
		m.styles.KeyHelp.Render("r") + " " + m.styles.KeyDesc.Render("refresh"),
		m.styles.KeyHelp.Render("?") + " " + m.styles.KeyDesc.Render("help"),
		m.styles.KeyHelp.Render("q") + " " + m.styles.KeyDesc.Render("quit"),
	}

	bar := strings.Join(keys, "  •  ")
	return m.styles.FooterBar.Width(m.width - 2).Render(bar)
}

func (m Model) renderConfirmModal() string {
	var sb strings.Builder
	sb.WriteString(m.styles.ModalTitle.Render(m.confirmTitle))
	sb.WriteString("\n\n")

	if m.confirmWarning != "" {
		sb.WriteString(m.styles.ModalWarn.Render(m.confirmWarning))
		sb.WriteString("\n\n")
	}

	sb.WriteString(m.confirmPrompt)
	sb.WriteString("\n\n")
	sb.WriteString(m.styles.KeyHelp.Render("[y] Yes"))
	sb.WriteString("    ")
	sb.WriteString(m.styles.KeyDesc.Render("[n] No / Esc"))

	return m.styles.ModalBox.Width(min(m.width-10, 60)).Render(sb.String())
}

func (m Model) renderResultModal() string {
	var sb strings.Builder
	sb.WriteString(m.styles.ModalTitle.Render(m.resultTitle))
	sb.WriteString("\n\n")

	if m.resultIsError {
		sb.WriteString(m.styles.Danger.Render(m.resultMessage))
	} else {
		sb.WriteString(m.styles.Success.Render(m.resultMessage))
	}

	sb.WriteString("\n\n")
	sb.WriteString(m.styles.KeyDesc.Render("Press Enter or Esc to dismiss."))

	return m.styles.ModalBox.Width(min(m.width-10, 65)).Render(sb.String())
}

func (m Model) renderInputModal() string {
	var sb strings.Builder
	sb.WriteString(m.styles.ModalTitle.Render(m.inputTitle))
	sb.WriteString("\n\n")
	sb.WriteString(m.inputPrompt)
	sb.WriteString("\n")
	sb.WriteString(m.styles.SelectedItem.Render(fmt.Sprintf(" > %s ", m.inputValue)))
	sb.WriteString("\n\n")
	sb.WriteString(m.styles.KeyDesc.Render("Enter: submit   Esc: cancel"))

	return m.styles.ModalBox.Width(min(m.width-10, 55)).Render(sb.String())
}

func (m Model) overlayModal(modal string) string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
