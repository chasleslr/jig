package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinearConfig_ShouldSyncPlanOnSave(t *testing.T) {
	tests := []struct {
		name     string
		config   LinearConfig
		expected bool
	}{
		{
			name:     "nil SyncPlanOnSave defaults to true",
			config:   LinearConfig{SyncPlanOnSave: nil},
			expected: true,
		},
		{
			name:     "explicit true returns true",
			config:   LinearConfig{SyncPlanOnSave: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit false returns false",
			config:   LinearConfig{SyncPlanOnSave: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldSyncPlanOnSave()
			if got != tt.expected {
				t.Errorf("ShouldSyncPlanOnSave() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLinearConfig_GetPlanLabelName(t *testing.T) {
	tests := []struct {
		name     string
		config   LinearConfig
		expected string
	}{
		{
			name:     "empty string defaults to jig-plan",
			config:   LinearConfig{PlanLabelName: ""},
			expected: "jig-plan",
		},
		{
			name:     "custom label name is returned",
			config:   LinearConfig{PlanLabelName: "my-custom-label"},
			expected: "my-custom-label",
		},
		{
			name:     "label with special characters is preserved",
			config:   LinearConfig{PlanLabelName: "plan/v2"},
			expected: "plan/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetPlanLabelName()
			if got != tt.expected {
				t.Errorf("GetPlanLabelName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLinearConfig_ShouldCreateIssueOnSave(t *testing.T) {
	tests := []struct {
		name     string
		config   LinearConfig
		expected bool
	}{
		{
			name:     "nil CreateIssueOnSave defaults to true",
			config:   LinearConfig{CreateIssueOnSave: nil},
			expected: true,
		},
		{
			name:     "explicit true returns true",
			config:   LinearConfig{CreateIssueOnSave: boolPtr(true)},
			expected: true,
		},
		{
			name:     "explicit false returns false",
			config:   LinearConfig{CreateIssueOnSave: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ShouldCreateIssueOnSave()
			if got != tt.expected {
				t.Errorf("ShouldCreateIssueOnSave() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

func TestJigDir(t *testing.T) {
	t.Run("uses JIG_HOME when set", func(t *testing.T) {
		// Save original value
		orig := os.Getenv("JIG_HOME")
		defer func() {
			if orig != "" {
				os.Setenv("JIG_HOME", orig)
			} else {
				os.Unsetenv("JIG_HOME")
			}
		}()

		// Set custom JIG_HOME
		customDir := "/custom/jig/home"
		os.Setenv("JIG_HOME", customDir)

		dir, err := JigDir()
		if err != nil {
			t.Fatalf("JigDir() error = %v", err)
		}
		if dir != customDir {
			t.Errorf("JigDir() = %q, want %q", dir, customDir)
		}
	})

	t.Run("falls back to ~/.jig when JIG_HOME not set", func(t *testing.T) {
		// Save original value
		orig := os.Getenv("JIG_HOME")
		defer func() {
			if orig != "" {
				os.Setenv("JIG_HOME", orig)
			} else {
				os.Unsetenv("JIG_HOME")
			}
		}()

		// Unset JIG_HOME
		os.Unsetenv("JIG_HOME")

		dir, err := JigDir()
		if err != nil {
			t.Fatalf("JigDir() error = %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".jig")
		if dir != expected {
			t.Errorf("JigDir() = %q, want %q", dir, expected)
		}
	})
}
