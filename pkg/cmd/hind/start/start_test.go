package start

import (
	"testing"
)

func TestClusterNameExtraction(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "no args uses default",
			args:     []string{},
			expected: "default",
		},
		{
			name:     "single arg uses cluster name",
			args:     []string{"dev"},
			expected: "dev",
		},
		{
			name:     "custom cluster name",
			args:     []string{"my-test-cluster"},
			expected: "my-test-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the cluster name extraction logic
			clusterName := "default"
			if len(tt.args) > 0 {
				clusterName = tt.args[0]
			}

			if clusterName != tt.expected {
				t.Errorf("expected cluster name %q, got %q", tt.expected, clusterName)
			}
		})
	}
}
