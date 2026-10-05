package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"devgit/internal/database"
	"devgit/internal/git"
	"github.com/spf13/cobra"
)

// HistoryFlags holds filtering options for command history listing.
type HistoryFlags struct {
	Limit       int
	OnlySuccess bool
	OnlyFailed  bool
	Keep        int
}

// NewHistoryCmd creates and returns the 'history' command and subcommands.
func NewHistoryCmd() *cobra.Command {
	var flags HistoryFlags

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "View and manage DevGit command execution history",
		Long: `Inspect recently executed DevGit commands, filter by status,
and manage history retention in the local SQLite database.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunHistoryList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), flags)
		},
	}

	historyCmd.Flags().IntVarP(&flags.Limit, "limit", "n", 20, "Maximum number of history records to show")
	historyCmd.Flags().BoolVar(&flags.OnlySuccess, "success", false, "Show only successful commands")
	historyCmd.Flags().BoolVar(&flags.OnlyFailed, "failed", false, "Show only failed commands")

	// Subcommand: devgit history list
	listCmd := &cobra.Command{
		Use:           "list",
		Short:         "List recent command execution history",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunHistoryList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), flags)
		},
	}
	listCmd.Flags().IntVarP(&flags.Limit, "limit", "n", 20, "Maximum number of history records to show")
	listCmd.Flags().BoolVar(&flags.OnlySuccess, "success", false, "Show only successful commands")
	listCmd.Flags().BoolVar(&flags.OnlyFailed, "failed", false, "Show only failed commands")

	// Subcommand: devgit history clear
	var clearFlags HistoryFlags
	clearCmd := &cobra.Command{
		Use:           "clear",
		Short:         "Clear or prune command execution history",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunHistoryClear(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), clearFlags)
		},
	}
	clearCmd.Flags().IntVar(&clearFlags.Keep, "keep", 0, "Retain the N most recent history entries")

	historyCmd.AddCommand(listCmd)
	historyCmd.AddCommand(clearCmd)

	return historyCmd
}

// RunHistoryList queries and formats command history from SQLite.
func RunHistoryList(ctx context.Context, out, errOut io.Writer, flags HistoryFlags) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Database unavailable: %v\n", err)
		return err
	}
	defer db.Close()

	records, err := db.ListHistory(ctx, database.HistoryListOptions{
		Limit:       flags.Limit,
		OnlySuccess: flags.OnlySuccess,
		OnlyFailed:  flags.OnlyFailed,
	})
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to query command history: %v\n", err)
		return err
	}

	if len(records) == 0 {
		fmt.Fprintln(out, "No command history found.")
		return nil
	}

	fmt.Fprintln(out, "DevGit Command History")
	fmt.Fprintln(out)

	for i, r := range records {
		status := "✓"
		if !r.Success {
			status = "❌"
		}

		timeStr := formatRelativeTime(r.ExecutedAt)
		durStr := fmt.Sprintf("%dms", r.DurationMS)

		repoInfo := ""
		if r.RepositoryPath != "" {
			repoInfo = fmt.Sprintf("  (%s)", r.RepositoryPath)
		}

		fmt.Fprintf(out, "  [%d] %-24s  %s  %6s    %-12s%s\n",
			i+1,
			r.Command,
			status,
			durStr,
			timeStr,
			repoInfo,
		)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Showing %d recent command(s).\n", len(records))
	return nil
}

// RunHistoryClear removes or prunes records from command history.
func RunHistoryClear(ctx context.Context, out, errOut io.Writer, flags HistoryFlags) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Database unavailable: %v\n", err)
		return err
	}
	defer db.Close()

	if flags.Keep > 0 {
		deleted, err := db.PruneHistory(ctx, flags.Keep, 0)
		if err != nil {
			fmt.Fprintf(errOut, "❌ Failed to prune history: %v\n", err)
			return err
		}
		fmt.Fprintf(out, "✓ Kept %d most recent command(s) (%d record(s) removed).\n", flags.Keep, deleted)
		return nil
	}

	deleted, err := db.ClearHistory(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to clear history: %v\n", err)
		return err
	}

	fmt.Fprintf(out, "✓ Command history cleared (%d record(s) removed).\n", deleted)
	return nil
}

// RecordCommandExecutionSafe records a completed command execution into SQLite metadata safely.
// Any database errors are caught and suppressed so Git operations are never interrupted.
func RecordCommandExecutionSafe(cmdName string, duration time.Duration, success bool, workDir string) {
	defer func() {
		_ = recover()
	}()

	db, err := database.OpenDefault()
	if err != nil {
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	var repoID *int64
	if client, err := git.NewClient(workDir); err == nil {
		if inside, err := client.IsInsideWorkTree(ctx); err == nil && inside {
			if root, err := client.Root(ctx); err == nil && root != "" {
				if repo, err := db.GetOrCreateRepository(ctx, root); err == nil && repo != nil {
					repoID = &repo.ID
				}
			}
		}
	}

	fullCmd := "devgit " + cmdName
	_, _ = db.RecordCommand(ctx, database.HistoryEntry{
		RepositoryID: repoID,
		Command:      fullCmd,
		Success:      success,
		ExecutedAt:   time.Now().UTC(),
		DurationMS:   duration.Milliseconds(),
	})
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}
