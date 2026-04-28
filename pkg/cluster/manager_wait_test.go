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
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func TestWaitForContainersRunning_ReturnsContextErrorPromptly(t *testing.T) {
	m := &Manager{
		logger: &log.Logger{Handler: discard.New()},
		provider: &mock.ClientStub{
			InspectContainerFn: func(_ context.Context, name string) (*provider.ContainerInfo, error) {
				return &provider.ContainerInfo{Name: name, Status: "exited"}, nil
			},
			InspectNetworkFn: func(_ context.Context, name string) (*provider.NetworkInfo, error) {
				return &provider.NetworkInfo{Name: name}, nil
			},
		},
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
