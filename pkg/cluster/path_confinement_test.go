package cluster

import (
	"strings"
	"testing"
)

func TestValidateClusterName(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		wantErr     bool
	}{
		{
			name:        "valid simple name",
			clusterName: "default",
			wantErr:     false,
		},
		{
			name:        "valid with punctuation",
			clusterName: "dev-cluster_01",
			wantErr:     false,
		},
		{
			name:        "empty name",
			clusterName: "",
			wantErr:     true,
		},
		{
			name:        "whitespace name",
			clusterName: "   ",
			wantErr:     true,
		},
		{
			name:        "unix traversal",
			clusterName: "../../etc",
			wantErr:     true,
		},
		{
			name:        "windows traversal",
			clusterName: "..\\..\\windows",
			wantErr:     true,
		},
		{
			name:        "absolute unix path",
			clusterName: "/tmp/escape",
			wantErr:     true,
		},
		{
			name:        "clean resolves up",
			clusterName: "../cluster",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClusterName(tt.clusterName)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateClusterName(%q) expected error, got nil", tt.clusterName)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateClusterName(%q) expected no error, got %v", tt.clusterName, err)
			}
		})
	}
}

func TestSetActiveCluster_RejectsTraversalName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := SetActiveCluster("../../etc")
	if err == nil {
		t.Fatal("expected error for traversal cluster name, got nil")
	}

	if !strings.Contains(err.Error(), "invalid cluster name") {
		t.Fatalf("expected invalid cluster name error, got %v", err)
	}
}
