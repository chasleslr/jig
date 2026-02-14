package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout [ISSUE]",
	Short: "Create or switch to an issue's worktree",
	Long: `Create or check out a worktree for an issue.

If a worktree already exists for the issue, outputs its path.
Otherwise, creates a new worktree with a branch named after the issue.

If no ISSUE is provided, prompts to select from active issues.

Shell integration tip:
  cd $(jig checkout ENG-123)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCheckout,
}

var (
	checkoutQuiet bool
	checkoutBase  string
)

func init() {
	checkoutCmd.Flags().BoolVarP(&checkoutQuiet, "quiet", "q", false, "only output the path (for scripting)")
	checkoutCmd.Flags().StringVarP(&checkoutBase, "base", "b", "", "base branch for new worktree (default: main)")
}

func runCheckout(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	var issueID string
	if len(args) > 0 {
		issueID = args[0]
	}

	// Initialize state
	if err := state.InitWorktreeState(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// If no issue provided, prompt to select from active issues
	if issueID == "" {
		if !ui.IsInteractive() {
			return fmt.Errorf("issue ID is required")
		}

		var options []ui.SelectOption

		// Add existing worktrees
		worktrees, _ := state.DefaultWorktreeState.List()
		for _, wt := range worktrees {
			options = append(options, ui.SelectOption{
				Label:       wt.IssueID,
				Value:       wt.IssueID,
				Description: fmt.Sprintf("Existing worktree at %s", wt.Path),
			})
		}

		// Add issues from tracker
		t, err := getTracker(cfg)
		if err == nil {
			issues, err := t.SearchIssues(ctx, "")
			if err == nil {
				// Build a set of existing issue IDs to avoid duplicates
				existing := make(map[string]bool)
				for _, wt := range worktrees {
					existing[wt.IssueID] = true
				}

				for _, issue := range issues {
					if !existing[issue.Identifier] {
						options = append(options, ui.SelectOption{
							Label:       fmt.Sprintf("%s: %s", issue.Identifier, issue.Title),
							Value:       issue.Identifier,
							Description: string(issue.Status),
						})
					}
				}
			}
		}

		if len(options) == 0 {
			return fmt.Errorf("no issues found - provide an ISSUE argument")
		}

		selected, err := ui.RunSelect("Select issue:", options)
		if err != nil {
			return fmt.Errorf("failed to select issue: %w", err)
		}
		if selected == "" {
			return fmt.Errorf("no issue selected")
		}
		issueID = selected
	}

	// Check if we already have a worktree for this issue
	info, err := state.DefaultWorktreeState.Get(issueID)
	if err != nil {
		return fmt.Errorf("failed to check worktree state: %w", err)
	}

	if info != nil {
		// Check if the path still exists
		if _, err := os.Stat(info.Path); err == nil {
			// In interactive mode, confirm the switch
			if ui.IsInteractive() && !checkoutQuiet {
				confirmed, err := ui.RunConfirmWithDefault(
					fmt.Sprintf("Worktree already exists at %s. Switch to it?", info.Path),
					true,
				)
				if err != nil {
					return fmt.Errorf("failed to confirm: %w", err)
				}
				if !confirmed {
					return nil
				}
			}

			if checkoutQuiet {
				fmt.Println(info.Path)
			} else {
				printSuccess(fmt.Sprintf("Worktree exists at %s", info.Path))
				fmt.Printf("\nTo enter:\n  cd %s\n", info.Path)
			}
			// Update last used time
			state.DefaultWorktreeState.UpdateLastUsed(issueID)
			return nil
		}
		// Path no longer exists, clean up tracking
		state.DefaultWorktreeState.Untrack(issueID)
	}

	// Try to get issue title from cache for better branch name
	var title string
	if err := state.Init(); err == nil {
		// First try looking up a plan by the issue ID directly
		if plan, _, _ := lookupPlanByID(issueID); plan != nil {
			title = plan.Title
		} else if meta, _ := state.DefaultCache.GetIssueMetadata(issueID); meta != nil && meta.PlanID != "" {
			// Fall back to cached plan ID from metadata
			if plan, _ := state.DefaultCache.GetPlan(meta.PlanID); plan != nil {
				title = plan.Title
			}
		}
	}

	if title == "" {
		title = "feature"
	}

	// Generate branch name
	branchName := git.GenerateBranchName(issueID, title)

	if !checkoutQuiet {
		printInfo(fmt.Sprintf("Creating worktree for %s...", issueID))
	}

	// Create the worktree
	worktreePath, err := git.CreateWorktree(issueID, branchName)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Track the worktree
	repoRoot, _ := git.GetWorktreeRoot()
	wtInfo := &state.WorktreeInfo{
		IssueID:  issueID,
		Path:     worktreePath,
		Branch:   branchName,
		RepoPath: repoRoot,
	}
	if err := state.DefaultWorktreeState.Track(wtInfo); err != nil {
		if !checkoutQuiet {
			printWarning(fmt.Sprintf("Could not track worktree: %v", err))
		}
	}

	// Save metadata
	meta := &state.IssueMetadata{
		IssueID:      issueID,
		WorktreePath: worktreePath,
		BranchName:   branchName,
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		if !checkoutQuiet {
			printWarning(fmt.Sprintf("Could not save metadata: %v", err))
		}
	}

	if checkoutQuiet {
		fmt.Println(worktreePath)
	} else {
		printSuccess(fmt.Sprintf("Worktree created at %s", worktreePath))
		fmt.Printf("Branch: %s\n", branchName)
		fmt.Printf("\nTo enter:\n  cd %s\n", worktreePath)
	}

	return nil
}

// Helper function available for shell integration
func init() {
	// Add a completions subcommand for shell integration
	checkoutCmd.AddCommand(&cobra.Command{
		Use:    "__complete_issues",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// List tracked issues for completion
			if err := state.InitWorktreeState(); err != nil {
				return nil
			}

			worktrees, err := state.DefaultWorktreeState.List()
			if err != nil {
				return nil
			}

			for _, wt := range worktrees {
				fmt.Println(wt.IssueID)
			}

			// Also list from tracker if configured
			cfg := config.Get()
			t, err := getTracker(cfg)
			if err != nil {
				return nil
			}

			issues, err := t.SearchIssues(cmd.Context(), "")
			if err != nil {
				return nil
			}

			for _, issue := range issues {
				fmt.Println(issue.Identifier)
			}

			return nil
		},
	})
}
