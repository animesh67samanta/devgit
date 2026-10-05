package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings available across the TUI.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	NextTab key.Binding
	PrevTab key.Binding
	Enter   key.Binding
	Back    key.Binding
	Refresh key.Binding
	Help    key.Binding
	Quit    key.Binding

	// View jump keys
	ViewDashboard key.Binding
	ViewStatus    key.Binding
	ViewBranches  key.Binding
	ViewLog       key.Binding
	ViewDiff      key.Binding
	ViewStashes   key.Binding

	// Actions
	ActionPush   key.Binding
	ActionPull   key.Binding
	ActionCommit key.Binding
	ActionCreate key.Binding
	ActionDelete key.Binding
	ActionApply  key.Binding
	ActionPop    key.Binding
	ConfirmYes   key.Binding
	ConfirmNo    key.Binding
}

// DefaultKeyMap returns the standard navigation and action keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next tab"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		ViewDashboard: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "dashboard"),
		),
		ViewStatus: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "status"),
		),
		ViewBranches: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "branches"),
		),
		ViewLog: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "log"),
		),
		ViewDiff: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "diff"),
		),
		ViewStashes: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "stashes"),
		),
		ActionPush: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "push"),
		),
		ActionPull: key.NewBinding(
			key.WithKeys("P", "u"),
			key.WithHelp("P/u", "pull"),
		),
		ActionCommit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "commit"),
		),
		ActionCreate: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new branch"),
		),
		ActionDelete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete/drop"),
		),
		ActionApply: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "apply stash"),
		),
		ActionPop: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "pop stash"),
		),
		ConfirmYes: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y", "yes"),
		),
		ConfirmNo: key.NewBinding(
			key.WithKeys("n", "N"),
			key.WithHelp("n", "no"),
		),
	}
}
