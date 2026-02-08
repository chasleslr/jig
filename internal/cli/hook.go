package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// HookInput represents the JSON input from Claude Code hooks
type HookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
	ToolName       string `json:"tool_name,omitempty"`
}

var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Hook commands for Claude Code integration",
	Hidden: true, // Hidden since these are invoked by Claude, not users
}

var hookExitPlanModeCmd = &cobra.Command{
	Use:   "exit-plan-mode",
	Short: "Hook for ExitPlanMode tool",
	Long: `This hook is invoked by Claude Code when exiting plan mode.
It prompts the user to save their plan before exiting.`,
	RunE: runHookExitPlanMode,
}

var hookMarkSkipSaveCmd = &cobra.Command{
	Use:    "mark-skip-save",
	Short:  "Mark that user wants to skip saving",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")
		if sessionID != "" {
			return createSessionMarker(sessionID, "skip-save")
		}
		return createMarker("skip-save")
	},
}

var hookMarkPlanSavedCmd = &cobra.Command{
	Use:    "mark-plan-saved",
	Short:  "Mark that plan has been saved",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session")
		if sessionID != "" {
			return createSessionMarker(sessionID, "plan-saved")
		}
		return createMarker("plan-saved")
	},
}

func init() {
	hookCmd.AddCommand(hookExitPlanModeCmd)
	hookCmd.AddCommand(hookMarkSkipSaveCmd)
	hookCmd.AddCommand(hookMarkPlanSavedCmd)

	// Add session flag to marker commands for session-scoped markers
	hookMarkSkipSaveCmd.Flags().String("session", "", "Session ID for session-scoped markers")
	hookMarkPlanSavedCmd.Flags().String("session", "", "Session ID for session-scoped markers")
}

// readHookInput reads and parses the JSON input from Claude Code via stdin
func readHookInput() (*HookInput, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	// If stdin is empty, return an empty HookInput (for backward compatibility)
	if len(data) == 0 {
		return &HookInput{}, nil
	}

	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse hook input JSON: %w", err)
	}

	return &input, nil
}

func runHookExitPlanMode(cmd *cobra.Command, args []string) error {
	// Read JSON input from Claude Code via stdin
	hookInput, err := readHookInput()
	if err != nil {
		// Log error but continue with empty input for backward compatibility
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		hookInput = &HookInput{}
	}

	// Get session ID for session-scoped markers
	sessionID := hookInput.SessionID

	// Check for "plan saved" marker (session-scoped if we have session ID)
	markerPath := getSessionMarkerPath(sessionID, "plan-saved")
	if _, err := os.Stat(markerPath); err == nil {
		// Plan already saved, allow exit and clean up marker
		os.Remove(markerPath)
		outputHookResponse("allow", "Plan has been saved. You may now exit plan mode.")
		return nil
	}

	// Check for "skip save" marker (user chose not to save)
	skipMarkerPath := getSessionMarkerPath(sessionID, "skip-save")
	if _, err := os.Stat(skipMarkerPath); err == nil {
		// User chose to skip, allow exit and clean up marker
		os.Remove(skipMarkerPath)
		outputHookResponse("allow", "")
		return nil
	}

	// Find plan file by extracting slug from session transcript
	planFile := findClaudePlan(hookInput.TranscriptPath)

	if planFile == "" {
		// No plan file found, allow exit
		outputHookResponse("allow", "")
		return nil
	}

	// Plan exists but not saved - deny and prompt user to save or discard
	outputHookResponse("deny", buildExitPlanModePrompt(planFile, sessionID))
	return nil
}

// findPlanFile looks for a plan file in the current directory
func findPlanFile() string {
	candidates := []string{"plan.md", "PLAN.md", "plan.markdown"}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	// Also check for any file matching *-plan.md
	matches, _ := filepath.Glob("*-plan.md")
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

// findClaudePlan looks for the plan file associated with the given session
// by extracting the slug from the session JSONL log.
// Returns empty string if no slug found or plan file doesn't exist.
func findClaudePlan(transcriptPath string) string {
	if transcriptPath == "" {
		return ""
	}

	slug := extractSlugFromSession(transcriptPath)
	if slug == "" {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	planPath := filepath.Join(homeDir, ".claude", "plans", slug+".md")
	if _, err := os.Stat(planPath); err == nil {
		return planPath
	}

	return ""
}

// extractSlugFromSession parses the session JSONL and extracts the slug field.
// Returns the most recent (last) slug found, or empty string if none.
func extractSlugFromSession(transcriptPath string) string {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lastSlug string
	for scanner.Scan() {
		var entry struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			if entry.Slug != "" {
				lastSlug = entry.Slug
			}
		}
	}
	return lastSlug
}

// getMarkerPath returns the path to a marker file (legacy, non-session-scoped)
func getMarkerPath(name string) string {
	// Store markers in .jig directory
	jigDir := ".jig"
	os.MkdirAll(jigDir, 0755)
	return filepath.Join(jigDir, name+".marker")
}

// getSessionMarkerPath returns the path to a session-scoped marker file
// If sessionID is empty, falls back to non-session-scoped markers
func getSessionMarkerPath(sessionID, name string) string {
	if sessionID == "" {
		return getMarkerPath(name)
	}

	// Store session-scoped markers in .jig/sessions/<session-id>/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return getMarkerPath(name)
	}

	sessionsDir := filepath.Join(homeDir, ".jig", "sessions", sessionID)
	os.MkdirAll(sessionsDir, 0755)
	return filepath.Join(sessionsDir, name+".marker")
}

// createMarker creates a marker file (legacy, non-session-scoped)
func createMarker(name string) error {
	path := getMarkerPath(name)
	return os.WriteFile(path, []byte{}, 0644)
}

// createSessionMarker creates a session-scoped marker file
func createSessionMarker(sessionID, name string) error {
	path := getSessionMarkerPath(sessionID, name)
	return os.WriteFile(path, []byte{}, 0644)
}

// HookSpecificOutput represents the inner payload for Claude Code hook responses
// When permissionDecision is set, it overrides Claude's default permission behavior
type HookSpecificOutput struct {
	// HookEventName identifies which hook event this response is for
	// Values: "PreToolUse", "PostToolUse", etc.
	HookEventName string `json:"hookEventName"`

	// PermissionDecision controls whether the tool is allowed to run
	// Values: "allow", "deny"
	PermissionDecision string `json:"permissionDecision,omitempty"`

	// PermissionDecisionReason is shown to Claude to explain the decision
	// Claude can use this to inform the user or take action
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// HookOutput represents the JSON output for Claude Code hooks
// The response must be wrapped in hookSpecificOutput per the Claude Code protocol
type HookOutput struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// outputHookResponse outputs a JSON hook response to stdout
// The response uses the hookSpecificOutput wrapper format required by Claude Code
func outputHookResponse(decision, reason string) {
	output := HookOutput{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       decision,
			PermissionDecisionReason: reason,
		},
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}

// readPlanSummary reads the plan file and returns a brief summary
func readPlanSummary(planFile string) string {
	content, err := os.ReadFile(planFile)
	if err != nil {
		return ""
	}

	// Return first 500 chars as summary, or full content if shorter
	text := string(content)
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}

// buildExitPlanModePrompt builds the prompt for Claude to show the user
func buildExitPlanModePrompt(planFile, sessionID string) string {
	// Build the marker command with session ID if available
	markSkipCmd := "jig hook mark-skip-save"
	if sessionID != "" {
		markSkipCmd = fmt.Sprintf("jig hook mark-skip-save --session %s", sessionID)
	}

	// Read plan summary for context
	planSummary := readPlanSummary(planFile)
	planContext := ""
	if planSummary != "" {
		planContext = fmt.Sprintf(`
Plan file: %s
Plan preview:
---
%s
---

`, planFile, planSummary)
	}

	return fmt.Sprintf(`Before exiting plan mode, ask the user what they want to do with their plan.
%s
Use the AskUserQuestion tool with these options:

question: "What would you like to do with your plan?"
header: "Save Plan"
options:
  1. label: "Save the plan"
     description: "Save the plan to jig for future implementation"
  2. label: "Exit without saving"
     description: "Discard the plan and exit"

Based on the user's choice:

- If "Save the plan":
  1. Run: jig plan save %s
  2. After saving, call ExitPlanMode again

- If "Exit without saving":
  1. Run: %s
  2. Call ExitPlanMode again
`, planContext, planFile, markSkipCmd)
}
