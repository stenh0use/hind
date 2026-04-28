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
	"github.com/stenh0use/hind/pkg/provider"
)

func TestNormalizeContainerStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "running passthrough", input: provider.Running.String(), expected: provider.Running.String()},
		{name: "stopped passthrough", input: provider.Stopped.String(), expected: provider.Stopped.String()},
		{name: "exited maps to stopped", input: "exited", expected: provider.Stopped.String()},
		{name: "uppercase exited maps to stopped", input: "EXITED", expected: provider.Stopped.String()},
		{name: "unknown passthrough", input: "restarting", expected: "restarting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeContainerStatus(tt.input); got != tt.expected {
				t.Fatalf("normalizeContainerStatus(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

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
