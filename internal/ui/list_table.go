package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListAction represents an action taken on a list item
type ListAction string

const (
	ListActionSelect   ListAction = "select"
	ListActionOpenPR   ListAction = "open_pr"
	ListActionCheckout ListAction = "checkout"
	ListActionDetails  ListAction = "details"
	ListActionQuit     ListAction = "quit"
)

// ListItem represents an item in the list
type ListItem struct {
	IssueID    string
	Title      string
	Branch     string
	Path       string
	PlanStatus string
	PRNumber   int
	PRState    string
	PRURL      string
	LastActive time.Time
	Exists     bool
}

// ListResult is returned when an action is taken
type ListResult struct {
	Action ListAction
	Item   *ListItem
}

// listTableModel is the Bubble Tea model for the list table
type listTableModel struct {
	title    string
	items    []ListItem
	cursor   int
	height   int
	scroll   int
	result   *ListResult
	quitting bool
}

// listTableColumns defines the column widths
var listTableColumns = []TableColumn{
	{Title: "ISSUE", Width: 15},
	{Title: "TITLE", Width: 35},
	{Title: "STATUS", Width: 12},
	{Title: "PR", Width: 10},
}

// helpBarStyle for the help bar at bottom
var helpBarStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

// helpKeyStyle for highlighting keys in help bar
var helpKeyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205")).
	Bold(true)

// newListTable creates a new list table model
func newListTable(title string, items []ListItem) listTableModel {
	return listTableModel{
		title:  title,
		items:  items,
		height: 10,
	}
}

// Init implements tea.Model
func (m listTableModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m listTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.result = &ListResult{Action: ListActionQuit}
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				if m.cursor >= m.scroll+m.height {
					m.scroll = m.cursor - m.height + 1
				}
			}

		case "home", "g":
			m.cursor = 0
			m.scroll = 0

		case "end", "G":
			m.cursor = len(m.items) - 1
			if m.cursor >= m.height {
				m.scroll = m.cursor - m.height + 1
			}

		case "enter":
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				m.result = &ListResult{
					Action: ListActionSelect,
					Item:   &item,
				}
				m.quitting = true
				return m, tea.Quit
			}

		case "o":
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				m.result = &ListResult{
					Action: ListActionOpenPR,
					Item:   &item,
				}
				m.quitting = true
				return m, tea.Quit
			}

		case "c":
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				m.result = &ListResult{
					Action: ListActionCheckout,
					Item:   &item,
				}
				m.quitting = true
				return m, tea.Quit
			}

		case "d":
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				m.result = &ListResult{
					Action: ListActionDetails,
					Item:   &item,
				}
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.height = max(msg.Height-6, 3)
	}

	return m, nil
}

// View implements tea.Model
func (m listTableModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(selectedStyle.Render(m.title))
		b.WriteString("\n\n")
	}

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Rows
	if len(m.items) == 0 {
		b.WriteString(unselectedStyle.Render("  No items"))
		b.WriteString("\n")
	} else {
		end := min(m.scroll+m.height, len(m.items))

		for i := m.scroll; i < end; i++ {
			b.WriteString(m.renderRow(i, i == m.cursor))
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(m.items) > m.height {
			scrollInfo := fmt.Sprintf(" (%d-%d of %d)", m.scroll+1, end, len(m.items))
			b.WriteString(unselectedStyle.Render(scrollInfo))
			b.WriteString("\n")
		}
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderHeader renders the table header
func (m listTableModel) renderHeader() string {
	var parts []string
	for _, col := range listTableColumns {
		cell := truncateOrPad(col.Title, col.Width)
		parts = append(parts, cell)
	}
	return tableHeaderStyle.Render("  " + strings.Join(parts, " │ "))
}

// renderRow renders a single row
func (m listTableModel) renderRow(idx int, selected bool) string {
	item := m.items[idx]

	// Build cells
	title := item.Title
	if len(title) > 33 {
		title = title[:30] + "..."
	}
	if title == "" {
		title = "-"
	}

	status := item.PlanStatus
	if status == "" {
		status = "-"
	}

	pr := "-"
	if item.PRNumber > 0 {
		pr = fmt.Sprintf("#%d", item.PRNumber)
	}

	cells := []string{
		truncateOrPad(item.IssueID, listTableColumns[0].Width),
		truncateOrPad(title, listTableColumns[1].Width),
		truncateOrPad(status, listTableColumns[2].Width),
		truncateOrPad(pr, listTableColumns[3].Width),
	}

	content := strings.Join(cells, " │ ")

	cursor := "  "
	if selected {
		cursor = tableCursorStyle.Render("> ")
		content = tableSelectedRowStyle.Render(content)
	} else {
		content = tableNormalRowStyle.Render(content)
	}

	return cursor + content
}

// renderHelpBar renders the help bar at bottom
func (m listTableModel) renderHelpBar() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"o", "open PR"},
		{"c", "checkout"},
		{"d", "details"},
		{"enter", "select"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := helpKeyStyle.Render(k.key) + helpBarStyle.Render(":"+k.desc)
		parts = append(parts, part)
	}

	return strings.Join(parts, helpBarStyle.Render("  "))
}

// RunListTable runs the list table component and returns the result
func RunListTable(title string, items []ListItem) (*ListResult, error) {
	if len(items) == 0 {
		return &ListResult{Action: ListActionQuit}, nil
	}

	m := newListTable(title, items)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(listTableModel)
	if model.result != nil {
		return model.result, nil
	}

	return &ListResult{Action: ListActionQuit}, nil
}
