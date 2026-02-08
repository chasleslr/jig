package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charleslr/jig/internal/plan"
)

// ClaudeRunner implements the Runner interface for Claude Code
type ClaudeRunner struct {
	command  string
	skillDir string
}

// NewClaudeRunner creates a new Claude Code runner
func NewClaudeRunner(command, skillDir string) *ClaudeRunner {
	if command == "" {
		command = "claude"
	}
	if skillDir == "" {
		skillDir = ".claude/skills"
	}
	return &ClaudeRunner{
		command:  command,
		skillDir: skillDir,
	}
}

// Name returns the runner identifier
func (r *ClaudeRunner) Name() string {
	return "claude"
}

// Available checks if Claude Code is installed
func (r *ClaudeRunner) Available() bool {
	_, err := exec.LookPath(r.command)
	return err == nil
}

// Prepare sets up the context for Claude Code
func (r *ClaudeRunner) Prepare(ctx context.Context, opts *PrepareOpts) error {
	if opts.WorktreeDir == "" {
		return fmt.Errorf("worktree directory is required")
	}

	// Create the .jig directory for implementation context
	jigDir := filepath.Join(opts.WorktreeDir, ".jig")
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		return fmt.Errorf("failed to create .jig directory: %w", err)
	}

	// Handle planning-specific context files
	if opts.PromptType == PromptTypePlan {
		if err := r.writePlanningContext(jigDir, opts); err != nil {
			return err
		}
	}

	// Write the plan to .jig/plan.md if available
	if opts.Plan != nil {
		planContent, err := plan.Serialize(opts.Plan)
		if err != nil {
			return fmt.Errorf("failed to serialize plan: %w", err)
		}
		planPath := filepath.Join(jigDir, "plan.md")
		if err := os.WriteFile(planPath, planContent, 0644); err != nil {
			return fmt.Errorf("failed to write plan file: %w", err)
		}

		// Also write issue metadata as JSON for easy parsing
		issueMeta := map[string]interface{}{
			"issue_id":   opts.Plan.ID,
			"title":      opts.Plan.Title,
			"status":     opts.Plan.Status,
			"author":     opts.Plan.Author,
			"created_at": opts.Plan.Created,
			"updated_at": opts.Plan.Updated,
		}
		metaData, err := json.MarshalIndent(issueMeta, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize issue metadata: %w", err)
		}
		metaPath := filepath.Join(jigDir, "issue.json")
		if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
			return fmt.Errorf("failed to write issue metadata: %w", err)
		}
	}

	// Copy .claude/commands/ from the main repo to the worktree
	// so that Claude Code can find the /jig:implement skill
	if err := r.copyClaudeCommands(opts.WorktreeDir); err != nil {
		// Non-fatal - log but continue
		fmt.Fprintf(os.Stderr, "Warning: could not copy Claude commands: %v\n", err)
	}

	return nil
}

// writePlanningContext writes the planning-specific context files to .jig/sessions/<session-id>/
// The session ID is passed directly to the skill invocation to avoid race conditions
func (r *ClaudeRunner) writePlanningContext(jigDir string, opts *PrepareOpts) error {
	// Use session ID if provided, otherwise fall back to "default"
	sessionID := opts.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// Create session-specific directory for parallel planning support
	sessionDir := filepath.Join(jigDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Write planning goal context if provided
	if opts.PlanGoal != "" {
		content := fmt.Sprintf("# Planning Goal\n\n%s\n", opts.PlanGoal)
		contextPath := filepath.Join(sessionDir, "planning-context.md")
		if err := os.WriteFile(contextPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write planning context: %w", err)
		}
	}

	// Write issue context if provided (from Linear, etc.)
	if opts.IssueContext != "" {
		contextPath := filepath.Join(sessionDir, "issue-context.md")
		if err := os.WriteFile(contextPath, []byte(opts.IssueContext), 0644); err != nil {
			return fmt.Errorf("failed to write issue context: %w", err)
		}
	}

	return nil
}

// copyClaudeCommands copies the .claude/commands directory from the git root to the worktree
func (r *ClaudeRunner) copyClaudeCommands(worktreeDir string) error {
	// Find the main repo root (where .claude/commands lives)
	// The worktree's git dir points back to the main repo
	cmd := exec.Command("git", "-C", worktreeDir, "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find git common dir: %w", err)
	}

	gitCommonDir := strings.TrimSpace(string(output))
	// The common dir is typically .git, and the repo root is its parent
	mainRepoRoot := filepath.Dir(gitCommonDir)

	srcCommandsDir := filepath.Join(mainRepoRoot, ".claude", "commands")
	if _, err := os.Stat(srcCommandsDir); os.IsNotExist(err) {
		// No commands to copy
		return nil
	}

	dstCommandsDir := filepath.Join(worktreeDir, ".claude", "commands")
	if err := os.MkdirAll(filepath.Dir(dstCommandsDir), 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Copy the commands directory recursively
	return copyDir(srcCommandsDir, dstCommandsDir)
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// Launch starts Claude Code as a subprocess with TTY passthrough.
// It blocks until Claude Code exits and returns information about the session.
func (r *ClaudeRunner) Launch(ctx context.Context, opts *LaunchOpts) (*LaunchResult, error) {
	startTime := time.Now()

	// Build arguments
	var args []string

	// If a prompt is provided and we want non-interactive mode, use -p flag
	if opts.Prompt != "" && !opts.Interactive {
		args = append(args, "-p", opts.Prompt)
	}

	// If an initial prompt is provided for interactive mode, pass as positional arg
	// This sends the message and continues interactively
	if opts.InitialPrompt != "" && opts.Interactive {
		args = append(args, opts.InitialPrompt)
	}

	// Use plan mode for planning sessions
	if opts.PlanMode {
		args = append(args, "--permission-mode", "plan")
	} else if opts.AutoAcceptEdits {
		// Auto-accept file edits for implementation sessions
		args = append(args, "--permission-mode", "acceptEdits")
	}

	// Add system prompt if provided
	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}

	// Add any extra arguments
	args = append(args, opts.Args...)

	// Create the command
	cmd := exec.CommandContext(ctx, r.command, args...)

	// Set working directory if specified
	if opts.WorktreeDir != "" {
		cmd.Dir = opts.WorktreeDir
	}

	// Pass through stdin for interactive TTY support
	cmd.Stdin = os.Stdin

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to complete
	err := cmd.Run()

	result := &LaunchResult{
		Duration: time.Since(startTime),
		ExitCode: 0,
	}

	if err != nil {
		// Check if it's an exit error to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			// Non-zero exit is not necessarily an error (user may have quit)
			return result, nil
		}
		return result, fmt.Errorf("failed to run claude: %w", err)
	}

	return result, nil
}

func buildImplementSkill(opts *PrepareOpts) string {
	content := `# Jig Implementation Session

You are implementing a planned feature. Follow the plan carefully.

## Guidelines

1. **Follow the plan**: Implement exactly what's specified in the plan
2. **Test as you go**: Ensure each change works before moving on
3. **Commit logically**: Make atomic commits that match the plan

`
	if opts.Plan != nil {
		content += fmt.Sprintf("## Plan: %s\n\n", opts.Plan.Title)

		if opts.Plan.ProposedSolution != "" {
			content += fmt.Sprintf("### Proposed Solution\n%s\n\n", opts.Plan.ProposedSolution)
		}
	}

	return content
}

func buildReviewSkill(opts *PrepareOpts) string {
	content := `# Jig Review Session

You are addressing PR review comments.

## Guidelines

1. **Address all comments**: Go through each review comment systematically
2. **Explain your changes**: When making changes, explain why
3. **Ask for clarification**: If a comment is unclear, ask before changing
4. **Don't over-engineer**: Make minimal changes to address the feedback

## Review Comments

`
	if opts.ExtraVars != nil {
		if comments, ok := opts.ExtraVars["pr_comments"]; ok {
			content += comments + "\n"
		}
	}

	return content
}

func buildLeadReviewSkill(opts *PrepareOpts) string {
	content := `# Jig Lead Engineer Review

You are reviewing a plan as a lead engineer.

## Review Focus

1. **Architecture**: Is the proposed architecture sound?
2. **Scalability**: Will this approach scale?
3. **Maintainability**: Is the code easy to maintain?
4. **Best practices**: Does it follow team conventions?
5. **Technical debt**: Does it introduce unnecessary technical debt?

## Plan to Review

`
	if opts.Plan != nil {
		content += fmt.Sprintf("### %s\n\n", opts.Plan.Title)
		if opts.Plan.ProblemStatement != "" {
			content += fmt.Sprintf("**Problem:**\n%s\n\n", opts.Plan.ProblemStatement)
		}
		if opts.Plan.ProposedSolution != "" {
			content += fmt.Sprintf("**Solution:**\n%s\n\n", opts.Plan.ProposedSolution)
		}
	}

	return content
}

func buildSecurityReviewSkill(opts *PrepareOpts) string {
	content := `# Jig Security Review

You are reviewing a plan from a security perspective.

## Review Focus

1. **Authentication/Authorization**: Are access controls appropriate?
2. **Data validation**: Is all input validated?
3. **Injection attacks**: SQL, XSS, command injection risks?
4. **Secrets management**: How are secrets handled?
5. **Data exposure**: Is sensitive data properly protected?
6. **Dependencies**: Are there vulnerable dependencies?

## Plan to Review

`
	if opts.Plan != nil {
		content += fmt.Sprintf("### %s\n\n", opts.Plan.Title)
		if opts.Plan.ProblemStatement != "" {
			content += fmt.Sprintf("**Problem:**\n%s\n\n", opts.Plan.ProblemStatement)
		}
		if opts.Plan.ProposedSolution != "" {
			content += fmt.Sprintf("**Solution:**\n%s\n\n", opts.Plan.ProposedSolution)
		}
	}

	return content
}

func init() {
	// Register the Claude runner with default settings
	Register(NewClaudeRunner("", ""))
}
