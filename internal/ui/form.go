package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	formLabelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	formFocusedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	formValueStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	formDescStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	formSectionStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	formSavedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	formInputPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

// FormField represents a single field in a form
type FormField struct {
	Key         string // Config key (e.g., "linear.api_key")
	Label       string // Display label
	Description string // Help text
	Value       string // Current value
	Placeholder string // Placeholder when empty
	Secret      bool   // Mask the value (for API keys, etc.)
	Section     string // Group fields by section
}

// FormModel is an interactive form component
type FormModel struct {
	fields    []FormField
	cursor    int
	editing   bool
	textInput textinput.Model
	saved     bool
	cancelled bool
	width     int
}

// NewForm creates a new form with the given fields
func NewForm(fields []FormField) FormModel {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 50

	return FormModel{
		fields:    fields,
		textInput: ti,
		width:     60,
	}
}

// Init implements tea.Model
func (m FormModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.width > 80 {
			m.width = 80
		}

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				// Save the edited value
				m.fields[m.cursor].Value = m.textInput.Value()
				m.editing = false
				return m, nil

			case "esc":
				// Cancel editing
				m.editing = false
				return m, nil
			}

			// Pass to text input
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}

		case "enter", "e":
			// Start editing
			m.editing = true
			field := m.fields[m.cursor]
			m.textInput.SetValue(field.Value)
			m.textInput.Placeholder = field.Placeholder
			if field.Secret {
				m.textInput.EchoMode = textinput.EchoPassword
			} else {
				m.textInput.EchoMode = textinput.EchoNormal
			}
			m.textInput.Focus()
			return m, textinput.Blink

		case "s", "ctrl+s":
			// Save and exit
			m.saved = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model
func (m FormModel) View() string {
	if m.saved {
		return formSavedStyle.Render("✓ Configuration saved\n")
	}
	if m.cancelled {
		return "Configuration unchanged\n"
	}

	var b strings.Builder

	b.WriteString(formSectionStyle.Render("Jig Configuration"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 40))
	b.WriteString("\n\n")

	currentSection := ""

	for i, field := range m.fields {
		// Section header
		if field.Section != "" && field.Section != currentSection {
			currentSection = field.Section
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(formSectionStyle.Render(fmt.Sprintf("▸ %s", currentSection)))
			b.WriteString("\n\n")
		}

		// Field
		cursor := "  "
		if i == m.cursor {
			cursor = formFocusedStyle.Render("> ")
		}

		label := field.Label
		if i == m.cursor {
			label = formFocusedStyle.Render(label)
		} else {
			label = formLabelStyle.Render(label)
		}

		b.WriteString(cursor)
		b.WriteString(label)
		b.WriteString(": ")

		// Value or input
		if m.editing && i == m.cursor {
			b.WriteString(m.textInput.View())
		} else {
			value := field.Value
			if value == "" {
				value = formLabelStyle.Render("(not set)")
			} else if field.Secret {
				value = strings.Repeat("•", min(len(value), 20))
			}
			if i == m.cursor {
				value = formValueStyle.Render(value)
			}
			b.WriteString(value)
		}

		b.WriteString("\n")

		// Description (only for focused field)
		if i == m.cursor && field.Description != "" && !m.editing {
			b.WriteString("    ")
			b.WriteString(formDescStyle.Render(field.Description))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.editing {
		b.WriteString(formLabelStyle.Render("enter save • esc cancel"))
	} else {
		b.WriteString(formLabelStyle.Render("↑/↓ navigate • enter edit • s save • q quit"))
	}

	return b.String()
}

// Fields returns the current field values
func (m FormModel) Fields() []FormField {
	return m.fields
}

// Saved returns whether the form was saved
func (m FormModel) Saved() bool {
	return m.saved
}

// Cancelled returns whether the form was cancelled
func (m FormModel) Cancelled() bool {
	return m.cancelled
}

// GetValue returns the value of a field by key
func (m FormModel) GetValue(key string) string {
	for _, f := range m.fields {
		if f.Key == key {
			return f.Value
		}
	}
	return ""
}

// RunForm runs the form and returns the updated fields
func RunForm(fields []FormField) ([]FormField, bool, error) {
	m := NewForm(fields)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, false, err
	}

	model := result.(FormModel)
	return model.Fields(), model.Saved(), nil
}
