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

Shows:
- Issue ID and title
- Branch name
- Worktree path
- PR status (if any)
- Last activity`,
	RunE: runList,
}

var (
	listLong        bool
	listJSON        bool
	listInteractive bool
)

func init() {
	listCmd.Flags().BoolVarP(&listLong, "long", "l", false, "show detailed information")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().BoolVarP(&listInteractive, "interactive", "i", false, "interactive selection mode")
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
	type Item struct {
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

	items := make(map[string]*Item)

	// Add worktrees
	for _, wt := range worktrees {
		item := &Item{
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
		if item, ok := items[p.ID]; ok {
			item.Title = p.Title
			item.PlanStatus = string(p.Status)
		} else {
			items[p.ID] = &Item{
				IssueID:    p.ID,
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
	var sortedItems []*Item
	for _, item := range items {
		sortedItems = append(sortedItems, item)
	}
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].LastActive.After(sortedItems[j].LastActive)
	})

	// Interactive mode
	if listInteractive && ui.IsInteractive() {
		options := make([]ui.SelectOption, len(sortedItems))
		for i, item := range sortedItems {
			label := item.IssueID
			if item.Title != "" {
				label = fmt.Sprintf("%s: %s", item.IssueID, item.Title)
				if len(label) > 50 {
					label = label[:47] + "..."
				}
			}
			desc := ""
			if item.PRNumber > 0 {
				desc = fmt.Sprintf("PR #%d (%s)", item.PRNumber, item.PRState)
			} else if item.PlanStatus != "" {
				desc = item.PlanStatus
			}
			options[i] = ui.SelectOption{
				Label:       label,
				Value:       item.IssueID,
				Description: desc,
			}
		}

		selected, err := ui.RunSelect("Select an issue:", options)
		if err != nil {
			return fmt.Errorf("failed to select: %w", err)
		}
		if selected != "" {
			// Output the selected issue ID (useful for scripting)
			fmt.Println(selected)
		}
		return nil
	}

	// Print
	if listJSON {
		// Simple JSON output for scripting
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

	if listLong {
		// Detailed list
		for _, item := range sortedItems {
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
	} else {
		// Compact list
		fmt.Printf("%-15s %-30s %-10s %s\n", "ISSUE", "TITLE", "STATUS", "PR")
		fmt.Println(strings.Repeat("-", 70))

		for _, item := range sortedItems {
			title := item.Title
			if len(title) > 28 {
				title = title[:25] + "..."
			}
			if title == "" {
				title = "-"
			}

			status := item.PlanStatus
			if status == "" {
				status = "-"
			}

			pr := "-"
			if item.PRNumber > 0 {
				pr = fmt.Sprintf("#%d", item.PRNumber)
			}

			fmt.Printf("%-15s %-30s %-10s %s\n", item.IssueID, title, status, pr)
		}
	}

	return nil
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
