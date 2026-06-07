package plan

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

func testCluster() domain.Cluster {
	return domain.Cluster{
		Name:    domain.Name("demo"),
		Network: domain.Network{Name: domain.NetworkName("hind.demo")},
		Nodes: []domain.Node{
			{Name: domain.NodeName("hind.demo.consul.01"), Network: domain.NetworkName("hind.demo")},
			{Name: domain.NodeName("hind.demo.nomad.01"), Network: domain.NetworkName("hind.demo")},
		},
	}
}

func TestPlanStartGoal(t *testing.T) {
	tests := []struct {
		name          string
		snapshot      runtime.Snapshot
		wantCreateNet bool
		wantCreateOps int
		wantStartOps  int
		wantRecreate  int
		wantNoop      bool
	}{
		{name: "absent creates network and containers", snapshot: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}, wantCreateNet: true, wantCreateOps: 2},
		{name: "stopped containers are started", snapshot: runtime.Snapshot{Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")}, Containers: map[domain.NodeName]runtime.ContainerResource{domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerStopped}, domain.NodeName("hind.demo.nomad.01"): {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerStopped}}}, wantStartOps: 2},
		{name: "unhealthy and unknown are recreated", snapshot: runtime.Snapshot{Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")}, Containers: map[domain.NodeName]runtime.ContainerResource{domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerUnhealthy}, domain.NodeName("hind.demo.nomad.01"): {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerUnknown}}}, wantRecreate: 2},
		{name: "running is no-op", snapshot: runtime.Snapshot{Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")}, Containers: map[domain.NodeName]runtime.ContainerResource{domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerRunning}, domain.NodeName("hind.demo.nomad.01"): {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerRunning}}}, wantNoop: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := Build(testCluster(), tt.snapshot, GoalStart)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCreateNet, countKind(p, OpCreateNetwork) > 0, "create network mismatch")
			assert.Equal(t, tt.wantCreateOps, countKind(p, OpCreateContainer), "create ops mismatch")
			assert.Equal(t, tt.wantStartOps, countKind(p, OpStartContainer), "start ops mismatch")
			assert.Equal(t, tt.wantRecreate, countKind(p, OpRecreateContainer), "recreate ops mismatch")
			assert.Equal(t, tt.wantNoop, p.Noop, "noop mismatch")
		})
	}
}

func TestPlanDeleteOrdering(t *testing.T) {
	s := runtime.Snapshot{Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")}, Containers: map[domain.NodeName]runtime.ContainerResource{domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerRunning}}}
	p, err := Build(testCluster(), s, GoalDelete)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(p.Operations), 2, "expected at least 2 operations")
	assert.Equal(t, OpDeleteNetwork, p.Operations[len(p.Operations)-1].Kind, "last op must be delete network")
}

func TestValidateConflict(t *testing.T) {
	tests := []struct {
		name    string
		plan    Plan
		wantErr string
	}{
		{
			name: "duplicate operation id",
			plan: Plan{Goal: GoalStart, Operations: []Operation{
				{ID: "1", Kind: OpCreateNetwork, Resource: ResourceRef{Kind: ResourceNetwork, Name: "n"}},
				{ID: "1", Kind: OpCreateContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
			}},
			wantErr: "duplicate operation id",
		},
		{
			name: "duplicate create target",
			plan: Plan{Goal: GoalStart, Operations: []Operation{
				{ID: "1", Kind: OpCreateContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
				{ID: "2", Kind: OpCreateContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
			}},
			wantErr: "duplicate create target",
		},
		{
			name: "create container requires network create dependency when network absent",
			plan: Plan{Goal: GoalStart, Operations: []Operation{
				{ID: "1", Kind: OpCreateNetwork, Resource: ResourceRef{Kind: ResourceNetwork, Name: "n"}},
				{ID: "2", Kind: OpCreateContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
			}},
			wantErr: "must depend on create_network",
		},
		{
			name: "delete network depends on all container deletes",
			plan: Plan{Goal: GoalDelete, Operations: []Operation{
				{ID: "1", Kind: OpDeleteContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
				{ID: "2", Kind: OpDeleteNetwork, Resource: ResourceRef{Kind: ResourceNetwork, Name: "n"}},
			}},
			wantErr: "must depend on all delete_container operations",
		},
		{
			name: "incompatible operation pair on same resource",
			plan: Plan{Goal: GoalStart, Operations: []Operation{
				{ID: "1", Kind: OpStartContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
				{ID: "2", Kind: OpDeleteContainer, Resource: ResourceRef{Kind: ResourceContainer, Name: "a"}},
			}},
			wantErr: "incompatible operations on same resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.plan)
			require.Error(t, err, "expected conflict error")
			if tt.wantErr != "" {
				assert.True(t, strings.Contains(err.Error(), tt.wantErr), "error=%q want substring=%q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestBuildDeleteNetworkDependsOnContainerDeletes(t *testing.T) {
	s := runtime.Snapshot{Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")}, Containers: map[domain.NodeName]runtime.ContainerResource{domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerRunning}, domain.NodeName("hind.demo.nomad.01"): {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerRunning}}}
	p, err := Build(testCluster(), s, GoalDelete)
	require.NoError(t, err)
	var deleteContainerIDs []OperationID
	var deleteNetwork Operation
	for _, op := range p.Operations {
		if op.Kind == OpDeleteContainer {
			deleteContainerIDs = append(deleteContainerIDs, op.ID)
		}
		if op.Kind == OpDeleteNetwork {
			deleteNetwork = op
		}
	}
	require.Len(t, deleteContainerIDs, 2, "delete container ops count mismatch")
	for _, id := range deleteContainerIDs {
		assert.True(t, hasID(deleteNetwork.DependsOn, id), "delete network missing dependency on %s", id)
	}
}

func TestBuildStartContainerDependsOnCreatedNetwork(t *testing.T) {
	s := runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}
	desired := testCluster()
	desired.Nodes[0].Image = domain.ImageRef{Name: "hashicorp/consul", Tag: "1.20"}
	desired.Nodes[0].Ports = []domain.PortMapping{{ContainerPort: 8500, HostPort: 8500}}
	desired.Nodes[0].Environment = map[string]string{"CONSUL_BIND": "0.0.0.0"}
	desired.Nodes[0].Devices = []string{"/dev/net/tun"}
	p, err := Build(desired, s, GoalStart)
	require.NoError(t, err)
	var createNetworkID OperationID
	for _, op := range p.Operations {
		if op.Kind == OpCreateNetwork {
			createNetworkID = op.ID
			require.NotNil(t, op.Spec, "create network operation missing spec")
			assert.Equal(t, string(desired.Network.Name), op.Spec.Network, "create network spec network mismatch")
		}
	}
	require.NotEmpty(t, createNetworkID, "expected create network operation")
	for _, op := range p.Operations {
		if op.Kind != OpCreateContainer {
			continue
		}
		assert.True(t, hasID(op.DependsOn, createNetworkID), "create container %s missing create network dependency", op.Resource.Name)
		require.NotNil(t, op.Spec, "create container %s missing spec", op.Resource.Name)
		if op.Resource.Name == string(desired.Nodes[0].Name) {
			assert.Equal(t, "hashicorp/consul:1.20", op.Spec.Image)
			assert.Equal(t, string(desired.Nodes[0].Network), op.Spec.Network)
			assert.Len(t, op.Spec.Ports, 1)
			assert.Equal(t, int32(8500), op.Spec.Ports[0])
			assert.Equal(t, "0.0.0.0", op.Spec.Environment["CONSUL_BIND"])
			assert.Equal(t, []string{"/dev/net/tun"}, op.Spec.Devices)
		}
	}
}

func TestBuildStartCreateContainerIncludesClusterLabel(t *testing.T) {
	desired := testCluster()
	snapshot := runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}

	p, err := Build(desired, snapshot, GoalStart)
	require.NoError(t, err)

	for _, op := range p.Operations {
		if op.Kind != OpCreateContainer {
			continue
		}
		require.NotNil(t, op.Spec, "create container %s missing spec", op.Resource.Name)
		require.NotNil(t, op.Spec.Labels, "create container %s missing labels map", op.Resource.Name)
		assert.Equal(t, string(desired.Name), op.Spec.Labels["hind.cluster"], "create container %s hind.cluster label mismatch", op.Resource.Name)
	}
}

func TestBuildStartRecreateContainerIncludesClusterLabel(t *testing.T) {
	desired := testCluster()
	snapshot := runtime.Snapshot{
		Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")},
		Containers: map[domain.NodeName]runtime.ContainerResource{
			domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerUnhealthy},
			domain.NodeName("hind.demo.nomad.01"):  {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerUnknown},
		},
	}

	p, err := Build(desired, snapshot, GoalStart)
	require.NoError(t, err)

	assert.Equal(t, 2, countKind(p, OpRecreateContainer), "recreate ops count mismatch")

	for _, op := range p.Operations {
		if op.Kind != OpRecreateContainer {
			continue
		}
		require.NotNil(t, op.Spec, "recreate container %s missing spec", op.Resource.Name)
		require.NotNil(t, op.Spec.Labels, "recreate container %s missing labels map", op.Resource.Name)
		assert.Equal(t, string(desired.Name), op.Spec.Labels["hind.cluster"], "recreate container %s hind.cluster label mismatch", op.Resource.Name)
	}
}

func TestHasDriftOwnedFieldsPolicy(t *testing.T) {
	base := runtime.ContainerResource{SpecHash: ""}
	assert.False(t, hasDrift("", base), "expected no drift when desired hash absent")
	assert.False(t, hasDrift("abc", base), "expected no drift when actual hash absent")
	assert.False(t, hasDrift("abc", runtime.ContainerResource{SpecHash: "abc"}), "expected no drift for equal hashes")
	assert.True(t, hasDrift("abc", runtime.ContainerResource{SpecHash: "def"}), "expected drift for hash mismatch")
}

func TestPlanStartGoal_AllStopped_StartsAll(t *testing.T) {
	s := runtime.Snapshot{
		Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")},
		Containers: map[domain.NodeName]runtime.ContainerResource{
			domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerStopped},
			domain.NodeName("hind.demo.nomad.01"):  {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerStopped},
		},
	}
	p, err := Build(testCluster(), s, GoalStart)
	require.NoError(t, err)
	assert.Equal(t, 2, countKind(p, OpStartContainer), "start ops count mismatch")
	assert.Equal(t, 0, countKind(p, OpCreateContainer), "create ops must be 0 (containers already exist; must not recreate)")
	assert.Equal(t, 0, countKind(p, OpCreateNetwork), "create network ops must be 0 (network already exists)")
}

func TestPlanDeleteGoal_AllStopped_DeletesAll(t *testing.T) {
	s := runtime.Snapshot{
		Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")},
		Containers: map[domain.NodeName]runtime.ContainerResource{
			domain.NodeName("hind.demo.consul.01"): {Name: domain.NodeName("hind.demo.consul.01"), Status: runtime.ContainerStopped},
			domain.NodeName("hind.demo.nomad.01"):  {Name: domain.NodeName("hind.demo.nomad.01"), Status: runtime.ContainerStopped},
		},
	}
	p, err := Build(testCluster(), s, GoalDelete)
	require.NoError(t, err)
	assert.Equal(t, 2, countKind(p, OpDeleteContainer), "delete container ops count mismatch")
	assert.Equal(t, 1, countKind(p, OpDeleteNetwork), "delete network ops count mismatch")
	last := p.Operations[len(p.Operations)-1]
	assert.Equal(t, OpDeleteNetwork, last.Kind, "last op kind mismatch")
}

func hasID(ids []OperationID, target OperationID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func countKind(p Plan, kind OperationKind) int {
	n := 0
	for _, op := range p.Operations {
		if op.Kind == kind {
			n++
		}
	}
	return n
}
