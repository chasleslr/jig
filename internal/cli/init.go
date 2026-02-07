package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize jig in the current project",
	Long: `Initialize jig in the current project by setting up Claude Code hooks.

This command adds the necessary hook configuration to .claude/settings.json
so that Claude prompts to save plans when exiting plan mode.`,
	RunE: runInit,
}

var initForce bool

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing hook configuration")
}

// ClaudeSettings represents the .claude/settings.json structure
type ClaudeSettings struct {
	Hooks *ClaudeHooks `json:"hooks,omitempty"`
	// Preserve other fields
	Other map[string]interface{} `json:"-"`
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
	if !initForce && hasJigHook(rawSettings) {
		printInfo("Jig hooks already configured in .claude/settings.json")
		fmt.Println("Use --force to overwrite existing configuration")
		return nil
	}

	// Add or update the hooks
	hooks := getOrCreateHooks(rawSettings)

	// Remove any existing jig hooks first
	hooks["PreToolUse"] = filterNonJigHooks(hooks["PreToolUse"])

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

	// Write back
	data, err := json.MarshalIndent(rawSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	printSuccess("Jig initialized successfully")
	fmt.Println()
	fmt.Println("Added to .claude/settings.json:")
	fmt.Println("  - PreToolUse hook for ExitPlanMode")
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
