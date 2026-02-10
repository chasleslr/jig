package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charleslr/jig/internal/plan"
	"github.com/charleslr/jig/internal/state"
)

// PlanTableColumns returns the standard columns for displaying plans
func PlanTableColumns() []TableColumn {
	return []TableColumn{
		{Title: "ID", Width: 18},
		{Title: "Title", Width: 40},
		{Title: "Status", Width: 12},
	}
}

// PlanToTableRow converts a plan to a table row
func PlanToTableRow(p *plan.Plan) TableRow {
	status := string(p.Status)
	if status == "" {
		status = "draft"
	}

	return TableRow{
		Cells: []string{
			p.ID,
			p.Title,
			status,
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

// CachedPlanTableColumns returns columns for displaying cached plans with sync status
func CachedPlanTableColumns() []TableColumn {
	return []TableColumn{
		{Title: "ID", Width: 18},
		{Title: "Title", Width: 36},
		{Title: "Status", Width: 12},
		{Title: "Synced", Width: 10},
	}
}

// CachedPlanToTableRow converts a cached plan to a table row with sync status
func CachedPlanToTableRow(cp *state.CachedPlan) TableRow {
	if cp == nil || cp.Plan == nil {
		return TableRow{}
	}

	status := string(cp.Plan.Status)
	if status == "" {
		status = "draft"
	}

	syncStatus := "-"
	if cp.Plan.IssueID != "" {
		if cp.NeedsSync() {
			syncStatus = "no"
		} else {
			syncStatus = "yes"
		}
	}

	return TableRow{
		Cells: []string{
			cp.Plan.ID,
			cp.Plan.Title,
			status,
			syncStatus,
		},
		Value: cp.Plan,
	}
}

// CachedPlansToTableRows converts a slice of cached plans to table rows
func CachedPlansToTableRows(plans []*state.CachedPlan) []TableRow {
	rows := make([]TableRow, len(plans))
	for i, cp := range plans {
		rows[i] = CachedPlanToTableRow(cp)
	}
	return rows
}

// RunCachedPlanTable runs an interactive table for selecting a cached plan
func RunCachedPlanTable(title string, plans []*state.CachedPlan) (*plan.Plan, bool, error) {
	if len(plans) == 0 {
		return nil, false, nil
	}

	columns := CachedPlanTableColumns()
	rows := CachedPlansToTableRows(plans)

	m := NewTable(title, columns, rows)
	result, err := runTableProgram(m)
	if err != nil {
		return nil, false, err
	}

	model := result.(TableModel)

	if model.WasCancelled() {
		return nil, false, nil
	}

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
