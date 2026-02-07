package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charleslr/jig/internal/plan"
)

// PlanTableColumns returns the standard columns for displaying plans
func PlanTableColumns() []TableColumn {
	return []TableColumn{
		{Title: "ID", Width: 18},
		{Title: "Title", Width: 35},
		{Title: "Status", Width: 12},
		{Title: "Phases", Width: 8},
	}
}

// PlanToTableRow converts a plan to a table row
func PlanToTableRow(p *plan.Plan) TableRow {
	status := string(p.Status)
	if status == "" {
		status = "draft"
	}

	phases := fmt.Sprintf("%d", len(p.Phases))
	if len(p.Phases) > 0 {
		completed := len(p.GetCompletedPhases())
		phases = fmt.Sprintf("%d/%d", completed, len(p.Phases))
	}

	return TableRow{
		Cells: []string{
			p.ID,
			p.Title,
			status,
			phases,
		},
		Value: p,
	}
}

// PlansToTableRows converts a slice of plans to table rows
func PlansToTableRows(plans []*plan.Plan) []TableRow {
	rows := make([]TableRow, len(plans))
	for i, p := range plans {
		rows[i] = PlanToTableRow(p)
	}
	return rows
}

// RunPlanTable runs an interactive table for selecting a plan
// Returns the selected plan and whether a selection was made (false if cancelled)
func RunPlanTable(title string, plans []*plan.Plan) (*plan.Plan, bool, error) {
	if len(plans) == 0 {
		return nil, false, nil
	}

	columns := PlanTableColumns()
	rows := PlansToTableRows(plans)

	m := NewTable(title, columns, rows)
	result, err := runTableProgram(m)
	if err != nil {
		return nil, false, err
	}

	model := result.(TableModel)

	// Check if user cancelled (pressed q/esc)
	if model.WasCancelled() {
		return nil, false, nil
	}

	// Get the selected row
	row := model.SelectedRow()
	if row == nil || row.Value == nil {
		return nil, false, nil
	}

	selectedPlan, ok := row.Value.(*plan.Plan)
	if !ok {
		return nil, false, nil
	}

	return selectedPlan, true, nil
}

// runTableProgram is a helper to run the table tea program
func runTableProgram(m TableModel) (tea.Model, error) {
	p := tea.NewProgram(m)
	return p.Run()
}
