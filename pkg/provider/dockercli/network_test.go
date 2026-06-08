package dockercli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListNetworks_ReturnsListSpecificErrorTextOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\nexit 2\n"
	err := os.WriteFile(dockerBin, []byte(script), 0o755)
	require.NoError(t, err, "failed to write fake docker binary")

	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err, "failed to set PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	_, err = c.ListNetworks(context.Background(), nil)
	require.Error(t, err, "ListNetworks() expected error, got nil")

	assert.Contains(t, err.Error(), "failed to list networks")
	assert.NotContains(t, err.Error(), "failed to inspect network")
}
