package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charleslr/jig/internal/tracker"
)

// PlanPromptAction represents an action the user can take in the plan prompt menu
type PlanPromptAction int

const (
	PlanPromptActionStart PlanPromptAction = iota
	PlanPromptActionViewContext
	PlanPromptActionAddInstructions
	PlanPromptActionCancel
)

// PlanPromptResult holds the result from the plan prompt flow
type PlanPromptResult struct {
	Action       PlanPromptAction
	Instructions string // Additional instructions if provided
}

var (
	planPromptTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	planPromptSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	planPromptQuestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("12")).
				MarginTop(1)

	planPromptInstructionsStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("10")).
					MarginTop(1)
)

// planPromptState tracks the current state of the plan prompt flow
type planPromptState int

const (
	stateMenu planPromptState = iota
	stateViewContext
	stateAddInstructions
)

// PlanPromptModel is an interactive menu for the plan prompt flow
type PlanPromptModel struct {
	issue        *tracker.Issue
	state        planPromptState
	cursor       int
	instructions string
	result       *PlanPromptResult
	quitting     bool

	// Sub-models
	textArea         TextAreaModel
	contextViewport  viewport.Model
	contextReady     bool
	contextContent   string // Cached rendered content
	width            int
	height           int
}

type menuOption struct {
	label       string
	description string
	action      PlanPromptAction
}

var menuOptions = []menuOption{
	{label: "Start planning", description: "Launch the planning session with issue context", action: PlanPromptActionStart},
	{label: "View issue context", description: "Preview the issue details that will be sent", action: PlanPromptActionViewContext},
	{label: "Add instructions", description: "Provide additional instructions for the planning session", action: PlanPromptActionAddInstructions},
}

// NewPlanPrompt creates a new plan prompt model
func NewPlanPrompt(issue *tracker.Issue) PlanPromptModel {
	return PlanPromptModel{
		issue: issue,
		state: stateMenu,
	}
}

// Init implements tea.Model
func (m PlanPromptModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m PlanPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size for all states
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 4 // Title + separator + padding
		footerHeight := 3 // Help text + padding

		if !m.contextReady {
			m.contextViewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.contextViewport.YPosition = headerHeight
			m.contextReady = true
		} else {
			m.contextViewport.Width = msg.Width
			m.contextViewport.Height = msg.Height - headerHeight - footerHeight
		}

		// Update viewport content if we're in context view and have cached content
		if m.state == stateViewContext && m.contextContent != "" {
			m.contextViewport.SetContent(m.contextContent)
		}
	}

	switch m.state {
	case stateMenu:
		return m.updateMenu(msg)
	case stateViewContext:
		return m.updateViewContext(msg)
	case stateAddInstructions:
		return m.updateAddInstructions(msg)
	}
	return m, nil
}

func (m PlanPromptModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.result = &PlanPromptResult{Action: PlanPromptActionCancel}
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(menuOptions)-1 {
				m.cursor++
			}

		case "enter", " ":
			action := menuOptions[m.cursor].action
			switch action {
			case PlanPromptActionStart:
				m.result = &PlanPromptResult{
					Action:       PlanPromptActionStart,
					Instructions: m.instructions,
				}
				m.quitting = true
				return m, tea.Quit

			case PlanPromptActionViewContext:
				m.state = stateViewContext
				// Render content once and cache it
				if m.contextContent == "" {
					m.contextContent = m.renderContextContent()
				}
				if m.contextReady {
					m.contextViewport.SetContent(m.contextContent)
					m.contextViewport.GotoTop()
				}
				return m, nil

			case PlanPromptActionAddInstructions:
				m.state = stateAddInstructions
				m.textArea = NewTextArea("Additional instructions:").
					WithPlaceholder("Enter any additional context or instructions for the planning session...").
					WithSize(70, 6)
				if m.instructions != "" {
					// Pre-fill with existing instructions
					m.textArea.textarea.SetValue(m.instructions)
				}
				return m, m.textArea.Init()
			}
		}
	}
	return m, nil
}

func (m PlanPromptModel) updateViewContext(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Return to menu
			m.state = stateMenu
			return m, nil
		}
	}

	// Handle viewport updates (scrolling)
	m.contextViewport, cmd = m.contextViewport.Update(msg)
	return m, cmd
}

func (m PlanPromptModel) updateAddInstructions(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel and return to menu without saving
			m.state = stateMenu
			return m, nil

		case "ctrl+d":
			// Save instructions and start planning immediately
			m.instructions = m.textArea.Value()
			m.result = &PlanPromptResult{
				Action:       PlanPromptActionStart,
				Instructions: m.instructions,
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Update the text area
	newModel, cmd := m.textArea.Update(msg)
	m.textArea = newModel.(TextAreaModel)

	// Check if text area quit (submitted or cancelled)
	if m.textArea.submitted {
		// Save instructions and start planning immediately
		m.instructions = m.textArea.Value()
		m.result = &PlanPromptResult{
			Action:       PlanPromptActionStart,
			Instructions: m.instructions,
		}
		m.quitting = true
		return m, tea.Quit
	}
	if m.textArea.cancelled {
		// Return to menu without saving
		m.state = stateMenu
		return m, nil
	}

	return m, cmd
}

// View implements tea.Model
func (m PlanPromptModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case stateViewContext:
		return m.viewContext()
	case stateAddInstructions:
		return m.viewAddInstructions()
	default:
		return m.viewMenu()
	}
}

func (m PlanPromptModel) viewMenu() string {
	var b strings.Builder

	// Title with issue info
	b.WriteString(planPromptTitleStyle.Render("Planning: "))
	b.WriteString(issueIdentifierStyle.Render(m.issue.Identifier))
	b.WriteString(planPromptSubtitleStyle.Render(" - "))
	b.WriteString(planPromptSubtitleStyle.Render(m.issue.Title))
	b.WriteString("\n\n")

	// Show instructions indicator if we have any
	if m.instructions != "" {
		instructionsPreview := m.instructions
		if len(instructionsPreview) > 50 {
			instructionsPreview = instructionsPreview[:47] + "..."
		}
		instructionsPreview = strings.ReplaceAll(instructionsPreview, "\n", " ")
		b.WriteString(planPromptInstructionsStyle.Render("✓ Instructions added: "))
		b.WriteString(unselectedStyle.Render(instructionsPreview))
		b.WriteString("\n\n")
	}

	// Question
	b.WriteString(planPromptQuestionStyle.Render("What would you like to do?"))
	b.WriteString("\n\n")

	// Menu options
	for i, opt := range menuOptions {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		label := opt.label
		if m.cursor == i {
			label = selectedStyle.Render(label)
		} else {
			label = unselectedStyle.Render(label)
		}

		b.WriteString(cursor)
		b.WriteString(label)

		if opt.description != "" && m.cursor == i {
			b.WriteString("\n    ")
			b.WriteString(unselectedStyle.Render(opt.description))
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("↑/↓ to move, enter to select, q to quit"))

	return b.String()
}

func (m PlanPromptModel) viewContext() string {
	var b strings.Builder

	// Header
	b.WriteString(planPromptTitleStyle.Render("Issue Context"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Use cached content if available, otherwise render
	if m.contextContent != "" {
		b.WriteString(m.contextContent)
	} else {
		b.WriteString(m.renderContextContent())
	}

	// Footer with help
	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("q/esc to return to menu"))

	return b.String()
}

// renderContextContent renders the issue context for the viewport
func (m PlanPromptModel) renderContextContent() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	return RenderIssueContextWithWidth(m.issue, width)
}

func (m PlanPromptModel) viewAddInstructions() string {
	var b strings.Builder

	b.WriteString(planPromptTitleStyle.Render("Add Instructions"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	b.WriteString(m.textArea.View())

	return b.String()
}

// Result returns the result of the plan prompt
func (m PlanPromptModel) Result() *PlanPromptResult {
	return m.result
}

// RunPlanPrompt runs the plan prompt flow and returns the result
func RunPlanPrompt(issue *tracker.Issue) (*PlanPromptResult, error) {
	m := NewPlanPrompt(issue)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(PlanPromptModel)
	if model.Result() == nil {
		return &PlanPromptResult{Action: PlanPromptActionCancel}, nil
	}

	return model.Result(), nil
}
