package dockercli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/config"
)

func TestCreateContainer_UsesImageNameWhenTagAndDigestUnset(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "docker-args.txt")
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$DOCKER_ARGS_FILE\"\nprintf 'container-id\n'\n"
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

	oldArgsFile := os.Getenv("DOCKER_ARGS_FILE")
	if err := os.Setenv("DOCKER_ARGS_FILE", argsFile); err != nil {
		t.Fatalf("failed to set DOCKER_ARGS_FILE: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("DOCKER_ARGS_FILE", oldArgsFile)
	})

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	cfg := config.Node{
		Name: "hind.test.consul.01",
		Image: config.Image{
			Name: "docker.io/stenh0use/hind.consul",
		},
	}

	if _, err := c.CreateContainer(context.Background(), cfg); err != nil {
		t.Fatalf("CreateContainer() error = %v", err)
	}

	argsOut, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read docker args file: %v", err)
	}

	args := strings.Split(strings.TrimSpace(string(argsOut)), "\n")
	if len(args) == 0 {
		t.Fatal("expected docker args, got none")
	}

	last := args[len(args)-1]
	if last != cfg.Image.Name {
		t.Fatalf("image arg = %q, want %q", last, cfg.Image.Name)
	}
	if last == cfg.Name {
		t.Fatalf("image arg unexpectedly fell back to container name %q", cfg.Name)
	}
}
