package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
)

var prCmd = &cobra.Command{
	Use:   "pr [ISSUE]",
	Short: "Create a PR and record it in metadata",
	Long: `Create a pull request using gh and record the PR info in issue metadata.

This command:
1. Creates a draft PR (or non-draft with --no-draft)
2. Records the PR number and URL in issue metadata
3. Enables 'jig merge ISSUE' to work without being on the branch

If ISSUE is provided, uses plan info for title/body if available.
Otherwise, auto-detects issue from current branch name.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPR,
}

var (
	prDraft bool
	prTitle string
	prBody  string
	prBase  string
)

func init() {
	prCmd.Flags().BoolVarP(&prDraft, "draft", "d", true, "create as draft PR")
	prCmd.Flags().StringVarP(&prTitle, "title", "t", "", "PR title (auto from plan if not provided)")
	prCmd.Flags().StringVarP(&prBody, "body", "b", "", "PR body")
	prCmd.Flags().StringVar(&prBase, "base", "", "base branch (default: main)")
}

func runPR(cmd *cobra.Command, args []string) error {
	return runPRWithClient(args, git.DefaultClient)
}

// runPRWithClient is the main implementation that accepts a git.Client for testing.
func runPRWithClient(args []string, client git.Client) error {
	// Check if gh is available
	if !client.Available() {
		return fmt.Errorf("GitHub CLI (gh) is not available or not authenticated")
	}

	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}

	// Determine issue ID
	var issueID string
	if len(args) > 0 {
		issueID = args[0]
	} else {
		// Try to detect from current branch
		branch, err := client.GetCurrentBranch()
		if err == nil && branch != "" {
			// Try to extract issue ID from branch name (e.g., NUM-123-feature-name)
			parts := strings.SplitN(branch, "-", 3)
			if len(parts) >= 2 {
				issueID = parts[0] + "-" + parts[1]
			}
		}
	}

	// Get current branch for the PR
	currentBranch, err := client.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if a PR already exists for this branch
	existingPR, err := client.GetPRForBranch(currentBranch)
	if err == nil && existingPR != nil {
		printInfo(fmt.Sprintf("PR #%d already exists for branch %s", existingPR.Number, currentBranch))
		printInfo(existingPR.URL)

		// Update metadata if we have an issue ID
		if issueID != "" {
			if err := updatePRMetadata(issueID, currentBranch, existingPR.Number, existingPR.URL); err != nil {
				printWarning(fmt.Sprintf("Could not update metadata: %v", err))
			} else {
				printSuccess("Metadata updated with PR info")
			}
		}
		return nil
	}

	// Get PR title and body
	title := prTitle
	body := prBody

	// If no title provided, try to get from plan
	if title == "" && issueID != "" {
		plan, _ := state.DefaultCache.GetPlan(issueID)
		if plan != nil {
			title = plan.Title
			if body == "" && plan.ProblemStatement != "" {
				body = fmt.Sprintf("Implements: %s\n\n## Problem\n%s\n\n## Solution\n%s", issueID, plan.ProblemStatement, plan.ProposedSolution)
			}
		}
	}

	// If still no title, use branch name
	if title == "" {
		title = currentBranch
	}

	// If no body, create a minimal one
	if body == "" {
		if issueID != "" {
			body = fmt.Sprintf("Implements: %s", issueID)
		} else {
			body = "PR created via jig pr"
		}
	}

	// Determine base branch
	baseBranch := prBase
	if baseBranch == "" {
		baseBranch = "main"
	}

	printInfo(fmt.Sprintf("Creating PR from %s to %s...", currentBranch, baseBranch))

	// Create the PR
	pr, err := client.CreatePR(title, body, baseBranch, prDraft)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	printSuccess(fmt.Sprintf("Created PR #%d", pr.Number))
	fmt.Println(pr.URL)

	// Record in metadata if we have an issue ID
	if issueID != "" {
		if err := updatePRMetadata(issueID, currentBranch, pr.Number, pr.URL); err != nil {
			printWarning(fmt.Sprintf("Could not update metadata: %v", err))
		} else {
			printSuccess("Metadata updated with PR info")
		}
	}

	return nil
}

func updatePRMetadata(issueID, branchName string, prNumber int, prURL string) error {
	meta, err := state.DefaultCache.GetIssueMetadata(issueID)
	if err != nil {
		return err
	}

	if meta == nil {
		meta = &state.IssueMetadata{
			IssueID: issueID,
		}
	}

	meta.BranchName = branchName
	meta.PRNumber = prNumber
	meta.PRURL = prURL

	return state.DefaultCache.SaveIssueMetadata(meta)
}
