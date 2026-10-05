package cmd

import (
	"context"
	"fmt"
	"io"

	"devgit/internal/config"
	"devgit/internal/database"
	"devgit/internal/git"
	"devgit/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// NewTuiCmd creates and returns the 'tui' command.
func NewTuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tui",
		Short:         "Launch the interactive terminal UI",
		Long:          `Open an interactive, keyboard-driven terminal dashboard for DevGit to inspect status, branches, commits, diffs, and stashes.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunTUI(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), "")
		},
	}

	return cmd
}

// RunTUI initializes and runs the Bubble Tea interactive terminal UI.
func RunTUI(ctx context.Context, in io.Reader, out, errOut io.Writer, workDir string) error {
	client, err := git.NewClient(workDir)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error initializing Git client: %v\n", err)
		return err
	}

	inside, err := client.IsInsideWorkTree(ctx)
	if err != nil || !inside {
		fmt.Fprintln(errOut, notRepoMessage)
		return git.ErrNotRepository
	}

	cfg, err := config.Load(config.LoadOptions{WorkDir: workDir})
	if err != nil {
		fmt.Fprintf(errOut, "❌ %v\n", err)
		return err
	}

	var db *database.DB
	if d, err := database.OpenDefault(); err == nil {
		db = d
		defer db.Close()
	}

	model := tui.NewModelWithConfigAndDB(client, cfg, db)
	program := tea.NewProgram(model, tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())

	_, err = program.Run()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Error running TUI: %v\n", err)
		return err
	}

	return nil
}
