package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagerRejectsTraversalAndAbsolutePaths(t *testing.T) {
	tests := []struct {
		name    string
		op      func(*Manager) error
		wantErr bool
	}{
		{
			name: "write rejects traversal",
			op: func(m *Manager) error {
				return m.WriteFile("../../escape.txt", []byte("x"))
			},
			wantErr: true,
		},
		{
			name: "read rejects traversal",
			op: func(m *Manager) error {
				_, err := m.ReadFile("../cluster.json")
				return err
			},
			wantErr: true,
		},
		{
			name: "ensure dir rejects traversal",
			op: func(m *Manager) error {
				return m.EnsureDir("../../cluster")
			},
			wantErr: true,
		},
		{
			name: "remove dir rejects traversal",
			op: func(m *Manager) error {
				return m.RemoveDir("../cluster")
			},
			wantErr: true,
		},
		{
			name: "write rejects absolute",
			op: func(m *Manager) error {
				return m.WriteFile("/tmp/escape.txt", []byte("x"))
			},
			wantErr: true,
		},
		{
			name: "valid nested relative path",
			op: func(m *Manager) error {
				return m.WriteFile("cluster/default/cluster.json", []byte("{}"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			m, err := New(root)
			require.NoError(t, err)

			err = tt.op(m)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
