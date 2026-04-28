package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/file"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/mock"
)

func newManagerForBehaviorTests(t *testing.T, clusterName string, cfg *config.Cluster, stub *mock.ClientStub) *Manager {
	t.Helper()

	root := t.TempDir()
	fm, err := file.New(root)
	if err != nil {
		t.Fatalf("file.New() error = %v", err)
	}

	return &Manager{
		logger:     &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		provider:   stub,
		config:     cfg,
		fm:         fm,
		configFile: file.JoinPath(ClusterConfigDir, clusterName, ClusterConfigFile),
	}
}

func TestManagerStart_ReturnsErrorWhenPersistedConfigInvalid(t *testing.T) {
	t.Parallel()

	m := newManagerForBehaviorTests(t, "demo", &config.Cluster{Name: "demo"}, &mock.ClientStub{})

	if err := m.fm.WriteFile(m.configFile, []byte("{")); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := m.Start(context.Background())
	if err == nil {
		t.Fatal("Start() expected error, got nil")
	}
	if result != StartResultCreated {
		t.Fatalf("Start() result = %v, want %v on error", result, StartResultCreated)
	}
	if !strings.Contains(err.Error(), "failed to load cluster config") {
		t.Fatalf("Start() error = %q, want load-config context", err)
	}
}

func TestManagerStart_UsesPersistedConfigForReconcile(t *testing.T) {
	t.Parallel()

	persistedOnlyNode := "hind.demo.client.03"
	wantErr := errors.New("persisted node inspected")

	stub := &mock.ClientStub{
		InspectNetworkFn: func(context.Context, string) (*provider.NetworkInfo, error) {
			return &provider.NetworkInfo{Name: "hind.demo"}, nil
		},
		InspectContainerFn: func(_ context.Context, name string) (*provider.ContainerInfo, error) {
			if name == persistedOnlyNode {
				return nil, wantErr
			}
			return nil, nil
		},
	}

	m := newManagerForBehaviorTests(t, "demo", &config.Cluster{
		Name: "demo",
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01"},
		},
		Network: config.Network{Name: "hind.demo-default"},
	}, stub)

	persisted := &config.Cluster{
		Name: "demo",
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01"},
			{Name: persistedOnlyNode},
		},
		Network: config.Network{Name: "hind.demo"},
	}

	data, err := json.Marshal(persisted)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := m.fm.WriteFile(m.configFile, data); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err = m.Start(context.Background())
	if err == nil {
		t.Fatal("Start() expected reconcile error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Start() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestManagerGet_ReturnsErrorWhenNoPersistedConfigAndNoDefaults(t *testing.T) {
	t.Parallel()

	m := newManagerForBehaviorTests(t, "demo", &config.Cluster{}, &mock.ClientStub{})

	_, err := m.Get(context.Background())
	if err == nil {
		t.Fatal("Get() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cluster config not found") {
		t.Fatalf("Get() error = %q, want missing-config error", err)
	}
}

func TestManagerStop_ReturnsWrappedStopContainerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("stop failed")
	nodeName := "hind.demo.consul.01"

	stub := &mock.ClientStub{
		InspectContainerFn: func(context.Context, string) (*provider.ContainerInfo, error) {
			return &provider.ContainerInfo{Name: nodeName, Status: provider.Running.String()}, nil
		},
		StopContainerFn: func(context.Context, string) error {
			return wantErr
		},
	}

	m := newManagerForBehaviorTests(t, "demo", &config.Cluster{
		Name: "demo",
		Nodes: []config.Node{
			{Name: nodeName},
		},
	}, stub)

	err := m.Stop(context.Background())
	if err == nil {
		t.Fatal("Stop() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Stop() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "failed to stop container") {
		t.Fatalf("Stop() error = %q, want stop-container context", err)
	}
}

func TestManagerDelete_ReturnsWrappedStopContainerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("stop failed")
	nodeName := "hind.demo.consul.01"

	stub := &mock.ClientStub{
		InspectContainerFn: func(context.Context, string) (*provider.ContainerInfo, error) {
			return &provider.ContainerInfo{Name: nodeName, Status: provider.Running.String()}, nil
		},
		StopContainerFn: func(context.Context, string) error {
			return wantErr
		},
	}

	m := newManagerForBehaviorTests(t, "demo", &config.Cluster{
		Name: "demo",
		Nodes: []config.Node{
			{Name: nodeName},
		},
		Network: config.Network{Name: "hind.demo"},
	}, stub)

	err := m.Delete(context.Background())
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Delete() error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), "failed to stop container") {
		t.Fatalf("Delete() error = %q, want stop-container context", err)
	}
}

func TestList_ReturnsErrorWhenClusterPathIsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	baseDir := filepath.Join(home, DefaultConfigParentDir, DefaultConfigName)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	clusterPath := filepath.Join(baseDir, ClusterConfigDir)
	if err := os.WriteFile(clusterPath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := List()
	if err == nil {
		t.Fatal("List() expected error when cluster path is a file, got nil")
	}
}
