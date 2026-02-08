package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/charleslr/jig/internal/config"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "jig",
		Short: "A workflow orchestrator for software engineering",
		Long: `Jig is a workflow orchestrator that manages the lifecycle of plans,
issues, worktrees, and PRs, delegating AI execution to external tools.`,
		Example: `  jig plan                # create a new plan
  jig implement AI-123    # implement the plan associated with an issue
  jig review AI-123       # address PR review comments for an issue
  jig merge               # merge the PR for a plan`,
		SilenceErrors: true,
		RunE:          handleUnknownCommand,
	}
)

// Execute runs the root command
func Execute() error {
	// Enable Cobra's built-in command suggestions
	rootCmd.SuggestionsMinimumDistance = 2

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.jig/config.toml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(implementCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(mergeCmd)
	rootCmd.AddCommand(checkoutCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(amendCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if err := config.Init(cfgFile); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
	}
}

// Helper functions for CLI

func exitWithError(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}

func printSuccess(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func printInfo(msg string) {
	fmt.Printf("→ %s\n", msg)
}

func printWarning(msg string) {
	fmt.Printf("⚠ %s\n", msg)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("jig v0.1.0")
	},
}

// handleUnknownCommand handles the case when no subcommand is provided
func handleUnknownCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	// Unknown subcommands are handled by Cobra's built-in suggestion feature
	// (enabled via SuggestionsMinimumDistance in Execute)
	return fmt.Errorf("unknown command %q", args[0])
}
