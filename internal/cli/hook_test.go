package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadHookInput(t *testing.T) {
	// Test with valid JSON input
	t.Run("valid JSON", func(t *testing.T) {
		input := HookInput{
			SessionID:      "test-session-123",
			TranscriptPath: "/path/to/transcript.jsonl",
			Cwd:            "/test/dir",
			PermissionMode: "default",
			HookEventName:  "PreToolUse",
			ToolName:       "ExitPlanMode",
		}

		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("failed to marshal test input: %v", err)
		}

		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}

		// Write test data and close
		go func() {
			w.Write(data)
			w.Close()
		}()

		// Temporarily replace stdin
		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		result, err := readHookInput()
		if err != nil {
			t.Fatalf("readHookInput failed: %v", err)
		}

		if result.SessionID != "test-session-123" {
			t.Errorf("expected session_id 'test-session-123', got '%s'", result.SessionID)
		}
		if result.ToolName != "ExitPlanMode" {
			t.Errorf("expected tool_name 'ExitPlanMode', got '%s'", result.ToolName)
		}
	})

	// Test with empty stdin (backward compatibility)
	t.Run("empty stdin", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		w.Close() // Close immediately to simulate empty stdin

		oldStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		result, err := readHookInput()
		if err != nil {
			t.Fatalf("readHookInput failed on empty stdin: %v", err)
		}

		if result.SessionID != "" {
			t.Errorf("expected empty session_id, got '%s'", result.SessionID)
		}
	})
}

func TestGetSessionMarkerPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	t.Run("with session ID", func(t *testing.T) {
		path := getSessionMarkerPath("test-session-123", "plan-saved")
		expected := filepath.Join(homeDir, ".jig", "sessions", "test-session-123", "plan-saved.marker")
		if path != expected {
			t.Errorf("expected path '%s', got '%s'", expected, path)
		}
	})

	t.Run("empty session ID falls back to legacy", func(t *testing.T) {
		path := getSessionMarkerPath("", "plan-saved")
		expected := filepath.Join(".jig", "plan-saved.marker")
		if path != expected {
			t.Errorf("expected path '%s', got '%s'", expected, path)
		}
	})
}

func TestFindClaudePlan(t *testing.T) {
	// Create a temporary plans directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	plansDir := filepath.Join(homeDir, ".claude", "plans")
	if _, err := os.Stat(plansDir); os.IsNotExist(err) {
		t.Skip("~/.claude/plans directory does not exist")
	}

	// Just test that the function doesn't crash and returns something reasonable
	result := findClaudePlan("test-session")
	// Result can be empty or a path, both are valid
	if result != "" && filepath.Ext(result) != ".md" {
		t.Errorf("expected empty string or .md file, got '%s'", result)
	}
}

func TestFindPlanFile(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	t.Run("no plan file", func(t *testing.T) {
		result := findPlanFile()
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("plan.md exists", func(t *testing.T) {
		err := os.WriteFile("plan.md", []byte("# Plan"), 0644)
		if err != nil {
			t.Fatalf("failed to create plan.md: %v", err)
		}
		defer os.Remove("plan.md")

		result := findPlanFile()
		if result != "plan.md" {
			t.Errorf("expected 'plan.md', got '%s'", result)
		}
	})

	t.Run("*-plan.md pattern", func(t *testing.T) {
		err := os.WriteFile("feature-plan.md", []byte("# Plan"), 0644)
		if err != nil {
			t.Fatalf("failed to create feature-plan.md: %v", err)
		}
		defer os.Remove("feature-plan.md")

		result := findPlanFile()
		if result != "feature-plan.md" {
			t.Errorf("expected 'feature-plan.md', got '%s'", result)
		}
	})
}

func TestCreateSessionMarker(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	sessionID := "test-session-" + time.Now().Format("20060102150405")
	sessionsDir := filepath.Join(homeDir, ".jig", "sessions", sessionID)
	defer os.RemoveAll(sessionsDir)

	err = createSessionMarker(sessionID, "test-marker")
	if err != nil {
		t.Fatalf("createSessionMarker failed: %v", err)
	}

	markerPath := filepath.Join(sessionsDir, "test-marker.marker")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Errorf("marker file was not created at '%s'", markerPath)
	}
}

func TestBuildExitPlanModePrompt(t *testing.T) {
	t.Run("with session ID", func(t *testing.T) {
		prompt := buildExitPlanModePrompt("/path/to/plan.md", "test-session-123")
		if !contains(prompt, "--session test-session-123") {
			t.Error("prompt should contain session flag when session ID is provided")
		}
		if !contains(prompt, "/path/to/plan.md") {
			t.Error("prompt should contain plan file path")
		}
	})

	t.Run("without session ID", func(t *testing.T) {
		prompt := buildExitPlanModePrompt("/path/to/plan.md", "")
		if contains(prompt, "--session") {
			t.Error("prompt should not contain session flag when session ID is empty")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestHookOutput(t *testing.T) {
	t.Run("JSON marshaling", func(t *testing.T) {
		output := HookOutput{
			PermissionDecision:       "deny",
			PermissionDecisionReason: "Test reason",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("failed to marshal HookOutput: %v", err)
		}

		// Verify it contains expected fields
		jsonStr := string(data)
		if !contains(jsonStr, `"permissionDecision":"deny"`) {
			t.Errorf("JSON should contain permissionDecision field, got: %s", jsonStr)
		}
		if !contains(jsonStr, `"permissionDecisionReason":"Test reason"`) {
			t.Errorf("JSON should contain permissionDecisionReason field, got: %s", jsonStr)
		}
	})

	t.Run("omitempty for empty fields", func(t *testing.T) {
		output := HookOutput{
			PermissionDecision: "allow",
		}

		data, err := json.Marshal(output)
		if err != nil {
			t.Fatalf("failed to marshal HookOutput: %v", err)
		}

		jsonStr := string(data)
		if contains(jsonStr, "permissionDecisionReason") {
			t.Errorf("JSON should omit empty permissionDecisionReason, got: %s", jsonStr)
		}
	})
}

func TestReadPlanSummary(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "jig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("short plan file", func(t *testing.T) {
		planContent := "# My Plan\n\nThis is a short plan."
		planFile := filepath.Join(tmpDir, "short-plan.md")
		err := os.WriteFile(planFile, []byte(planContent), 0644)
		if err != nil {
			t.Fatalf("failed to write plan file: %v", err)
		}

		summary := readPlanSummary(planFile)
		if summary != planContent {
			t.Errorf("expected full content for short plan, got: %s", summary)
		}
	})

	t.Run("long plan file truncated", func(t *testing.T) {
		// Create content longer than 500 chars
		planContent := "# My Plan\n\n"
		for i := 0; i < 100; i++ {
			planContent += "This is line number " + string(rune('0'+i%10)) + " of the plan content.\n"
		}

		planFile := filepath.Join(tmpDir, "long-plan.md")
		err := os.WriteFile(planFile, []byte(planContent), 0644)
		if err != nil {
			t.Fatalf("failed to write plan file: %v", err)
		}

		summary := readPlanSummary(planFile)
		if len(summary) > 510 { // 500 + "..."
			t.Errorf("expected truncated summary, got length: %d", len(summary))
		}
		if !contains(summary, "...") {
			t.Error("truncated summary should end with ...")
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		summary := readPlanSummary("/nonexistent/path/plan.md")
		if summary != "" {
			t.Errorf("expected empty string for non-existent file, got: %s", summary)
		}
	})
}
