package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/skills"
	"github.com/charleslr/jig/internal/tracker"
	"github.com/charleslr/jig/internal/tracker/linear"
	"github.com/charleslr/jig/internal/ui"
)

// OnboardingResult holds the collected onboarding data
type OnboardingResult struct {
	Tracker        string
	LinearAPIKey   string
	TeamID         string
	TeamName       string
	ProjectID      string
	ProjectName    string
	BranchPattern  string
	WorktreeDir    string
	InstallSkills  bool
	SkillsLocation string // "global" or "project"
}

// RunOnboarding runs the interactive onboarding wizard
func RunOnboarding() (*OnboardingResult, error) {
	if !ui.IsInteractive() {
		return nil, fmt.Errorf("onboarding requires an interactive terminal")
	}

	result := &OnboardingResult{}

	// State for dynamic steps
	var linearClient *linear.Client
	var teams []tracker.Team
	var projects []tracker.Project

	steps := []ui.WizardStep{
		// Step 1: Welcome
		{
			ID:          "welcome",
			Title:       "Welcome to Jig!",
			Description: "Let's set up your workflow orchestrator.",
			Type:        ui.StepTypeWelcome,
		},

		// Step 2: Issue Tracker Selection
		{
			ID:          "tracker",
			Title:       "Issue Tracker",
			Description: "Which issue tracker do you use?",
			Type:        ui.StepTypeSelect,
			Options: []ui.SelectOption{
				{Label: "Linear", Value: "linear", Description: "Connect to Linear for issue tracking"},
				{Label: "None", Value: "none", Description: "Skip issue tracker integration"},
			},
		},

		// Step 3: Linear API Key
		{
			ID:          "linear_api_key",
			Title:       "Linear API Key",
			Description: "Enter your Linear API key (from https://linear.app/settings/api)",
			Type:        ui.StepTypeInput,
			Placeholder: "lin_api_...",
			Secret:      true,
			Validate: func(value string) error {
				if value == "" {
					return fmt.Errorf("API key is required")
				}
				if !strings.HasPrefix(value, "lin_api_") {
					return fmt.Errorf("API key should start with 'lin_api_'")
				}
				return nil
			},
			ShouldSkip: func(results map[string]string) bool {
				return results["tracker"] != "linear"
			},
		},

		// Step 4: Validate Linear API Key
		{
			ID:          "validate_linear",
			Title:       "Validating API Key",
			Type:        ui.StepTypeSpinner,
			ActionLabel: "Connecting to Linear...",
			Action: func() error {
				apiKey := result.LinearAPIKey
				if apiKey == "" {
					return fmt.Errorf("no API key provided")
				}

				linearClient = linear.NewClient(apiKey, "", "")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				var err error
				teams, err = linearClient.GetTeams(ctx)
				if err != nil {
					return fmt.Errorf("failed to connect to Linear: %w", err)
				}

				if len(teams) == 0 {
					return fmt.Errorf("no teams found in your Linear workspace")
				}

				return nil
			},
			ShouldSkip: func(results map[string]string) bool {
				// Store the API key before this step runs
				if key, ok := results["linear_api_key"]; ok {
					result.LinearAPIKey = key
				}
				return results["tracker"] != "linear"
			},
		},

		// Step 5: Team Selection
		{
			ID:          "team",
			Title:       "Select Team",
			Description: "Which Linear team should jig use by default?",
			Type:        ui.StepTypeSelect,
			Options:     []ui.SelectOption{}, // Will be populated dynamically
			ShouldSkip: func(results map[string]string) bool {
				return results["tracker"] != "linear"
			},
		},

		// Step 6: Fetch Projects
		{
			ID:          "fetch_projects",
			Title:       "Loading Projects",
			Type:        ui.StepTypeSpinner,
			ActionLabel: "Fetching projects...",
			Action: func() error {
				if linearClient == nil || result.TeamID == "" {
					return nil
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				var err error
				projects, err = linearClient.GetProjects(ctx, result.TeamID)
				if err != nil {
					return fmt.Errorf("failed to fetch projects: %w", err)
				}

				return nil
			},
			ShouldSkip: func(results map[string]string) bool {
				// Store team ID before this step
				if teamVal, ok := results["team"]; ok && teamVal != "" {
					result.TeamID = teamVal
					for _, t := range teams {
						if t.ID == teamVal {
							result.TeamName = t.Name
							break
						}
					}
				}
				return results["tracker"] != "linear"
			},
		},

		// Step 7: Project Selection
		{
			ID:          "project",
			Title:       "Default Project (Optional)",
			Description: "Select a default project for new issues, or skip.",
			Type:        ui.StepTypeSelect,
			Options:     []ui.SelectOption{}, // Will be populated dynamically
			ShouldSkip: func(results map[string]string) bool {
				return results["tracker"] != "linear"
			},
		},

		// Step 8: Branch Pattern
		{
			ID:          "branch_pattern",
			Title:       "Branch Naming Pattern",
			Description: "Pattern for git branches. Use {issue_id} and {slug} placeholders.",
			Type:        ui.StepTypeInput,
			Placeholder: "{issue_id}-{slug}",
		},

		// Step 9: Worktree Directory
		{
			ID:          "worktree_dir",
			Title:       "Worktree Directory",
			Description: "Where should jig create git worktrees?",
			Type:        ui.StepTypeInput,
			Placeholder: getDefaultWorktreeDir(),
		},

		// Step 10: Install Claude Skills
		{
			ID:          "install_skills",
			Title:       "Claude Code Integration",
			Description: "Install jig skills for Claude Code?",
			Type:        ui.StepTypeConfirm,
		},

		// Step 11: Skills Location
		{
			ID:          "skills_location",
			Title:       "Skills Installation Location",
			Description: "Where should jig skills be installed?",
			Type:        ui.StepTypeSelect,
			Options: []ui.SelectOption{
				{
					Label:       "Global (Recommended)",
					Value:       "global",
					Description: "Install to ~/.claude/commands/jig/ (available in all projects)",
				},
				{
					Label:       "Project",
					Value:       "project",
					Description: "Install to ./.claude/commands/jig/ (this project only)",
				},
			},
			ShouldSkip: func(results map[string]string) bool {
				return results["install_skills"] != "yes"
			},
		},

		// Step 12: Summary
		{
			ID:    "summary",
			Title: "Setup Complete!",
			Type:  ui.StepTypeSummary,
			SummaryFunc: func(results map[string]string) string {
				var b strings.Builder

				b.WriteString("Configuration saved to ~/.jig/config.toml\n\n")

				if results["tracker"] == "linear" {
					b.WriteString(fmt.Sprintf("  Issue Tracker: Linear\n"))
					if result.TeamName != "" {
						b.WriteString(fmt.Sprintf("  Team: %s\n", result.TeamName))
					}
					if result.ProjectName != "" {
						b.WriteString(fmt.Sprintf("  Project: %s\n", result.ProjectName))
					}
				} else {
					b.WriteString("  Issue Tracker: None\n")
				}

				branchPattern := results["branch_pattern"]
				if branchPattern == "" {
					branchPattern = "{issue_id}-{slug}"
				}
				b.WriteString(fmt.Sprintf("  Branch Pattern: %s\n", branchPattern))

				worktreeDir := results["worktree_dir"]
				if worktreeDir == "" {
					worktreeDir = getDefaultWorktreeDir()
				}
				b.WriteString(fmt.Sprintf("  Worktree Dir: %s\n", worktreeDir))

				if results["install_skills"] == "yes" {
					location := results["skills_location"]
					if location == "global" {
						b.WriteString("  Claude Skills: Installed globally (~/.claude/commands/jig/)\n")
					} else if location == "project" {
						b.WriteString("  Claude Skills: Installed for this project (./.claude/commands/jig/)\n")
					} else {
						b.WriteString("  Claude Skills: Installed\n")
					}
				}

				b.WriteString("\nGet started:\n")
				b.WriteString("  jig plan    # Start planning a new feature\n")
				b.WriteString("  jig new     # Create a new issue\n")

				return b.String()
			},
		},
	}

	// Create a custom wizard that can update options dynamically
	results, completed, err := runOnboardingWizard(steps, result, &teams, &projects)
	if err != nil {
		return nil, err
	}

	if !completed {
		return nil, nil
	}

	// Populate result from collected data
	result.Tracker = results["tracker"]

	branchPattern := results["branch_pattern"]
	if branchPattern == "" {
		branchPattern = "{issue_id}-{slug}"
	}
	result.BranchPattern = branchPattern

	worktreeDir := results["worktree_dir"]
	if worktreeDir == "" {
		worktreeDir = getDefaultWorktreeDir()
	}
	result.WorktreeDir = worktreeDir

	result.InstallSkills = results["install_skills"] == "yes"

	// Store skills location preference (default to global if not specified)
	result.SkillsLocation = results["skills_location"]
	if result.SkillsLocation == "" {
		result.SkillsLocation = "global"
	}

	// Store project info
	if projectVal, ok := results["project"]; ok && projectVal != "" && projectVal != "skip" {
		result.ProjectID = projectVal
		for _, p := range projects {
			if p.ID == projectVal {
				result.ProjectName = p.Name
				break
			}
		}
	}

	return result, nil
}

// runOnboardingWizard runs a wizard with dynamic option updates
func runOnboardingWizard(steps []ui.WizardStep, result *OnboardingResult, teams *[]tracker.Team, projects *[]tracker.Project) (map[string]string, bool, error) {
	// We need to run the wizard in a way that allows dynamic updates
	// For now, we'll run it step by step

	results := make(map[string]string)

	for i := 0; i < len(steps); i++ {
		step := &steps[i]

		// Check if we should skip this step
		if step.ShouldSkip != nil && step.ShouldSkip(results) {
			continue
		}

		// Update options dynamically for certain steps
		if step.ID == "team" && len(*teams) > 0 {
			options := make([]ui.SelectOption, len(*teams))
			for j, t := range *teams {
				options[j] = ui.SelectOption{
					Label:       fmt.Sprintf("%s (%s)", t.Name, t.Key),
					Value:       t.ID,
					Description: fmt.Sprintf("Team key: %s", t.Key),
				}
			}
			step.Options = options
		}

		if step.ID == "project" && len(*projects) > 0 {
			options := make([]ui.SelectOption, len(*projects)+1)
			options[0] = ui.SelectOption{
				Label:       "Skip",
				Value:       "skip",
				Description: "Don't set a default project",
			}
			for j, p := range *projects {
				options[j+1] = ui.SelectOption{
					Label: p.Name,
					Value: p.ID,
				}
			}
			step.Options = options
		} else if step.ID == "project" && len(*projects) == 0 {
			// No projects found, skip this step
			continue
		}

		// Run the individual step
		value, cancelled, err := runSingleStep(*step, results)
		if err != nil {
			return results, false, err
		}
		if cancelled {
			return results, false, nil
		}

		// Store the result
		if step.ID != "" && value != "" {
			results[step.ID] = value
		}

		// Update result struct for spinner actions that need it
		if step.ID == "linear_api_key" {
			result.LinearAPIKey = value
		}
		if step.ID == "team" {
			result.TeamID = value
			for _, t := range *teams {
				if t.ID == value {
					result.TeamName = t.Name
					break
				}
			}
		}
	}

	return results, true, nil
}

// runSingleStep runs a single wizard step and returns the result
func runSingleStep(step ui.WizardStep, results map[string]string) (string, bool, error) {
	switch step.Type {
	case ui.StepTypeWelcome:
		return runWelcomeStep(step)

	case ui.StepTypeSelect:
		return runSelectStep(step)

	case ui.StepTypeInput:
		return runInputStep(step)

	case ui.StepTypeSpinner:
		return runSpinnerStep(step, results)

	case ui.StepTypeConfirm:
		return runConfirmStep(step)

	case ui.StepTypeSummary:
		return runSummaryStep(step, results)

	default:
		return "", false, nil
	}
}

func runWelcomeStep(step ui.WizardStep) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardTitleStyle().Render(step.Title))
	if step.Description != "" {
		fmt.Println(ui.WizardSubtitleStyle().Render(step.Description))
	}
	fmt.Println()

	// Simple press enter to continue
	fmt.Print("Press Enter to continue...")
	var input string
	_, _ = fmt.Scanln(&input) // Ignore error - just waiting for any input

	return "", false, nil
}

func runSelectStep(step ui.WizardStep) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardTitleStyle().Render(step.Title))
	if step.Description != "" {
		fmt.Println(ui.WizardSubtitleStyle().Render(step.Description))
	}
	fmt.Println()

	value, err := ui.RunSelect("", step.Options)
	if err != nil {
		return "", false, err
	}
	if value == "" {
		return "", true, nil
	}

	return value, false, nil
}

func runInputStep(step ui.WizardStep) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardTitleStyle().Render(step.Title))
	if step.Description != "" {
		fmt.Println(ui.WizardSubtitleStyle().Render(step.Description))
	}
	fmt.Println()

	var value string
	var err error

	if step.Secret {
		// Use secret input for sensitive data
		if step.Validate != nil {
			value, err = ui.RunSecretInputWithValidation("", step.Validate)
		} else {
			value, err = ui.RunSecretInput("")
		}
	} else {
		if step.Validate != nil {
			value, err = ui.RunInputWithValidation("", step.Placeholder, step.Validate)
		} else {
			value, err = ui.RunInput("", step.Placeholder)
		}
	}

	if err != nil {
		return "", false, err
	}

	// Use placeholder as default if empty (only for non-secret fields)
	if value == "" && !step.Secret {
		value = step.Placeholder
	}

	return value, false, nil
}

func runSpinnerStep(step ui.WizardStep, results map[string]string) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardTitleStyle().Render(step.Title))
	fmt.Println()

	actionLabel := step.ActionLabel
	if actionLabel == "" {
		actionLabel = "Processing..."
	}

	err := ui.RunWithSpinner(actionLabel, step.Action)
	if err != nil {
		return "", false, err
	}

	return "", false, nil
}

func runConfirmStep(step ui.WizardStep) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardTitleStyle().Render(step.Title))
	if step.Description != "" {
		fmt.Println(ui.WizardSubtitleStyle().Render(step.Description))
	}
	fmt.Println()

	confirmed, err := ui.RunConfirmWithDefault("", true)
	if err != nil {
		return "", false, err
	}

	if confirmed {
		return "yes", false, nil
	}
	return "no", false, nil
}

func runSummaryStep(step ui.WizardStep, results map[string]string) (string, bool, error) {
	fmt.Println()
	fmt.Println(ui.WizardSuccessStyle().Render(step.Title))
	fmt.Println()

	if step.SummaryFunc != nil {
		fmt.Println(step.SummaryFunc(results))
	}

	return "", false, nil
}

func getDefaultWorktreeDir() string {
	jigDir, _ := config.JigDir()
	return filepath.Join(jigDir, "worktrees")
}

// SaveOnboardingResult saves the onboarding result to configuration
func SaveOnboardingResult(result *OnboardingResult) error {
	// Save credentials securely
	if result.LinearAPIKey != "" {
		store, err := config.NewStore()
		if err != nil {
			return fmt.Errorf("failed to create credential store: %w", err)
		}
		if err := store.SetLinearAPIKey(result.LinearAPIKey); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
	}

	// Save configuration
	if err := config.Set("default.tracker", result.Tracker); err != nil {
		return fmt.Errorf("failed to set tracker: %w", err)
	}
	if err := config.Set("default.runner", "claude"); err != nil {
		return fmt.Errorf("failed to set runner: %w", err)
	}

	if result.TeamID != "" {
		if err := config.Set("linear.team_id", result.TeamID); err != nil {
			return fmt.Errorf("failed to set team_id: %w", err)
		}
	}
	if result.ProjectID != "" {
		if err := config.Set("linear.default_project", result.ProjectID); err != nil {
			return fmt.Errorf("failed to set default_project: %w", err)
		}
	}

	if err := config.Set("git.branch_pattern", result.BranchPattern); err != nil {
		return fmt.Errorf("failed to set branch_pattern: %w", err)
	}
	if err := config.Set("git.worktree_dir", result.WorktreeDir); err != nil {
		return fmt.Errorf("failed to set worktree_dir: %w", err)
	}

	// Save skills location preference
	if result.SkillsLocation != "" {
		if err := config.Set("claude.skills_location", result.SkillsLocation); err != nil {
			return fmt.Errorf("failed to set skills_location: %w", err)
		}
	}

	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// InstallClaudeSkills installs jig hooks and skills for Claude Code
// location can be "global" (user-level) or "project" (project-level)
func InstallClaudeSkills(location string) error {
	// Run the existing init logic to set up hooks
	if err := setupClaudeHooks(); err != nil {
		return fmt.Errorf("failed to set up Claude hooks: %w", err)
	}

	// Determine commands directory based on location
	var commandsDir string
	if location == "global" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		commandsDir = filepath.Join(homeDir, ".claude", "commands", "jig")
	} else {
		// Default to project-level
		commandsDir = ".claude/commands/jig"
	}

	// Ensure directory exists
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Write embedded skill files
	if err := installSkillFiles(commandsDir, initForce); err != nil {
		return fmt.Errorf("failed to install skill files: %w", err)
	}

	return nil
}

// InstallSkillFiles installs Claude Code skill files (exported for use in init.go)
// location can be "global" (user-level) or "project" (project-level)
func InstallSkillFiles(location string, force bool) error {
	// Determine commands directory based on location
	var commandsDir string
	if location == "global" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		commandsDir = filepath.Join(homeDir, ".claude", "commands", "jig")
	} else {
		// Default to project-level
		commandsDir = ".claude/commands/jig"
	}

	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	return installSkillFiles(commandsDir, force)
}

// installSkillFiles writes embedded skill files to the commands directory
func installSkillFiles(commandsDir string, force bool) error {
	skillFiles := []string{"plan.md", "implement.md"}

	for _, skillFile := range skillFiles {
		content, err := skills.EmbeddedSkills.ReadFile(skillFile)
		if err != nil {
			return fmt.Errorf("failed to read embedded skill %s: %w", skillFile, err)
		}

		destPath := filepath.Join(commandsDir, skillFile)

		// Check if file exists (skip unless force)
		if _, err := os.Stat(destPath); err == nil && !force {
			continue // File exists, don't overwrite
		}

		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", skillFile, err)
		}
	}

	return nil
}

// setupClaudeHooks sets up the Claude Code hooks for jig integration
func setupClaudeHooks() error {
	// Ensure .claude directory exists
	claudeDir := ".claude"
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Read existing settings if present
	var rawSettings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &rawSettings); err != nil {
			return fmt.Errorf("failed to parse existing settings.json: %w", err)
		}
	} else {
		rawSettings = make(map[string]interface{})
	}

	// Add or update the hooks
	hooks := getOrCreateHooksMap(rawSettings)

	// Remove any existing jig hooks first
	hooks["PreToolUse"] = filterNonJigHooksFromList(hooks["PreToolUse"])

	// Add the jig ExitPlanMode hook
	preToolUse := hooks["PreToolUse"].([]interface{})
	preToolUse = append(preToolUse, map[string]interface{}{
		"matcher": "ExitPlanMode",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "jig hook exit-plan-mode",
			},
		},
	})
	hooks["PreToolUse"] = preToolUse
	rawSettings["hooks"] = hooks

	// Add permissions for jig commands
	addJigPermissions(rawSettings)

	// Write back
	data, err := json.MarshalIndent(rawSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	return nil
}

// getOrCreateHooksMap gets or creates the hooks section
func getOrCreateHooksMap(settings map[string]interface{}) map[string]interface{} {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	if _, ok := hooks["PreToolUse"]; !ok {
		hooks["PreToolUse"] = []interface{}{}
	}

	return hooks
}

// filterNonJigHooksFromList removes jig hooks from the list
func filterNonJigHooksFromList(preToolUse interface{}) []interface{} {
	list, ok := preToolUse.([]interface{})
	if !ok {
		return []interface{}{}
	}

	var filtered []interface{}
	for _, hook := range list {
		h, ok := hook.(map[string]interface{})
		if !ok {
			filtered = append(filtered, hook)
			continue
		}

		// Check if this is a jig hook
		isJigHook := false
		if hooksList, ok := h["hooks"].([]interface{}); ok {
			for _, hc := range hooksList {
				if config, ok := hc.(map[string]interface{}); ok {
					if cmd, ok := config["command"].(string); ok {
						if cmd == "jig hook exit-plan-mode" {
							isJigHook = true
							break
						}
					}
				}
			}
		}

		if !isJigHook {
			filtered = append(filtered, hook)
		}
	}

	return filtered
}
