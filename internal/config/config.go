package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for jig
type Config struct {
	Default DefaultConfig         `mapstructure:"default"`
	Linear  LinearConfig          `mapstructure:"linear"`
	GitHub  GitHubConfig          `mapstructure:"github"`
	Claude  ClaudeConfig          `mapstructure:"claude"`
	Runners map[string]RunnerSpec `mapstructure:"runners"`
	Review  ReviewConfig          `mapstructure:"review"`
	Git     GitConfig             `mapstructure:"git"`
	Repos   map[string]RepoConfig `mapstructure:"repos"`
}

// DefaultConfig holds default settings
type DefaultConfig struct {
	Tracker string `mapstructure:"tracker"`
	Runner  string `mapstructure:"runner"`
}

// LinearConfig holds Linear API configuration
type LinearConfig struct {
	APIKey            string `mapstructure:"api_key"`
	TeamID            string `mapstructure:"team_id"`
	DefaultProject    string `mapstructure:"default_project"`
	SyncPlanOnSave    *bool  `mapstructure:"sync_plan_on_save"`    // default: true
	CreateIssueOnSave *bool  `mapstructure:"create_issue_on_save"` // default: true
	PlanLabelName     string `mapstructure:"plan_label_name"`      // default: "jig-plan"
}

// GitHubConfig holds GitHub configuration (uses gh CLI auth)
type GitHubConfig struct {
	// GitHub configuration is handled by gh CLI
}

// ClaudeConfig holds Claude Code configuration
type ClaudeConfig struct {
	SkillsLocation string `mapstructure:"skills_location"` // "global" or "project"
}

// RunnerSpec defines configuration for an external coding tool
type RunnerSpec struct {
	Command   string `mapstructure:"command"`
	SkillDir  string `mapstructure:"skill_dir"`
	PromptArg string `mapstructure:"prompt_arg"`
}

// ReviewConfig holds review workflow configuration
type ReviewConfig struct {
	DefaultReviewers  []string `mapstructure:"default_reviewers"`
	OptionalReviewers []string `mapstructure:"optional_reviewers"`
}

// GitConfig holds git-related configuration
type GitConfig struct {
	BranchPattern string `mapstructure:"branch_pattern"`
	WorktreeDir   string `mapstructure:"worktree_dir"`
}

// RepoConfig holds per-repository configuration
type RepoConfig struct {
	Path           string `mapstructure:"path"`
	TrackerProject string `mapstructure:"tracker_project"`
}

var (
	cfg     *Config
	cfgFile string
)

// JigDir returns the path to the jig configuration directory.
// It checks the JIG_HOME environment variable first, then falls back to ~/.jig.
func JigDir() (string, error) {
	if jigHome := os.Getenv("JIG_HOME"); jigHome != "" {
		return jigHome, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".jig"), nil
}

// Init initializes the configuration system
func Init(customCfgFile string) error {
	cfgFile = customCfgFile

	jigDir, err := JigDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(jigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set config file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(jigDir)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found is okay, we'll use defaults
	}

	// Unmarshal into struct
	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Default settings
	viper.SetDefault("default.tracker", "linear")
	viper.SetDefault("default.runner", "claude")

	// Default runners
	viper.SetDefault("runners.claude.command", "claude")
	viper.SetDefault("runners.claude.skill_dir", ".claude/skills")

	viper.SetDefault("runners.codex.command", "codex")

	viper.SetDefault("runners.opencode.command", "opencode")

	// Default review settings
	viper.SetDefault("review.default_reviewers", []string{"lead", "security"})
	viper.SetDefault("review.optional_reviewers", []string{"performance", "accessibility"})

	// Default git settings
	viper.SetDefault("git.branch_pattern", "{issue_id}-{slug}")

	// Default Linear sync settings
	viper.SetDefault("linear.sync_plan_on_save", true)
	viper.SetDefault("linear.create_issue_on_save", true)
	viper.SetDefault("linear.plan_label_name", "jig-plan")

	// Default Claude Code settings
	viper.SetDefault("claude.skills_location", "global")

	jigDir, _ := JigDir()
	viper.SetDefault("git.worktree_dir", filepath.Join(jigDir, "worktrees"))
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		// Return default config if not initialized
		return &Config{
			Default: DefaultConfig{
				Tracker: "linear",
				Runner:  "claude",
			},
		}
	}
	return cfg
}

// Set sets a configuration value
func Set(key string, value interface{}) error {
	viper.Set(key, value)
	return Save()
}

// Save writes the configuration to disk
func Save() error {
	jigDir, err := JigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(jigDir, "config.toml")
	return viper.WriteConfigAs(configPath)
}

// GetRunner returns the configuration for a specific runner
func (c *Config) GetRunner(name string) *RunnerSpec {
	if c.Runners == nil {
		return nil
	}
	if spec, ok := c.Runners[name]; ok {
		return &spec
	}
	return nil
}

// GetDefaultRunner returns the configuration for the default runner
func (c *Config) GetDefaultRunner() *RunnerSpec {
	return c.GetRunner(c.Default.Runner)
}

// PromptsDir returns the path to the user prompts directory
func PromptsDir() (string, error) {
	jigDir, err := JigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(jigDir, "prompts"), nil
}

// CacheDir returns the path to the cache directory
func CacheDir() (string, error) {
	jigDir, err := JigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(jigDir, "cache"), nil
}

// ShouldSyncPlanOnSave returns whether plans should be synced to Linear on save
func (c *LinearConfig) ShouldSyncPlanOnSave() bool {
	if c.SyncPlanOnSave == nil {
		return true // default
	}
	return *c.SyncPlanOnSave
}

// GetPlanLabelName returns the label name to use for plans in Linear
func (c *LinearConfig) GetPlanLabelName() string {
	if c.PlanLabelName == "" {
		return "jig-plan" // default
	}
	return c.PlanLabelName
}

// ShouldCreateIssueOnSave returns whether a Linear issue should be auto-created when saving a plan without a linked issue
func (c *LinearConfig) ShouldCreateIssueOnSave() bool {
	if c.CreateIssueOnSave == nil {
		return true // default
	}
	return *c.CreateIssueOnSave
}
