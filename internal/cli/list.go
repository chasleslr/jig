package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/charleslr/jig/internal/git"
	"github.com/charleslr/jig/internal/state"
	"github.com/charleslr/jig/internal/ui"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all active plans and worktrees",
	Long: `List all jig-managed worktrees and their status.

By default, shows an interactive table when running in a TTY with these shortcuts:
  o      - Open PR in browser (if PR exists)
  c      - Output worktree path for checkout/cd
  d      - Show detailed info for selected item
  enter  - Select and output issue ID
  q/esc  - Quit

Use --long for non-interactive detailed output, or --json for machine-readable output.
In non-TTY environments (pipes), falls back to --long output automatically.`,
	RunE: runList,
}

var (
	listLong bool
	listJSON bool
)

func init() {
	listCmd.Flags().BoolVarP(&listLong, "long", "l", false, "show detailed information")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
}

// listItem represents an item in the list display
type listItem struct {
	IssueID    string
	Title      string
	Branch     string
	Path       string
	PlanStatus string
	PRNumber   int
	PRState    string
	LastActive time.Time
	Exists     bool
}

func runList(cmd *cobra.Command, args []string) error {
	// Initialize state
	if err := state.Init(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	if err := state.InitWorktreeState(); err != nil {
		return fmt.Errorf("failed to initialize worktree state: %w", err)
	}

	// Get all tracked worktrees
	worktrees, err := state.DefaultWorktreeState.List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Get all cached plans
	plans, err := state.DefaultCache.ListPlans()
	if err != nil {
		printWarning(fmt.Sprintf("Could not list plans: %v", err))
	}

	// Build a combined list
	items := make(map[string]*listItem)

	// Add worktrees
	for _, wt := range worktrees {
		item := &listItem{
			IssueID:    wt.IssueID,
			Branch:     wt.Branch,
			Path:       wt.Path,
			LastActive: wt.LastUsedAt,
		}

		// Check if path exists
		if _, err := os.Stat(wt.Path); err == nil {
			item.Exists = true
		}

		items[wt.IssueID] = item
	}

	// Add plans
	for _, p := range plans {
		// Use IssueID when available (new format), fallback to ID (old format)
		issueID := p.IssueID
		if issueID == "" {
			issueID = p.ID
		}

		if item, ok := items[issueID]; ok {
			item.Title = p.Title
			item.PlanStatus = string(p.Status)
		} else {
			items[issueID] = &listItem{
				IssueID:    issueID,
				Title:      p.Title,
				PlanStatus: string(p.Status),
			}
		}
	}

	// Add PR info
	if git.GHAvailable() {
		for _, item := range items {
			if item.Branch != "" {
				pr, _ := git.GetPRForBranch(item.Branch)
				if pr != nil {
					item.PRNumber = pr.Number
					item.PRState = pr.State
				}
			}
		}
	}

	if len(items) == 0 {
		fmt.Println("No active work found")
		fmt.Println()
		fmt.Println("Get started:")
		fmt.Println("  jig new --title \"My feature\"    # Create a plan")
		fmt.Println("  jig implement ENG-123            # Start implementation")
		return nil
	}

	// Sort by last active (most recent first)
	var sortedItems []*listItem
	for _, item := range items {
		sortedItems = append(sortedItems, item)
	}
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].LastActive.After(sortedItems[j].LastActive)
	})

	// JSON output (explicit flag only)
	if listJSON {
		fmt.Println("[")
		for i, item := range sortedItems {
			comma := ","
			if i == len(sortedItems)-1 {
				comma = ""
			}
			fmt.Printf(`  {"issue_id": "%s", "title": "%s", "branch": "%s", "path": "%s"}%s%s`,
				item.IssueID,
				escapeJSON(item.Title),
				item.Branch,
				item.Path,
				comma,
				"\n",
			)
		}
		fmt.Println("]")
		return nil
	}

	// Long format (non-interactive) - explicit flag or non-TTY fallback
	if listLong || !ui.IsInteractive() {
		return printLongFormat(sortedItems)
	}

	// Default: interactive table
	return runInteractiveList(sortedItems)
}

// printLongFormat outputs detailed list information
func printLongFormat(items []*listItem) error {
	for _, item := range items {
		fmt.Printf("%s\n", item.IssueID)

		if item.Title != "" {
			fmt.Printf("  Title:  %s\n", item.Title)
		}
		if item.Branch != "" {
			fmt.Printf("  Branch: %s\n", item.Branch)
		}
		if item.Path != "" {
			exists := "✓"
			if !item.Exists {
				exists = "✗ (missing)"
			}
			fmt.Printf("  Path:   %s %s\n", item.Path, exists)
		}
		if item.PlanStatus != "" {
			fmt.Printf("  Plan:   %s\n", item.PlanStatus)
		}
		if item.PRNumber > 0 {
			fmt.Printf("  PR:     #%d (%s)\n", item.PRNumber, item.PRState)
		}
		if !item.LastActive.IsZero() {
			fmt.Printf("  Active: %s\n", formatRelativeTime(item.LastActive))
		}
		fmt.Println()
	}
	return nil
}

// runInteractiveList runs the interactive list table
func runInteractiveList(items []*listItem) error {
	listItems := make([]ui.ListItem, len(items))
	for i, item := range items {
		listItems[i] = ui.ListItem{
			IssueID:    item.IssueID,
			Title:      item.Title,
			Branch:     item.Branch,
			Path:       item.Path,
			PlanStatus: item.PlanStatus,
			PRNumber:   item.PRNumber,
			PRState:    item.PRState,
			LastActive: item.LastActive,
			Exists:     item.Exists,
		}
	}

	result, err := ui.RunListTable("Active Work", listItems)
	if err != nil {
		return err
	}

	switch result.Action {
	case ui.ListActionSelect:
		fmt.Println(result.Item.IssueID)
	case ui.ListActionOpenPR:
		if result.Item.PRNumber > 0 {
			return git.OpenPRInBrowser(result.Item.Branch)
		}
		fmt.Fprintln(os.Stderr, "No PR exists for this item")
	case ui.ListActionCheckout:
		if result.Item.Path != "" && result.Item.Exists {
			fmt.Println(result.Item.Path)
		} else {
			fmt.Fprintln(os.Stderr, "No worktree exists for this item")
		}
	case ui.ListActionDetails:
		printItemDetails(result.Item)
	}
	return nil
}

// printItemDetails prints detailed information about a list item
func printItemDetails(item *ui.ListItem) {
	fmt.Printf("%s\n", item.IssueID)
	if item.Title != "" {
		fmt.Printf("  Title:  %s\n", item.Title)
	}
	if item.Branch != "" {
		fmt.Printf("  Branch: %s\n", item.Branch)
	}
	if item.Path != "" {
		exists := "✓"
		if !item.Exists {
			exists = "✗ (missing)"
		}
		fmt.Printf("  Path:   %s %s\n", item.Path, exists)
	}
	if item.PlanStatus != "" {
		fmt.Printf("  Plan:   %s\n", item.PlanStatus)
	}
	if item.PRNumber > 0 {
		fmt.Printf("  PR:     #%d (%s)\n", item.PRNumber, item.PRState)
	}
	if !item.LastActive.IsZero() {
		fmt.Printf("  Active: %s\n", formatRelativeTime(item.LastActive))
	}
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
