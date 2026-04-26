package dockercli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
)

func TestListNetworks_ReturnsListSpecificErrorTextOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\nexit 2\n"
	if err := os.WriteFile(dockerBin, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake docker binary: %v", err)
	}

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tmpDir+":"+oldPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	_, err := c.ListNetworks(context.Background(), nil)
	if err == nil {
		t.Fatal("ListNetworks() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to list networks") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "failed to list networks")
	}
	if strings.Contains(err.Error(), "failed to inspect network") {
		t.Fatalf("error = %q, should not contain inspect wording", err.Error())
	}
}
