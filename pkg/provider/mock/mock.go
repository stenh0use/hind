package mock

import (
	"context"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

// ClientStub is a stub implementation of provider.Client for testing.
type ClientStub struct {
	CreateContainerFn  func(context.Context, config.Node) (string, error)
	StartContainerFn   func(context.Context, string) error
	StopContainerFn    func(context.Context, string) error
	DeleteContainerFn  func(context.Context, string) error
	InspectContainerFn func(context.Context, string) (*provider.ContainerInfo, error)
	ListContainersFn   func(context.Context, []string) ([]provider.ContainerInfo, error)
	CreateNetworkFn    func(context.Context, config.Network) (string, error)
	DeleteNetworkFn    func(context.Context, string) error
	ListNetworksFn     func(context.Context, []string) ([]provider.NetworkInfo, error)
	InspectNetworkFn   func(context.Context, string) (*provider.NetworkInfo, error)
}

func (c *ClientStub) CreateContainer(ctx context.Context, cfg config.Node) (string, error) {
	if c.CreateContainerFn != nil {
		return c.CreateContainerFn(ctx, cfg)
	}
	return "", nil
}

func (c *ClientStub) StartContainer(ctx context.Context, name string) error {
	if c.StartContainerFn != nil {
		return c.StartContainerFn(ctx, name)
	}
	return nil
}

func (c *ClientStub) StopContainer(ctx context.Context, name string) error {
	if c.StopContainerFn != nil {
		return c.StopContainerFn(ctx, name)
	}
	return nil
}

func (c *ClientStub) DeleteContainer(ctx context.Context, name string) error {
	if c.DeleteContainerFn != nil {
		return c.DeleteContainerFn(ctx, name)
	}
	return nil
}

func (c *ClientStub) InspectContainer(ctx context.Context, name string) (*provider.ContainerInfo, error) {
	if c.InspectContainerFn != nil {
		return c.InspectContainerFn(ctx, name)
	}
	return nil, nil
}

func (c *ClientStub) ListContainers(ctx context.Context, filters []string) ([]provider.ContainerInfo, error) {
	if c.ListContainersFn != nil {
		return c.ListContainersFn(ctx, filters)
	}
	return nil, nil
}

func (c *ClientStub) CreateNetwork(ctx context.Context, cfg config.Network) (string, error) {
	if c.CreateNetworkFn != nil {
		return c.CreateNetworkFn(ctx, cfg)
	}
	return "", nil
}

func (c *ClientStub) DeleteNetwork(ctx context.Context, name string) error {
	if c.DeleteNetworkFn != nil {
		return c.DeleteNetworkFn(ctx, name)
	}
	return nil
}

func (c *ClientStub) ListNetworks(ctx context.Context, filters []string) ([]provider.NetworkInfo, error) {
	if c.ListNetworksFn != nil {
		return c.ListNetworksFn(ctx, filters)
	}
	return nil, nil
}

func (c *ClientStub) InspectNetwork(ctx context.Context, name string) (*provider.NetworkInfo, error) {
	if c.InspectNetworkFn != nil {
		return c.InspectNetworkFn(ctx, name)
	}
	return nil, nil
}

var _ provider.Client = (*ClientStub)(nil)
