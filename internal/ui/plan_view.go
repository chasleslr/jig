package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charleslr/jig/internal/plan"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	statusStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("235"))
)

// PlanViewModel displays a plan in a nice format
type PlanViewModel struct {
	plan     *plan.Plan
	scroll   int
	height   int
	quitting bool
}

// NewPlanView creates a new plan view
func NewPlanView(p *plan.Plan) PlanViewModel {
	return PlanViewModel{
		plan:   p,
		height: 20,
	}
}

// Init implements tea.Model
func (m PlanViewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PlanViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}

		case "down", "j":
			m.scroll++

		case "home", "g":
			m.scroll = 0

		case "end", "G":
			m.scroll = 100 // Will be clamped
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height - 4
	}

	return m, nil
}

// View implements tea.Model
func (m PlanViewModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(m.plan.Title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Status
	b.WriteString(sectionStyle.Render("Status: "))
	b.WriteString(formatPlanStatus(m.plan.Status))
	b.WriteString("\n\n")

	// Progress
	if len(m.plan.Phases) > 0 {
		progress := m.plan.Progress()
		b.WriteString(sectionStyle.Render("Progress: "))
		b.WriteString(renderProgressBar(progress, 30))
		b.WriteString(fmt.Sprintf(" %.0f%%", progress))
		b.WriteString("\n\n")
	}

	// Full markdown body (preserving all content)
	if m.plan.RawContent != "" {
		body := extractMarkdownBody(m.plan.RawContent)
		if body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
	}

	// Help
	b.WriteString(unselectedStyle.Render("↑/↓ to scroll, q to quit"))

	return b.String()
}

func formatPlanStatus(status plan.Status) string {
	switch status {
	case plan.StatusDraft:
		return statusStyle.Copy().Background(lipgloss.Color("240")).Render("DRAFT")
	case plan.StatusReviewing:
		return statusStyle.Copy().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0")).Render("REVIEWING")
	case plan.StatusApproved:
		return statusStyle.Copy().Background(lipgloss.Color("10")).Foreground(lipgloss.Color("0")).Render("APPROVED")
	case plan.StatusInProgress:
		return statusStyle.Copy().Background(lipgloss.Color("12")).Render("IN PROGRESS")
	case plan.StatusComplete:
		return statusStyle.Copy().Background(lipgloss.Color("10")).Foreground(lipgloss.Color("0")).Render("COMPLETE")
	default:
		return statusStyle.Render(string(status))
	}
}

func renderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(bar)
}

// extractMarkdownBody extracts the markdown body from raw content (after frontmatter)
func extractMarkdownBody(rawContent string) string {
	// Look for the closing frontmatter delimiter "---"
	// The frontmatter is between the first "---" and second "---"
	const delimiter = "---"

	// Find the first delimiter
	firstIdx := strings.Index(rawContent, delimiter)
	if firstIdx == -1 {
		return rawContent // No frontmatter, return as-is
	}

	// Find the second delimiter (closing frontmatter)
	rest := rawContent[firstIdx+len(delimiter):]
	secondIdx := strings.Index(rest, delimiter)
	if secondIdx == -1 {
		return rawContent // Malformed frontmatter, return as-is
	}

	// Body starts after the second delimiter
	body := rest[secondIdx+len(delimiter):]
	return strings.TrimSpace(body)
}

// ShowPlan displays a plan in an interactive view
func ShowPlan(p *plan.Plan) error {
	m := NewPlanView(p)
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// RenderPlanSummary returns a simple text summary of a plan
func RenderPlanSummary(p *plan.Plan) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Plan: %s\n", p.Title))
	b.WriteString(fmt.Sprintf("Status: %s\n", p.Status))

	if len(p.Phases) > 0 {
		b.WriteString(fmt.Sprintf("Progress: %.0f%% (%d/%d phases)\n",
			p.Progress(),
			len(p.GetCompletedPhases()),
			len(p.Phases)))
	}

	return b.String()
}
