package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var mergeCmd = &cobra.Command{
	Use:   "merge [ISSUE]",
	Short: "Merge an approved PR",
	Long: `Merge a pull request after validating it's ready.

This command:
1. Checks for unresolved review comments
2. Validates CI status
3. Checks if stacked PRs' dependencies are merged
4. Merges the PR via gh
5. Updates the tracker issue status
6. Cleans up the local worktree`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMerge,
}

var (
	mergeMethod       string
	mergeDeleteBranch bool
	mergeForce        bool
	mergePR           int
)

var mergeMethodSet bool

func init() {
	mergeCmd.Flags().StringVarP(&mergeMethod, "method", "m", "squash", "merge method (merge, squash, rebase)")
	mergeCmd.Flags().BoolVar(&mergeDeleteBranch, "delete-branch", true, "delete branch after merge")
	mergeCmd.Flags().BoolVar(&mergeForce, "force", false, "skip validation checks")
	mergeCmd.Flags().IntVar(&mergePR, "pr", 0, "PR number (auto-detected if not specified)")
}

func runMerge(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	// Check if gh is available
	if !git.GHAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is not available or not authenticated")
	}

	// Determine issue ID
	var issueID string
	if len(args) > 0 {
		issueID = args[0]
	}

	// Get PR number
	prNumber := mergePR
	if prNumber == 0 {
		// Try to get from current branch
		pr, err := git.GetPR()
		if err != nil {
			return fmt.Errorf("no PR found for current branch (use --pr to specify)")
		}
		prNumber = pr.Number
	}

	printInfo(fmt.Sprintf("Checking PR #%d...", prNumber))

	// Get PR details
	pr, err := git.GetPRByNumber(prNumber)
	if err != nil {
		return fmt.Errorf("failed to get PR: %w", err)
	}

	// Extract issue ID from branch if not provided
	if issueID == "" {
		parts := strings.SplitN(pr.HeadRefName, "-", 3)
		if len(parts) >= 2 {
			issueID = parts[0] + "-" + parts[1]
		}
	}

	// Prompt for merge method if not explicitly set via flag and interactive
	if !cmd.Flags().Changed("method") && ui.IsInteractive() {
		options := []ui.SelectOption{
			{Label: "Squash and merge", Value: "squash", Description: "Combine all commits into one"},
			{Label: "Create merge commit", Value: "merge", Description: "Preserve all commits"},
			{Label: "Rebase and merge", Value: "rebase", Description: "Rebase commits onto base branch"},
		}
		selected, err := ui.RunSelect("Merge method:", options)
		if err != nil {
			return fmt.Errorf("failed to select merge method: %w", err)
		}
		if selected != "" {
			mergeMethod = selected
		}
	}

	// Validation checks (unless forced)
	if !mergeForce {
		var validationIssues []string

		// Check if PR is draft
		if pr.IsDraft {
			validationIssues = append(validationIssues, "PR is still a draft")
		}

		// Check for unresolved comments
		comments, err := git.GetPRReviewThreads(prNumber)
		if err == nil && len(comments) > 0 {
			validationIssues = append(validationIssues, fmt.Sprintf("%d unresolved review comment(s)", len(comments)))
		}

		// Check CI status
		ciStatus, err := git.GetCIStatus()
		if err == nil {
			switch ciStatus {
			case "failure":
				validationIssues = append(validationIssues, "CI checks are failing")
			case "pending":
				printWarning("CI checks are still running")
			}
		}

		// Check if mergeable
		if pr.Mergeable == "CONFLICTING" {
			return fmt.Errorf("PR has merge conflicts - resolve them first")
		}

		// If there are validation issues, prompt for confirmation
		if len(validationIssues) > 0 {
			if ui.IsInteractive() {
				fmt.Println()
				printWarning("The following issues were found:")
				for _, issue := range validationIssues {
					fmt.Printf("  - %s\n", issue)
				}
				fmt.Println()

				confirmed, err := ui.RunConfirm("Proceed with merge anyway?")
				if err != nil {
					return fmt.Errorf("failed to confirm: %w", err)
				}
				if !confirmed {
					return fmt.Errorf("merge cancelled")
				}
			} else {
				return fmt.Errorf("validation issues found: %s (use --force to override)", strings.Join(validationIssues, ", "))
			}
		}
	}

	printInfo(fmt.Sprintf("Merging PR #%d (%s)...", prNumber, pr.Title))

	// Check if a worktree exists for this branch BEFORE merging
	// If it does, we need to merge without --delete-branch, then clean up manually
	var worktreeInfo *state.WorktreeInfo
	if err := state.InitWorktreeState(); err == nil {
		worktreeInfo, _ = state.DefaultWorktreeState.GetByBranch(pr.HeadRefName)
	}

	// Merge the PR
	// If a worktree exists for the branch, don't use --delete-branch flag
	// because git will refuse to delete a branch that's checked out in a worktree
	deleteBranchOnMerge := mergeDeleteBranch && worktreeInfo == nil
	if err := git.MergePR(prNumber, mergeMethod, deleteBranchOnMerge); err != nil {
		return fmt.Errorf("failed to merge PR: %w", err)
	}

	printSuccess(fmt.Sprintf("PR #%d merged!", prNumber))

	// Clean up worktree if it exists (must be done before branch deletion)
	if worktreeInfo != nil {
		printInfo(fmt.Sprintf("Cleaning up worktree at %s...", worktreeInfo.Path))

		// Check if we're inside the worktree being removed
		// If so, change to the main repo root first to avoid "no such file or directory" errors
		insideWorktree, _ := git.IsInsidePath(worktreeInfo.Path)
		if insideWorktree {
			mainRepoRoot, err := git.GetMainRepoRoot()
			if err == nil {
				printInfo(fmt.Sprintf("Changing to %s before cleanup...", mainRepoRoot))
				_ = os.Chdir(mainRepoRoot)
			}
		}

		if err := git.RemoveWorktree(worktreeInfo.Path); err != nil {
			printWarning(fmt.Sprintf("Could not remove worktree: %v", err))
		} else {
			state.DefaultWorktreeState.Untrack(worktreeInfo.IssueID)
			printSuccess("Worktree removed")
		}

		// Now delete the branch manually if requested
		if mergeDeleteBranch {
			if err := git.DeleteBranch(pr.HeadRefName, false); err != nil {
				printWarning(fmt.Sprintf("Could not delete local branch: %v", err))
			}
		}
	}

	// Update plan status (both local cache and tracker) if we have an issue ID
	if issueID != "" {
		printInfo(fmt.Sprintf("Updating issue %s status...", issueID))

		// Initialize state and try to update the plan (with remote fallback)
		if err := state.Init(); err == nil {
			if p, _, err := lookupPlanByIDWithFallback(issueID, &LookupPlanOptions{
				FetchFromRemote: true,
				Config:          cfg,
				Context:         ctx,
			}); err == nil && p != nil {
				// Create tracker syncer if configured
				var syncer state.TrackerSyncer
				if cfg.Default.Tracker == "linear" {
					syncer = getLinearSyncer(cfg)
				}

				// Use PlanStatusManager to complete the plan
				mgr := state.NewPlanStatusManager(state.DefaultCache, syncer)
				result, err := mgr.Complete(ctx, p)
				if err != nil {
					printWarning(fmt.Sprintf("Could not update plan status: %v", err))
				} else {
					if result.CacheSaved {
						printSuccess("Plan status updated to Complete")
					}
					if result.TrackerSynced {
						printSuccess("Issue status synced to tracker")
					} else if result.TrackerError != nil {
						printWarning(fmt.Sprintf("Could not sync to tracker: %v", result.TrackerError))
					}
				}
			} else {
				// No local plan, just update tracker directly
				t, err := getTracker(cfg)
				if err != nil {
					printWarning(fmt.Sprintf("Could not connect to tracker: %v", err))
				} else {
					if err := t.TransitionIssue(ctx, issueID, "done"); err != nil {
						printWarning(fmt.Sprintf("Could not update issue status: %v", err))
					} else {
						printSuccess("Issue status updated to Done")
					}
				}
			}
		} else {
			// State init failed, fall back to tracker-only update
			t, err := getTracker(cfg)
			if err != nil {
				printWarning(fmt.Sprintf("Could not connect to tracker: %v", err))
			} else {
				if err := t.TransitionIssue(ctx, issueID, "done"); err != nil {
					printWarning(fmt.Sprintf("Could not update issue status: %v", err))
				} else {
					printSuccess("Issue status updated to Done")
				}
			}
		}
	}

	fmt.Println()
	fmt.Printf("PR merged: %s\n", pr.URL)

	return nil
}
