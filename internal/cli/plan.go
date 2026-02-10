package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
  jig plan save < plan.md
  jig plan save --session 12345 plan.md`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlanSave,
}

var planSaveSessionID string
var planSaveNoSync bool

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

	planSaveCmd.Flags().StringVar(&planSaveSessionID, "session", "", "session ID for tracking the saved plan (used by jig plan)")
	planSaveCmd.Flags().BoolVar(&planSaveNoSync, "no-sync", false, "don't sync plan to Linear")
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

	cfg := config.Get()
	ctx := context.Background()

	// Create Linear issue if plan has no linked issue and creation is enabled
	if !planSaveNoSync && shouldCreateIssueForPlan(cfg, p) {
		issueID, err := createIssueForPlan(ctx, cfg, p)
		if err != nil {
			printWarning(fmt.Sprintf("Could not create Linear issue: %v", err))
		} else {
			p.IssueID = issueID
			printSuccess(fmt.Sprintf("Created Linear issue %s", issueID))
		}
	}

	// Save to cache (with potentially updated IssueID)
	if err := state.DefaultCache.SavePlan(p); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	// Sync to Linear if applicable (adds comment to existing linked issue)
	if !planSaveNoSync && shouldSyncToLinear(cfg, p) {
		if err := syncPlanToLinear(ctx, cfg, p); err != nil {
			printWarning(fmt.Sprintf("Could not sync to Linear: %v", err))
		} else {
			printSuccess(fmt.Sprintf("Plan synced to Linear issue %s", p.IssueID))
		}
	}

	// Mark plan as saved (for hook to detect)
	markPlanSaved()

	// Write the plan ID to the session directory for tracking
	// This allows runPlanNew to know which plan was saved during this session
	if planSaveSessionID != "" {
		writeSavedPlanID(planSaveSessionID, p.ID)
	}

	printSuccess(fmt.Sprintf("Plan saved: %s", p.ID))
	fmt.Printf("  Title: %s\n", p.Title)
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

// writeSavedPlanID writes the plan ID to the session directory for tracking
// This allows the parent planning session to know which plan was saved
func writeSavedPlanID(sessionID, planID string) {
	sessionDir := filepath.Join(".jig", "sessions", sessionID)
	os.MkdirAll(sessionDir, 0755)
	os.WriteFile(filepath.Join(sessionDir, "saved-plan-id"), []byte(planID), 0644)
}

// readSavedPlanID reads the plan ID that was saved during a planning session
// Returns empty string if no plan was saved or the file doesn't exist
func readSavedPlanID(sessionID string) string {
	sessionDir := filepath.Join(".jig", "sessions", sessionID)
	data, err := os.ReadFile(filepath.Join(sessionDir, "saved-plan-id"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// displaySavedPlanNextSteps checks if a plan was saved during the session and displays next steps
// Returns true if a plan was found and displayed, false otherwise
func displaySavedPlanNextSteps(sessionID string) bool {
	savedPlanID := readSavedPlanID(sessionID)
	if savedPlanID == "" {
		return false
	}
	printSuccess(fmt.Sprintf("Plan saved: %s", savedPlanID))
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  - View plan: jig plan show %s\n", savedPlanID)
	fmt.Printf("  - Start implementation: jig implement %s\n", savedPlanID)
	return true
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

		fmt.Printf("  %s\n", p.ID)
		fmt.Printf("    Title: %s\n", p.Title)
		fmt.Printf("    Status: %s\n", status)
		fmt.Println()
	}

	return nil
}

func runPlanNew(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	var issueID string
	var issueContext string
	var additionalInstructions string
	var issue *tracker.Issue

	// Get issue context if provided
	if len(args) > 0 {
		issueID = args[0]

		// Fetch issue with spinner for better UX during API latency
		fetchResult := fetchIssueWithDeps(ctx, issueID, defaultIssueFetchDeps(cfg))
		issue = fetchResult.Issue

		if fetchResult.Err != nil {
			printWarning(fmt.Sprintf("Could not fetch issue: %v", fetchResult.Err))
		} else {
			// Show interactive menu if we have an issue and are in interactive mode
			if ui.IsInteractive() {
				result, err := ui.RunPlanPrompt(issue)
				if err != nil {
					return fmt.Errorf("failed to run plan prompt: %w", err)
				}

				if result.Action == ui.PlanPromptActionCancel {
					return nil // User cancelled
				}

				// Collect any additional instructions
				additionalInstructions = result.Instructions

				// Show confirmation of what was captured
				printSuccess(fmt.Sprintf("Issue loaded: %s", issue.Identifier))
				if additionalInstructions != "" {
					printInfo("Custom instructions added")
				}
			}

			result := processIssueForPlan(issue, planNewTitle)
			issueContext = result.IssueContext
			planNewTitle = result.Title
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

	// Include additional instructions in the goal if provided
	if additionalInstructions != "" {
		if planGoal != "" {
			planGoal = planGoal + "\n\n## Additional Instructions\n\n" + additionalInstructions
		} else {
			planGoal = additionalInstructions
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

	// Always generate a local plan ID
	planID := fmt.Sprintf("PLAN-%d", time.Now().Unix())

	// Create initial plan metadata (the actual plan content is created by Claude)
	p := plan.NewPlan(planID, planNewTitle, author)

	// Link to issue if provided
	if issueID != "" {
		p.IssueID = issueID
	}
	p.ProblemStatement = "TODO: Define the problem being solved"
	p.ProposedSolution = "TODO: Describe the proposed solution"

	// Initialize cache (plan will be saved when Claude runs `jig plan save`)
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
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

	// Generate a unique session ID for parallel planning support
	sessionID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Prepare the runner context (writes planning context files to .jig/sessions/<session-id>/)
	prepOpts := &runner.PrepareOpts{
		Plan:         p,
		WorktreeDir:  cwd,
		PromptType:   runner.PromptTypePlan,
		PlanGoal:     planGoal,
		IssueContext: issueContext,
		SessionID:    sessionID,
	}
	if err := r.Prepare(ctx, prepOpts); err != nil {
		return fmt.Errorf("failed to prepare runner: %w", err)
	}

	printInfo(fmt.Sprintf("Launching %s for planning...", runnerName))
	fmt.Println()

	// Launch the runner in plan mode with the /jig:plan skill
	// Pass the session ID so the skill reads from the correct session directory
	// This avoids race conditions when multiple planning sessions run in parallel
	_, err = r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir:   cwd,
		InitialPrompt: fmt.Sprintf("/jig:plan %s", sessionID),
		Interactive:   true,
		PlanMode:      true,
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo("Planning session ended")

	// Check if a plan was saved during the session by reading from the session directory
	// This is more reliable than guessing based on cache contents
	if displaySavedPlanNextSteps(sessionID) {
		return nil
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

// issueResult holds the result of processing a fetched issue
type issueResult struct {
	IssueContext string
	Title        string // Updated title (from issue if original was empty)
}

// issueFetchDeps holds injectable dependencies for fetching issues
type issueFetchDeps struct {
	isInteractive func() bool
	runSpinner    func(message string, fn func() error) error
	getTracker    func() (tracker.Tracker, error)
}

// fetchIssueResult holds the result of fetching an issue
type fetchIssueResult struct {
	Issue *tracker.Issue
	Err   error
}

// fetchIssueWithDeps fetches an issue using the provided dependencies.
// In interactive mode, it uses a spinner for better UX during API latency.
// In non-interactive mode, it fetches directly.
func fetchIssueWithDeps(ctx context.Context, issueID string, deps issueFetchDeps) fetchIssueResult {
	var issue *tracker.Issue
	var fetchErr error

	if deps.isInteractive() {
		fetchErr = deps.runSpinner(fmt.Sprintf("Fetching issue %s", issueID), func() error {
			t, err := deps.getTracker()
			if err != nil {
				return err
			}
			issue, err = t.GetIssue(ctx, issueID)
			return err
		})
	} else {
		t, err := deps.getTracker()
		if err != nil {
			fetchErr = err
		} else {
			issue, fetchErr = t.GetIssue(ctx, issueID)
		}
	}

	return fetchIssueResult{Issue: issue, Err: fetchErr}
}

// defaultIssueFetchDeps returns the production dependencies for fetching issues
func defaultIssueFetchDeps(cfg *config.Config) issueFetchDeps {
	return issueFetchDeps{
		isInteractive: ui.IsInteractive,
		runSpinner:    ui.RunWithSpinner,
		getTracker:    func() (tracker.Tracker, error) { return getTracker(cfg) },
	}
}

// processIssueForPlan processes a successfully fetched issue and returns
// the issue context and updated title for use in plan creation.
// If currentTitle is non-empty, it is preserved; otherwise the issue's title is used.
func processIssueForPlan(issue *tracker.Issue, currentTitle string) issueResult {
	result := issueResult{
		IssueContext: formatIssueContext(issue),
		Title:        currentTitle,
	}

	if currentTitle == "" {
		result.Title = issue.Title
	}

	return result
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

// shouldSyncToLinear returns true if the plan should be synced to Linear
func shouldSyncToLinear(cfg *config.Config, p *plan.Plan) bool {
	// Check if Linear is the configured tracker
	if cfg.Default.Tracker != "linear" {
		return false
	}

	// Check if sync is enabled in config
	if !cfg.Linear.ShouldSyncPlanOnSave() {
		return false
	}

	// Only sync plans that are linked to an issue
	if !p.HasLinkedIssue() {
		return false
	}

	// Check if Linear API key is configured
	store, err := config.NewStore()
	if err != nil {
		return false
	}
	apiKey, _ := store.GetLinearAPIKey()
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return false
	}

	return true
}

// syncPlanToLinear syncs a plan to its associated Linear issue
func syncPlanToLinear(ctx context.Context, cfg *config.Config, p *plan.Plan) error {
	syncer, err := getLinearPlanSyncer(cfg)
	if err != nil {
		return err
	}
	labelName := cfg.Linear.GetPlanLabelName()
	return syncPlanWithSyncer(ctx, syncer, p, labelName)
}

// syncPlanWithSyncer syncs a plan using the provided PlanSyncer (for testability)
func syncPlanWithSyncer(ctx context.Context, syncer tracker.PlanSyncer, p *plan.Plan, labelName string) error {
	return syncer.SyncPlanToIssue(ctx, p, labelName)
}

// getLinearPlanSyncer creates a Linear client configured as a PlanSyncer
func getLinearPlanSyncer(cfg *config.Config) (tracker.PlanSyncer, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to get config store: %w", err)
	}
	apiKey, err := store.GetLinearAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get Linear API key: %w", err)
	}
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Linear API key not configured")
	}

	return linear.NewClient(apiKey, cfg.Linear.TeamID, cfg.Linear.DefaultProject), nil
}

// shouldCreateIssueForPlan returns true if a new Linear issue should be created for this plan
func shouldCreateIssueForPlan(cfg *config.Config, p *plan.Plan) bool {
	// Don't create if plan already has a linked issue
	if p.HasLinkedIssue() {
		return false
	}

	// Check if Linear is the configured tracker
	if cfg.Default.Tracker != "linear" {
		return false
	}

	// Check if issue creation is enabled in config
	if !cfg.Linear.ShouldCreateIssueOnSave() {
		return false
	}

	// Check if Linear API key is configured
	store, err := config.NewStore()
	if err != nil {
		return false
	}
	apiKey, _ := store.GetLinearAPIKey()
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return false
	}

	return true
}

// createIssueForPlan creates a new Linear issue from the plan and returns the issue identifier
func createIssueForPlan(ctx context.Context, cfg *config.Config, p *plan.Plan) (string, error) {
	client, err := getLinearClient(cfg)
	if err != nil {
		return "", err
	}

	issue, err := client.CreateIssueFromPlan(ctx, p)
	if err != nil {
		return "", err
	}
	return issue.Identifier, nil
}

// getLinearClient creates a Linear client
func getLinearClient(cfg *config.Config) (*linear.Client, error) {
	store, err := config.NewStore()
	if err != nil {
		return nil, fmt.Errorf("failed to get config store: %w", err)
	}
	apiKey, err := store.GetLinearAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get Linear API key: %w", err)
	}
	if apiKey == "" {
		apiKey = cfg.Linear.APIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Linear API key not configured")
	}

	return linear.NewClient(apiKey, cfg.Linear.TeamID, cfg.Linear.DefaultProject), nil
}
