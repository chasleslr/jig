package skills

import (
	"testing"
)

func TestEmbeddedSkills(t *testing.T) {
	t.Run("plan.md is embedded", func(t *testing.T) {
		content, err := EmbeddedSkills.ReadFile("plan.md")
		if err != nil {
			t.Fatalf("failed to read plan.md: %v", err)
		}

		if len(content) == 0 {
			t.Error("plan.md should not be empty")
		}

		// Check for expected content
		contentStr := string(content)
		if len(contentStr) < 100 {
			t.Error("plan.md content seems too short")
		}

		// Should contain skill metadata
		if !contains(contentStr, "description:") {
			t.Error("plan.md should contain description metadata")
		}

		if !contains(contentStr, "/jig:plan") {
			t.Error("plan.md should mention /jig:plan")
		}
	})

	t.Run("implement.md is embedded", func(t *testing.T) {
		content, err := EmbeddedSkills.ReadFile("implement.md")
		if err != nil {
			t.Fatalf("failed to read implement.md: %v", err)
		}

		if len(content) == 0 {
			t.Error("implement.md should not be empty")
		}

		// Check for expected content
		contentStr := string(content)
		if len(contentStr) < 100 {
			t.Error("implement.md content seems too short")
		}

		// Should contain skill metadata
		if !contains(contentStr, "description:") {
			t.Error("implement.md should contain description metadata")
		}

		if !contains(contentStr, "/jig:implement") {
			t.Error("implement.md should mention /jig:implement")
		}
	})

	t.Run("non-existent file returns error", func(t *testing.T) {
		_, err := EmbeddedSkills.ReadFile("nonexistent.md")
		if err == nil {
			t.Error("expected error when reading non-existent file")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
