package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/prompt"
	"github.com/charleslr/jig/internal/runner"
	"github.com/charleslr/jig/internal/state"
)

var reviewCmd = &cobra.Command{
	Use:   "review [ISSUE]",
	Short: "Address PR review comments",
	Long: `Fetch unresolved PR review comments and launch your coding tool to address them.

If ISSUE is not provided, jig will try to detect the issue from the
current branch or worktree.

This command:
1. Fetches unresolved review comments from the PR
2. Prepares review context (comments, plan)
3. Launches your configured coding tool`,
	Args: cobra.MaximumNArgs(1),
	RunE: runReview,
}

var (
	reviewRunner   string
	reviewNoLaunch bool
	reviewPR       int
)

func init() {
	reviewCmd.Flags().StringVarP(&reviewRunner, "runner", "r", "", "coding tool to use (default from config)")
	reviewCmd.Flags().BoolVar(&reviewNoLaunch, "no-launch", false, "show comments but don't launch tool")
	reviewCmd.Flags().IntVar(&reviewPR, "pr", 0, "PR number (auto-detected if not specified)")
}

func runReview(cmd *cobra.Command, args []string) error {
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
	} else {
		// Try to detect from current worktree
		if err := state.InitWorktreeState(); err == nil {
			cwd, _ := git.GetWorktreeRoot()
			if info, err := state.DefaultWorktreeState.GetByPath(cwd); err == nil && info != nil {
				issueID = info.IssueID
			}
		}

		// Try to detect from branch name
		if issueID == "" {
			branch, _ := git.GetCurrentBranch()
			if branch != "" {
				// Extract issue ID from branch name (e.g., "ENG-123-feature" -> "ENG-123")
				parts := strings.SplitN(branch, "-", 3)
				if len(parts) >= 2 {
					issueID = parts[0] + "-" + parts[1]
				}
			}
		}
	}

	// Get PR number
	prNumber := reviewPR
	if prNumber == 0 {
		// Try to get from current branch
		pr, err := git.GetPR()
		if err != nil {
			return fmt.Errorf("no PR found for current branch (use --pr to specify)")
		}
		prNumber = pr.Number
	}

	printInfo(fmt.Sprintf("Fetching review comments for PR #%d...", prNumber))

	// Get PR details
	pr, err := git.GetPRByNumber(prNumber)
	if err != nil {
		return fmt.Errorf("failed to get PR: %w", err)
	}

	// Get unresolved review comments
	comments, err := git.GetPRReviewThreads(prNumber)
	if err != nil {
		printWarning(fmt.Sprintf("Could not fetch review threads: %v", err))
		// Fall back to regular comments
		comments, err = git.GetPRComments(prNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR comments: %w", err)
		}
	}

	if len(comments) == 0 {
		printSuccess("No unresolved review comments!")
		return nil
	}

	printInfo(fmt.Sprintf("Found %d unresolved comment(s)", len(comments)))

	// Format comments for display and prompt
	var commentStrs []string
	for i, c := range comments {
		commentStr := fmt.Sprintf("### Comment %d\n", i+1)
		commentStr += fmt.Sprintf("**Author:** %s\n", c.Author)
		if c.Path != "" {
			commentStr += fmt.Sprintf("**File:** %s", c.Path)
			if c.Line > 0 {
				commentStr += fmt.Sprintf(":%d", c.Line)
			}
			commentStr += "\n"
		}
		commentStr += fmt.Sprintf("\n%s\n", c.Body)
		commentStrs = append(commentStrs, commentStr)
	}

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Try to get plan from cache
	var p interface{}
	if issueID != "" {
		if plan, err := state.DefaultCache.GetPlan(issueID); err == nil && plan != nil {
			p = plan
		}
	}

	// Determine worktree path
	worktreePath, err := git.GetWorktreeRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Determine which runner to use
	runnerName := reviewRunner
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

	// Load and render the review prompt
	promptMgr, err := prompt.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize prompt manager: %w", err)
	}

	vars := &prompt.Vars{
		PRNumber:   fmt.Sprintf("%d", prNumber),
		PRTitle:    pr.Title,
		PRBody:     pr.Body,
		PRComments: commentStrs,
		BranchName: pr.HeadRefName,
	}
	if p != nil {
		// vars.Plan = p // Need type assertion
	}

	promptContent, err := promptMgr.LoadAndRender(prompt.TypeReview, vars)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Prepare the runner context
	prepOpts := &runner.PrepareOpts{
		WorktreeDir: worktreePath,
		PromptType:  runner.PromptTypeReview,
		ExtraVars: map[string]string{
			"pr_comments": strings.Join(commentStrs, "\n---\n"),
		},
	}
	if err := r.Prepare(ctx, prepOpts); err != nil {
		return fmt.Errorf("failed to prepare runner: %w", err)
	}

	if reviewNoLaunch {
		fmt.Println("\n## Unresolved Review Comments")
		fmt.Println()
		for _, c := range commentStrs {
			fmt.Println(c)
			fmt.Println("---")
		}
		fmt.Printf("\nTo address these comments, run:\n")
		fmt.Printf("  %s\n", runnerName)
		return nil
	}

	printInfo(fmt.Sprintf("Launching %s to address review comments...", runnerName))
	fmt.Println()
	fmt.Println("Address each review comment and commit your changes.")
	fmt.Println("After pushing, the reviewers will be notified.")
	fmt.Println()

	// Launch the runner as a subprocess (blocks until it exits)
	result, err := r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir: worktreePath,
		Prompt:      promptContent,
		Interactive: true,
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo(fmt.Sprintf("Review session ended (duration: %s, exit code: %d)", result.Duration.Round(time.Second), result.ExitCode))

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Push your changes: git push\n")
	fmt.Printf("  2. Reviewers will be notified\n")
	fmt.Printf("  3. When approved: jig merge %s\n", issueID)

	return nil
}
