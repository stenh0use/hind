package cluster

import (
	"context"
	"testing"

	"github.com/stenh0use/hind/pkg/config"
	"github.com/stenh0use/hind/pkg/provider"
)

func TestReconcilePlan_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		plan ReconcilePlan
		want bool
	}{
		{
			name: "empty plan",
			plan: ReconcilePlan{},
			want: true,
		},
		{
			name: "has network to create",
			plan: ReconcilePlan{
				NetworkToCreate: &config.Network{Name: "test"},
			},
			want: false,
		},
		{
			name: "has containers to create",
			plan: ReconcilePlan{
				ContainersToCreate: []config.Node{{Name: "test"}},
			},
			want: false,
		},
		{
			name: "has containers to start",
			plan: ReconcilePlan{
				ContainersToStart: []string{"test"},
			},
			want: false,
		},
		{
			name: "has containers to recreate",
			plan: ReconcilePlan{
				ContainersToRecreate: []RecreateAction{{ExistingName: "test"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateReconcilePlan_NewCluster(t *testing.T) {
	m := &Manager{
		config: &config.Cluster{
			Name:    "test",
			Network: config.Network{Name: "hind.test"},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01", Kind: config.ConsulNode},
				{Name: "hind.test.nomad.01", Kind: config.NomadNode},
			},
		},
	}

	actual := &ActualState{
		Network:    nil,
		Containers: map[string]*provider.ContainerInfo{},
	}

	plan, err := m.calculateReconcilePlan(context.Background(), actual)
	if err != nil {
		t.Fatalf("calculateReconcilePlan() error = %v", err)
	}

	if plan.NetworkToCreate == nil {
		t.Error("NetworkToCreate should not be nil for new cluster")
	}

	if len(plan.ContainersToCreate) != 2 {
		t.Errorf("ContainersToCreate = %d, want 2", len(plan.ContainersToCreate))
	}

	if len(plan.ContainersToStart) != 0 {
		t.Errorf("ContainersToStart should be empty, got %d", len(plan.ContainersToStart))
	}

	if len(plan.ContainersToRecreate) != 0 {
		t.Errorf("ContainersToRecreate should be empty, got %d", len(plan.ContainersToRecreate))
	}
}

func TestCalculateReconcilePlan_AllRunning(t *testing.T) {
	m := &Manager{
		config: &config.Cluster{
			Name:    "test",
			Network: config.Network{Name: "hind.test"},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01", Kind: config.ConsulNode},
			},
		},
	}

	actual := &ActualState{
		Network: &provider.NetworkInfo{Name: "hind.test"},
		Containers: map[string]*provider.ContainerInfo{
			"hind.test.consul.01": {
				Name:   "hind.test.consul.01",
				Status: provider.Running.String(),
			},
		},
	}

	plan, err := m.calculateReconcilePlan(context.Background(), actual)
	if err != nil {
		t.Fatalf("calculateReconcilePlan() error = %v", err)
	}

	if !plan.IsEmpty() {
		t.Error("Plan should be empty when all containers are running")
	}
}

func TestCalculateReconcilePlan_StoppedContainers(t *testing.T) {
	m := &Manager{
		config: &config.Cluster{
			Name:    "test",
			Network: config.Network{Name: "hind.test"},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01", Kind: config.ConsulNode},
				{Name: "hind.test.nomad.01", Kind: config.NomadNode},
			},
		},
	}

	actual := &ActualState{
		Network: &provider.NetworkInfo{Name: "hind.test"},
		Containers: map[string]*provider.ContainerInfo{
			"hind.test.consul.01": {
				Name:   "hind.test.consul.01",
				Status: "exited",
			},
			"hind.test.nomad.01": {
				Name:   "hind.test.nomad.01",
				Status: "exited",
			},
		},
	}

	plan, err := m.calculateReconcilePlan(context.Background(), actual)
	if err != nil {
		t.Fatalf("calculateReconcilePlan() error = %v", err)
	}

	if len(plan.ContainersToStart) != 2 {
		t.Errorf("ContainersToStart = %d, want 2", len(plan.ContainersToStart))
	}

	if len(plan.ContainersToCreate) != 0 {
		t.Errorf("ContainersToCreate should be empty, got %d", len(plan.ContainersToCreate))
	}

	if len(plan.ContainersToRecreate) != 0 {
		t.Errorf("ContainersToRecreate should be empty, got %d", len(plan.ContainersToRecreate))
	}
}

func TestCalculateReconcilePlan_UnhealthyContainers(t *testing.T) {
	m := &Manager{
		config: &config.Cluster{
			Name:    "test",
			Network: config.Network{Name: "hind.test"},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01", Kind: config.ConsulNode},
			},
		},
	}

	actual := &ActualState{
		Network: &provider.NetworkInfo{Name: "hind.test"},
		Containers: map[string]*provider.ContainerInfo{
			"hind.test.consul.01": {
				Name:   "hind.test.consul.01",
				Status: provider.Error.String(),
			},
		},
	}

	plan, err := m.calculateReconcilePlan(context.Background(), actual)
	if err != nil {
		t.Fatalf("calculateReconcilePlan() error = %v", err)
	}

	if len(plan.ContainersToRecreate) != 1 {
		t.Errorf("ContainersToRecreate = %d, want 1", len(plan.ContainersToRecreate))
	}

	if plan.ContainersToRecreate[0].Reason != "unhealthy" {
		t.Errorf("Recreate reason = %s, want 'unhealthy'", plan.ContainersToRecreate[0].Reason)
	}

	if len(plan.ContainersToCreate) != 0 {
		t.Errorf("ContainersToCreate should be empty, got %d", len(plan.ContainersToCreate))
	}
}

func TestCalculateReconcilePlan_MixedStates(t *testing.T) {
	m := &Manager{
		config: &config.Cluster{
			Name:    "test",
			Network: config.Network{Name: "hind.test"},
			Nodes: []config.Node{
				{Name: "hind.test.consul.01", Kind: config.ConsulNode},
				{Name: "hind.test.nomad.01", Kind: config.NomadNode},
				{Name: "hind.test.vault.01", Kind: config.VaultNode},
				{Name: "hind.test.client.01", Kind: config.NomadNode},
			},
		},
	}

	actual := &ActualState{
		Network: &provider.NetworkInfo{Name: "hind.test"},
		Containers: map[string]*provider.ContainerInfo{
			"hind.test.consul.01": {
				Name:   "hind.test.consul.01",
				Status: provider.Running.String(),
			},
			"hind.test.nomad.01": {
				Name:   "hind.test.nomad.01",
				Status: "exited",
			},
			"hind.test.vault.01": {
				Name:   "hind.test.vault.01",
				Status: provider.Error.String(),
			},
			// client.01 doesn't exist in actual state
		},
	}

	plan, err := m.calculateReconcilePlan(context.Background(), actual)
	if err != nil {
		t.Fatalf("calculateReconcilePlan() error = %v", err)
	}

	// One running (no action)
	// One stopped (should start)
	if len(plan.ContainersToStart) != 1 {
		t.Errorf("ContainersToStart = %d, want 1", len(plan.ContainersToStart))
	}

	// One unhealthy (should recreate)
	if len(plan.ContainersToRecreate) != 1 {
		t.Errorf("ContainersToRecreate = %d, want 1", len(plan.ContainersToRecreate))
	}

	// One missing (should create)
	if len(plan.ContainersToCreate) != 1 {
		t.Errorf("ContainersToCreate = %d, want 1", len(plan.ContainersToCreate))
	}

	// Network exists (no action)
	if plan.NetworkToCreate != nil {
		t.Error("NetworkToCreate should be nil when network exists")
	}
}
