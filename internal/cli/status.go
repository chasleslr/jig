package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status [ISSUE]",
	Short: "Show status of current issue/worktree",
	Long: `Display the status of the current issue or a specific issue.

Shows:
- Plan status and phase progress
- Linear issue status
- PR status and unresolved comments
- Worktree information`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	if err := state.InitWorktreeState(); err != nil {
		return fmt.Errorf("failed to initialize worktree state: %w", err)
	}

	var issueID string
	var worktreeInfo *state.WorktreeInfo

	if len(args) > 0 {
		issueID = args[0]
		worktreeInfo, _ = state.DefaultWorktreeState.Get(issueID)
	} else {
		// Try to detect from current directory
		cwd, err := git.GetWorktreeRoot()
		if err == nil {
			worktreeInfo, _ = state.DefaultWorktreeState.GetByPath(cwd)
			if worktreeInfo != nil {
				issueID = worktreeInfo.IssueID
			}
		}

		// Try to detect from branch name
		if issueID == "" {
			branch, _ := git.GetCurrentBranch()
			if branch != "" {
				parts := strings.SplitN(branch, "-", 3)
				if len(parts) >= 2 {
					issueID = parts[0] + "-" + parts[1]
				}
			}
		}
	}

	// If still no issue ID, try to prompt from active worktrees
	if issueID == "" && ui.IsInteractive() {
		worktrees, err := state.DefaultWorktreeState.List()
		if err == nil && len(worktrees) > 0 {
			options := make([]ui.SelectOption, len(worktrees))
			for i, wt := range worktrees {
				desc := wt.Branch
				if wt.Path != "" {
					desc = wt.Path
				}
				options[i] = ui.SelectOption{
					Label:       wt.IssueID,
					Value:       wt.IssueID,
					Description: desc,
				}
			}

			selected, err := ui.RunSelect("Select issue to show status:", options)
			if err != nil {
				return fmt.Errorf("failed to select issue: %w", err)
			}
			if selected != "" {
				issueID = selected
				worktreeInfo, _ = state.DefaultWorktreeState.Get(issueID)
			}
		}
	}

	if issueID == "" {
		return fmt.Errorf("could not detect issue - provide ISSUE argument or run from a worktree")
	}

	fmt.Printf("Issue: %s\n", issueID)
	fmt.Println(strings.Repeat("=", 40))

	// Show worktree info
	if worktreeInfo != nil {
		fmt.Println()
		fmt.Println("## Worktree")
		fmt.Printf("  Path:   %s\n", worktreeInfo.Path)
		fmt.Printf("  Branch: %s\n", worktreeInfo.Branch)
	} else {
		// Check if worktree exists but not tracked
		wtInfo, _ := state.DefaultWorktreeState.GetByBranch(issueID)
		if wtInfo != nil {
			fmt.Println()
			fmt.Println("## Worktree")
			fmt.Printf("  Path:   %s\n", wtInfo.Path)
			fmt.Printf("  Branch: %s\n", wtInfo.Branch)
		}
	}

	// Show plan info from cache
	plan, _ := state.DefaultCache.GetPlan(issueID)
	if plan != nil {
		fmt.Println()
		fmt.Println("## Plan")
		fmt.Printf("  Title:  %s\n", plan.Title)
		fmt.Printf("  Status: %s\n", plan.Status)

		if len(plan.Phases) > 0 {
			fmt.Println()
			fmt.Println("  Phases:")
			for _, phase := range plan.Phases {
				status := "⬜"
				switch phase.Status {
				case "in-progress":
					status = "🔄"
				case "complete":
					status = "✅"
				case "blocked":
					status = "🚫"
				}
				deps := ""
				if len(phase.DependsOn) > 0 {
					deps = fmt.Sprintf(" (depends on: %s)", strings.Join(phase.DependsOn, ", "))
				}
				fmt.Printf("    %s %s%s\n", status, phase.Title, deps)
			}
			fmt.Printf("\n  Progress: %.0f%%\n", plan.Progress())
		}
	}

	// Show tracker info
	t, err := getTracker(cfg)
	if err == nil {
		issue, err := t.GetIssue(ctx, issueID)
		if err == nil {
			fmt.Println()
			fmt.Println("## Tracker")
			fmt.Printf("  Status:   %s\n", issue.Status)
			fmt.Printf("  Priority: %d\n", issue.Priority)
			if issue.Assignee != "" {
				fmt.Printf("  Assignee: %s\n", issue.Assignee)
			}
			if issue.URL != "" {
				fmt.Printf("  URL:      %s\n", issue.URL)
			}
		}
	}

	// Show PR info
	if git.GHAvailable() {
		branch := ""
		if worktreeInfo != nil {
			branch = worktreeInfo.Branch
		}
		if branch == "" {
			branch, _ = git.GetCurrentBranch()
		}

		if branch != "" {
			pr, err := git.GetPRForBranch(branch)
			if err == nil && pr != nil {
				fmt.Println()
				fmt.Println("## Pull Request")
				fmt.Printf("  #%d: %s\n", pr.Number, pr.Title)
				fmt.Printf("  State:     %s\n", pr.State)
				if pr.IsDraft {
					fmt.Printf("  Draft:     yes\n")
				}
				fmt.Printf("  Mergeable: %s\n", pr.Mergeable)
				fmt.Printf("  URL:       %s\n", pr.URL)

				// Check for unresolved comments
				comments, _ := git.GetPRReviewThreads(pr.Number)
				if len(comments) > 0 {
					fmt.Printf("\n  ⚠ %d unresolved review comment(s)\n", len(comments))
					fmt.Printf("  Run 'jig review' to address them\n")
				}

				// Check CI status
				ciStatus, _ := git.GetCIStatus()
				if ciStatus != "" && ciStatus != "unknown" {
					statusIcon := "✓"
					if ciStatus == "failure" {
						statusIcon = "✗"
					} else if ciStatus == "pending" {
						statusIcon = "○"
					}
					fmt.Printf("\n  CI Status: %s %s\n", statusIcon, ciStatus)
				}
			}
		}
	}

	// Show metadata
	meta, _ := state.DefaultCache.GetIssueMetadata(issueID)
	if meta != nil && meta.LastActive.Year() > 1 {
		fmt.Println()
		fmt.Println("## Activity")
		fmt.Printf("  Last active: %s\n", meta.LastActive.Format("2006-01-02 15:04"))
	}

	return nil
}
