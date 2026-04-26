package cluster

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

type stubProviderClient struct{}

func (s *stubProviderClient) CreateContainer(context.Context, config.Node) (string, error) {
	return "", nil
}

func (s *stubProviderClient) StartContainer(context.Context, string) error {
	return nil
}

func (s *stubProviderClient) StopContainer(context.Context, string) error {
	return nil
}

func (s *stubProviderClient) DeleteContainer(context.Context, string) error {
	return nil
}

func (s *stubProviderClient) InspectContainer(context.Context, string) (*provider.ContainerInfo, error) {
	return nil, nil
}

func (s *stubProviderClient) ListContainers(context.Context, []string) ([]provider.ContainerInfo, error) {
	return nil, nil
}

func (s *stubProviderClient) CreateNetwork(context.Context, config.Network) (string, error) {
	return "", nil
}

func (s *stubProviderClient) DeleteNetwork(context.Context, string) error {
	return nil
}

func (s *stubProviderClient) ListNetworks(context.Context, []string) ([]provider.NetworkInfo, error) {
	return nil, nil
}

func (s *stubProviderClient) InspectNetwork(context.Context, string) (*provider.NetworkInfo, error) {
	return nil, nil
}

func TestNewUsesInjectedProvider(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}
	injectedProvider := &stubProviderClient{}

	manager, err := New(logger, "di-seam", injectedProvider)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if manager.Provider() != injectedProvider {
		t.Fatalf("Provider() did not return injected provider")
	}
}

func TestNewReturnsErrorWhenProviderIsNil(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}

	manager, err := New(logger, "di-seam", nil)
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}

	if manager != nil {
		t.Fatal("New() manager = non-nil, want nil")
	}
}
