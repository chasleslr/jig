package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// MultiSelectModel is a multi-selection list component
type MultiSelectModel struct {
	title    string
	options  []SelectOption
	cursor   int
	selected map[int]bool
	quitting bool
}

// NewMultiSelect creates a new multi-selection list
func NewMultiSelect(title string, options []SelectOption) MultiSelectModel {
	return MultiSelectModel{
		title:    title,
		options:  options,
		selected: make(map[int]bool),
	}
}

// WithAllSelected initializes with all options selected
func (m MultiSelectModel) WithAllSelected() MultiSelectModel {
	for i := range m.options {
		m.selected[i] = true
	}
	return m
}

// Init implements tea.Model
func (m MultiSelectModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m MultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.selected = nil // Clear selection on cancel
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

		case " ":
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "a":
			// Select all
			for i := range m.options {
				m.selected[i] = true
			}

		case "n":
			// Select none
			m.selected = make(map[int]bool)

		case "enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model
func (m MultiSelectModel) View() string {
	if m.quitting {
		count := len(m.SelectedIndices())
		return fmt.Sprintf("Selected %d item(s)\n", count)
	}

	var b strings.Builder

	b.WriteString(m.title)
	b.WriteString("\n\n")

	for i, opt := range m.options {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = selectedStyle.Render("[x]")
		}

		label := opt.Label
		if m.cursor == i {
			label = selectedStyle.Render(label)
		} else {
			label = unselectedStyle.Render(label)
		}

		b.WriteString(cursor)
		b.WriteString(checkbox)
		b.WriteString(" ")
		b.WriteString(label)

		if opt.Description != "" && m.cursor == i {
			b.WriteString("\n      ")
			b.WriteString(unselectedStyle.Render(opt.Description))
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("↑/↓ move, space toggle, a all, n none, enter confirm, q cancel"))

	return b.String()
}

// SelectedIndices returns the indices of selected options
func (m MultiSelectModel) SelectedIndices() []int {
	var indices []int
	for i := range m.options {
		if m.selected[i] {
			indices = append(indices, i)
		}
	}
	return indices
}

// SelectedValues returns the values of selected options
func (m MultiSelectModel) SelectedValues() []string {
	var values []string
	for i, opt := range m.options {
		if m.selected[i] {
			values = append(values, opt.Value)
		}
	}
	return values
}

// RunMultiSelect runs the multi-select component and returns selected values
func RunMultiSelect(title string, options []SelectOption) ([]string, error) {
	m := NewMultiSelect(title, options)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(MultiSelectModel)
	return model.SelectedValues(), nil
}

// RunMultiSelectWithDefault runs the multi-select with all options pre-selected
func RunMultiSelectWithDefault(title string, options []SelectOption) ([]string, error) {
	m := NewMultiSelect(title, options).WithAllSelected()
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(MultiSelectModel)
	return model.SelectedValues(), nil
}
