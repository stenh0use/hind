package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImageWriteFiles_WritesEmbeddedBuildContext(t *testing.T) {
	tests := []struct {
		name        string
		imageName   string
		expectFiles []string
	}{
		{
			name:      "consul build context",
			imageName: "consul",
			expectFiles: []string{
				"Dockerfile",
			},
		},
		{
			name:      "nomad build context",
			imageName: "nomad",
			expectFiles: []string{
				"Dockerfile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)

			imageFiles, err := New(tt.imageName)
			if err != nil {
				t.Fatalf("New(%q) error = %v", tt.imageName, err)
			}

			if err := imageFiles.WriteFiles(); err != nil {
				t.Fatalf("WriteFiles() error = %v", err)
			}

			for _, rel := range tt.expectFiles {
				fullPath := filepath.Join(imageFiles.BuildDir(), rel)
				if _, err := os.Stat(fullPath); err != nil {
					t.Fatalf("expected file %q to exist: %v", fullPath, err)
				}
			}
		})
	}
}
