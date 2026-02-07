package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	textareaPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	textareaHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// TextAreaModel is a multi-line text input component
type TextAreaModel struct {
	textarea  textarea.Model
	prompt    string
	hint      string
	submitted bool
	cancelled bool
}

// NewTextArea creates a new multi-line text area with a prompt
func NewTextArea(prompt string) TextAreaModel {
	ta := textarea.New()
	ta.Placeholder = "Describe what you want to plan..."
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(6)
	ta.ShowLineNumbers = false

	return TextAreaModel{
		textarea: ta,
		prompt:   prompt,
		hint:     "ctrl+d to submit, esc to cancel",
	}
}

// WithPlaceholder sets the placeholder text
func (m TextAreaModel) WithPlaceholder(placeholder string) TextAreaModel {
	m.textarea.Placeholder = placeholder
	return m
}

// WithSize sets the width and height
func (m TextAreaModel) WithSize(width, height int) TextAreaModel {
	m.textarea.SetWidth(width)
	m.textarea.SetHeight(height)
	return m
}

// WithHint sets the hint text shown below the input
func (m TextAreaModel) WithHint(hint string) TextAreaModel {
	m.hint = hint
	return m
}

// Init implements tea.Model
func (m TextAreaModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update implements tea.Model
func (m TextAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.cancelled = true
			return m, tea.Quit

		case "ctrl+d":
			// Submit on ctrl+d
			m.submitted = true
			return m, tea.Quit
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m TextAreaModel) View() string {
	if m.submitted {
		value := strings.TrimSpace(m.textarea.Value())
		// Show a truncated version of what was entered
		display := value
		if len(display) > 60 {
			display = display[:57] + "..."
		}
		display = strings.ReplaceAll(display, "\n", " ")
		return fmt.Sprintf("%s %s\n", textareaPromptStyle.Render(m.prompt), display)
	}

	var b strings.Builder

	b.WriteString(textareaPromptStyle.Render(m.prompt))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n")
	b.WriteString(textareaHintStyle.Render(m.hint))

	return b.String()
}

// Value returns the entered value
func (m TextAreaModel) Value() string {
	return strings.TrimSpace(m.textarea.Value())
}

// Cancelled returns whether the input was cancelled
func (m TextAreaModel) Cancelled() bool {
	return m.cancelled
}

// RunTextArea runs the textarea component and returns the entered value
func RunTextArea(prompt string) (string, error) {
	m := NewTextArea(prompt)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(TextAreaModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}

// RunTextAreaWithOptions runs the textarea with custom options
func RunTextAreaWithOptions(prompt, placeholder string, width, height int) (string, error) {
	m := NewTextArea(prompt).
		WithPlaceholder(placeholder).
		WithSize(width, height)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(TextAreaModel)
	if model.Cancelled() {
		return "", nil
	}

	return model.Value(), nil
}
