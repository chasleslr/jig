package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/runner"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/tracker"
	"github.com/charleslr/jig/internal/tracker/linear"
	"github.com/charleslr/jig/internal/ui"
)

var implementCmd = &cobra.Command{
	Use:   "implement [ISSUE]",
	Short: "Implement a plan",
	Long: `Set up a worktree and launch your coding tool to implement a plan.

This command:
1. Fetches the plan from the tracker or local cache
2. Creates or checks out a worktree for the issue
3. Prepares implementation context (plan, prompts)
4. Launches your configured coding tool

If no ISSUE is provided and the terminal is interactive, shows a table
of cached plans to select from.

After your implementation session, use 'gh pr create' to open a PR.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImplement,
}

var (
	implRunner       string
	implNoLaunch     bool
	implNoAutoAccept bool
)

func init() {
	implementCmd.Flags().StringVarP(&implRunner, "runner", "r", "", "coding tool to use (default from config)")
	implementCmd.Flags().BoolVar(&implNoLaunch, "no-launch", false, "set up worktree but don't launch tool")
	implementCmd.Flags().BoolVar(&implNoAutoAccept, "no-auto-accept", false, "disable automatic acceptance of file edits")
}

func runImplement(cmd *cobra.Command, args []string) error {
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
	var p *plan.Plan

	// If no issue ID provided, show interactive plan selection
	if len(args) == 0 {
		if !ui.IsInteractive() {
			return fmt.Errorf("ISSUE argument is required in non-interactive mode")
		}

		// List cached plans
		plans, err := state.DefaultCache.ListPlans()
		if err != nil {
			return fmt.Errorf("failed to list plans: %w", err)
		}

		if len(plans) == 0 {
			return fmt.Errorf("no cached plans found. Create a plan with 'jig new' or 'jig plan save'")
		}

		// Show interactive table selection
		selectedPlan, ok, err := ui.RunPlanTable("Select a plan to implement:", plans)
		if err != nil {
			return fmt.Errorf("failed to select plan: %w", err)
		}
		if !ok || selectedPlan == nil {
			return nil // User cancelled
		}

		p = selectedPlan
		issueID = p.ID
	} else {
		issueID = args[0]

		// Try to get plan from cache first
		var err error
		p, err = state.DefaultCache.GetPlan(issueID)
		if err != nil {
			printWarning(fmt.Sprintf("Could not read cached plan: %v", err))
		}

		// If not in cache, try to fetch from tracker
		if p == nil {
			printInfo(fmt.Sprintf("Fetching plan for %s...", issueID))
			t, err := getTracker(cfg)
			if err != nil {
				printWarning(fmt.Sprintf("Could not connect to tracker: %v", err))
			} else {
				issue, err := t.GetIssue(ctx, issueID)
				if err != nil {
					printWarning(fmt.Sprintf("Could not fetch issue: %v", err))
				} else {
					// Create a minimal plan from the issue
					p = createPlanFromIssue(issue)
				}
			}
		}
	}

	// Transition plan to in-progress if it's draft or approved
	if p != nil && (p.Status == plan.StatusDraft || p.Status == plan.StatusApproved) {
		// Create tracker syncer if configured
		var syncer state.TrackerSyncer
		if cfg.Default.Tracker == "linear" {
			syncer = getLinearSyncer(cfg)
		}

		// Use PlanStatusManager for atomic cache + tracker update
		mgr := state.NewPlanStatusManager(state.DefaultCache, syncer)
		result, err := mgr.StartProgress(ctx, p)
		if err != nil {
			printWarning(fmt.Sprintf("Could not transition plan to in-progress: %v", err))
		} else {
			if result.TrackerError != nil {
				printWarning(fmt.Sprintf("Could not sync status to tracker: %v", result.TrackerError))
			}
		}
	}

	// Determine branch name
	branchName := git.GenerateBranchName(issueID, "implementation")
	if p != nil {
		branchName = git.GenerateBranchName(issueID, p.Title)
	}

	// Create or get worktree
	var worktreePath string
	if ui.IsInteractive() {
		var createErr error
		err := ui.RunWithSpinner(fmt.Sprintf("Setting up worktree for %s", issueID), func() error {
			worktreePath, createErr = git.CreateWorktree(issueID, branchName)
			return createErr
		})
		if err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		printInfo(fmt.Sprintf("Setting up worktree for %s...", issueID))
		var createErr error
		worktreePath, createErr = git.CreateWorktree(issueID, branchName)
		if createErr != nil {
			return fmt.Errorf("failed to create worktree: %w", createErr)
		}
		printSuccess(fmt.Sprintf("Worktree ready at %s", worktreePath))
	}

	// Track the worktree
	repoRoot, _ := git.GetWorktreeRoot()
	wtInfo := &state.WorktreeInfo{
		IssueID:  issueID,
		Path:     worktreePath,
		Branch:   branchName,
		RepoPath: repoRoot,
	}
	if p != nil {
		wtInfo.PlanID = p.ID
	}
	if err := state.DefaultWorktreeState.Track(wtInfo); err != nil {
		printWarning(fmt.Sprintf("Could not track worktree: %v", err))
	}

	// Save issue metadata
	meta := &state.IssueMetadata{
		IssueID:      issueID,
		WorktreePath: worktreePath,
		BranchName:   branchName,
	}
	if p != nil {
		meta.PlanID = p.ID
	}
	if err := state.DefaultCache.SaveIssueMetadata(meta); err != nil {
		printWarning(fmt.Sprintf("Could not save metadata: %v", err))
	}

	// Determine which runner to use
	runnerName := implRunner
	if runnerName == "" {
		runnerName = cfg.Default.Runner
	}
	if runnerName == "" {
		runnerName = "claude"
	}

	// Get the runner
	r, err := runner.Get(runnerName)
	if err != nil {
		return fmt.Errorf("runner not found: %s", runnerName)
	}

	if !r.Available() {
		return fmt.Errorf("runner '%s' is not available (not installed or not in PATH)", runnerName)
	}

	// Prepare the runner context (writes plan to .jig/plan.md)
	prepOpts := &runner.PrepareOpts{
		Plan:        p,
		WorktreeDir: worktreePath,
		PromptType:  runner.PromptTypeImplement,
	}
	if err := r.Prepare(ctx, prepOpts); err != nil {
		return fmt.Errorf("failed to prepare runner: %w", err)
	}

	if implNoLaunch {
		printSuccess("Worktree ready for implementation")
		fmt.Printf("\nWorktree: %s\n", worktreePath)
		fmt.Printf("Branch: %s\n", branchName)
		fmt.Printf("\nTo start implementing:\n")
		fmt.Printf("  cd %s && %s /jig:implement\n", worktreePath, runnerName)
		return nil
	}

	printInfo(fmt.Sprintf("Launching %s for implementation...", runnerName))
	fmt.Println()

	// Launch the runner with the /jig:implement skill
	// The skill will read the plan from .jig/plan.md
	_, err = r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir:     worktreePath,
		InitialPrompt:   fmt.Sprintf("/jig:implement %s", issueID),
		Interactive:     true,
		AutoAcceptEdits: !implNoAutoAccept,
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo("Implementation session ended")

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Commit your changes: git add . && git commit\n")
	fmt.Printf("  2. Create a draft PR: gh pr create --draft\n")
	fmt.Printf("  3. Address feedback: jig review %s\n", issueID)

	return nil
}

// createPlanFromIssue creates a minimal plan from a tracker issue
func createPlanFromIssue(issue *tracker.Issue) *plan.Plan {
	if issue == nil {
		return nil
	}

	// Generate a local plan ID, link to the issue via IssueID
	planID := fmt.Sprintf("PLAN-%d", time.Now().Unix())
	p := plan.NewPlan(planID, issue.Title, issue.Assignee)
	p.IssueID = issue.Identifier
	p.ProblemStatement = issue.Description
	return p
}

// getLinearSyncer creates a Linear client that implements TrackerSyncer.
// Returns nil if Linear is not properly configured.
func getLinearSyncer(cfg *config.Config) state.TrackerSyncer {
	store, err := config.NewStore()
	if err != nil {
		return nil
	}

	apiKey, err := store.GetLinearAPIKey()
	if err != nil {
		return nil
	}
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return nil
	}

	return linear.NewClient(apiKey, cfg.Linear.TeamID, cfg.Linear.DefaultProject)
}
