package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/file"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func TestWaitForContainersRunning_ReturnsContextErrorPromptly(t *testing.T) {
	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	clusterCfg := &config.Cluster{
		Name: "test",
		Network: config.Network{
			Name: "hind.test",
		},
		Nodes: []config.Node{
			{Name: "hind.test.consul.01"},
		},
	}

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
		config:     clusterCfg,
		fm:         fm,
		configFile: file.JoinPath(ClusterConfigDir, "test", ClusterConfigFile),
	}

	data, err := json.Marshal(clusterCfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := fm.WriteFile(m.configFile, data); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err = m.waitForContainersRunning(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForContainersRunning() error = %v, want context.Canceled", err)
	}

	if elapsed > 200*time.Millisecond {
		t.Fatalf("waitForContainersRunning() took %s, expected prompt return after canceled context", elapsed)
	}
}
