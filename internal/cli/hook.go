package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

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
		return createMarker("skip-save")
	},
}

var hookMarkPlanSavedCmd = &cobra.Command{
	Use:    "mark-plan-saved",
	Short:  "Mark that plan has been saved",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createMarker("plan-saved")
	},
}

func init() {
	hookCmd.AddCommand(hookExitPlanModeCmd)
	hookCmd.AddCommand(hookMarkSkipSaveCmd)
	hookCmd.AddCommand(hookMarkPlanSavedCmd)
}

func runHookExitPlanMode(cmd *cobra.Command, args []string) error {
	// Check for plan file in current directory
	planFile := findPlanFile()

	// Check for "plan saved" marker
	markerPath := getMarkerPath("plan-saved")
	if _, err := os.Stat(markerPath); err == nil {
		// Plan already saved, allow exit
		fmt.Println("Plan has been saved. You may now exit plan mode.")
		return nil // exit 0 = allow
	}

	// Check for "skip save" marker (user chose not to save)
	skipMarkerPath := getMarkerPath("skip-save")
	if _, err := os.Stat(skipMarkerPath); err == nil {
		// User chose to skip, allow exit and clean up marker
		os.Remove(skipMarkerPath)
		return nil // exit 0 = allow
	}

	if planFile == "" {
		// No plan file found, allow exit
		return nil // exit 0 = allow
	}

	// Plan exists but not saved - block and prompt user
	fmt.Print(buildExitPlanModePrompt(planFile))
	os.Exit(2) // exit 2 = block
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

// getMarkerPath returns the path to a marker file
func getMarkerPath(name string) string {
	// Store markers in .jig directory
	jigDir := ".jig"
	os.MkdirAll(jigDir, 0755)
	return filepath.Join(jigDir, name+".marker")
}

// createMarker creates a marker file
func createMarker(name string) error {
	path := getMarkerPath(name)
	return os.WriteFile(path, []byte{}, 0644)
}

// buildExitPlanModePrompt builds the prompt for Claude to show the user
func buildExitPlanModePrompt(planFile string) string {
	return fmt.Sprintf(`BLOCKED: Before exiting plan mode, ask the user what they want to do with their plan.

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
  1. Run: jig hook mark-skip-save
  2. Call ExitPlanMode again
`, planFile)
}
