package cluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	persistencefs "github.com/stenh0use/hind/pkg/cluster/persistence/fs"
)

func TestValidateName(t *testing.T) {
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
			err := domain.ValidateName(tt.clusterName)
			if tt.wantErr {
				assert.Error(t, err, "domain.ValidateName(%q) expected error", tt.clusterName)
			} else {
				assert.NoError(t, err, "domain.ValidateName(%q) expected no error", tt.clusterName)
			}
		})
	}
}

func TestSetActiveCluster_RejectsTraversalName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	repo, err := persistencefs.NewRepository()
	require.NoError(t, err)

	err = repo.SetActive(context.Background(), "../../etc")
	require.Error(t, err, "expected error for traversal cluster name")
	assert.Contains(t, err.Error(), "invalid cluster name")
}
