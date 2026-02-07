package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	wizardTitleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	wizardSubtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	wizardStepStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	wizardSuccessStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	wizardErrorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	wizardDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	wizardHelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// WizardTitleStyle returns the title style for external use
func WizardTitleStyle() lipgloss.Style {
	return wizardTitleStyle
}

// WizardSubtitleStyle returns the subtitle style for external use
func WizardSubtitleStyle() lipgloss.Style {
	return wizardSubtitleStyle
}

// WizardSuccessStyle returns the success style for external use
func WizardSuccessStyle() lipgloss.Style {
	return wizardSuccessStyle
}

// WizardStepType defines the type of wizard step
type WizardStepType int

const (
	StepTypeWelcome WizardStepType = iota
	StepTypeSelect
	StepTypeInput
	StepTypeSpinner
	StepTypeConfirm
	StepTypeSummary
)

// WizardStep defines a single step in the wizard
type WizardStep struct {
	ID          string
	Title       string
	Description string
	Type        WizardStepType

	// For select steps
	Options []SelectOption

	// For input steps
	Placeholder string
	Secret      bool
	Validate    func(string) error

	// For spinner steps
	Action      func() error
	ActionLabel string

	// For summary steps
	SummaryFunc func(results map[string]string) string

	// Skip condition - if true, skip this step
	ShouldSkip func(results map[string]string) bool
}

// WizardModel orchestrates a multi-step wizard
type WizardModel struct {
	steps       []WizardStep
	currentStep int
	results     map[string]string
	err         error

	// Sub-models
	selectModel SelectModel
	inputModel  InputModel
	spinner     spinner.Model
	confirmYes  bool
	confirmCursor int

	// State
	completed  bool
	cancelled  bool
	showingSpinner bool
	spinnerDone    bool
	textInput  textinput.Model
	width      int
}

// NewWizard creates a new wizard with the given steps
func NewWizard(steps []WizardStep) WizardModel {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return WizardModel{
		steps:     steps,
		results:   make(map[string]string),
		textInput: ti,
		spinner:   s,
		width:     60,
	}
}

// Init implements tea.Model
func (m WizardModel) Init() tea.Cmd {
	return m.initCurrentStep()
}

func (m WizardModel) initCurrentStep() tea.Cmd {
	if m.currentStep >= len(m.steps) {
		return nil
	}

	step := m.steps[m.currentStep]

	// Check if we should skip this step
	if step.ShouldSkip != nil && step.ShouldSkip(m.results) {
		m.currentStep++
		return m.initCurrentStep()
	}

	switch step.Type {
	case StepTypeInput:
		m.textInput.SetValue("")
		m.textInput.Placeholder = step.Placeholder
		if step.Secret {
			m.textInput.EchoMode = textinput.EchoPassword
		} else {
			m.textInput.EchoMode = textinput.EchoNormal
		}
		m.textInput.Focus()
		return textinput.Blink

	case StepTypeSpinner:
		m.showingSpinner = true
		m.spinnerDone = false
		return tea.Batch(m.spinner.Tick, m.runStepAction())

	case StepTypeConfirm:
		m.confirmCursor = 0 // Default to Yes
		return nil

	default:
		return nil
	}
}

// WizardActionDone signals a spinner action completed
type WizardActionDone struct {
	Err error
}

func (m WizardModel) runStepAction() tea.Cmd {
	step := m.steps[m.currentStep]
	return func() tea.Msg {
		if step.Action != nil {
			err := step.Action()
			return WizardActionDone{Err: err}
		}
		return WizardActionDone{}
	}
}

// Update implements tea.Model
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.width > 80 {
			m.width = 80
		}

	case WizardActionDone:
		m.showingSpinner = false
		m.spinnerDone = true
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			// Auto-advance after successful spinner action
			return m.nextStep()
		}
		return m, nil

	case spinner.TickMsg:
		if m.showingSpinner {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		// Handle cancellation
		if msg.String() == "ctrl+c" {
			m.cancelled = true
			return m, tea.Quit
		}

		// Handle based on current step type
		if m.currentStep >= len(m.steps) {
			// On summary/final step
			if msg.String() == "enter" || msg.String() == "q" {
				m.completed = true
				return m, tea.Quit
			}
			return m, nil
		}

		step := m.steps[m.currentStep]

		switch step.Type {
		case StepTypeWelcome:
			if msg.String() == "enter" {
				return m.nextStep()
			}
			if msg.String() == "q" || msg.String() == "esc" {
				m.cancelled = true
				return m, tea.Quit
			}

		case StepTypeSelect:
			return m.handleSelectUpdate(msg)

		case StepTypeInput:
			return m.handleInputUpdate(msg)

		case StepTypeConfirm:
			return m.handleConfirmUpdate(msg)

		case StepTypeSpinner:
			// Spinner is automatic, no key handling needed (except cancel)
			if msg.String() == "q" || msg.String() == "esc" {
				m.cancelled = true
				return m, tea.Quit
			}

		case StepTypeSummary:
			if msg.String() == "enter" {
				m.completed = true
				return m, tea.Quit
			}
		}
	}

	return m, cmd
}

func (m WizardModel) handleSelectUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	step := m.steps[m.currentStep]

	// Initialize selectModel cursor if needed
	if len(step.Options) > 0 {
		switch msg.String() {
		case "up", "k":
			if m.selectModel.cursor > 0 {
				m.selectModel.cursor--
			}
		case "down", "j":
			if m.selectModel.cursor < len(step.Options)-1 {
				m.selectModel.cursor++
			}
		case "enter", " ":
			// Store the result
			m.results[step.ID] = step.Options[m.selectModel.cursor].Value
			m.selectModel.cursor = 0
			return m.nextStep()
		case "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m WizardModel) handleInputUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	step := m.steps[m.currentStep]

	switch msg.String() {
	case "enter":
		value := m.textInput.Value()

		// Validate if we have a validator
		if step.Validate != nil {
			if err := step.Validate(value); err != nil {
				m.err = err
				return m, nil
			}
		}

		// Store the result
		m.results[step.ID] = value
		m.err = nil
		return m.nextStep()

	case "esc":
		m.cancelled = true
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		m.err = nil
		return m, cmd
	}
}

func (m WizardModel) handleConfirmUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	step := m.steps[m.currentStep]

	switch msg.String() {
	case "y", "Y":
		m.results[step.ID] = "yes"
		return m.nextStep()

	case "n", "N":
		m.results[step.ID] = "no"
		return m.nextStep()

	case "left", "h":
		m.confirmCursor = 0

	case "right", "l":
		m.confirmCursor = 1

	case "tab":
		m.confirmCursor = (m.confirmCursor + 1) % 2

	case "enter", " ":
		if m.confirmCursor == 0 {
			m.results[step.ID] = "yes"
		} else {
			m.results[step.ID] = "no"
		}
		return m.nextStep()

	case "esc", "q":
		m.cancelled = true
		return m, tea.Quit
	}

	return m, nil
}

func (m WizardModel) nextStep() (tea.Model, tea.Cmd) {
	m.currentStep++

	// Skip steps that should be skipped
	for m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]
		if step.ShouldSkip != nil && step.ShouldSkip(m.results) {
			m.currentStep++
		} else {
			break
		}
	}

	if m.currentStep >= len(m.steps) {
		m.completed = true
		return m, tea.Quit
	}

	return m, m.initCurrentStep()
}

// View implements tea.Model
func (m WizardModel) View() string {
	if m.cancelled {
		return wizardDimStyle.Render("Setup cancelled.\n")
	}

	if m.completed || m.currentStep >= len(m.steps) {
		return m.viewSummary()
	}

	step := m.steps[m.currentStep]

	var b strings.Builder

	// Progress indicator
	progress := fmt.Sprintf("Step %d of %d", m.currentStep+1, len(m.steps))
	b.WriteString(wizardDimStyle.Render(progress))
	b.WriteString("\n\n")

	// Title
	b.WriteString(wizardTitleStyle.Render(step.Title))
	b.WriteString("\n")

	// Description
	if step.Description != "" {
		b.WriteString(wizardSubtitleStyle.Render(step.Description))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Step-specific content
	switch step.Type {
	case StepTypeWelcome:
		b.WriteString(m.viewWelcome())

	case StepTypeSelect:
		b.WriteString(m.viewSelect(step))

	case StepTypeInput:
		b.WriteString(m.viewInput(step))

	case StepTypeSpinner:
		b.WriteString(m.viewSpinner(step))

	case StepTypeConfirm:
		b.WriteString(m.viewConfirm(step))

	case StepTypeSummary:
		b.WriteString(m.viewSummary())
	}

	return b.String()
}

func (m WizardModel) viewWelcome() string {
	var b strings.Builder
	b.WriteString(wizardHelpStyle.Render("Press enter to continue, q to quit"))
	return b.String()
}

func (m WizardModel) viewSelect(step WizardStep) string {
	var b strings.Builder

	for i, opt := range step.Options {
		cursor := "  "
		if i == m.selectModel.cursor {
			cursor = cursorStyle.Render("> ")
		}

		label := opt.Label
		if i == m.selectModel.cursor {
			label = selectedStyle.Render(label)
		} else {
			label = unselectedStyle.Render(label)
		}

		b.WriteString(cursor)
		b.WriteString(label)

		if opt.Description != "" && i == m.selectModel.cursor {
			b.WriteString("\n    ")
			b.WriteString(wizardDimStyle.Render(opt.Description))
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardHelpStyle.Render("↑/↓ to move, enter to select, q to quit"))

	return b.String()
}

func (m WizardModel) viewInput(step WizardStep) string {
	var b strings.Builder

	b.WriteString(m.textInput.View())
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("Error: %s", m.err.Error())))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardHelpStyle.Render("enter to confirm, esc to cancel"))

	return b.String()
}

func (m WizardModel) viewSpinner(step WizardStep) string {
	var b strings.Builder

	if m.showingSpinner {
		label := step.ActionLabel
		if label == "" {
			label = "Processing..."
		}
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(label)
	} else if m.spinnerDone {
		if m.err != nil {
			b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("✗ %s", m.err.Error())))
		} else {
			b.WriteString(wizardSuccessStyle.Render("✓ Done"))
		}
	}

	b.WriteString("\n")

	return b.String()
}

func (m WizardModel) viewConfirm(step WizardStep) string {
	var b strings.Builder

	yesStyle := unselectedStyle
	noStyle := unselectedStyle

	if m.confirmCursor == 0 {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString("[")
	b.WriteString(yesStyle.Render("Yes"))
	b.WriteString("] [")
	b.WriteString(noStyle.Render("No"))
	b.WriteString("]")

	b.WriteString("\n\n")
	b.WriteString(wizardHelpStyle.Render("y/n to select, ←/→ to move, enter to confirm"))

	return b.String()
}

func (m WizardModel) viewSummary() string {
	var b strings.Builder

	b.WriteString(wizardSuccessStyle.Render("Setup Complete!"))
	b.WriteString("\n\n")

	// Check if last step has a summary function
	if len(m.steps) > 0 {
		lastStep := m.steps[len(m.steps)-1]
		if lastStep.Type == StepTypeSummary && lastStep.SummaryFunc != nil {
			b.WriteString(lastStep.SummaryFunc(m.results))
		}
	}

	return b.String()
}

// Results returns the collected results
func (m WizardModel) Results() map[string]string {
	return m.results
}

// Completed returns whether the wizard was completed
func (m WizardModel) Completed() bool {
	return m.completed
}

// Cancelled returns whether the wizard was cancelled
func (m WizardModel) Cancelled() bool {
	return m.cancelled
}

// RunWizard runs the wizard and returns the collected results
func RunWizard(steps []WizardStep) (map[string]string, bool, error) {
	m := NewWizard(steps)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, false, err
	}

	model := result.(WizardModel)
	return model.Results(), model.Completed(), nil
}
