package cli

import (
	"testing"
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
