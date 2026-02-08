package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/charleslr/jig/internal/tracker"
)

var (
	issueTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	issueIdentifierStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("12")).
				Bold(true)

	issueLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	issueStatusStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(lipgloss.Color("235"))

	issueDescriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))
)

// IssueViewModel displays a tracker issue in a nice format
type IssueViewModel struct {
	issue    *tracker.Issue
	scroll   int
	height   int
	width    int
	quitting bool
}

// NewIssueView creates a new issue view
func NewIssueView(issue *tracker.Issue) IssueViewModel {
	return IssueViewModel{
		issue:  issue,
		height: 20,
		width:  80,
	}
}

// Init implements tea.Model
func (m IssueViewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m IssueViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.width = msg.Width
	}

	return m, nil
}

// View implements tea.Model
func (m IssueViewModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Identifier and title
	b.WriteString(issueIdentifierStyle.Render(m.issue.Identifier))
	b.WriteString(" ")
	b.WriteString(issueTitleStyle.Render(m.issue.Title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Status
	b.WriteString(sectionStyle.Render("Status: "))
	b.WriteString(formatIssueStatus(m.issue.Status))
	b.WriteString("\n\n")

	// Labels
	if len(m.issue.Labels) > 0 {
		b.WriteString(sectionStyle.Render("Labels: "))
		b.WriteString(issueLabelStyle.Render(strings.Join(m.issue.Labels, ", ")))
		b.WriteString("\n\n")
	}

	// Description
	if m.issue.Description != "" {
		b.WriteString(sectionStyle.Render("Description"))
		b.WriteString("\n")
		b.WriteString(issueDescriptionStyle.Render(wrapText(m.issue.Description, 60)))
		b.WriteString("\n\n")
	}

	// URL
	if m.issue.URL != "" {
		b.WriteString(sectionStyle.Render("URL: "))
		b.WriteString(issueLabelStyle.Render(m.issue.URL))
		b.WriteString("\n\n")
	}

	// Help
	b.WriteString(unselectedStyle.Render("↑/↓ to scroll, q/esc to close"))

	return b.String()
}

func formatIssueStatus(status tracker.Status) string {
	switch status {
	case tracker.StatusBacklog:
		return issueStatusStyle.Copy().Background(lipgloss.Color("240")).Render("BACKLOG")
	case tracker.StatusTodo:
		return issueStatusStyle.Copy().Background(lipgloss.Color("12")).Render("TODO")
	case tracker.StatusInProgress:
		return issueStatusStyle.Copy().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0")).Render("IN PROGRESS")
	case tracker.StatusInReview:
		return issueStatusStyle.Copy().Background(lipgloss.Color("13")).Render("IN REVIEW")
	case tracker.StatusDone:
		return issueStatusStyle.Copy().Background(lipgloss.Color("10")).Foreground(lipgloss.Color("0")).Render("DONE")
	case tracker.StatusCanceled:
		return issueStatusStyle.Copy().Background(lipgloss.Color("9")).Render("CANCELED")
	default:
		return issueStatusStyle.Render(string(status))
	}
}

// ShowIssue displays an issue in an interactive view
func ShowIssue(issue *tracker.Issue) error {
	m := NewIssueView(issue)
	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// RenderIssueContext returns a formatted string of the issue context
// suitable for display in a non-interactive context
func RenderIssueContext(issue *tracker.Issue) string {
	return RenderIssueContextWithWidth(issue, 80)
}

// RenderIssueContextWithWidth returns a formatted string of the issue context
// with a specified width for markdown rendering
func RenderIssueContextWithWidth(issue *tracker.Issue, width int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("%s - %s\n", issue.Identifier, issue.Title))
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("Status: %s\n", issue.Status))

	if len(issue.Labels) > 0 {
		b.WriteString(fmt.Sprintf("Labels: %s\n", strings.Join(issue.Labels, ", ")))
	}

	if issue.Description != "" {
		b.WriteString("\nDescription:\n")
		b.WriteString(renderMarkdown(issue.Description, width))
		b.WriteString("\n")
	}

	if issue.URL != "" {
		b.WriteString(fmt.Sprintf("\nURL: %s\n", issue.URL))
	}

	return b.String()
}

// renderMarkdown renders markdown content using glamour
// Falls back to plain text wrapping if glamour fails
func renderMarkdown(content string, width int) string {
	// Use WithStandardStyle("dark") instead of WithAutoStyle() which is slow inside bubbletea
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return wrapText(content, width)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return wrapText(content, width)
	}

	return strings.TrimSpace(rendered)
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= width {
			result.WriteString(line)
			continue
		}

		// Wrap long lines
		words := strings.Fields(line)
		currentLine := ""
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			}
		}
		if currentLine != "" {
			result.WriteString(currentLine)
		}
	}

	return result.String()
}
