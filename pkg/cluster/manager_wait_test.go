package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

type waitFakeProvider struct{}

func (f *waitFakeProvider) CreateContainer(ctx context.Context, cfg config.Node) (string, error) {
	return "", nil
}
func (f *waitFakeProvider) StartContainer(ctx context.Context, name string) error { return nil }
func (f *waitFakeProvider) StopContainer(ctx context.Context, name string) error  { return nil }
func (f *waitFakeProvider) DeleteContainer(ctx context.Context, name string) error {
	return nil
}
func (f *waitFakeProvider) InspectContainer(ctx context.Context, name string) (*provider.ContainerInfo, error) {
	return &provider.ContainerInfo{Name: name, Status: "exited"}, nil
}
func (f *waitFakeProvider) ListContainers(ctx context.Context, filters []string) ([]provider.ContainerInfo, error) {
	return nil, nil
}
func (f *waitFakeProvider) CreateNetwork(ctx context.Context, cfg config.Network) (string, error) {
	return "", nil
}
func (f *waitFakeProvider) DeleteNetwork(ctx context.Context, name string) error { return nil }
func (f *waitFakeProvider) ListNetworks(ctx context.Context, filters []string) ([]provider.NetworkInfo, error) {
	return nil, nil
}
func (f *waitFakeProvider) InspectNetwork(ctx context.Context, name string) (*provider.NetworkInfo, error) {
	return &provider.NetworkInfo{Name: name}, nil
}

func TestWaitForContainersRunning_ReturnsContextErrorPromptly(t *testing.T) {
	m := &Manager{
		logger:   &log.Logger{Handler: discard.New()},
		provider: &waitFakeProvider{},
		config: &config.Cluster{
			Name: "test",
			Network: config.Network{
				Name: "hind.test",
			},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01"},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := m.waitForContainersRunning(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForContainersRunning() error = %v, want context.Canceled", err)
	}

	if elapsed > 200*time.Millisecond {
		t.Fatalf("waitForContainersRunning() took %s, expected prompt return after canceled context", elapsed)
	}
}
