package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/charleslr/jig/internal/config"
	"github.com/charleslr/jig/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage jig configuration",
	Long: `View and modify jig configuration.

Configuration is stored in ~/.jig/config.toml

Run without subcommands to interactively edit configuration.`,
	RunE: runConfigEdit,
}

var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := viper.Get(key)
		if value == nil {
			return fmt.Errorf("key not found: %s", key)
		}
		fmt.Println(value)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Handle special cases for credentials
		if key == "linear.api_key" {
			store, err := config.NewStore()
			if err != nil {
				return err
			}
			if err := store.SetLinearAPIKey(value); err != nil {
				return err
			}
			printSuccess("Linear API key saved securely")
			return nil
		}

		if err := config.Set(key, value); err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}

		printSuccess(fmt.Sprintf("Set %s = %s", key, value))
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()
		printSettings("", settings)
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive configuration setup",
	Long: `Walk through setting up jig configuration interactively.

Same as running 'jig config' without arguments.`,
	RunE: runConfigEdit,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configInitCmd)
}

func printSettings(prefix string, settings map[string]interface{}) {
	for key, value := range settings {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			printSettings(fullKey, v)
		default:
			fmt.Printf("%s = %v\n", fullKey, value)
		}
	}
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	if !ui.IsInteractive() {
		// Non-interactive: just list config
		settings := viper.AllSettings()
		printSettings("", settings)
		return nil
	}

	// Get current config values
	cfg := config.Get()
	jigDir, _ := config.JigDir()
	defaultWtDir := jigDir + "/worktrees"

	// Get Linear API key from secure store (show masked if set)
	linearAPIKey := ""
	store, err := config.NewStore()
	if err == nil {
		if key, _ := store.GetLinearAPIKey(); key != "" {
			linearAPIKey = key
		}
	}

	// Build form fields with current values
	fields := []ui.FormField{
		{
			Key:         "default.tracker",
			Label:       "Issue Tracker",
			Description: "Issue tracking service (linear or none)",
			Value:       cfg.Default.Tracker,
			Placeholder: "linear",
			Section:     "General",
		},
		{
			Key:         "linear.api_key",
			Label:       "Linear API Key",
			Description: "Get from https://linear.app/settings/api",
			Value:       linearAPIKey,
			Placeholder: "lin_api_...",
			Secret:      true,
			Section:     "Linear",
		},
		{
			Key:         "linear.team_id",
			Label:       "Linear Team ID",
			Description: "Your Linear team identifier (optional)",
			Value:       cfg.Linear.TeamID,
			Placeholder: "",
			Section:     "Linear",
		},
		{
			Key:         "linear.default_project",
			Label:       "Default Project",
			Description: "Default Linear project for new issues (optional)",
			Value:       cfg.Linear.DefaultProject,
			Placeholder: "",
			Section:     "Linear",
		},
		{
			Key:         "git.branch_pattern",
			Label:       "Branch Pattern",
			Description: "Pattern for branch names: {issue_id}, {slug}",
			Value:       cfg.Git.BranchPattern,
			Placeholder: "{issue_id}-{slug}",
			Section:     "Git",
		},
		{
			Key:         "git.worktree_dir",
			Label:       "Worktree Directory",
			Description: "Where to create git worktrees",
			Value:       cfg.Git.WorktreeDir,
			Placeholder: defaultWtDir,
			Section:     "Git",
		},
	}

	// Run the form
	updatedFields, saved, err := ui.RunForm(fields)
	if err != nil {
		return fmt.Errorf("form error: %w", err)
	}

	if !saved {
		return nil
	}

	// Apply the changes
	for _, field := range updatedFields {
		// Handle special cases
		if field.Key == "linear.api_key" {
			if field.Value != "" && field.Value != linearAPIKey {
				// API key changed, save to secure store
				if store != nil {
					if err := store.SetLinearAPIKey(field.Value); err != nil {
						printWarning(fmt.Sprintf("Could not save API key: %v", err))
					}
				}
			}
			continue
		}

		// Set regular config values
		if field.Value != "" {
			config.Set(field.Key, field.Value)
		}
	}

	// Ensure default runner is set
	config.Set("default.runner", "claude")

	// Save configuration
	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Path command for shell integration
var configPathCmd = &cobra.Command{
	Use:    "path",
	Short:  "Print the configuration directory path",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := config.JigDir()
		if err != nil {
			os.Exit(1)
		}
		fmt.Println(dir)
	},
}

func init() {
	configCmd.AddCommand(configPathCmd)
}
