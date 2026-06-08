package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)

			err = imageFiles.WriteFiles()
			require.NoError(t, err)

			for _, rel := range tt.expectFiles {
				fullPath := filepath.Join(imageFiles.BuildDir(), rel)
				_, err := os.Stat(fullPath)
				require.NoError(t, err, "expected file %q to exist", fullPath)
			}
		})
	}
}
