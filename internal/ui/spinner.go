package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel is a simple spinner component
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	err      error
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SpinnerModel{
		spinner: s,
		message: message,
	}
}

// Init implements tea.Model
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case SpinnerDoneMsg:
		m.quitting = true
		return m, tea.Quit

	case SpinnerErrorMsg:
		m.err = msg.Err
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model
func (m SpinnerModel) View() string {
	if m.quitting {
		if m.err != nil {
			return fmt.Sprintf("✗ %s: %v\n", m.message, m.err)
		}
		return fmt.Sprintf("✓ %s\n", m.message)
	}
	return fmt.Sprintf("%s %s\n", m.spinner.View(), m.message)
}

// SpinnerDoneMsg signals the spinner is done
type SpinnerDoneMsg struct{}

// SpinnerErrorMsg signals an error occurred
type SpinnerErrorMsg struct {
	Err error
}

// RunWithSpinner runs a function while showing a spinner
func RunWithSpinner(message string, fn func() error) error {
	p := tea.NewProgram(NewSpinner(message))

	// Run the function in a goroutine
	errChan := make(chan error, 1)
	go func() {
		err := fn()
		if err != nil {
			p.Send(SpinnerErrorMsg{Err: err})
		} else {
			p.Send(SpinnerDoneMsg{})
		}
		errChan <- err
	}()

	// Run the spinner
	if _, err := p.Run(); err != nil {
		return err
	}

	// Return the function's error
	return <-errChan
}

// SimpleSpinner shows a spinner for a duration (for demos)
func SimpleSpinner(message string, duration time.Duration) {
	p := tea.NewProgram(NewSpinner(message))

	go func() {
		time.Sleep(duration)
		p.Send(SpinnerDoneMsg{})
	}()

	p.Run()
}
