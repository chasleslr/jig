package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/prompt"
	"github.com/charleslr/jig/internal/runner"
	"github.com/charleslr/jig/internal/state"
)

var amendCmd = &cobra.Command{
	Use:   "amend ISSUE",
	Short: "Amend an approved plan",
	Long: `Amend an existing plan and trigger a new review cycle.

Any change to an approved plan requires a new review cycle.
This command:
1. Opens the plan for editing
2. Launches your coding tool for interactive editing
3. Runs a full review cycle on the amended plan
4. Updates the tracker issue on approval`,
	Args: cobra.ExactArgs(1),
	RunE: runAmend,
}

var (
	amendRunner   string
	amendNoReview bool
)

func init() {
	amendCmd.Flags().StringVarP(&amendRunner, "runner", "r", "", "coding tool to use (default from config)")
	amendCmd.Flags().BoolVar(&amendNoReview, "no-review", false, "skip the review cycle (use with caution)")
}

func runAmend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	issueID := args[0]

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Get the plan from cache
	p, err := state.DefaultCache.GetPlan(issueID)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if p == nil {
		return fmt.Errorf("plan not found: %s", issueID)
	}

	// Check if plan can be amended
	if p.Status == "draft" {
		printWarning("Plan is still a draft - use 'jig new' to continue editing")
	}

	printInfo(fmt.Sprintf("Amending plan: %s", p.Title))
	fmt.Printf("Current status: %s\n", p.Status)
	fmt.Println()

	// Transition to reviewing status
	originalStatus := p.Status
	p.Status = "reviewing"
	if err := state.DefaultCache.SavePlan(p); err != nil {
		return fmt.Errorf("failed to update plan status: %w", err)
	}

	// Get the plan markdown for editing
	planMD, err := state.DefaultCache.GetPlanMarkdown(issueID)
	if err != nil || planMD == "" {
		return fmt.Errorf("failed to get plan content: %w", err)
	}

	// Write to a temp file for editing
	cacheDir, _ := config.CacheDir()
	editPath := fmt.Sprintf("%s/plans/%s-edit.md", cacheDir, issueID)
	if err := os.WriteFile(editPath, []byte(planMD), 0644); err != nil {
		return fmt.Errorf("failed to write edit file: %w", err)
	}

	// Determine which runner to use
	runnerName := amendRunner
	if runnerName == "" {
		runnerName = cfg.Default.Runner
	}
	if runnerName == "" {
		runnerName = "claude"
	}

	// Get the runner from the injected registry
	r, err := deps.RunnerRegistry.Get(runnerName)
	if err != nil {
		return fmt.Errorf("runner not found: %s", runnerName)
	}

	if !r.Available() {
		return fmt.Errorf("runner '%s' is not available", runnerName)
	}

	// Load and render the planning prompt with amendment context
	promptMgr, err := prompt.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	amendmentContext := fmt.Sprintf(`## Amendment Session

You are amending an existing plan. The current plan is at:
%s

Please make your changes to the plan. After editing:
1. Save the plan file
2. Exit this session
3. The plan will be sent for review

### Important
- Any changes to an approved plan require review
- Keep changes focused and well-justified
- Add a note explaining why the amendment is needed

### Previous Status
This plan was previously: %s

`, editPath, originalStatus)

	promptContent, err := promptMgr.LoadAndRender(prompt.TypePlan, &prompt.Vars{
		Plan:         p,
		IssueContext: amendmentContext,
	})
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Prepare the runner context
	if err := r.Prepare(ctx, &runner.PrepareOpts{
		Plan:        p,
		WorktreeDir: cwd,
		PromptType:  runner.PromptTypePlan,
	}); err != nil {
		return fmt.Errorf("failed to prepare runner: %w", err)
	}

	printInfo(fmt.Sprintf("Launching %s for plan amendment...", runnerName))
	fmt.Println()
	fmt.Printf("Edit the plan at: %s\n", editPath)
	fmt.Println()
	fmt.Println("After your editing session:")
	if amendNoReview {
		fmt.Println("  - Changes will be saved without review")
	} else {
		fmt.Println("  - The plan will be sent for review")
		fmt.Println("  - Run 'jig review-plan' to complete the review cycle")
	}
	fmt.Println()

	// Launch the runner as a subprocess (blocks until it exits)
	result, err := r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir: cwd,
		Prompt:      promptContent,
		Interactive: true,
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo(fmt.Sprintf("Amend session ended (duration: %s, exit code: %d)", result.Duration.Round(time.Second), result.ExitCode))

	if amendNoReview {
		printSuccess("Plan amended (no review required)")
	} else {
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  - Run 'jig review-plan %s' to start the review cycle\n", p.ID)
	}

	return nil
}
