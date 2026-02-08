package cli

import (
	"context"
	"fmt"

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
	Short: "Implement a plan or phase",
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
	implPhase        string
	implRunner       string
	implNoLaunch     bool
	implNoAutoAccept bool
)

func init() {
	implementCmd.Flags().StringVarP(&implPhase, "phase", "p", "", "specific phase to implement")
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
		if err := p.TransitionTo(plan.StatusInProgress); err != nil {
			printWarning(fmt.Sprintf("Could not transition plan to in-progress: %v", err))
		} else {
			if err := state.DefaultCache.SavePlan(p); err != nil {
				printWarning(fmt.Sprintf("Could not save plan status: %v", err))
			}
			// Sync status to tracker (non-blocking)
			syncPlanStatusToTracker(ctx, cfg, p)
		}
	}

	// Determine branch name
	branchName := git.GenerateBranchName(issueID, "implementation")
	if p != nil {
		branchName = git.GenerateBranchName(issueID, p.Title)
	}

	// Check for specific phase
	var selectedPhase *plan.Phase

	// If no phase specified but plan has multiple phases, prompt for selection
	if implPhase == "" && p != nil && len(p.Phases) > 1 && ui.IsInteractive() {
		// Build options from available (non-blocked) phases
		var options []ui.SelectOption
		for _, ph := range p.Phases {
			if !ph.IsBlocked(p.Phases) {
				label := ph.Title
				if ph.Status == "complete" {
					label += " (completed)"
				}
				options = append(options, ui.SelectOption{
					Label:       label,
					Value:       ph.ID,
					Description: fmt.Sprintf("Phase %s", ph.ID),
				})
			}
		}
		if len(options) > 0 {
			selected, err := ui.RunSelect("Select phase to implement:", options)
			if err != nil {
				return fmt.Errorf("failed to select phase: %w", err)
			}
			if selected != "" {
				implPhase = selected
			}
		}
	}

	if implPhase != "" && p != nil {
		selectedPhase = p.GetPhase(implPhase)
		if selectedPhase == nil {
			return fmt.Errorf("phase not found: %s", implPhase)
		}
		if selectedPhase.IsBlocked(p.Phases) {
			return fmt.Errorf("phase %s is blocked by dependencies", implPhase)
		}
		// Update branch name for phase
		if selectedPhase.IssueID != "" {
			branchName = git.GenerateBranchName(selectedPhase.IssueID, selectedPhase.Title)
		}
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
	if selectedPhase != nil {
		wtInfo.PhaseID = selectedPhase.ID
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
		Phase:       selectedPhase,
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

	p := plan.NewPlan(issue.Identifier, issue.Title, issue.Assignee)
	p.ProblemStatement = issue.Description
	return p
}

// syncPlanStatusToTracker syncs the plan status to the configured tracker
// This is non-blocking - failures are logged as warnings but don't abort the operation
func syncPlanStatusToTracker(ctx context.Context, cfg *config.Config, p *plan.Plan) {
	if cfg.Default.Tracker != "linear" {
		return // Only Linear is supported for now
	}

	store, err := config.NewStore()
	if err != nil {
		printWarning(fmt.Sprintf("Could not sync status to tracker: %v", err))
		return
	}

	apiKey, err := store.GetLinearAPIKey()
	if err != nil {
		printWarning(fmt.Sprintf("Could not sync status to tracker: %v", err))
		return
	}
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return // No API key configured, skip sync
	}

	client := linear.NewClient(apiKey, cfg.Linear.TeamID, cfg.Linear.DefaultProject)
	if err := client.SyncPlanStatus(ctx, p); err != nil {
		printWarning(fmt.Sprintf("Could not sync status to tracker: %v", err))
	}
}
