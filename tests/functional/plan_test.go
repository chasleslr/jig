package functional

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charleslr/jig/tests/functional/testenv"
)

func TestPlanSave(t *testing.T) {
	tests := []struct {
		name        string
		planContent string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid plan from stdin",
			planContent: testenv.ValidPlan("TEST-001", "Test Plan"),
			expectError: false,
		},
		{
			name:        "minimal valid plan",
			planContent: testenv.MinimalPlan("TEST-002"),
			expectError: false,
		},
		{
			name:        "invalid plan missing frontmatter",
			planContent: testenv.InvalidPlan(),
			expectError: true,
			errorMsg:    "invalid plan format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testenv.New(t)
			defer env.Cleanup()

			result := env.RunJigWithStdin(tt.planContent, "plan", "save", "--no-sync")

			if tt.expectError {
				env.AssertFailure(result)
				if tt.errorMsg != "" {
					env.AssertOutputContains(result, tt.errorMsg)
				}
			} else {
				env.AssertSuccess(result)
			}
		})
	}
}

func TestPlanSaveFromFile(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Write a plan file
	planContent := testenv.ValidPlan("TEST-FILE", "Plan from File")
	env.WriteFile("my-plan.md", planContent)

	// Save from file
	result := env.RunJig("plan", "save", "--no-sync", "my-plan.md")
	env.AssertSuccess(result)

	// Verify the plan was saved
	listResult := env.RunJig("plan", "list")
	env.AssertSuccess(listResult)
	env.AssertStdoutContains(listResult, "TEST-FILE")
}

func TestPlanList(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Initially should show no plans (or empty state)
	result := env.RunJig("plan", "list")
	env.AssertSuccess(result)

	// Save multiple plans
	plans := []struct {
		id    string
		title string
	}{
		{"LIST-001", "First Plan"},
		{"LIST-002", "Second Plan"},
		{"LIST-003", "Third Plan"},
	}

	for _, p := range plans {
		content := testenv.ValidPlan(p.id, p.title)
		saveResult := env.RunJigWithStdin(content, "plan", "save", "--no-sync")
		env.AssertSuccess(saveResult)
	}

	// List plans and verify all are present
	listResult := env.RunJig("plan", "list")
	env.AssertSuccess(listResult)

	for _, p := range plans {
		env.AssertStdoutContains(listResult, p.id)
	}
}

func TestPlanShow(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Save a plan
	planContent := testenv.ValidPlan("SHOW-001", "Plan to Show")
	saveResult := env.RunJigWithStdin(planContent, "plan", "save", "--no-sync")
	env.AssertSuccess(saveResult)

	// Show the plan with --raw flag (non-interactive)
	showResult := env.RunJig("plan", "show", "SHOW-001", "--raw")
	env.AssertSuccess(showResult)

	// Verify content is present
	env.AssertStdoutContains(showResult, "Plan to Show")
	env.AssertStdoutContains(showResult, "Problem Statement")
	env.AssertStdoutContains(showResult, "Proposed Solution")
}

func TestPlanShowNonExistent(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Try to show a non-existent plan
	result := env.RunJig("plan", "show", "NONEXISTENT", "--raw")
	env.AssertFailure(result)
	env.AssertOutputContains(result, "not found")
}

func TestPlanSaveEmptyInput(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Try to save empty input
	result := env.RunJigWithStdin("", "plan", "save", "--no-sync")
	env.AssertFailure(result)
}

func TestPlanCacheIsolation(t *testing.T) {
	// Create two separate environments
	env1 := testenv.New(t)
	defer env1.Cleanup()

	env2 := testenv.New(t)
	defer env2.Cleanup()

	// Save a plan in env1
	plan1 := testenv.ValidPlan("ISO-001", "Isolated Plan 1")
	result1 := env1.RunJigWithStdin(plan1, "plan", "save", "--no-sync")
	env1.AssertSuccess(result1)

	// Save a different plan in env2
	plan2 := testenv.ValidPlan("ISO-002", "Isolated Plan 2")
	result2 := env2.RunJigWithStdin(plan2, "plan", "save", "--no-sync")
	env2.AssertSuccess(result2)

	// Verify env1 only has its plan
	list1 := env1.RunJig("plan", "list")
	env1.AssertSuccess(list1)
	if !strings.Contains(list1.Stdout, "ISO-001") {
		t.Error("env1 should have ISO-001")
	}
	if strings.Contains(list1.Stdout, "ISO-002") {
		t.Error("env1 should NOT have ISO-002")
	}

	// Verify env2 only has its plan
	list2 := env2.RunJig("plan", "list")
	env2.AssertSuccess(list2)
	if !strings.Contains(list2.Stdout, "ISO-002") {
		t.Error("env2 should have ISO-002")
	}
	if strings.Contains(list2.Stdout, "ISO-001") {
		t.Error("env2 should NOT have ISO-001")
	}
}

func TestPlanSaveCreatesCache(t *testing.T) {
	env := testenv.New(t)
	defer env.Cleanup()

	// Verify cache directory doesn't exist initially
	cacheDir := env.CacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		t.Fatal("cache directory should not exist initially")
	}

	// Save a plan
	plan := testenv.ValidPlan("CACHE-001", "Cache Test")
	result := env.RunJigWithStdin(plan, "plan", "save", "--no-sync")
	env.AssertSuccess(result)

	// Verify cache directory now exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("cache directory should exist after saving plan")
	}

	// Verify plan file exists in cache
	planFile := filepath.Join(cacheDir, "plans", "CACHE-001.md")
	if _, err := os.Stat(planFile); os.IsNotExist(err) {
		t.Errorf("plan file should exist at %s", planFile)
	}
}
