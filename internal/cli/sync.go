package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
)

var syncCmd = &cobra.Command{
	Use:   "sync [ISSUE]",
	Short: "Sync PR info from GitHub",
	Long: `Sync pull request information from GitHub into issue metadata.

This command fetches PR info (number, URL) from GitHub for tracked issues
and stores it in the local metadata cache. This enables commands like
'jig merge ISSUE' to work without being on the PR branch.

Examples:
  jig sync              # Sync current issue (from branch)
  jig sync NUM-123      # Sync specific issue
  jig sync --all        # Sync all tracked issues`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

var syncAll bool

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all tracked issues")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Check if gh is available
	if !git.GHAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is not available or not authenticated")
	}

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	if syncAll {
		return syncAllIssues()
	}

	// Determine issue ID
	var issueID string
	if len(args) > 0 {
		issueID = args[0]
	} else {
		// Try to detect from current branch
		branch, err := git.GetCurrentBranch()
		if err == nil && branch != "" {
			// Try to extract issue ID from branch name (e.g., NUM-123-feature-name)
			parts := strings.SplitN(branch, "-", 3)
			if len(parts) >= 2 {
				issueID = parts[0] + "-" + parts[1]
			}
		}
	}

	if issueID == "" {
		return fmt.Errorf("could not detect issue - provide ISSUE argument or run from a worktree")
	}

	return syncIssue(issueID)
}

func syncIssue(issueID string) error {
	printInfo(fmt.Sprintf("Syncing PR info for %s...", issueID))

	prNumber, err := state.DefaultCache.SyncPRForIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	if prNumber == 0 {
		printWarning(fmt.Sprintf("No PR found for %s", issueID))
		return nil
	}

	printSuccess(fmt.Sprintf("Found and synced PR #%d for %s", prNumber, issueID))
	return nil
}

func syncAllIssues() error {
	printInfo("Syncing all tracked issues...")

	metadata, err := state.DefaultCache.ListIssueMetadata()
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if len(metadata) == 0 {
		printWarning("No tracked issues found")
		return nil
	}

	var synced, notFound, alreadySynced, failed int

	for _, meta := range metadata {
		if meta.PRNumber > 0 {
			alreadySynced++
			continue
		}

		if meta.BranchName == "" {
			failed++
			continue
		}

		prNumber, err := state.DefaultCache.SyncPRForIssue(meta.IssueID)
		if err != nil {
			printWarning(fmt.Sprintf("%s: sync failed - %v", meta.IssueID, err))
			failed++
			continue
		}

		if prNumber == 0 {
			notFound++
			continue
		}

		printSuccess(fmt.Sprintf("%s: synced PR #%d", meta.IssueID, prNumber))
		synced++
	}

	fmt.Println()
	fmt.Printf("Summary: %d synced, %d already synced, %d no PR found, %d failed\n",
		synced, alreadySynced, notFound, failed)

	return nil
}
