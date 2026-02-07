package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/runner"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/tracker"
	"github.com/charleslr/jig/internal/tracker/linear"
	"github.com/charleslr/jig/internal/ui"
)

var planCmd = &cobra.Command{
	Use:   "plan [ISSUE_ID]",
	Short: "Create and manage implementation plans",
	Long: `Create and manage implementation plans.

When run without a subcommand, creates a new plan (same as 'jig plan new').
If ISSUE_ID is provided, the plan will be seeded with context from the
existing Linear issue.

Subcommands:
  new      Create a new plan
  list     List cached plans
  show     Show a cached plan
  save     Save a plan from file or stdin
  import   Import a plan from a file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlanNew,
}

var planNewCmd = &cobra.Command{
	Use:   "new [ISSUE_ID]",
	Short: "Create a new plan",
	Long: `Create a new implementation plan, optionally from an existing issue.

If ISSUE_ID is provided, the plan will be seeded with context from the
existing Linear issue. Otherwise, a blank planning session is started.

After creating the plan, jig launches your configured coding tool
for an interactive planning session.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlanNew,
}

var (
	planNewTitle    string
	planNewGoal     string
	planNewRunner   string
	planNewNoLaunch bool
)

var planSaveCmd = &cobra.Command{
	Use:   "save [FILE]",
	Short: "Save a plan from file or stdin",
	Long: `Save an implementation plan to jig's cache.

If FILE is provided, reads the plan from that file.
If no FILE is provided, reads from stdin.

This command is typically invoked by Claude Code after creating a plan.

Examples:
  jig plan save plan.md
  cat plan.md | jig plan save
  jig plan save < plan.md`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlanSave,
}

var planImportCmd = &cobra.Command{
	Use:   "import <FILE>",
	Short: "Import a plan from a file",
	Long: `Import an implementation plan from a markdown file.

This is a convenience alias for 'jig plan save <FILE>'.

Example:
  jig plan import ./my-plan.md`,
	Args: cobra.ExactArgs(1),
	RunE: runPlanImport,
}

var planShowCmd = &cobra.Command{
	Use:   "show <PLAN_ID>",
	Short: "Show a cached plan",
	Long: `Display a cached plan in an interactive viewer.

Use --raw to output the raw markdown instead.`,
	Args: cobra.ExactArgs(1),
	RunE: runPlanShow,
}

var planShowRaw bool

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached plans",
	Long:  `List all plans in jig's cache.`,
	RunE:  runPlanList,
}

func init() {
	// Add flags to both planCmd and planNewCmd
	planCmd.Flags().StringVarP(&planNewTitle, "title", "t", "", "plan title (optional, will be generated if not provided)")
	planCmd.Flags().StringVarP(&planNewGoal, "goal", "g", "", "what you want to plan (can also be provided interactively)")
	planCmd.Flags().StringVarP(&planNewRunner, "runner", "r", "", "coding tool to use (default from config)")
	planCmd.Flags().BoolVar(&planNewNoLaunch, "no-launch", false, "don't launch the coding tool")

	planNewCmd.Flags().StringVarP(&planNewTitle, "title", "t", "", "plan title (optional, will be generated if not provided)")
	planNewCmd.Flags().StringVarP(&planNewGoal, "goal", "g", "", "what you want to plan (can also be provided interactively)")
	planNewCmd.Flags().StringVarP(&planNewRunner, "runner", "r", "", "coding tool to use (default from config)")
	planNewCmd.Flags().BoolVar(&planNewNoLaunch, "no-launch", false, "don't launch the coding tool")

	planCmd.AddCommand(planNewCmd)
	planCmd.AddCommand(planSaveCmd)
	planCmd.AddCommand(planImportCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planListCmd)

	planShowCmd.Flags().BoolVar(&planShowRaw, "raw", false, "output raw markdown instead of interactive view")
}

func runPlanSave(cmd *cobra.Command, args []string) error {
	var content []byte
	var err error

	if len(args) > 0 {
		// Read from file
		content, err = os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	} else {
		// Read from stdin
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	}

	if len(content) == 0 {
		return fmt.Errorf("no plan content provided")
	}

	// Validate the plan structure before parsing
	if err := plan.ValidateStructure(content); err != nil {
		return fmt.Errorf("invalid plan format: %w", err)
	}

	// Parse the plan
	p, err := plan.Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Save to cache
	if err := state.DefaultCache.SavePlan(p); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	// Mark plan as saved (for hook to detect)
	markPlanSaved()

	printSuccess(fmt.Sprintf("Plan saved: %s", p.ID))
	fmt.Printf("  Title: %s\n", p.Title)
	fmt.Printf("  Phases: %d\n", len(p.Phases))
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  - View plan: jig plan show %s\n", p.ID)
	fmt.Printf("  - Start implementation: jig implement %s\n", p.ID)

	return nil
}

// markPlanSaved creates a marker file indicating the plan has been saved
func markPlanSaved() {
	jigDir := ".jig"
	os.MkdirAll(jigDir, 0755)
	os.WriteFile(filepath.Join(jigDir, "plan-saved.marker"), []byte{}, 0644)
}

func runPlanImport(cmd *cobra.Command, args []string) error {
	// Just delegate to save with the file argument
	return runPlanSave(cmd, args)
}

func runPlanShow(cmd *cobra.Command, args []string) error {
	planID := args[0]

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// If --raw flag or non-interactive, output raw markdown
	if planShowRaw || !ui.IsInteractive() {
		content, err := state.DefaultCache.GetPlanMarkdown(planID)
		if err != nil {
			return fmt.Errorf("failed to read plan: %w", err)
		}
		if content == "" {
			return fmt.Errorf("plan not found: %s", planID)
		}
		fmt.Println(content)
		return nil
	}

	// Get parsed plan for interactive view
	p, err := state.DefaultCache.GetPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to read plan: %w", err)
	}
	if p == nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	// Show interactive plan view
	return ui.ShowPlan(p)
}

func runPlanList(cmd *cobra.Command, args []string) error {
	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	plans, err := state.DefaultCache.ListPlans()
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	if len(plans) == 0 {
		fmt.Println("No plans cached.")
		return nil
	}

	// If interactive, show table with selection
	if ui.IsInteractive() {
		selectedPlan, ok, err := ui.RunPlanTable("Cached plans:", plans)
		if err != nil {
			return fmt.Errorf("failed to display plans: %w", err)
		}
		if ok && selectedPlan != nil {
			// Show the selected plan's details
			return ui.ShowPlan(selectedPlan)
		}
		return nil
	}

	// Non-interactive: print plain text list
	fmt.Printf("Cached plans (%d):\n\n", len(plans))
	for _, p := range plans {
		status := string(p.Status)
		if status == "" {
			status = "draft"
		}
		phases := fmt.Sprintf("%d phases", len(p.Phases))
		if len(p.Phases) == 1 {
			phases = "1 phase"
		}

		fmt.Printf("  %s\n", p.ID)
		fmt.Printf("    Title: %s\n", p.Title)
		fmt.Printf("    Status: %s | %s\n", status, phases)
		fmt.Println()
	}

	return nil
}

func runPlanNew(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	var issueID string
	var issueContext string

	// Get issue context if provided
	if len(args) > 0 {
		issueID = args[0]
		printInfo(fmt.Sprintf("Fetching issue context for %s...", issueID))

		t, err := getTracker(cfg)
		if err != nil {
			printWarning(fmt.Sprintf("Could not connect to tracker: %v", err))
		} else {
			issue, err := t.GetIssue(ctx, issueID)
			if err != nil {
				printWarning(fmt.Sprintf("Could not fetch issue: %v", err))
			} else {
				issueContext = formatIssueContext(issue)
				if planNewTitle == "" {
					planNewTitle = issue.Title
				}
			}
		}
	}

	// Get planning goal - prompt interactively if not provided
	planGoal := planNewGoal
	if planGoal == "" && issueContext == "" {
		if ui.IsInteractive() {
			goal, err := ui.RunTextArea("What would you like to plan?")
			if err != nil {
				return fmt.Errorf("failed to get planning goal: %w", err)
			}
			if goal == "" {
				return fmt.Errorf("planning goal is required")
			}
			planGoal = goal
		} else {
			return fmt.Errorf("planning goal is required (use --goal or provide an ISSUE_ID)")
		}
	}

	// Use title if provided, otherwise use a placeholder (LLM will generate proper title)
	if planNewTitle == "" {
		if issueID != "" {
			// Title was already set from issue above
		} else {
			planNewTitle = "New Plan"
		}
	}

	// Determine author (from git config)
	author := getGitAuthor()

	// Generate a plan ID if none provided
	planID := issueID
	if planID == "" {
		planID = fmt.Sprintf("PLAN-%d", time.Now().Unix())
	}

	// Create initial plan
	p := plan.NewPlan(planID, planNewTitle, author)
	p.ProblemStatement = "TODO: Define the problem being solved"
	p.ProposedSolution = "TODO: Describe the proposed solution"

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Save initial plan to cache
	if err := state.DefaultCache.SavePlan(p); err != nil {
		printWarning(fmt.Sprintf("Could not cache plan: %v", err))
	}

	// Determine which runner to use (currently only Claude is supported)
	runnerName := planNewRunner
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

	// Get current working directory for the planning session
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if planNewNoLaunch {
		printSuccess("Plan initialized")
		fmt.Printf("\nPlan ID: %s\n", p.ID)
		fmt.Printf("Plan cached at: ~/.jig/cache/plans/%s.md\n", p.ID)
		fmt.Printf("\nTo launch planning session manually:\n")
		fmt.Printf("  cd %s && %s\n", cwd, runnerName)
		return nil
	}

	// Prepare the runner context (writes planning context files to .jig/)
	prepOpts := &runner.PrepareOpts{
		Plan:         p,
		WorktreeDir:  cwd,
		PromptType:   runner.PromptTypePlan,
		PlanGoal:     planGoal,
		IssueContext: issueContext,
	}
	if err := r.Prepare(ctx, prepOpts); err != nil {
		return fmt.Errorf("failed to prepare runner: %w", err)
	}

	printInfo(fmt.Sprintf("Launching %s for planning...", runnerName))
	fmt.Println()

	// Launch the runner in plan mode with the /jig:plan skill
	// The skill will read context from .jig/planning-context.md and .jig/issue-context.md
	_, err = r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir:   cwd,
		InitialPrompt: "/jig:plan",
		Interactive:   true,
		PlanMode:      true,
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo("Planning session ended")

	// Check if a plan was saved during the session
	if err := state.Init(); err == nil {
		plans, _ := state.DefaultCache.ListPlans()
		if len(plans) > 0 {
			// Find the most recent plan
			latest := plans[len(plans)-1]
			printSuccess(fmt.Sprintf("Plan saved: %s", latest.ID))
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  - View plan: jig plan show %s\n", latest.ID)
			fmt.Printf("  - Start implementation: jig implement %s\n", latest.ID)
			return nil
		}
	}

	// No plan saved - provide manual instructions
	fmt.Printf("\nIf you created a plan, save it with:\n")
	fmt.Printf("  jig plan save <plan-file.md>\n")

	return nil
}

// getTracker returns the configured tracker client
func getTracker(cfg *config.Config) (tracker.Tracker, error) {
	switch cfg.Default.Tracker {
	case "linear":
		store, err := config.NewStore()
		if err != nil {
			return nil, err
		}
		apiKey, err := store.GetLinearAPIKey()
		if err != nil {
			return nil, err
		}
		if apiKey == "" {
			apiKey = cfg.Linear.APIKey
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Linear API key not configured")
		}
		return linear.NewClient(apiKey, cfg.Linear.TeamID, cfg.Linear.DefaultProject), nil
	default:
		return nil, fmt.Errorf("unknown tracker: %s", cfg.Default.Tracker)
	}
}

// formatIssueContext formats an issue for inclusion in a prompt
func formatIssueContext(issue *tracker.Issue) string {
	ctx := fmt.Sprintf("## Existing Issue\n\n")
	ctx += fmt.Sprintf("**ID:** %s\n", issue.Identifier)
	ctx += fmt.Sprintf("**Title:** %s\n", issue.Title)
	ctx += fmt.Sprintf("**Status:** %s\n", issue.Status)

	if issue.Description != "" {
		ctx += fmt.Sprintf("\n**Description:**\n%s\n", issue.Description)
	}

	if len(issue.Labels) > 0 {
		ctx += fmt.Sprintf("\n**Labels:** %v\n", issue.Labels)
	}

	return ctx
}

// getGitAuthor returns the git author name
func getGitAuthor() string {
	// Try to get from git config
	// For now, return a placeholder
	if name := os.Getenv("USER"); name != "" {
		return name
	}
	return "unknown"
}
