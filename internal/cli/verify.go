package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/runner"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [ISSUE]",
	Short: "Verify implementation against plan acceptance criteria",
	Long: `Verify that the implementation meets all acceptance criteria defined in the plan.

This command:
1. Loads the plan from the worktree context
2. Extracts acceptance criteria from the plan
3. Launches your coding tool to verify each criterion
4. Generates a verification report with pass/fail status

If no ISSUE is provided, uses the current worktree context.

Run this after implementation is complete but before creating a PR.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerify,
}

var (
	verifyRunner   string
	verifyNoLaunch bool
)

func init() {
	verifyCmd.Flags().StringVarP(&verifyRunner, "runner", "r", "", "coding tool to use (default from config)")
	verifyCmd.Flags().BoolVar(&verifyNoLaunch, "no-launch", false, "show verification instructions but don't launch tool")
}

func runVerify(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg := config.Get()

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	var issueID string

	// Get issue ID from args or current worktree
	if len(args) > 0 {
		issueID = args[0]
	} else {
		// Try to detect from current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Check if we're in a worktree with .jig context
		jigDir := filepath.Join(cwd, ".jig")
		if _, err := os.Stat(jigDir); os.IsNotExist(err) {
			return fmt.Errorf("not in a jig worktree (no .jig directory found). Run this from a worktree or specify ISSUE")
		}

		// Read issue metadata from .jig/issue.json
		issueJSONPath := filepath.Join(jigDir, "issue.json")
		data, err := os.ReadFile(issueJSONPath)
		if err != nil {
			return fmt.Errorf("failed to read issue metadata: %w", err)
		}

		var issueMeta struct {
			IssueID string `json:"issue_id"`
		}
		if err := json.Unmarshal(data, &issueMeta); err != nil {
			return fmt.Errorf("failed to parse issue metadata: %w", err)
		}
		issueID = issueMeta.IssueID
	}

	// Check that plan exists in worktree
	cwd, _ := os.Getwd()
	planPath := filepath.Join(cwd, ".jig", "plan.md")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return fmt.Errorf("no plan found at %s. Run this from a worktree set up by 'jig implement'", planPath)
	}

	// Determine which runner to use
	runnerName := verifyRunner
	if runnerName == "" {
		runnerName = cfg.Default.Runner
	}
	if runnerName == "" {
		runnerName = "claude"
	}

	if verifyNoLaunch {
		printSuccess("Ready for verification")
		fmt.Printf("\nIssue: %s\n", issueID)
		fmt.Printf("Plan: %s\n", planPath)
		fmt.Printf("\nTo verify:\n")
		fmt.Printf("  %s /jig:verify %s\n", runnerName, issueID)
		return nil
	}

	// Get the runner (only needed if we're going to launch)
	r, err := runner.Get(runnerName)
	if err != nil {
		return fmt.Errorf("runner not found: %s", runnerName)
	}

	if !r.Available() {
		return fmt.Errorf("runner '%s' is not available (not installed or not in PATH)", runnerName)
	}

	if ui.IsInteractive() {
		printInfo(fmt.Sprintf("Launching %s for verification...", runnerName))
	}
	fmt.Println()

	// Launch the runner with the /jig:verify skill
	_, err = r.Launch(ctx, &runner.LaunchOpts{
		WorktreeDir:     cwd,
		InitialPrompt:   fmt.Sprintf("/jig:verify %s", issueID),
		Interactive:     true,
		AutoAcceptEdits: false, // Verification should be read-only
	})
	if err != nil {
		return fmt.Errorf("failed to launch runner: %w", err)
	}

	// Post-session processing
	fmt.Println()
	printInfo("Verification session ended")

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  If all criteria passed:\n")
	fmt.Printf("    1. Create a PR: gh pr create --draft\n")
	fmt.Printf("    2. Address feedback: jig review %s\n", issueID)
	fmt.Printf("\n  If issues found:\n")
	fmt.Printf("    1. Fix the issues\n")
	fmt.Printf("    2. Re-verify: jig verify %s\n", issueID)

	return nil
}
