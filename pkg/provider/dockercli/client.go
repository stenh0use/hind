package dockercli

import (
	"context"
	"os/exec"

	"github.com/apex/log"
	"github.com/stenh0use/hind/pkg/provider"
)

const clientBin = "docker"

// Client provides an interface to the Docker API for cluster operations
type Client struct {
	logger *log.Logger
}

// New creates a new Docker client
func New(logger *log.Logger) provider.Client {
	return &Client{
		logger: logger,
	}
}

func baseClientCmd(ctx context.Context, arg ...string) *exec.Cmd {
	return exec.CommandContext(
		ctx,
		clientBin,
		arg...,
	)
}
