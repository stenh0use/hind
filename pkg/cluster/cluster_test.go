package cluster

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/config"
)

func TestCountClientNodes(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []config.Node
		expected int
	}{
		{
			name:     "no nodes",
			nodes:    []config.Node{},
			expected: 0,
		},
		{
			name: "only server nodes",
			nodes: []config.Node{
				{Name: "server1", Role: config.Server},
				{Name: "server2", Role: config.Server},
			},
			expected: 0,
		},
		{
			name: "only client nodes",
			nodes: []config.Node{
				{Name: "client1", Role: config.Client},
				{Name: "client2", Role: config.Client},
				{Name: "client3", Role: config.Client},
			},
			expected: 3,
		},
		{
			name: "mixed server and client nodes",
			nodes: []config.Node{
				{Name: "server1", Role: config.Server},
				{Name: "client1", Role: config.Client},
				{Name: "client2", Role: config.Client},
				{Name: "server2", Role: config.Server},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				config: &config.Cluster{
					Nodes: tt.nodes,
				},
			}

			got := m.CountClientNodes()
			if got != tt.expected {
				t.Errorf("CountClientNodes() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetClientNodes(t *testing.T) {
	tests := []struct {
		name          string
		nodes         []config.Node
		expectedCount int
	}{
		{
			name:          "no nodes",
			nodes:         []config.Node{},
			expectedCount: 0,
		},
		{
			name: "only server nodes",
			nodes: []config.Node{
				{Name: "server1", Role: config.Server},
				{Name: "server2", Role: config.Server},
			},
			expectedCount: 0,
		},
		{
			name: "only client nodes",
			nodes: []config.Node{
				{Name: "client1", Role: config.Client},
				{Name: "client2", Role: config.Client},
			},
			expectedCount: 2,
		},
		{
			name: "mixed nodes",
			nodes: []config.Node{
				{Name: "server1", Role: config.Server},
				{Name: "client1", Role: config.Client},
				{Name: "server2", Role: config.Server},
				{Name: "client2", Role: config.Client},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				config: &config.Cluster{
					Nodes: tt.nodes,
				},
			}

			clients := m.getClientNodes()
			if len(clients) != tt.expectedCount {
				t.Errorf("getClientNodes() returned %d nodes, want %d", len(clients), tt.expectedCount)
			}

			// Verify all returned nodes are clients
			for _, node := range clients {
				if node.Role != config.Client {
					t.Errorf("getClientNodes() returned non-client node: %s (role: %s)", node.Name, node.Role)
				}
			}
		})
	}
}

func TestFindNodeConfigByName(t *testing.T) {
	nodes := []config.Node{
		{Name: "server1", Role: config.Server},
		{Name: "client1", Role: config.Client},
		{Name: "client2", Role: config.Client},
	}

	tests := []struct {
		name      string
		searchFor string
		wantFound bool
		wantRole  config.Role
	}{
		{
			name:      "find existing server node",
			searchFor: "server1",
			wantFound: true,
			wantRole:  config.Server,
		},
		{
			name:      "find existing client node",
			searchFor: "client1",
			wantFound: true,
			wantRole:  config.Client,
		},
		{
			name:      "node does not exist",
			searchFor: "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				config: &config.Cluster{
					Nodes: nodes,
				},
			}

			node := m.findNodeConfigByName(tt.searchFor)

			if tt.wantFound {
				if node == nil {
					t.Errorf("findNodeConfigByName(%q) = nil, want non-nil", tt.searchFor)
					return
				}
				if node.Name != tt.searchFor {
					t.Errorf("findNodeConfigByName(%q) returned node with name %q", tt.searchFor, node.Name)
				}
				if node.Role != tt.wantRole {
					t.Errorf("findNodeConfigByName(%q) returned node with role %s, want %s", tt.searchFor, node.Role, tt.wantRole)
				}
			} else {
				if node != nil {
					t.Errorf("findNodeConfigByName(%q) = %+v, want nil", tt.searchFor, node)
				}
			}
		})
	}
}

func TestStartResult(t *testing.T) {
	// Test that StartResult constants are defined
	results := []StartResult{
		StartResultCreated,
		StartResultResumed,
		StartResultAlreadyRunning,
	}

	// Verify they have different values
	seen := make(map[StartResult]bool)
	for _, r := range results {
		if seen[r] {
			t.Errorf("duplicate StartResult value: %v", r)
		}
		seen[r] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 unique StartResult values, got %d", len(seen))
	}
}

func TestListReturnsEmptyWhenConfigDirMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	clusters, err := List()
	if err != nil {
		t.Fatalf("List() returned error when config dir is missing: %v", err)
	}

	if len(clusters) != 0 {
		t.Fatalf("List() expected 0 clusters on first run, got %d", len(clusters))
	}
}

func TestAddClientNodes_UsesNextAvailableNumbering(t *testing.T) {
	version := release.Latest().Hind
	clusterConfig := &config.Cluster{
		Name:    "demo",
		Version: version,
		Network: config.Network{Name: "hind.demo"},
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01", Role: config.Server},
			{Name: "hind.demo.client.01", Role: config.Client},
			{Name: "hind.demo.client.03", Role: config.Client},
		},
	}

	m := &Manager{
		logger: &log.Logger{Handler: discard.New(), Level: log.ErrorLevel},
		config: clusterConfig,
	}

	if err := m.addClientNodes(1); err != nil {
		t.Fatalf("addClientNodes() error = %v", err)
	}

	var gotClientNames []string
	for _, node := range m.config.Nodes {
		if node.Role == config.Client {
			gotClientNames = append(gotClientNames, node.Name)
		}
	}
	slices.Sort(gotClientNames)

	wantClientNames := []string{
		"hind.demo.client.01",
		"hind.demo.client.03",
		"hind.demo.client.04",
	}
	if !slices.Equal(gotClientNames, wantClientNames) {
		t.Fatalf("client names = %v, want %v", gotClientNames, wantClientNames)
	}
}

func TestNewNomadClientNode_ReturnsConsistentClientConfig(t *testing.T) {
	node := newNomadClientNode("demo", "hind.demo", "1.8.0", 7)

	if node.Name != "hind.demo.client.07" {
		t.Fatalf("node.Name = %q, want %q", node.Name, "hind.demo.client.07")
	}
	if node.Kind != config.NomadNode {
		t.Fatalf("node.Kind = %q, want %q", node.Kind, config.NomadNode)
	}
	if node.Role != config.Client {
		t.Fatalf("node.Role = %q, want %q", node.Role, config.Client)
	}
	if node.Network != "hind.demo" {
		t.Fatalf("node.Network = %q, want %q", node.Network, "hind.demo")
	}
	if node.Image.Name != release.NomadClient.ImageName() {
		t.Fatalf("node.Image.Name = %q, want %q", node.Image.Name, release.NomadClient.ImageName())
	}
	if node.Image.Tag != "1.8.0" {
		t.Fatalf("node.Image.Tag = %q, want %q", node.Image.Tag, "1.8.0")
	}
	if len(node.Devices) != 1 || node.Devices[0] != "/dev/fuse" {
		t.Fatalf("node.Devices = %v, want [/dev/fuse]", node.Devices)
	}
	if node.Environment["CONSUL_AGENT_MODE"] != "client" {
		t.Fatalf("CONSUL_AGENT_MODE = %q, want %q", node.Environment["CONSUL_AGENT_MODE"], "client")
	}
	if node.Environment["CONSUL_SERVER_ADDRESS"] != "hind.demo.consul.01" {
		t.Fatalf("CONSUL_SERVER_ADDRESS = %q, want %q", node.Environment["CONSUL_SERVER_ADDRESS"], "hind.demo.consul.01")
	}
	if node.Environment["NOMAD_AGENT_MODE"] != "client" {
		t.Fatalf("NOMAD_AGENT_MODE = %q, want %q", node.Environment["NOMAD_AGENT_MODE"], "client")
	}
}

func TestNewClusterConfig_UsesClientNodeFactory(t *testing.T) {
	cfg, err := newClusterConfig("test", release.Latest().Hind)
	if err != nil {
		t.Fatalf("newClusterConfig() error = %v", err)
	}

	var clients []config.Node
	for _, node := range cfg.Nodes {
		if node.Role == config.Client {
			clients = append(clients, node)
		}
	}
	if len(clients) != 1 {
		t.Fatalf("client node count = %d, want 1", len(clients))
	}

	expected := newNomadClientNode("test", "hind.test", cfg.Version, 1)
	client := clients[0]
	if client.Name != expected.Name {
		t.Fatalf("client.Name = %q, want %q", client.Name, expected.Name)
	}
	if client.Image != expected.Image {
		t.Fatalf("client.Image = %+v, want %+v", client.Image, expected.Image)
	}
	if !slices.Equal(client.Devices, expected.Devices) {
		t.Fatalf("client.Devices = %v, want %v", client.Devices, expected.Devices)
	}
	if !slices.Equal(client.Ports, expected.Ports) {
		t.Fatalf("client.Ports = %v, want %v", client.Ports, expected.Ports)
	}
	if len(client.Volumes) != len(expected.Volumes) {
		t.Fatalf("client.Volumes len = %d, want %d", len(client.Volumes), len(expected.Volumes))
	}
	if len(client.Environment) != len(expected.Environment) {
		t.Fatalf("environment size = %d, want %d", len(client.Environment), len(expected.Environment))
	}
	for key, wantValue := range expected.Environment {
		if got := client.Environment[key]; got != wantValue {
			t.Fatalf("environment[%q] = %q, want %q", key, got, wantValue)
		}
	}
}

func TestSetClientCount_UsesClientNodeFactory(t *testing.T) {
	version := release.Latest().Hind
	clusterConfig := &config.Cluster{
		Name:    "demo",
		Version: version,
		Network: config.Network{Name: "hind.demo"},
		Nodes: []config.Node{
			{Name: "hind.demo.consul.01", Role: config.Server},
			{Name: "hind.demo.nomad.01", Role: config.Server},
			{Name: "hind.demo.client.01", Role: config.Client},
		},
	}

	m := &Manager{config: clusterConfig}

	if err := m.SetClientCount(context.Background(), 2); err != nil {
		t.Fatalf("SetClientCount() error = %v", err)
	}

	var clients []config.Node
	for _, node := range m.config.Nodes {
		if node.Role == config.Client {
			clients = append(clients, node)
		}
	}

	if len(clients) != 2 {
		t.Fatalf("client node count = %d, want 2", len(clients))
	}

	expectedClients := []config.Node{
		newNomadClientNode("demo", "hind.demo", version, 1),
		newNomadClientNode("demo", "hind.demo", version, 2),
	}

	if !reflect.DeepEqual(clients, expectedClients) {
		t.Fatalf("client nodes = %#v, want %#v", clients, expectedClients)
	}

	var nonClientNames []string
	for _, node := range m.config.Nodes {
		if node.Role != config.Client {
			nonClientNames = append(nonClientNames, node.Name)
		}
	}

	wantNonClientNames := []string{"hind.demo.consul.01", "hind.demo.nomad.01"}
	if !reflect.DeepEqual(nonClientNames, wantNonClientNames) {
		t.Fatalf("non-client names = %v, want %v", nonClientNames, wantNonClientNames)
	}
}

func TestSetClientCount_RejectsCountBelowOne(t *testing.T) {
	m := &Manager{config: &config.Cluster{Name: "demo"}}

	if err := m.SetClientCount(context.Background(), 0); err == nil {
		t.Fatal("SetClientCount() error = nil, want non-nil")
	}
}
