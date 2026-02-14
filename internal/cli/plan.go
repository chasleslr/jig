package cli

import (
	"context"
	"encoding/json"
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

var planSyncCmd = &cobra.Command{
	Use:   "sync [PLAN_ID]",
	Short: "Sync plans to Linear",
	Long: `Sync implementation plans to their associated Linear issues.

Without arguments, shows an interactive multi-select of all unsynced plans.
With PLAN_ID, syncs that specific plan immediately.

Plans are considered "unsynced" if they have a linked issue and either:
- Have never been synced
- Have been updated after the last sync

Examples:
  jig plan sync                    # Interactive multi-select
  jig plan sync NUM-123            # Sync specific plan`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPlanSync,
}

var planLinkCmd = &cobra.Command{
	Use:   "link <PLAN_ID> <ISSUE_ID>",
	Short: "Link a plan to an existing issue",
	Long: `Link an existing plan to an existing Linear issue.

This is useful for:
- Fixing orphaned plans that failed to link during creation
- Manually associating a plan with a different issue
- Recovering from failed issue creation

After linking, the plan content will be synced to the issue.

Examples:
  jig plan link PLAN-1234567890 NUM-123
  jig plan link my-plan-id NUM-456`,
	Args: cobra.ExactArgs(2),
	RunE: runPlanLink,
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
	planCmd.AddCommand(planSyncCmd)
	planCmd.AddCommand(planLinkCmd)

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

	// Auto-generate plan ID if not provided
	if p.ID == "" {
		p.ID = fmt.Sprintf("PLAN-%d", time.Now().Unix())
	}

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	cfg := config.Get()
	ctx := context.Background()

	// Read session metadata to check for linked issue
	sessionIssueID := ""
	if planSaveSessionID != "" {
		sessionIssueID = readSessionIssueID(planSaveSessionID)
	}

	// Link to issue from session metadata if provided
	if sessionIssueID != "" {
		p.IssueID = sessionIssueID
	}

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
		// Get cached hash for deduplication (empty string if not previously synced)
		cachedHash := ""
		if cached, err := state.DefaultCache.GetCachedPlan(p.ID); err == nil && cached != nil {
			cachedHash = cached.SyncedContentHash
		}

		result, err := syncPlanToLinearWithDedup(ctx, cfg, p, cachedHash)
		if err != nil {
			printWarning(fmt.Sprintf("Could not sync to Linear: %v", err))
		} else if result.Skipped {
			printInfo("Plan content unchanged, skipping sync")
		} else {
			// Mark plan as synced with the content hash
			if err := state.DefaultCache.MarkPlanSyncedWithHash(p.ID, result.ContentHash); err != nil {
				printWarning(fmt.Sprintf("Plan synced but failed to update sync timestamp: %v", err))
			}
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

// readSessionIssueID reads the issue ID from session metadata
// Returns empty string if no issue ID is set in the session
func readSessionIssueID(sessionID string) string {
	sessionDir := filepath.Join(".jig", "sessions", sessionID)
	metadataPath := filepath.Join(sessionDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return ""
	}

	var metadata struct {
		IssueID string `json:"issue_id"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ""
	}
	return metadata.IssueID
}

// writeSessionMetadata writes session metadata to track the linked issue
func writeSessionMetadata(sessionID, issueID string) error {
	sessionDir := filepath.Join(".jig", "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	metadata := map[string]interface{}{
		"issue_id":   issueID,
		"created_at": time.Now().Format(time.RFC3339),
		"command":    fmt.Sprintf("jig plan %s", issueID),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataPath := filepath.Join(sessionDir, "metadata.json")
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
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
	ctx := context.Background()
	cfg := config.Get()
	id := args[0]

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Look up plan by ID with remote fallback
	p, planID, err := lookupPlanByIDWithFallback(id, &LookupPlanOptions{
		FetchFromRemote: true,
		Config:          cfg,
		Context:         ctx,
	})
	if err != nil {
		return fmt.Errorf("failed to read plan: %w", err)
	}
	if p == nil {
		return fmt.Errorf("plan not found: %s", id)
	}

	// If --raw flag or non-interactive, output raw markdown
	if planShowRaw || !ui.IsInteractive() {
		content, err := state.DefaultCache.GetPlanMarkdown(planID)
		if err != nil {
			return fmt.Errorf("failed to read plan: %w", err)
		}
		if content == "" {
			return fmt.Errorf("plan not found: %s", id)
		}
		fmt.Println(content)
		return nil
	}

	// Show interactive plan view
	return ui.ShowPlan(p)
}

func runPlanList(cmd *cobra.Command, args []string) error {
	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	cachedPlans, err := state.DefaultCache.ListCachedPlans()
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	if len(cachedPlans) == 0 {
		fmt.Println("No plans cached.")
		return nil
	}

	// If interactive, show table with selection
	if ui.IsInteractive() {
		selectedPlan, ok, err := ui.RunCachedPlanTable("Cached plans:", cachedPlans)
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
	fmt.Printf("Cached plans (%d):\n\n", len(cachedPlans))
	for _, cp := range cachedPlans {
		if cp.Plan == nil {
			continue
		}
		status := string(cp.Plan.Status)
		if status == "" {
			status = "draft"
		}

		syncStatus := "-"
		if cp.Plan.IssueID != "" {
			if cp.NeedsSync() {
				syncStatus = "no"
			} else {
				syncStatus = "yes"
			}
		}

		fmt.Printf("  %s\n", cp.Plan.ID)
		fmt.Printf("    Title: %s\n", cp.Plan.Title)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Synced: %s\n", syncStatus)
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

	// Get the runner from the injected registry
	r, err := deps.RunnerRegistry.Get(runnerName)
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

	// Write session metadata if planning for an existing issue
	if issueID != "" {
		if err := writeSessionMetadata(sessionID, issueID); err != nil {
			printWarning(fmt.Sprintf("Could not write session metadata: %v", err))
		}
	}

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

// syncResult represents the result of a plan sync operation
type syncResult struct {
	Synced      bool   // true if sync was performed
	ContentHash string // the content hash (set regardless of whether sync was performed)
	Skipped     bool   // true if sync was skipped due to unchanged content
}

// syncPlanToLinearWithDedup syncs a plan to Linear with content-based deduplication.
// Returns syncResult indicating whether sync was performed or skipped.
func syncPlanToLinearWithDedup(ctx context.Context, cfg *config.Config, p *plan.Plan, cachedHash string) (syncResult, error) {
	// Compute the content hash
	contentHash := linear.ComputePlanContentHash(p)

	// Check if content is unchanged
	if cachedHash != "" && cachedHash == contentHash {
		return syncResult{Synced: false, ContentHash: contentHash, Skipped: true}, nil
	}

	// Content changed or never synced, proceed with sync
	if err := syncPlanToLinear(ctx, cfg, p); err != nil {
		return syncResult{ContentHash: contentHash}, err
	}

	return syncResult{Synced: true, ContentHash: contentHash, Skipped: false}, nil
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

// issueCreator is an interface for creating issues from plans (for testability)
type issueCreator interface {
	CreateIssueFromPlan(ctx context.Context, p *plan.Plan) (*tracker.Issue, error)
}

// createIssueForPlan creates a new Linear issue from the plan and returns the issue identifier
func createIssueForPlan(ctx context.Context, cfg *config.Config, p *plan.Plan) (string, error) {
	client, err := getLinearClient(cfg)
	if err != nil {
		return "", err
	}

	return createIssueForPlanWithCreator(ctx, client, p)
}

// createIssueForPlanWithCreator creates an issue using the provided creator (for testability)
func createIssueForPlanWithCreator(ctx context.Context, creator issueCreator, p *plan.Plan) (string, error) {
	issue, err := creator.CreateIssueFromPlan(ctx, p)
	if err != nil {
		return "", err
	}
	return issue.Identifier, nil
}

// getLinearClient creates a Linear client
func getLinearClient(cfg *config.Config) (*linear.Client, error) {
	return getLinearClientWithStore(cfg, config.NewStore)
}

// getLinearClientWithStore creates a Linear client using the provided store factory (for testability)
func getLinearClientWithStore(cfg *config.Config, newStore func() (*config.Store, error)) (*linear.Client, error) {
	store, err := newStore()
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

func runPlanSync(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	// Validate that Linear is configured as the tracker
	if cfg.Default.Tracker != "linear" {
		return fmt.Errorf("Linear is not configured as the tracker (current: %s)", cfg.Default.Tracker)
	}

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// If a plan ID is provided, sync that specific plan
	if len(args) > 0 {
		return syncSinglePlan(ctx, cfg, args[0])
	}

	// Interactive mode: show multi-select of unsynced plans
	if !ui.IsInteractive() {
		return fmt.Errorf("interactive mode required (no plan ID provided)")
	}

	return syncPlansInteractive(ctx, cfg)
}

// syncSinglePlan syncs a specific plan by ID
func syncSinglePlan(ctx context.Context, cfg *config.Config, planID string) error {
	syncer, err := getLinearPlanSyncer(cfg)
	if err != nil {
		return fmt.Errorf("failed to get syncer: %w", err)
	}
	labelName := cfg.Linear.GetPlanLabelName()

	deps := planSyncDeps{
		getCachedPlan:        state.DefaultCache.GetCachedPlan,
		markPlanSyncedWithHash: state.DefaultCache.MarkPlanSyncedWithHash,
		syncPlan: func(ctx context.Context, p *plan.Plan) error {
			return syncPlanWithSyncer(ctx, syncer, p, labelName)
		},
		computeContentHash: linear.ComputePlanContentHash,
	}

	return syncSinglePlanWithDeps(ctx, planID, deps)
}

// planSyncDeps holds dependencies for sync operations (for testability)
type planSyncDeps struct {
	getCachedPlan        func(id string) (*state.CachedPlan, error)
	markPlanSyncedWithHash func(id, hash string) error
	syncPlan             func(ctx context.Context, p *plan.Plan) error
	computeContentHash   func(p *plan.Plan) string
}

// syncSinglePlanWithDeps syncs a specific plan using injected dependencies
func syncSinglePlanWithDeps(ctx context.Context, planID string, deps planSyncDeps) error {
	cached, err := deps.getCachedPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if cached == nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	if cached.Plan == nil || cached.Plan.IssueID == "" {
		return fmt.Errorf("plan has no linked issue")
	}

	// Compute content hash for deduplication
	contentHash := deps.computeContentHash(cached.Plan)

	// Check if content is unchanged
	if cached.SyncedContentHash != "" && cached.SyncedContentHash == contentHash {
		printInfo("Plan content unchanged, skipping sync")
		return nil
	}

	// Sync to Linear
	if err := deps.syncPlan(ctx, cached.Plan); err != nil {
		return fmt.Errorf("failed to sync plan: %w", err)
	}

	// Mark as synced with content hash
	if err := deps.markPlanSyncedWithHash(planID, contentHash); err != nil {
		printWarning(fmt.Sprintf("Plan synced but failed to update sync timestamp: %v", err))
	}

	printSuccess(fmt.Sprintf("Plan synced to Linear issue %s", cached.Plan.IssueID))
	return nil
}

// syncPlansInteractive shows an interactive multi-select of unsynced plans
func syncPlansInteractive(ctx context.Context, cfg *config.Config) error {
	// Get all cached plans
	cachedPlans, err := state.DefaultCache.ListCachedPlans()
	if err != nil {
		return fmt.Errorf("failed to list plans: %w", err)
	}

	// Filter to plans that need sync
	var unsyncedPlans []*state.CachedPlan
	for _, cp := range cachedPlans {
		if cp.NeedsSync() {
			unsyncedPlans = append(unsyncedPlans, cp)
		}
	}

	if len(unsyncedPlans) == 0 {
		fmt.Println("No unsynced plans found.")
		return nil
	}

	// Build options for multi-select
	options := make([]ui.SelectOption, len(unsyncedPlans))
	for i, cp := range unsyncedPlans {
		syncStatus := "never synced"
		if cp.SyncedAt != nil {
			syncStatus = "updated since last sync"
		}
		options[i] = ui.SelectOption{
			Label:       fmt.Sprintf("%s - %s", cp.Plan.ID, cp.Plan.Title),
			Value:       cp.Plan.ID,
			Description: fmt.Sprintf("Issue: %s (%s)", cp.Plan.IssueID, syncStatus),
		}
	}

	// Show multi-select
	selectedIDs, err := ui.RunMultiSelect("Select plans to sync:", options)
	if err != nil {
		return fmt.Errorf("failed to run multi-select: %w", err)
	}

	if len(selectedIDs) == 0 {
		fmt.Println("No plans selected.")
		return nil
	}

	// Build a map of selected plan IDs to cached plans for syncing
	idToPlan := make(map[string]*state.CachedPlan)
	for _, cp := range unsyncedPlans {
		idToPlan[cp.Plan.ID] = cp
	}

	// Sync selected plans
	syncer, err := getLinearPlanSyncer(cfg)
	if err != nil {
		return fmt.Errorf("failed to get syncer: %w", err)
	}
	labelName := cfg.Linear.GetPlanLabelName()

	deps := planSyncDeps{
		getCachedPlan:        state.DefaultCache.GetCachedPlan,
		markPlanSyncedWithHash: state.DefaultCache.MarkPlanSyncedWithHash,
		syncPlan: func(ctx context.Context, p *plan.Plan) error {
			return syncPlanWithSyncer(ctx, syncer, p, labelName)
		},
		computeContentHash: linear.ComputePlanContentHash,
	}

	return syncSelectedPlansWithDeps(ctx, selectedIDs, idToPlan, deps)
}

// syncSelectedPlansWithDeps syncs the selected plans using injected dependencies
func syncSelectedPlansWithDeps(ctx context.Context, planIDs []string, idToPlan map[string]*state.CachedPlan, deps planSyncDeps) error {
	var successCount int
	var skippedCount int
	var failures []string

	for _, planID := range planIDs {
		cp, ok := idToPlan[planID]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: plan not found", planID))
			continue
		}

		// Compute content hash for deduplication
		contentHash := deps.computeContentHash(cp.Plan)

		// Check if content is unchanged
		if cp.SyncedContentHash != "" && cp.SyncedContentHash == contentHash {
			skippedCount++
			printInfo(fmt.Sprintf("Skipped %s (content unchanged)", planID))
			continue
		}

		// Sync to Linear
		if err := deps.syncPlan(ctx, cp.Plan); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", planID, err))
			continue
		}

		// Mark as synced with content hash
		if err := deps.markPlanSyncedWithHash(planID, contentHash); err != nil {
			printWarning(fmt.Sprintf("Plan %s synced but failed to update sync timestamp: %v", planID, err))
		}

		successCount++
		printSuccess(fmt.Sprintf("Synced %s to issue %s", planID, cp.Plan.IssueID))
	}

	// Report results
	if successCount > 0 {
		fmt.Printf("\nSuccessfully synced %d plan(s)\n", successCount)
	}
	if skippedCount > 0 {
		fmt.Printf("Skipped %d plan(s) (content unchanged)\n", skippedCount)
	}

	if len(failures) > 0 {
		fmt.Printf("\nFailed to sync %d plan(s):\n", len(failures))
		for _, f := range failures {
			fmt.Printf("  - %s\n", f)
		}
		return fmt.Errorf("failed to sync %d plan(s)", len(failures))
	}

	return nil
}

func runPlanLink(cmd *cobra.Command, args []string) error {
	planID := args[0]
	issueID := args[1]

	ctx := context.Background()
	cfg := config.Get()

	// Initialize cache
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Load plan from cache
	cached, err := state.DefaultCache.GetCachedPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if cached == nil || cached.Plan == nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	// Validate that the issue exists in Linear
	if cfg.Default.Tracker == "linear" {
		t, err := getTracker(cfg)
		if err != nil {
			printWarning(fmt.Sprintf("Could not connect to tracker: %v", err))
		} else {
			issue, err := t.GetIssue(ctx, issueID)
			if err != nil {
				return fmt.Errorf("Linear issue not found: %s (%w)", issueID, err)
			}
			printInfo(fmt.Sprintf("Found issue: %s - %s", issue.Identifier, issue.Title))
		}
	}

	// Link the plan to the issue
	cached.Plan.IssueID = issueID
	cached.IssueID = issueID

	// Save the updated plan
	if err := state.DefaultCache.SavePlan(cached.Plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	printSuccess(fmt.Sprintf("Linked plan %s to issue %s", planID, issueID))

	// Sync plan content to issue if Linear sync is enabled
	if cfg.Default.Tracker == "linear" && cfg.Linear.ShouldSyncPlanOnSave() {
		// Get cached hash for deduplication
		cachedHash := cached.SyncedContentHash

		result, err := syncPlanToLinearWithDedup(ctx, cfg, cached.Plan, cachedHash)
		if err != nil {
			printWarning(fmt.Sprintf("Could not sync plan to Linear: %v", err))
		} else if result.Skipped {
			printInfo("Plan content unchanged, skipping sync")
		} else {
			if err := state.DefaultCache.MarkPlanSyncedWithHash(planID, result.ContentHash); err != nil {
				printWarning(fmt.Sprintf("Plan synced but failed to update sync timestamp: %v", err))
			}
			printSuccess(fmt.Sprintf("Plan synced to Linear issue %s", issueID))
		}
	}

	return nil
}
