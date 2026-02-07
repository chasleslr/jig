package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModel is a yes/no confirmation component
type ConfirmModel struct {
	question  string
	confirmed bool
	answered  bool
	yesLabel  string
	noLabel   string
	cursor    int // 0 = yes, 1 = no
}

// NewConfirm creates a new confirmation dialog
func NewConfirm(question string) ConfirmModel {
	return ConfirmModel{
		question: question,
		yesLabel: "Yes",
		noLabel:  "No",
		cursor:   1, // Default to No for safety
	}
}

// WithLabels sets custom labels for Yes/No
func (m ConfirmModel) WithLabels(yes, no string) ConfirmModel {
	m.yesLabel = yes
	m.noLabel = no
	return m
}

// WithDefault sets the default selection (true = yes, false = no)
func (m ConfirmModel) WithDefault(defaultYes bool) ConfirmModel {
	if defaultYes {
		m.cursor = 0
	} else {
		m.cursor = 1
	}
	return m
}

// Init implements tea.Model
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.answered = true
			m.confirmed = false
			return m, tea.Quit

		case "y", "Y":
			m.answered = true
			m.confirmed = true
			return m, tea.Quit

		case "n", "N":
			m.answered = true
			m.confirmed = false
			return m, tea.Quit

		case "left", "h":
			m.cursor = 0

		case "right", "l":
			m.cursor = 1

		case "tab":
			m.cursor = (m.cursor + 1) % 2

		case "enter", " ":
			m.answered = true
			m.confirmed = (m.cursor == 0)
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model
func (m ConfirmModel) View() string {
	if m.answered {
		if m.confirmed {
			return fmt.Sprintf("%s %s\n", m.question, selectedStyle.Render(m.yesLabel))
		}
		return fmt.Sprintf("%s %s\n", m.question, unselectedStyle.Render(m.noLabel))
	}

	var b strings.Builder

	b.WriteString(m.question)
	b.WriteString(" ")

	yesStyle := unselectedStyle
	noStyle := unselectedStyle

	if m.cursor == 0 {
		yesStyle = selectedStyle
	} else {
		noStyle = selectedStyle
	}

	b.WriteString("[")
	b.WriteString(yesStyle.Render(m.yesLabel))
	b.WriteString("] [")
	b.WriteString(noStyle.Render(m.noLabel))
	b.WriteString("]")

	return b.String()
}

// Confirmed returns whether the user confirmed
func (m ConfirmModel) Confirmed() bool {
	return m.confirmed
}

// RunConfirm runs the confirm component and returns the result
func RunConfirm(question string) (bool, error) {
	m := NewConfirm(question)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return false, err
	}

	model := result.(ConfirmModel)
	return model.Confirmed(), nil
}

// RunConfirmWithDefault runs the confirm with a default value
func RunConfirmWithDefault(question string, defaultYes bool) (bool, error) {
	m := NewConfirm(question).WithDefault(defaultYes)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return false, err
	}

	model := result.(ConfirmModel)
	return model.Confirmed(), nil
}

// ConfirmStyle defines styling for the confirm dialog
var ConfirmStyle = struct {
	Question lipgloss.Style
	Yes      lipgloss.Style
	No       lipgloss.Style
}{
	Question: lipgloss.NewStyle().Bold(true),
	Yes:      lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
	No:       lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
}
