package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up stale worktrees",
	Long: `Remove worktrees that are no longer needed.

A worktree is considered stale if:
- Its directory no longer exists
- Its branch has been merged
- Its branch no longer exists

By default, prompts for confirmation before removing each worktree.`,
	RunE: runClean,
}

var (
	cleanAll   bool
	cleanDry   bool
	cleanForce bool
)

func init() {
	cleanCmd.Flags().BoolVarP(&cleanAll, "all", "a", false, "remove all stale worktrees without prompting")
	cleanCmd.Flags().BoolVar(&cleanDry, "dry-run", false, "show what would be removed without removing")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "force removal even if worktree has uncommitted changes")
}

func runClean(cmd *cobra.Command, args []string) error {
	// Initialize state
	if err := state.InitWorktreeState(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Prune git worktrees first
	printInfo("Pruning stale git worktree references...")
	if err := git.PruneWorktrees(); err != nil {
		printWarning(fmt.Sprintf("Could not prune worktrees: %v", err))
	}

	// Sync our state with git
	if err := state.DefaultWorktreeState.SyncWithGit(); err != nil {
		printWarning(fmt.Sprintf("Could not sync state: %v", err))
	}

	// Find stale worktrees
	stale, err := state.DefaultWorktreeState.FindStale()
	if err != nil {
		return fmt.Errorf("failed to find stale worktrees: %w", err)
	}

	if len(stale) == 0 {
		printSuccess("No stale worktrees found")
		return nil
	}

	printInfo(fmt.Sprintf("Found %d stale worktree(s)", len(stale)))
	fmt.Println()

	// Show what we found
	for i, info := range stale {
		reason := "unknown"

		// Check why it's stale
		if _, err := os.Stat(info.Path); os.IsNotExist(err) {
			reason = "directory does not exist"
		} else if merged, _ := git.IsBranchMerged(info.Branch); merged {
			reason = "branch has been merged"
		} else if exists, _ := git.BranchExists(info.Branch); !exists {
			reason = "branch no longer exists"
		}

		fmt.Printf("%d. %s\n", i+1, info.IssueID)
		fmt.Printf("   Path: %s\n", info.Path)
		fmt.Printf("   Branch: %s\n", info.Branch)
		fmt.Printf("   Reason: %s\n", reason)
		fmt.Println()
	}

	if cleanDry {
		fmt.Println("Dry run - no worktrees were removed")
		return nil
	}

	// Remove worktrees
	var toRemove []*state.WorktreeInfo
	if cleanAll {
		toRemove = stale
	} else if ui.IsInteractive() {
		// Build options for multi-select
		options := make([]ui.SelectOption, len(stale))
		for i, info := range stale {
			reason := "unknown"
			if _, err := os.Stat(info.Path); os.IsNotExist(err) {
				reason = "directory does not exist"
			} else if merged, _ := git.IsBranchMerged(info.Branch); merged {
				reason = "branch has been merged"
			} else if exists, _ := git.BranchExists(info.Branch); !exists {
				reason = "branch no longer exists"
			}

			options[i] = ui.SelectOption{
				Label:       info.IssueID,
				Value:       info.IssueID,
				Description: reason,
			}
		}

		selectedIDs, err := ui.RunMultiSelectWithDefault("Select worktrees to remove:", options)
		if err != nil {
			return fmt.Errorf("failed to select worktrees: %w", err)
		}

		// Map selected IDs back to WorktreeInfo
		selectedSet := make(map[string]bool)
		for _, id := range selectedIDs {
			selectedSet[id] = true
		}
		for _, info := range stale {
			if selectedSet[info.IssueID] {
				toRemove = append(toRemove, info)
			}
		}
	} else {
		// Non-interactive: prompt for each (fallback)
		for _, info := range stale {
			fmt.Printf("Remove worktree for %s? [y/N] ", info.IssueID)
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "Y" {
				toRemove = append(toRemove, info)
			}
		}
	}

	if len(toRemove) == 0 {
		printInfo("No worktrees selected for removal")
		return nil
	}

	// Actually remove them
	removed := 0
	for _, info := range toRemove {
		removeFunc := func() error {
			// Check if directory exists
			if _, err := os.Stat(info.Path); err == nil {
				// Remove git worktree
				if err := git.RemoveWorktree(info.Path); err != nil {
					if !cleanForce {
						return err
					}
					// Force remove the directory
					if err := os.RemoveAll(info.Path); err != nil {
						return err
					}
				}
			}

			// Remove from tracking
			if err := state.DefaultWorktreeState.Untrack(info.IssueID); err != nil {
				// Non-fatal
			}

			// Try to delete the branch if it exists and is merged
			if merged, _ := git.IsBranchMerged(info.Branch); merged {
				git.DeleteBranch(info.Branch, false)
			}

			return nil
		}

		if ui.IsInteractive() {
			err := ui.RunWithSpinner(fmt.Sprintf("Removing worktree for %s", info.IssueID), removeFunc)
			if err != nil {
				printWarning(fmt.Sprintf("Could not remove worktree: %v", err))
				continue
			}
		} else {
			printInfo(fmt.Sprintf("Removing worktree for %s...", info.IssueID))
			if err := removeFunc(); err != nil {
				printWarning(fmt.Sprintf("Could not remove worktree: %v", err))
				continue
			}
		}

		removed++
	}

	if removed > 0 {
		printSuccess(fmt.Sprintf("Removed %d worktree(s)", removed))
	}

	return nil
}
