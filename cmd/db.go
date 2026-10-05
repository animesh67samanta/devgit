package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"devgit/internal/database"
	"github.com/spf13/cobra"
)

// NewDBCmd creates and returns the 'db' command for database inspection and maintenance.
func NewDBCmd() *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Inspect and maintain the local DevGit SQLite database",
		Long: `View database diagnostics, tracked repositories, history retention,
and user preferences stored in the local SQLite metadata database.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDBStatus(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	// Subcommand: devgit db status
	statusCmd := &cobra.Command{
		Use:           "status",
		Short:         "Show database diagnostics, file size, and record counts",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDBStatus(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	// Subcommand: devgit db repos
	reposCmd := &cobra.Command{
		Use:           "repos",
		Short:         "List tracked Git repositories and access timestamps",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDBRepos(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	// Subcommand: devgit db prune [--keep N]
	var keepCount int
	pruneCmd := &cobra.Command{
		Use:           "prune",
		Short:         "Prune history and remove deleted repositories from metadata",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDBPrune(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), keepCount)
		},
	}
	pruneCmd.Flags().IntVar(&keepCount, "keep", 100, "Number of most recent history entries to retain")

	// Subcommand group: devgit db pref
	prefCmd := &cobra.Command{
		Use:   "pref",
		Short: "Manage persistent user and TUI preferences in SQLite",
	}

	prefListCmd := &cobra.Command{
		Use:           "list",
		Short:         "List all stored preferences",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDBPrefList(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	prefGetCmd := &cobra.Command{
		Use:           "get <key>",
		Short:         "Get the value of a stored preference",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Please specify a preference key.")
				return errors.New("missing preference key")
			}
			return RunDBPrefGet(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
		},
	}

	prefSetCmd := &cobra.Command{
		Use:           "set <key> <value>",
		Short:         "Set a persistent preference in SQLite",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				fmt.Fprintln(cmd.ErrOrStderr(), "❌ Usage: devgit db pref set <key> <value>")
				return errors.New("missing preference key or value")
			}
			val := strings.Join(args[1:], " ")
			return RunDBPrefSet(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0], val)
		},
	}

	prefCmd.AddCommand(prefListCmd)
	prefCmd.AddCommand(prefGetCmd)
	prefCmd.AddCommand(prefSetCmd)

	dbCmd.AddCommand(statusCmd)
	dbCmd.AddCommand(reposCmd)
	dbCmd.AddCommand(pruneCmd)
	dbCmd.AddCommand(prefCmd)

	return dbCmd
}

// RunDBStatus outputs database diagnostics and table counts.
func RunDBStatus(ctx context.Context, out, errOut io.Writer) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open DevGit database: %v\n", err)
		return err
	}
	defer db.Close()

	diag, err := db.Diagnostics(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to retrieve diagnostics: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "DevGit Database Diagnostics")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  Status:        Ready\n")
	fmt.Fprintf(out, "  Database File: %s\n", diag.Path)
	fmt.Fprintf(out, "  Schema:        Version %d\n", diag.SchemaVersion)
	fmt.Fprintf(out, "  File Size:     %s\n", formatBytes(diag.FileSizeBytes))
	walStr := "Disabled"
	if diag.WALEnabled {
		walStr = "Enabled (WAL)"
	}
	fmt.Fprintf(out, "  Journal Mode:  %s\n", walStr)
	fmt.Fprintf(out, "  Repositories:  %d tracked\n", diag.RepositoryCount)
	fmt.Fprintf(out, "  History:       %d command(s)\n", diag.HistoryCount)
	fmt.Fprintf(out, "  Preferences:   %d item(s)\n", diag.PreferenceCount)

	return nil
}

// RunDBRepos displays tracked repositories ordered by last accessed.
func RunDBRepos(ctx context.Context, out, errOut io.Writer) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open DevGit database: %v\n", err)
		return err
	}
	defer db.Close()

	repos, err := db.ListRecentRepositories(ctx, 50)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to list repositories: %v\n", err)
		return err
	}

	if len(repos) == 0 {
		fmt.Fprintln(out, "No tracked repositories found.")
		return nil
	}

	fmt.Fprintln(out, "Recent Repositories")
	fmt.Fprintln(out)

	for _, r := range repos {
		fmt.Fprintf(out, "  • %-40s (last used: %s)\n", r.Path, formatRelativeTime(r.LastUsedAt))
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Total: %d tracked repository workspace(s).\n", len(repos))
	return nil
}

// RunDBPrune cleans up stale repositories and limits command history to keepMax.
func RunDBPrune(ctx context.Context, out, errOut io.Writer, keepMax int) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open DevGit database: %v\n", err)
		return err
	}
	defer db.Close()

	prunedHistory, err := db.PruneHistory(ctx, keepMax, 0)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to prune history: %v\n", err)
		return err
	}

	removedRepos, err := db.CleanMissingRepositories(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to clean missing repositories: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "✓ Database maintenance complete.")
	fmt.Fprintf(out, "  Pruned %d old command history record(s).\n", prunedHistory)
	fmt.Fprintf(out, "  Removed %d missing repository record(s).\n", removedRepos)

	return nil
}

// RunDBPrefList prints all preferences.
func RunDBPrefList(ctx context.Context, out, errOut io.Writer) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open database: %v\n", err)
		return err
	}
	defer db.Close()

	prefs, err := db.ListPreferences(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "❌ Failed to list preferences: %v\n", err)
		return err
	}

	if len(prefs) == 0 {
		fmt.Fprintln(out, "No preferences set.")
		return nil
	}

	fmt.Fprintln(out, "DevGit Preferences")
	fmt.Fprintln(out)
	for k, v := range prefs {
		fmt.Fprintf(out, "  %s = %s\n", k, v)
	}
	return nil
}

// RunDBPrefGet gets a specific preference.
func RunDBPrefGet(ctx context.Context, out, errOut io.Writer, key string) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open database: %v\n", err)
		return err
	}
	defer db.Close()

	val, found, err := db.GetPreference(ctx, key)
	if err != nil {
		fmt.Fprintf(errOut, "❌ %v\n", err)
		return err
	}

	if !found {
		fmt.Fprintf(errOut, "❌ Preference not found: %s\n", key)
		return errors.New("preference not found")
	}

	fmt.Fprintln(out, val)
	return nil
}

// RunDBPrefSet sets a specific preference.
func RunDBPrefSet(ctx context.Context, out, errOut io.Writer, key, value string) error {
	db, err := database.OpenDefault()
	if err != nil {
		fmt.Fprintf(errOut, "❌ Unable to open database: %v\n", err)
		return err
	}
	defer db.Close()

	if err := db.SetPreference(ctx, key, value); err != nil {
		fmt.Fprintf(errOut, "❌ Failed to set preference: %v\n", err)
		return err
	}

	fmt.Fprintln(out, "✓ Preference saved.")
	fmt.Fprintf(out, "%s = %s\n", key, value)
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
