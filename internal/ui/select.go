package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
)

// SelectOption represents an option in a select list
type SelectOption struct {
	Label       string
	Value       string
	Description string
}

// SelectModel is a selection list component
type SelectModel struct {
	title    string
	options  []SelectOption
	cursor   int
	selected int
	quitting bool
}

// NewSelect creates a new selection list
func NewSelect(title string, options []SelectOption) SelectModel {
	return SelectModel{
		title:    title,
		options:  options,
		selected: -1,
	}
}

// Init implements tea.Model
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.cursor
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model
func (m SelectModel) View() string {
	if m.quitting && m.selected >= 0 {
		return fmt.Sprintf("Selected: %s\n", m.options[m.selected].Label)
	}

	var b strings.Builder

	b.WriteString(m.title)
	b.WriteString("\n\n")

	for i, opt := range m.options {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		label := opt.Label
		if m.cursor == i {
			label = selectedStyle.Render(label)
		} else {
			label = unselectedStyle.Render(label)
		}

		b.WriteString(cursor)
		b.WriteString(label)

		if opt.Description != "" && m.cursor == i {
			b.WriteString("\n    ")
			b.WriteString(unselectedStyle.Render(opt.Description))
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("↑/↓ to move, enter to select, q to quit"))

	return b.String()
}

// Selected returns the selected option index (-1 if none)
func (m SelectModel) Selected() int {
	return m.selected
}

// SelectedOption returns the selected option (nil if none)
func (m SelectModel) SelectedOption() *SelectOption {
	if m.selected < 0 || m.selected >= len(m.options) {
		return nil
	}
	return &m.options[m.selected]
}

// RunSelect runs the select component and returns the selected value
func RunSelect(title string, options []SelectOption) (string, error) {
	m := NewSelect(title, options)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return "", err
	}

	model := result.(SelectModel)
	if opt := model.SelectedOption(); opt != nil {
		return opt.Value, nil
	}

	return "", nil
}

// RunSelectIndex runs the select component and returns the selected index
func RunSelectIndex(title string, options []SelectOption) (int, error) {
	m := NewSelect(title, options)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return -1, err
	}

	model := result.(SelectModel)
	return model.Selected(), nil
}
