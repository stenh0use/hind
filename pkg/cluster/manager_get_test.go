package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

type stubProvider struct {
	inspectNetworkFn   func(ctx context.Context, name string) (*provider.NetworkInfo, error)
	inspectContainerFn func(ctx context.Context, name string) (*provider.ContainerInfo, error)
}

func (s *stubProvider) CreateContainer(ctx context.Context, cfg config.Node) (string, error) {
	return "", nil
}

func (s *stubProvider) StartContainer(ctx context.Context, name string) error {
	return nil
}

func (s *stubProvider) StopContainer(ctx context.Context, name string) error {
	return nil
}

func (s *stubProvider) DeleteContainer(ctx context.Context, name string) error {
	return nil
}

func (s *stubProvider) InspectContainer(ctx context.Context, name string) (*provider.ContainerInfo, error) {
	if s.inspectContainerFn != nil {
		return s.inspectContainerFn(ctx, name)
	}
	return nil, nil
}

func (s *stubProvider) ListContainers(ctx context.Context, filters []string) ([]provider.ContainerInfo, error) {
	return nil, nil
}

func (s *stubProvider) CreateNetwork(ctx context.Context, cfg config.Network) (string, error) {
	return "", nil
}

func (s *stubProvider) DeleteNetwork(ctx context.Context, name string) error {
	return nil
}

func (s *stubProvider) ListNetworks(ctx context.Context, filters []string) ([]provider.NetworkInfo, error) {
	return nil, nil
}

func (s *stubProvider) InspectNetwork(ctx context.Context, name string) (*provider.NetworkInfo, error) {
	if s.inspectNetworkFn != nil {
		return s.inspectNetworkFn(ctx, name)
	}
	return nil, nil
}

func TestManagerGet_NetworkNotFoundDoesNotPanic(t *testing.T) {
	t.Parallel()

	m := &Manager{
		provider: &stubProvider{
			inspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return nil, nil
			},
			inspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				return nil, nil
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
			Nodes: []config.Node{
				{Name: "hind.demo.consul.01"},
			},
		},
	}

	got, err := m.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil state")
	}
	if got.Network.Name != "" {
		t.Errorf("Get().Network.Name = %q, want empty string when network missing", got.Network.Name)
	}
	if len(got.Containers) != 0 {
		t.Errorf("Get().Containers len = %d, want 0", len(got.Containers))
	}
}

func TestManagerGet_ReturnsInspectNetworkError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("docker daemon unavailable")
	m := &Manager{
		provider: &stubProvider{
			inspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return nil, wantErr
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
		},
	}

	_, err := m.Get(context.Background())
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Get() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "failed to inspect network") {
		t.Fatalf("Get() error = %q, want context about network inspect", err)
	}
}

func TestManagerGet_ReturnsInspectContainerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("inspect container failed")
	m := &Manager{
		provider: &stubProvider{
			inspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return &provider.NetworkInfo{Name: name}, nil
			},
			inspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				return nil, wantErr
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
			Nodes:   []config.Node{{Name: "hind.demo.consul.01"}},
		},
	}

	_, err := m.Get(context.Background())
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Get() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "failed to inspect node 'hind.demo.consul.01'") {
		t.Fatalf("Get() error = %q, missing node context", err)
	}
}
