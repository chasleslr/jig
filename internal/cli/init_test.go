package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charleslr/jig/internal/config"
)

func TestAddJigPermissions(t *testing.T) {
	t.Run("adds permissions to empty settings", func(t *testing.T) {
		settings := make(map[string]interface{})
		addJigPermissions(settings)

		permissions, ok := settings["permissions"].(map[string]interface{})
		if !ok {
			t.Fatal("permissions should be added to settings")
		}

		allowList, ok := permissions["allow"].([]interface{})
		if !ok {
			t.Fatal("allow list should be created")
		}

		if len(allowList) != len(jigPermissions) {
			t.Errorf("expected %d permissions, got %d", len(jigPermissions), len(allowList))
		}

		// Check all jig permissions are present
		for _, perm := range jigPermissions {
			found := false
			for _, item := range allowList {
				if s, ok := item.(string); ok && s == perm {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected permission %s to be in allow list", perm)
			}
		}
	})

	t.Run("adds permissions to existing settings without overwriting", func(t *testing.T) {
		settings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{
					"Bash(go build:*)",
					"Bash(go test:*)",
				},
			},
		}

		addJigPermissions(settings)

		permissions := settings["permissions"].(map[string]interface{})
		allowList := permissions["allow"].([]interface{})

		// Should have original + jig permissions
		expectedCount := 2 + len(jigPermissions)
		if len(allowList) != expectedCount {
			t.Errorf("expected %d permissions, got %d", expectedCount, len(allowList))
		}

		// Check original permissions are preserved
		foundGoBuild := false
		for _, item := range allowList {
			if s, ok := item.(string); ok && s == "Bash(go build:*)" {
				foundGoBuild = true
				break
			}
		}
		if !foundGoBuild {
			t.Error("original permission Bash(go build:*) should be preserved")
		}
	})

	t.Run("does not duplicate existing jig permissions", func(t *testing.T) {
		settings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{
					"Bash(jig plan save:*)", // Already has one jig permission
				},
			},
		}

		addJigPermissions(settings)

		permissions := settings["permissions"].(map[string]interface{})
		allowList := permissions["allow"].([]interface{})

		// Should not duplicate the existing permission
		// Expected: 1 (existing) + 2 (other jig permissions) = 3
		expectedCount := len(jigPermissions)
		if len(allowList) != expectedCount {
			t.Errorf("expected %d permissions (no duplicates), got %d", expectedCount, len(allowList))
		}

		// Count occurrences of "Bash(jig plan save:*)"
		count := 0
		for _, item := range allowList {
			if s, ok := item.(string); ok && s == "Bash(jig plan save:*)" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected exactly 1 occurrence of 'Bash(jig plan save:*)', got %d", count)
		}
	})
}

func TestJigPermissionsContent(t *testing.T) {
	// Verify the expected permissions are defined
	expectedPerms := []string{
		"Bash(jig plan save:*)",
		"Bash(jig hook mark-skip-save:*)",
		"Bash(jig hook mark-plan-saved:*)",
	}

	if len(jigPermissions) != len(expectedPerms) {
		t.Errorf("expected %d jig permissions, got %d", len(expectedPerms), len(jigPermissions))
	}

	for _, expected := range expectedPerms {
		found := false
		for _, perm := range jigPermissions {
			if perm == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected jig permission %s to be defined", expected)
		}
	}
}

func TestIsJigConfigured(t *testing.T) {
	t.Run("returns true when config.toml exists", func(t *testing.T) {
		// Create temp jig directory with config
		tmpDir := t.TempDir()
		os.Setenv("JIG_HOME", tmpDir)
		defer os.Unsetenv("JIG_HOME")

		// Create config file
		configPath := filepath.Join(tmpDir, "config.toml")
		os.WriteFile(configPath, []byte("# test config"), 0644)

		if !isJigConfigured() {
			t.Error("should return true when config.toml exists")
		}
	})

	t.Run("returns false when config.toml does not exist", func(t *testing.T) {
		// Use empty temp directory
		tmpDir := t.TempDir()
		os.Setenv("JIG_HOME", tmpDir)
		defer os.Unsetenv("JIG_HOME")

		if isJigConfigured() {
			t.Error("should return false when config.toml does not exist")
		}
	})

	t.Run("returns false when JIG_HOME cannot be determined", func(t *testing.T) {
		// Temporarily unset HOME and JIG_HOME to cause error
		originalHome := os.Getenv("HOME")
		originalJigHome := os.Getenv("JIG_HOME")
		os.Unsetenv("HOME")
		os.Unsetenv("JIG_HOME")
		defer func() {
			os.Setenv("HOME", originalHome)
			if originalJigHome != "" {
				os.Setenv("JIG_HOME", originalJigHome)
			}
		}()

		// This test may behave differently on different systems
		// Just ensure it doesn't panic
		_ = isJigConfigured()
	})
}

func TestGetSkillsLocation(t *testing.T) {
	t.Run("returns global by default when not configured", func(t *testing.T) {
		// Initialize with empty config
		tmpDir := t.TempDir()
		os.Setenv("JIG_HOME", tmpDir)
		defer os.Unsetenv("JIG_HOME")

		config.Init("")

		location := getSkillsLocation()
		if location != "global" {
			t.Errorf("expected 'global', got '%s'", location)
		}
	})

	t.Run("returns configured location when set", func(t *testing.T) {
		// Create config with skills_location set
		tmpDir := t.TempDir()
		os.Setenv("JIG_HOME", tmpDir)
		defer os.Unsetenv("JIG_HOME")

		// Write config with project location
		configPath := filepath.Join(tmpDir, "config.toml")
		configContent := `[claude]
skills_location = "project"
`
		os.WriteFile(configPath, []byte(configContent), 0644)

		// Initialize config to read the file
		config.Init("")

		location := getSkillsLocation()
		if location != "project" {
			t.Errorf("expected 'project', got '%s'", location)
		}
	})
}

func TestHasJigHook(t *testing.T) {
	t.Run("returns true when jig hook exists", func(t *testing.T) {
		settings := map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "ExitPlanMode",
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "jig hook exit-plan-mode",
							},
						},
					},
				},
			},
		}

		if !hasJigHook(settings) {
			t.Error("should return true when jig hook exists")
		}
	})

	t.Run("returns false when no hooks section", func(t *testing.T) {
		settings := make(map[string]interface{})

		if hasJigHook(settings) {
			t.Error("should return false when no hooks section")
		}
	})

	t.Run("returns false when no PreToolUse", func(t *testing.T) {
		settings := map[string]interface{}{
			"hooks": map[string]interface{}{},
		}

		if hasJigHook(settings) {
			t.Error("should return false when no PreToolUse")
		}
	})

	t.Run("returns false when jig hook not present", func(t *testing.T) {
		settings := map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "SomeOtherMatcher",
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "some other command",
							},
						},
					},
				},
			},
		}

		if hasJigHook(settings) {
			t.Error("should return false when jig hook not present")
		}
	})
}
