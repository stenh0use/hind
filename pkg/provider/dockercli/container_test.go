package dockercli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/provider"
)

func TestNormalizeContainerStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected provider.Status
	}{
		// Clean typed values (from docker inspect State.Status and docker ls State column)
		{name: "running passthrough", input: "running", expected: provider.Running},
		{name: "stopped passthrough", input: "stopped", expected: provider.Stopped},
		{name: "exited maps to stopped", input: "exited", expected: provider.Stopped},
		{name: "uppercase exited maps to stopped", input: "EXITED", expected: provider.Stopped},

		// Raw docker CLI Status column strings (human-readable)
		{name: "up with duration", input: "Up 2 hours", expected: provider.Running},
		{name: "up with minutes", input: "Up 3 minutes", expected: provider.Running},
		{name: "bare up", input: "up", expected: provider.Running},
		{name: "exited with code and age", input: "Exited (0) 3 days ago", expected: provider.Stopped},
		{name: "exited 137 with age", input: "Exited (137) 5 minutes ago", expected: provider.Stopped},
		{name: "status prefix exited paren", input: "status: (exited)", expected: provider.Stopped},
		{name: "Status prefix running paren", input: "Status: (running)", expected: provider.Running},

		// Unknown states map to Error
		{name: "restarting maps to error", input: "restarting", expected: provider.Error},
		{name: "created maps to error", input: "created", expected: provider.Error},
		{name: "paused maps to error", input: "paused", expected: provider.Error},
		{name: "empty string maps to error", input: "", expected: provider.Error},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, normalizeContainerStatus(tt.input))
		})
	}
}

func TestListContainers_PassesAllFlag(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "docker-args.txt")
	dockerBin := filepath.Join(tmpDir, "docker")
	// Capture args, emit empty output so JSON parse loop is a no-op.
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$DOCKER_ARGS_FILE\"\n"
	err := os.WriteFile(dockerBin, []byte(script), 0o755)
	require.NoError(t, err, "write fake docker")

	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err, "set PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	oldArgsFile := os.Getenv("DOCKER_ARGS_FILE")
	err = os.Setenv("DOCKER_ARGS_FILE", argsFile)
	require.NoError(t, err, "set DOCKER_ARGS_FILE")
	t.Cleanup(func() { _ = os.Setenv("DOCKER_ARGS_FILE", oldArgsFile) })

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	_, err = c.ListContainers(context.Background(), []string{"label=hind.cluster=demo"})
	require.NoError(t, err)

	out, err := os.ReadFile(argsFile)
	require.NoError(t, err, "read args")
	args := strings.Split(strings.TrimSpace(string(out)), "\n")

	var sawAll bool
	for _, a := range args {
		if a == "--all" || a == "-a" {
			sawAll = true
			break
		}
	}
	assert.True(t, sawAll, "expected docker container ls invocation to include --all; got args=%v", args)
}

func TestDeleteContainer_PassesForceFlag(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "docker-args.txt")
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$DOCKER_ARGS_FILE\"\n"
	err := os.WriteFile(dockerBin, []byte(script), 0o755)
	require.NoError(t, err, "write fake docker")

	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err, "set PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	oldArgsFile := os.Getenv("DOCKER_ARGS_FILE")
	err = os.Setenv("DOCKER_ARGS_FILE", argsFile)
	require.NoError(t, err, "set DOCKER_ARGS_FILE")
	t.Cleanup(func() { _ = os.Setenv("DOCKER_ARGS_FILE", oldArgsFile) })

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	err = c.DeleteContainer(context.Background(), "hind.test.consul.01")
	require.NoError(t, err)

	out, err := os.ReadFile(argsFile)
	require.NoError(t, err, "read args")
	args := strings.Split(strings.TrimSpace(string(out)), "\n")

	var sawForce bool
	for _, a := range args {
		if a == "--force" || a == "-f" {
			sawForce = true
			break
		}
	}
	assert.True(t, sawForce, "expected docker container rm invocation to include --force; got args=%v", args)
}

func TestCreateContainer_IncludesDockerStderrOnCreateFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\necho 'docker: Error response from daemon: invalid volume specification' 1>&2\nexit 125\n"
	err := os.WriteFile(dockerBin, []byte(script), 0o755)
	require.NoError(t, err, "failed to write fake docker binary")

	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err, "failed to set PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	spec := provider.ContainerSpec{
		Name:  "hind.test.consul.01",
		Image: "docker.io/stenh0use/hind.consul",
	}

	_, err = c.CreateContainer(context.Background(), spec)
	require.Error(t, err, "expected CreateContainer to fail")
	assert.Contains(t, err.Error(), "invalid volume specification")
}

func TestCreateContainer_UsesImageWhenSet(t *testing.T) {
	tmpDir := t.TempDir()
	argsFile := filepath.Join(tmpDir, "docker-args.txt")
	dockerBin := filepath.Join(tmpDir, "docker")
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$DOCKER_ARGS_FILE\"\nprintf 'container-id\n'\n"
	err := os.WriteFile(dockerBin, []byte(script), 0o755)
	require.NoError(t, err, "failed to write fake docker binary")

	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err, "failed to set PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})

	oldArgsFile := os.Getenv("DOCKER_ARGS_FILE")
	err = os.Setenv("DOCKER_ARGS_FILE", argsFile)
	require.NoError(t, err, "failed to set DOCKER_ARGS_FILE")
	t.Cleanup(func() {
		_ = os.Setenv("DOCKER_ARGS_FILE", oldArgsFile)
	})

	c := &Client{logger: &log.Logger{Handler: discard.New()}}
	spec := provider.ContainerSpec{
		Name:  "hind.test.consul.01",
		Image: "docker.io/stenh0use/hind.consul",
	}

	_, err = c.CreateContainer(context.Background(), spec)
	require.NoError(t, err)

	argsOut, err := os.ReadFile(argsFile)
	require.NoError(t, err, "failed to read docker args file")

	args := strings.Split(strings.TrimSpace(string(argsOut)), "\n")
	require.NotEmpty(t, args, "expected docker args, got none")

	last := args[len(args)-1]
	assert.Equal(t, spec.Image, last)
	assert.NotEqual(t, spec.Name, last, "image arg unexpectedly fell back to container name %q", spec.Name)
}
