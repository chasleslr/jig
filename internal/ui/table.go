package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("240"))

	tableSelectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	tableNormalRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	tableCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))
)

// TableColumn defines a column in the table
type TableColumn struct {
	Title string
	Width int
}

// TableRow represents a row in the table
type TableRow struct {
	Cells []string
	Value interface{} // Optional value to return on selection
}

// TableModel is an interactive table component
type TableModel struct {
	title    string
	columns  []TableColumn
	rows     []TableRow
	cursor   int
	selected int
	quitting bool
	height   int
	scroll   int
}

// NewTable creates a new table model
func NewTable(title string, columns []TableColumn, rows []TableRow) TableModel {
	return TableModel{
		title:    title,
		columns:  columns,
		rows:     rows,
		selected: -1,
		height:   10, // Default visible rows
	}
}

// WithHeight sets the visible height (number of rows)
func (m TableModel) WithHeight(height int) TableModel {
	m.height = height
	return m
}

// Init implements tea.Model
func (m TableModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Scroll up if needed
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				// Scroll down if needed
				if m.cursor >= m.scroll+m.height {
					m.scroll = m.cursor - m.height + 1
				}
			}

		case "home", "g":
			m.cursor = 0
			m.scroll = 0

		case "end", "G":
			m.cursor = len(m.rows) - 1
			if m.cursor >= m.height {
				m.scroll = m.cursor - m.height + 1
			}

		case "enter", " ":
			if len(m.rows) > 0 {
				m.selected = m.cursor
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		// Adjust height based on terminal size, leaving room for header and help
		m.height = msg.Height - 6
		if m.height < 3 {
			m.height = 3
		}
	}

	return m, nil
}

// View implements tea.Model
func (m TableModel) View() string {
	if m.quitting {
		if m.selected >= 0 {
			// Show the selected row briefly
			row := m.rows[m.selected]
			return fmt.Sprintf("Selected: %s\n", strings.Join(row.Cells, " | "))
		}
		// User cancelled - return empty string
		return ""
	}

	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(selectedStyle.Render(m.title))
		b.WriteString("\n\n")
	}

	// Calculate total width and build header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Rows
	if len(m.rows) == 0 {
		b.WriteString(unselectedStyle.Render("  No items"))
		b.WriteString("\n")
	} else {
		// Calculate visible range
		end := m.scroll + m.height
		if end > len(m.rows) {
			end = len(m.rows)
		}

		for i := m.scroll; i < end; i++ {
			row := m.rows[i]
			b.WriteString(m.renderRow(row, i == m.cursor))
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(m.rows) > m.height {
			scrollInfo := fmt.Sprintf(" (%d-%d of %d)", m.scroll+1, end, len(m.rows))
			b.WriteString(unselectedStyle.Render(scrollInfo))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(unselectedStyle.Render("↑/↓ to move, enter to select, q to quit"))

	return b.String()
}

// renderHeader renders the table header
func (m TableModel) renderHeader() string {
	var parts []string
	for _, col := range m.columns {
		cell := truncateOrPad(col.Title, col.Width)
		parts = append(parts, cell)
	}
	return tableHeaderStyle.Render("  " + strings.Join(parts, " │ "))
}

// renderRow renders a single row
func (m TableModel) renderRow(row TableRow, selected bool) string {
	var parts []string
	for i, col := range m.columns {
		cell := ""
		if i < len(row.Cells) {
			cell = row.Cells[i]
		}
		cell = truncateOrPad(cell, col.Width)
		parts = append(parts, cell)
	}

	content := strings.Join(parts, " │ ")

	cursor := "  "
	if selected {
		cursor = tableCursorStyle.Render("> ")
		content = tableSelectedRowStyle.Render(content)
	} else {
		content = tableNormalRowStyle.Render(content)
	}

	return cursor + content
}

// truncateOrPad ensures a string is exactly the given width
func truncateOrPad(s string, width int) string {
	if len(s) > width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// Selected returns the selected row index (-1 if none)
func (m TableModel) Selected() int {
	return m.selected
}

// SelectedRow returns the selected row (nil if none)
func (m TableModel) SelectedRow() *TableRow {
	if m.selected < 0 || m.selected >= len(m.rows) {
		return nil
	}
	return &m.rows[m.selected]
}

// WasCancelled returns true if the user cancelled (pressed q/esc)
func (m TableModel) WasCancelled() bool {
	return m.quitting && m.selected < 0
}

// RunTable runs the table component and returns the selected row index
func RunTable(title string, columns []TableColumn, rows []TableRow) (int, error) {
	m := NewTable(title, columns, rows)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return -1, err
	}

	model := result.(TableModel)
	return model.Selected(), nil
}

// RunTableWithValue runs the table and returns the value from the selected row
func RunTableWithValue[T any](title string, columns []TableColumn, rows []TableRow) (T, bool, error) {
	var zero T

	m := NewTable(title, columns, rows)
	p := tea.NewProgram(m)

	result, err := p.Run()
	if err != nil {
		return zero, false, err
	}

	model := result.(TableModel)
	if row := model.SelectedRow(); row != nil && row.Value != nil {
		if val, ok := row.Value.(T); ok {
			return val, true, nil
		}
	}

	return zero, false, nil
}
