package provider

import (
	"context"

	"github.com/stenh0use/hind/pkg/config"
)

type Client interface {
	// Container methods
	// Create and start a node
	CreateContainer(ctx context.Context, cfg ContainerSpec) (string, error)
	// Start a node if it is stopped
	StartContainer(ctx context.Context, name string) error
	// Stop a node if it is running
	StopContainer(ctx context.Context, name string) error
	// Force stop a node immediately
	KillContainer(ctx context.Context, name string) error
	// Delete a node
	DeleteContainer(ctx context.Context, name string) error
	// Inspect node state
	InspectContainer(ctx context.Context, name string) (*ContainerInfo, error)
	// List nodes
	ListContainers(ctx context.Context, filters []string) ([]ContainerInfo, error)

	// Image methods
	BuildImage(ctx context.Context, opts BuildImageOptions) (BuildImageResult, error)
	TagExists(ctx context.Context, name string, tag string) (bool, error)
	PullImage(ctx context.Context, name string, tag string) error

	// Network methods
	// Create a new docker network
	CreateNetwork(ctx context.Context, cfg config.Network) (string, error)
	// Delete a network
	DeleteNetwork(ctx context.Context, name string) error
	// List networks
	ListNetworks(ctx context.Context, filters []string) ([]NetworkInfo, error)
	// Inspect network state
	InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error)
}
