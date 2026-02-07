package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	inputPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	inputErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// InputModel is a text input component
type InputModel struct {
	textInput    textinput.Model
	prompt       string
	defaultValue string
	validate     func(string) error
	err          error
	submitted    bool
	cancelled    bool
}

// NewInput creates a new text input with a prompt
func NewInput(prompt string, defaultValue string) InputModel {
	ti := textinput.New()
	ti.Placeholder = defaultValue
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return InputModel{
		textInput:    ti,
		prompt:       prompt,
		defaultValue: defaultValue,
	}
}

// WithValidation sets a validation function for the input
func (m InputModel) WithValidation(validate func(string) error) InputModel {
	m.validate = validate
	return m
}

// WithPlaceholder sets the placeholder text
func (m InputModel) WithPlaceholder(placeholder string) InputModel {
	m.textInput.Placeholder = placeholder
	return m
}

// WithWidth sets the input width
func (m InputModel) WithWidth(width int) InputModel {
	m.textInput.Width = width
	return m
}

// WithSecret sets the input to password mode (masks characters)
func (m InputModel) WithSecret() InputModel {
	m.textInput.EchoMode = textinput.EchoPassword
	return m
}

// Init implements tea.Model
func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			value := m.textInput.Value()
			if value == "" {
				value = m.defaultValue
			}

			// Validate if we have a validator
			if m.validate != nil {
				if err := m.validate(value); err != nil {
					m.err = err
					return m, nil
				}
			}

			m.submitted = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	m.err = nil // Clear error on new input
	return m, cmd
}

// View implements tea.Model
func (m InputModel) View() string {
	if m.submitted {
		value := m.textInput.Value()
		if value == "" {
			value = m.defaultValue
		}
		return fmt.Sprintf("%s %s\n", inputPromptStyle.Render(m.prompt), value)
	}

	var b strings.Builder

	b.WriteString(inputPromptStyle.Render(m.prompt))
	b.WriteString(" ")
	b.WriteString(m.textInput.View())

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(inputErrorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())))
	}

	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("enter to confirm, esc to cancel"))

	return b.String()
}

// Value returns the entered value (or default if empty)
func (m InputModel) Value() string {
	value := m.textInput.Value()
	if value == "" {
		return m.defaultValue
	}
	return value
}

// Cancelled returns whether the input was cancelled
func (m InputModel) Cancelled() bool {
	return m.cancelled
}

// RunInput runs the input component and returns the entered value
func RunInput(prompt string, defaultValue string) (string, error) {
	m := NewInput(prompt, defaultValue)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(InputModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}

// RunInputWithValidation runs the input component with validation
func RunInputWithValidation(prompt, defaultValue string, validate func(string) error) (string, error) {
	m := NewInput(prompt, defaultValue).WithValidation(validate)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(InputModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}

// RunSecretInput runs the input component in password mode
func RunSecretInput(prompt string) (string, error) {
	m := NewInput(prompt, "").WithSecret()
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(InputModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}

// RunSecretInputWithValidation runs secret input with validation
func RunSecretInputWithValidation(prompt string, validate func(string) error) (string, error) {
	m := NewInput(prompt, "").WithSecret().WithValidation(validate)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(InputModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}
