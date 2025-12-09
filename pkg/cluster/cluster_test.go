package cluster

import (
	"testing"

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
