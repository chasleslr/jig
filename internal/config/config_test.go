package config

import "testing"

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

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
