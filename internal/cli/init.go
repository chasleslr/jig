package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize jig in the current project",
	Long: `Initialize jig in the current project.

In interactive mode, this runs a setup wizard that configures:
- Issue tracker integration (Linear)
- Git settings (branch patterns, worktree directory)
- Claude Code hooks and skills

In non-interactive mode (or with --hooks-only), only sets up Claude Code hooks.`,
	RunE: runInit,
}

var (
	initForce     bool
	initHooksOnly bool
)

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing hook configuration")
	initCmd.Flags().BoolVar(&initHooksOnly, "hooks-only", false, "only install Claude Code hooks, skip full setup")
}

// ClaudeSettings represents the .claude/settings.json structure
type ClaudeSettings struct {
	Hooks       *ClaudeHooks `json:"hooks,omitempty"`
	Permissions *Permissions `json:"permissions,omitempty"`
	// Preserve other fields
	Other map[string]interface{} `json:"-"`
}

type Permissions struct {
	Allow []string `json:"allow,omitempty"`
}

type ClaudeHooks struct {
	PreToolUse []PreToolUseHook `json:"PreToolUse,omitempty"`
}

type PreToolUseHook struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookConfig `json:"hooks"`
}

type HookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if already configured
	isConfigured := isJigConfigured()

	// If already configured and not forcing, just update hooks and skills
	if isConfigured && !initForce {
		return runQuickUpdate()
	}

	// If interactive and not hooks-only, run full onboarding
	if ui.IsInteractive() && !initHooksOnly {
		return runInteractiveInit()
	}

	// Otherwise, just set up hooks (non-interactive mode)
	return runHooksOnlyInit()
}

// isJigConfigured checks if jig has been configured (config.toml exists)
func isJigConfigured() bool {
	jigDir, err := config.JigDir()
	if err != nil {
		return false
	}
	configPath := filepath.Join(jigDir, "config.toml")
	_, err = os.Stat(configPath)
	return err == nil
}

// getSkillsLocation returns the preferred skills installation location from config
// Returns "global" if not configured
func getSkillsLocation() string {
	cfg := config.Get()
	if cfg.Claude.SkillsLocation != "" {
		return cfg.Claude.SkillsLocation
	}
	// Default to global if not configured
	return "global"
}

// runQuickUpdate updates hooks and skills without going through the full wizard
func runQuickUpdate() error {
	printInfo("Jig is already configured. Updating hooks and skills...")
	fmt.Println()

	// Install/update Claude skills (this handles both hooks and skill files)
	location := getSkillsLocation()
	if err := InstallClaudeSkills(location); err != nil {
		return fmt.Errorf("failed to update Claude skills: %w", err)
	}

	printSuccess("Hooks and skills updated successfully")
	fmt.Println()
	fmt.Println("Your jig setup is up to date!")
	fmt.Println()
	fmt.Println("Use 'jig init --force' to reconfigure from scratch.")

	return nil
}

// runInteractiveInit runs the full interactive onboarding wizard
func runInteractiveInit() error {
	result, err := RunOnboarding()
	if err != nil {
		return err
	}

	if result == nil {
		// User cancelled
		return nil
	}

	// Save the configuration
	if err := SaveOnboardingResult(result); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Install Claude skills if requested
	if result.InstallSkills {
		if err := InstallClaudeSkills(result.SkillsLocation); err != nil {
			printWarning(fmt.Sprintf("Could not install Claude skills: %v", err))
		} else {
			printSuccess("Claude Code hooks installed")
		}
	}

	return nil
}

// runHooksOnlyInit sets up only the Claude Code hooks
func runHooksOnlyInit() error {
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

	// Check if jig hook already exists
	hooksAlreadyExist := hasJigHook(rawSettings)
	location := getSkillsLocation()

	if !initForce && hooksAlreadyExist {
		// Hooks exist, just update skills without touching hooks
		printInfo("Jig hooks already configured. Updating skills...")
		fmt.Println()

		if err := InstallSkillFiles(location, initForce); err != nil {
			return fmt.Errorf("failed to update skills: %w", err)
		}

		printSuccess("Skills updated successfully")
		fmt.Println()
		fmt.Println("Use 'jig init --force' to reinstall hooks.")
		return nil
	}

	// Install Claude skills (hooks + skill files)
	if err := InstallClaudeSkills(location); err != nil {
		return fmt.Errorf("failed to install Claude skills: %w", err)
	}

	printSuccess("Jig initialized successfully")
	fmt.Println()
	fmt.Println("Added to .claude/settings.json:")
	fmt.Println("  - PreToolUse hook for ExitPlanMode")
	fmt.Println("  - Permissions for jig commands (auto-approved)")
	fmt.Println()
	fmt.Println("Installed Claude Code skills:")
	fmt.Println("  - /jig:plan")
	fmt.Println("  - /jig:implement")
	fmt.Println()
	fmt.Println("When you exit plan mode, Claude will now prompt you to save your plan.")
	fmt.Println()
	fmt.Println("Get started:")
	fmt.Println("  jig new    # Start a new planning session")

	return nil
}

// hasJigHook checks if jig hooks are already configured
func hasJigHook(settings map[string]interface{}) bool {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return false
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return false
	}

	for _, hook := range preToolUse {
		h, ok := hook.(map[string]interface{})
		if !ok {
			continue
		}
		hooksList, ok := h["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, hc := range hooksList {
			config, ok := hc.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, ok := config["command"].(string); ok {
				if cmd == "jig hook exit-plan-mode" {
					return true
				}
			}
		}
	}

	return false
}

// getOrCreateHooks gets or creates the hooks section
func getOrCreateHooks(settings map[string]interface{}) map[string]interface{} {
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

// filterNonJigHooks removes jig hooks from the list
func filterNonJigHooks(preToolUse interface{}) []interface{} {
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

// jigPermissions are the Bash command permissions needed for jig hooks to work seamlessly
var jigPermissions = []string{
	"Bash(jig plan save:*)",
	"Bash(jig hook mark-skip-save:*)",
	"Bash(jig hook mark-plan-saved:*)",
}

// addJigPermissions adds jig-specific permissions to the settings
func addJigPermissions(settings map[string]interface{}) {
	permissions, ok := settings["permissions"].(map[string]interface{})
	if !ok {
		permissions = make(map[string]interface{})
		settings["permissions"] = permissions
	}

	allowList, ok := permissions["allow"].([]interface{})
	if !ok {
		allowList = []interface{}{}
	}

	// Convert to string set for deduplication
	existing := make(map[string]bool)
	for _, item := range allowList {
		if s, ok := item.(string); ok {
			existing[s] = true
		}
	}

	// Add jig permissions if not already present
	for _, perm := range jigPermissions {
		if !existing[perm] {
			allowList = append(allowList, perm)
			existing[perm] = true
		}
	}

	permissions["allow"] = allowList
}
