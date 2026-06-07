package mock

import (
	"context"

	"github.com/stenh0use/hind/pkg/provider"
)

// ClientStub is a stub implementation of provider.Client for testing.
type ClientStub struct {
	CreateContainerFn  func(context.Context, provider.ContainerSpec) (string, error)
	StartContainerFn   func(context.Context, string) error
	StopContainerFn    func(context.Context, string) error
	KillContainerFn    func(context.Context, string) error
	DeleteContainerFn  func(context.Context, string) error
	InspectContainerFn func(context.Context, string) (*provider.ContainerInfo, error)
	ListContainersFn   func(context.Context, []string) ([]provider.ContainerInfo, error)
	BuildImageFn       func(context.Context, provider.BuildImageOptions) (provider.BuildImageResult, error)
	TagExistsFn        func(context.Context, string, string) (bool, error)
	PullImageFn        func(context.Context, string, string) error
	CreateNetworkFn    func(context.Context, provider.NetworkSpec) (string, error)
	DeleteNetworkFn    func(context.Context, string) error
	ListNetworksFn     func(context.Context, []string) ([]provider.NetworkInfo, error)
	InspectNetworkFn   func(context.Context, string) (*provider.NetworkInfo, error)
}

func (c *ClientStub) CreateContainer(ctx context.Context, cfg provider.ContainerSpec) (string, error) {
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

func (c *ClientStub) KillContainer(ctx context.Context, name string) error {
	if c.KillContainerFn != nil {
		return c.KillContainerFn(ctx, name)
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

// BuildImage returns provider.BuildImageResult{} when BuildImageFn is nil.
// That zero-value result is intentionally invalid per the B-013 build-image contract
// (Digest/ImageRef must be non-empty), so tests that assert on result fields must
// provide BuildImageFn explicitly.
func (c *ClientStub) BuildImage(ctx context.Context, opts provider.BuildImageOptions) (provider.BuildImageResult, error) {
	if c.BuildImageFn != nil {
		return c.BuildImageFn(ctx, opts)
	}
	return provider.BuildImageResult{}, nil
}

func (c *ClientStub) TagExists(ctx context.Context, name string, tag string) (bool, error) {
	if c.TagExistsFn != nil {
		return c.TagExistsFn(ctx, name, tag)
	}
	return false, nil
}

func (c *ClientStub) PullImage(ctx context.Context, name string, tag string) error {
	if c.PullImageFn != nil {
		return c.PullImageFn(ctx, name, tag)
	}
	return nil
}

func (c *ClientStub) CreateNetwork(ctx context.Context, cfg provider.NetworkSpec) (string, error) {
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
