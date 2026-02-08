package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// PlanViewModel displays a plan in a nice format with scrolling
type PlanViewModel struct {
	plan     *plan.Plan
	viewport viewport.Model
	renderer *glamour.TermRenderer
	width    int
	ready    bool
	quitting bool
}

// NewPlanView creates a new plan view
func NewPlanView(p *plan.Plan) PlanViewModel {
	return PlanViewModel{
		plan: p,
	}
}

// Init implements tea.Model
func (m PlanViewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PlanViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := m.headerHeight()
		footerHeight := 2 // help text + padding
		m.width = msg.Width

		if !m.ready {
			// Initialize glamour renderer with terminal width
			renderer, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(msg.Width),
			)
			if err == nil {
				m.renderer = renderer
			}

			// Initialize viewport with window size
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
			// Re-render content with new width
			if m.renderer != nil {
				renderer, err := glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					glamour.WithWordWrap(msg.Width),
				)
				if err == nil {
					m.renderer = renderer
					m.viewport.SetContent(m.renderContent())
				}
			}
		}
	}

	// Handle viewport updates (scrolling)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// headerHeight returns the height of the fixed header
func (m PlanViewModel) headerHeight() int {
	lines := 4 // title + separator + blank + status line
	if len(m.plan.Phases) > 0 {
		lines += 2 // progress line + blank
	}
	return lines
}

// renderHeader renders the fixed header (title, status, progress)
func (m PlanViewModel) renderHeader() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(m.plan.Title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Status
	b.WriteString(sectionStyle.Render("Status: "))
	b.WriteString(formatPlanStatus(m.plan.Status))
	b.WriteString("\n")

	// Progress
	if len(m.plan.Phases) > 0 {
		progress := m.plan.Progress()
		b.WriteString(sectionStyle.Render("Progress: "))
		b.WriteString(renderProgressBar(progress, 30))
		b.WriteString(fmt.Sprintf(" %.0f%%", progress))
		b.WriteString("\n")
	}

	return b.String()
}

// renderContent renders the scrollable content (markdown body)
func (m PlanViewModel) renderContent() string {
	if m.plan.RawContent == "" {
		return ""
	}

	body := extractMarkdownBody(m.plan.RawContent)
	if body == "" {
		return ""
	}

	// Use glamour to render markdown if available
	if m.renderer != nil {
		rendered, err := m.renderer.Render(body)
		if err == nil {
			return strings.TrimSpace(rendered)
		}
	}

	// Fallback to raw markdown
	return body
}

// renderFooter renders the help text
func (m PlanViewModel) renderFooter() string {
	info := fmt.Sprintf(" %3.f%% ", m.viewport.ScrollPercent()*100)
	return helpStyle.Render("↑/↓ scroll • q quit") + helpStyle.Render(info)
}

// View implements tea.Model
func (m PlanViewModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Loading..."
	}

	return fmt.Sprintf("%s\n%s\n%s", m.renderHeader(), m.viewport.View(), m.renderFooter())
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
