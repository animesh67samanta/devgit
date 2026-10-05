package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds the centralized visual styles for the TUI.
type Styles struct {
	// App container
	App lipgloss.Style

	// Header & Titles
	HeaderTitle lipgloss.Style
	HeaderPath  lipgloss.Style
	HeaderBox   lipgloss.Style

	// Tabs
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabsBar     lipgloss.Style

	// Cards & Sections
	CardTitle lipgloss.Style
	CardBox   lipgloss.Style
	PanelBox  lipgloss.Style

	// Semantic Indicators
	Success lipgloss.Style
	Warning lipgloss.Style
	Danger  lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style

	// Lists & Selections
	SelectedItem   lipgloss.Style
	NormalItem     lipgloss.Style
	CurrentBranch  lipgloss.Style
	UpstreamBranch lipgloss.Style

	// Modals & Dialogs
	ModalTitle lipgloss.Style
	ModalBox   lipgloss.Style
	ModalWarn  lipgloss.Style

	// Footer & Keybinds
	FooterBar lipgloss.Style
	KeyHelp   lipgloss.Style
	KeyDesc   lipgloss.Style
}

// DefaultStyles returns standard Lip Gloss styles with graceful fallback.
func DefaultStyles() Styles {
	var s Styles

	primaryColor := lipgloss.Color("#7D56F4")
	accentColor := lipgloss.Color("#04B575")
	warnColor := lipgloss.Color("#FFB86C")
	dangerColor := lipgloss.Color("#FF5555")
	mutedColor := lipgloss.Color("#6272A4")
	bgHighlight := lipgloss.Color("#282A36")
	borderDim := lipgloss.Color("#44475A")

	s.App = lipgloss.NewStyle().
		Padding(0, 1)

	s.HeaderTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primaryColor).
		Padding(0, 1)

	s.HeaderPath = lipgloss.NewStyle().
		Foreground(mutedColor).
		PaddingLeft(1)

	s.HeaderBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(borderDim).
		PaddingBottom(0).
		MarginBottom(1)

	s.TabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primaryColor).
		Padding(0, 2)

	s.TabInactive = lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgHighlight).
		Padding(0, 2)

	s.TabsBar = lipgloss.NewStyle().
		MarginBottom(1)

	s.CardTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8BE9FD")).
		MarginBottom(1)

	s.CardBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderDim).
		Padding(1).
		MarginRight(1).
		MarginBottom(1)

	s.PanelBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderDim).
		Padding(1)

	s.Success = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(warnColor).
		Bold(true)

	s.Danger = lipgloss.NewStyle().
		Foreground(dangerColor).
		Bold(true)

	s.Info = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8BE9FD"))

	s.Muted = lipgloss.NewStyle().
		Foreground(mutedColor)

	s.SelectedItem = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primaryColor).
		Padding(0, 1)

	s.NormalItem = lipgloss.NewStyle().
		Padding(0, 1)

	s.CurrentBranch = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	s.UpstreamBranch = lipgloss.NewStyle().
		Foreground(mutedColor)

	s.ModalTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primaryColor).
		Padding(0, 1).
		MarginBottom(1)

	s.ModalBox = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2)

	s.ModalWarn = lipgloss.NewStyle().
		Foreground(warnColor).
		Bold(true).
		MarginBottom(1)

	s.FooterBar = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderDim).
		PaddingTop(0).
		MarginTop(1)

	s.KeyHelp = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)

	s.KeyDesc = lipgloss.NewStyle().
		Foreground(mutedColor)

	return s
}
