package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/file"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func TestManagerGet_NetworkNotFoundDoesNotPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	persisted := &config.Cluster{
		Name:    "demo",
		Network: config.Network{Name: "hind.demo"},
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01"},
		},
	}
	persistedData, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	configPath := file.JoinPath(ClusterConfigDir, "demo", ClusterConfigFile)
	if err := fm.WriteFile(configPath, persistedData); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := &Manager{
		provider: &mock.ClientStub{
			InspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return nil, nil
			},
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				return nil, nil
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo-default"},
			Nodes: []config.Node{
				{Name: "hind.demo.client.01"},
			},
		},
		fm:         fm,
		configFile: configPath,
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
		provider: &mock.ClientStub{
			InspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
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
		provider: &mock.ClientStub{
			InspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return &provider.NetworkInfo{Name: name}, nil
			},
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
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

func TestManagerGet_UsesPersistedTopology(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	persisted := &config.Cluster{
		Name:    "demo",
		Network: config.Network{Name: "hind.demo"},
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01"},
			{Name: "hind.demo.nomad.01"},
			{Name: "hind.demo.client.01"},
			{Name: "hind.demo.client.02"},
			{Name: "hind.demo.client.03"},
		},
	}

	persistedData, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	configPath := file.JoinPath(ClusterConfigDir, "demo", ClusterConfigFile)
	if err := fm.WriteFile(configPath, persistedData); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	inspected := []string{}
	m := &Manager{
		provider: &mock.ClientStub{
			InspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				return &provider.NetworkInfo{Name: name}, nil
			},
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				inspected = append(inspected, name)
				return &provider.ContainerInfo{Name: name, Status: provider.Running.String()}, nil
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo-default"},
			Nodes: []config.Node{
				{Name: "hind.demo.consul.01"},
				{Name: "hind.demo.nomad.01"},
				{Name: "hind.demo.client.01"},
			},
		},
		fm:         fm,
		configFile: configPath,
	}

	state, err := m.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if state.Network.Name != "hind.demo" {
		t.Fatalf("Get().Network.Name = %q, want %q", state.Network.Name, "hind.demo")
	}

	if len(state.Containers) != len(persisted.Nodes) {
		t.Fatalf("Get().Containers len = %d, want %d", len(state.Containers), len(persisted.Nodes))
	}

	if !slices.Contains(inspected, "hind.demo.client.03") {
		t.Fatalf("Get() did not inspect persisted scaled node hind.demo.client.03; inspected=%v", inspected)
	}
}

func TestManagerStop_UsesPersistedTopology(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	persisted := &config.Cluster{
		Name:    "demo",
		Network: config.Network{Name: "hind.demo"},
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01"},
			{Name: "hind.demo.nomad.01"},
			{Name: "hind.demo.client.01"},
			{Name: "hind.demo.client.02"},
			{Name: "hind.demo.client.03"},
		},
	}
	persistedData, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	configPath := file.JoinPath(ClusterConfigDir, "demo", ClusterConfigFile)
	if err := fm.WriteFile(configPath, persistedData); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stopped := []string{}
	m := &Manager{
		logger: &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		provider: &mock.ClientStub{
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				return &provider.ContainerInfo{Name: name, Status: provider.Running.String()}, nil
			},
			StopContainerFn: func(ctx context.Context, name string) error {
				stopped = append(stopped, name)
				return nil
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo-default"},
			Nodes: []config.Node{
				{Name: "hind.demo.consul.01"},
				{Name: "hind.demo.nomad.01"},
				{Name: "hind.demo.client.01"},
			},
		},
		fm:         fm,
		configFile: configPath,
	}

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() unexpected error: %v", err)
	}

	if len(stopped) != len(persisted.Nodes) {
		t.Fatalf("Stop() stopped %d nodes, want %d", len(stopped), len(persisted.Nodes))
	}

	if !slices.Contains(stopped, "hind.demo.client.03") {
		t.Fatalf("Stop() did not stop persisted scaled node hind.demo.client.03; stopped=%v", stopped)
	}
}

func TestManagerLoadPersistedConfig_MissingFileKeepsDefaults(t *testing.T) {
	t.Parallel()

	m := &Manager{
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo-default"},
			Nodes:   []config.Node{{Name: "hind.demo.consul.01"}},
		},
	}

	if err := m.LoadPersistedConfig(); err != nil {
		t.Fatalf("LoadPersistedConfig() unexpected error: %v", err)
	}

	if m.config.Network.Name != "hind.demo-default" {
		t.Fatalf("LoadPersistedConfig() changed defaults unexpectedly; got network %q", m.config.Network.Name)
	}
}

func TestManagerLoadPersistedConfig_MissingAndNoDefaultsErrors(t *testing.T) {
	t.Parallel()

	m := &Manager{}
	if err := m.LoadPersistedConfig(); err == nil {
		t.Fatal("LoadPersistedConfig() expected error when no persisted file and no in-memory config")
	}
}

func TestManagerStop_PropagatesInspectContainerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("container inspect failed")
	m := &Manager{
		logger: &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		provider: &mock.ClientStub{
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				// Return nil info with a real error (e.g. docker daemon error)
				return nil, wantErr
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
			Nodes:   []config.Node{{Name: "hind.demo.consul.01"}},
		},
	}

	err := m.Stop(context.Background())
	if err == nil {
		t.Fatal("Stop() expected error when InspectContainer returns error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Stop() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestManagerDelete_PropagatesInspectContainerError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	wantErr := errors.New("container inspect failed")
	m := &Manager{
		logger: &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		provider: &mock.ClientStub{
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				// Return nil info with a real error (e.g. docker daemon error)
				return nil, wantErr
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
			Nodes:   []config.Node{{Name: "hind.demo.consul.01"}},
		},
		fm:         fm,
		configFile: file.JoinPath(ClusterConfigDir, "demo", ClusterConfigFile),
	}

	err = m.Delete(context.Background())
	if err == nil {
		t.Fatal("Delete() expected error when InspectContainer returns error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestManagerDelete_PropagatesInspectNetworkError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	wantErr := errors.New("network inspect failed")
	m := &Manager{
		logger: &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		provider: &mock.ClientStub{
			InspectContainerFn: func(ctx context.Context, name string) (*provider.ContainerInfo, error) {
				// Container does not exist — nil, nil is the not-found signal
				return nil, nil
			},
			InspectNetworkFn: func(ctx context.Context, name string) (*provider.NetworkInfo, error) {
				// Return nil info with a real error
				return nil, wantErr
			},
		},
		config: &config.Cluster{
			Name:    "demo",
			Network: config.Network{Name: "hind.demo"},
			Nodes:   []config.Node{},
		},
		fm:         fm,
		configFile: file.JoinPath(ClusterConfigDir, "demo", ClusterConfigFile),
	}

	err = m.Delete(context.Background())
	if err == nil {
		t.Fatal("Delete() expected error when InspectNetwork returns error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, wantErr)
	}
}
