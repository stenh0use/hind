package dockercli

import (
	"context"
	"io"
	"os/exec"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/provider"
)

const clientBin = "docker"

// CommandExecutor abstracts command execution for Docker operations.
// It allows tests to inject a fake executor without spawning real processes.
type CommandExecutor interface {
	Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error
	Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

type osCommandExecutor struct{}

func (osCommandExecutor) Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (osCommandExecutor) Output(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

// Client provides an interface to the Docker API for cluster operations.
type Client struct {
	logger   *log.Logger
	executor CommandExecutor
}

// New creates a new Docker client.
// It does not perform a buildx availability check; that happens inside BuildImage at call time.
func New(logger *log.Logger) provider.Client {
	return &Client{
		logger:   logger,
		executor: osCommandExecutor{},
	}
}

// newWithExecutor creates a Client with a custom executor for testing.
func newWithExecutor(logger *log.Logger, exec CommandExecutor) *Client {
	return &Client{
		logger:   logger,
		executor: exec,
	}
}

func baseClientCmd(ctx context.Context, arg ...string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		clientBin,
		arg...,
	)
}
