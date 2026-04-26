package file

import "testing"

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
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			err = tt.op(m)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestManagerGetPathRejectsEscape(t *testing.T) {
	root := t.TempDir()
	m, err := New(root)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if got := m.GetPath("../../escape"); got != "" {
		t.Fatalf("expected empty path for traversal input, got %q", got)
	}
}
