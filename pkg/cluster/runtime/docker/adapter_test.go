package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/plan"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
	"github.com/stenh0use/hind/pkg/provider"
)

type fakeClient struct {
	containers            []provider.ContainerInfo
	listContainersResults [][]provider.ContainerInfo
	network               *provider.NetworkInfo

	inspectNetworkErr  error
	listContainersErr  error
	listContainersErrs []error
	startErr           error
	stopErr            error
	deleteErr          error
	createErr          error
	createNetworkErr   error
	deleteNetworkErr   error

	calls         []string
	listFilters   [][]string
	listCallCount int
	last          provider.ContainerSpec
}

func (f *fakeClient) CreateContainer(_ context.Context, cfg provider.ContainerSpec) (string, error) {
	f.calls = append(f.calls, "CreateContainer")
	f.last = cfg
	return "id", f.createErr
}
func (f *fakeClient) StartContainer(_ context.Context, _ string) error {
	f.calls = append(f.calls, "StartContainer")
	return f.startErr
}
func (f *fakeClient) StopContainer(_ context.Context, _ string) error {
	f.calls = append(f.calls, "StopContainer")
	return f.stopErr
}
func (f *fakeClient) KillContainer(_ context.Context, _ string) error { return nil }
func (f *fakeClient) DeleteContainer(_ context.Context, _ string) error {
	f.calls = append(f.calls, "DeleteContainer")
	return f.deleteErr
}
func (f *fakeClient) InspectContainer(_ context.Context, _ string) (*provider.ContainerInfo, error) {
	return nil, nil
}
func (f *fakeClient) ListContainers(_ context.Context, filters []string) ([]provider.ContainerInfo, error) {
	f.calls = append(f.calls, "ListContainers")
	f.listCallCount++
	f.listFilters = append(f.listFilters, filters)
	idx := f.listCallCount - 1
	if idx < len(f.listContainersErrs) && f.listContainersErrs[idx] != nil {
		return nil, f.listContainersErrs[idx]
	}
	if idx < len(f.listContainersResults) {
		return f.listContainersResults[idx], nil
	}
	return f.containers, f.listContainersErr
}
func (f *fakeClient) BuildImage(_ context.Context, _ provider.BuildImageOptions) (provider.BuildImageResult, error) {
	return provider.BuildImageResult{}, nil
}
func (f *fakeClient) TagExists(_ context.Context, _, _ string) (bool, error) { return false, nil }
func (f *fakeClient) PullImage(_ context.Context, _, _ string) error         { return nil }
func (f *fakeClient) CreateNetwork(_ context.Context, _ provider.NetworkSpec) (string, error) {
	f.calls = append(f.calls, "CreateNetwork")
	return "nid", f.createNetworkErr
}
func (f *fakeClient) DeleteNetwork(_ context.Context, _ string) error {
	f.calls = append(f.calls, "DeleteNetwork")
	return f.deleteNetworkErr
}
func (f *fakeClient) ListNetworks(_ context.Context, _ []string) ([]provider.NetworkInfo, error) {
	return nil, nil
}
func (f *fakeClient) InspectNetwork(_ context.Context, _ string) (*provider.NetworkInfo, error) {
	f.calls = append(f.calls, "InspectNetwork")
	return f.network, f.inspectNetworkErr
}

func TestAdapterInspect_MapsSnapshotAndStatuses(t *testing.T) {
	fc := &fakeClient{network: &provider.NetworkInfo{Name: "hind.demo"}, containers: []provider.ContainerInfo{{Name: "hind.demo.consul.01", Status: provider.Running, Labels: map[string]string{"hind.specHash": "abc"}}, {Name: "hind.demo.client.01", Status: provider.NA, Labels: map[string]string{}}}}
	a := New(fc)

	snap, err := a.Inspect(context.Background(), runtime.Selector{Cluster: "demo"})
	require.NoError(t, err)
	require.NotNil(t, snap.Network)
	assert.Equal(t, "hind.demo", string(snap.Network.Name))
	assert.Equal(t, "abc", snap.Containers[domain.NodeName("hind.demo.consul.01")].SpecHash)
	assert.Equal(t, runtime.ContainerUnknown, snap.Containers[domain.NodeName("hind.demo.client.01")].Status)
}

func TestAdapterInspect_PropagatesNetworkErrors(t *testing.T) {
	a := New(&fakeClient{inspectNetworkErr: errors.New("boom")})
	_, err := a.Inspect(context.Background(), runtime.Selector{Cluster: "demo"})
	require.Error(t, err)
}

func TestAdapterInspect_FallsBackToNameFilterWhenLabelQueryEmpty(t *testing.T) {
	fc := &fakeClient{
		network: &provider.NetworkInfo{Name: "hind.demo"},
		listContainersResults: [][]provider.ContainerInfo{
			{},
			{{Name: "hind.demo.consul.01", Status: provider.Running, Labels: map[string]string{"hind.specHash": "abc"}}, {Name: "hind.demo.client.01", Status: provider.NA, Labels: map[string]string{}}},
		},
	}
	a := New(fc)

	snap, err := a.Inspect(context.Background(), runtime.Selector{Cluster: "demo"})
	require.NoError(t, err)
	assert.Len(t, snap.Containers, 2)
	assert.Equal(t, runtime.ContainerRunning, snap.Containers[domain.NodeName("hind.demo.consul.01")].Status)
	wantFilters := [][]string{{"label=hind.cluster=demo"}, {"name=hind.demo."}}
	assert.Equal(t, wantFilters, fc.listFilters)
}

func TestAdapterInspect_DoesNotFallbackWhenLabelQueryReturnsContainers(t *testing.T) {
	fc := &fakeClient{network: &provider.NetworkInfo{Name: "hind.demo"}, containers: []provider.ContainerInfo{{Name: "hind.demo.consul.01", Status: provider.Running, Labels: map[string]string{}}}}
	a := New(fc)

	_, err := a.Inspect(context.Background(), runtime.Selector{Cluster: "demo"})
	require.NoError(t, err)
	assert.Equal(t, 1, fc.listCallCount)
	wantFilters := [][]string{{"label=hind.cluster=demo"}}
	assert.Equal(t, wantFilters, fc.listFilters)
}

func TestAdapterInspect_FallbackErrorPropagation(t *testing.T) {
	fc := &fakeClient{network: &provider.NetworkInfo{Name: "hind.demo"}, listContainersResults: [][]provider.ContainerInfo{{}}, listContainersErrs: []error{nil, errors.New("fallback failed")}}
	a := New(fc)

	_, err := a.Inspect(context.Background(), runtime.Selector{Cluster: "demo"})
	require.Error(t, err)
	assert.Equal(t, "list containers for cluster \"demo\" via name fallback: fallback failed", err.Error())
}

func TestAdapterApply_MapsSupportedOperations(t *testing.T) {
	tests := []struct {
		name      string
		op        runtime.Operation
		wantCalls []string
		assert    func(t *testing.T, fc *fakeClient)
	}{
		{
			name:      "create_network",
			op:        runtime.Operation{Kind: string(plan.OpCreateNetwork), Resource: runtime.ResourceRef{Kind: runtime.ResourceNetwork, Name: "hind.demo"}, Spec: &runtime.ResourceSpec{Network: domain.NetworkName("hind.demo"), Labels: map[string]string{"hind.cluster": "demo"}}},
			wantCalls: []string{"CreateNetwork"},
		},
		{
			name:      "delete_network",
			op:        runtime.Operation{Kind: string(plan.OpDeleteNetwork), Resource: runtime.ResourceRef{Kind: runtime.ResourceNetwork, Name: "hind.demo"}},
			wantCalls: []string{"DeleteNetwork"},
		},
		{
			name:      "create_container",
			op:        runtime.Operation{Kind: string(plan.OpCreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: &runtime.ResourceSpec{Image: "hashicorp/consul:1.20", Network: domain.NetworkName("hind.demo"), Labels: map[string]string{"hind.specHash": "h1", "hind.cluster": "demo"}, Ports: []domain.PortMapping{{HostPort: 8500, ContainerPort: 8500}}, Environment: map[string]string{"A": "B"}}},
			wantCalls: []string{"CreateContainer"},
			assert: func(t *testing.T, fc *fakeClient) {
				t.Helper()
				assert.Equal(t, "h1", fc.last.Labels["hind.specHash"], "labels not preserved: %#v", fc.last.Labels)
				assert.Equal(t, "demo", fc.last.Labels["hind.cluster"], "cluster label not preserved: %#v", fc.last.Labels)
				assert.Equal(t, "hind.demo.consul.01", fc.last.Name, "name not mapped")
			},
		},
		{
			name:      "start_container",
			op:        runtime.Operation{Kind: string(plan.OpStartContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}},
			wantCalls: []string{"StartContainer"},
		},
		{
			name:      "stop_container",
			op:        runtime.Operation{Kind: string(plan.OpStopContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}},
			wantCalls: []string{"StopContainer"},
		},
		{
			name:      "delete_container",
			op:        runtime.Operation{Kind: string(plan.OpDeleteContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}},
			wantCalls: []string{"StopContainer", "DeleteContainer"},
		},
		{
			name:      "recreate_container",
			op:        runtime.Operation{Kind: string(plan.OpRecreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: &runtime.ResourceSpec{Image: "hashicorp/consul:1.20", Network: domain.NetworkName("hind.demo"), Labels: map[string]string{"hind.specHash": "h1"}, Ports: []domain.PortMapping{{HostPort: 8500, ContainerPort: 8500}}, Environment: map[string]string{"A": "B"}}},
			wantCalls: []string{"StopContainer", "DeleteContainer", "CreateContainer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &fakeClient{}
			a := New(fc)

			_, err := a.Apply(context.Background(), tt.op)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCalls, fc.calls)
			if tt.assert != nil {
				tt.assert(t, fc)
			}
		})
	}
}

func TestAdapterApply_PropagatesErrorsByOperationKind(t *testing.T) {
	baseSpec := &runtime.ResourceSpec{Image: "hashicorp/consul:1.20", Network: domain.NetworkName("hind.demo"), Labels: map[string]string{"hind.specHash": "h1"}}
	providerErr := errors.New("provider failure")

	tests := []struct {
		name      string
		op        runtime.Operation
		fc        *fakeClient
		wantErr   string
		wantCalls []string
	}{
		{name: "create_network", op: runtime.Operation{Kind: string(plan.OpCreateNetwork), Resource: runtime.ResourceRef{Kind: runtime.ResourceNetwork, Name: "hind.demo"}, Spec: &runtime.ResourceSpec{Network: domain.NetworkName("hind.demo")}}, fc: &fakeClient{createNetworkErr: providerErr}, wantErr: "create network \"hind.demo\": provider failure", wantCalls: []string{"CreateNetwork"}},
		{name: "delete_network", op: runtime.Operation{Kind: string(plan.OpDeleteNetwork), Resource: runtime.ResourceRef{Kind: runtime.ResourceNetwork, Name: "hind.demo"}}, fc: &fakeClient{deleteNetworkErr: providerErr}, wantErr: "delete network \"hind.demo\": provider failure", wantCalls: []string{"DeleteNetwork"}},
		{name: "create_container", op: runtime.Operation{Kind: string(plan.OpCreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: baseSpec}, fc: &fakeClient{createErr: providerErr}, wantErr: "create container \"hind.demo.consul.01\": provider failure", wantCalls: []string{"CreateContainer"}},
		{name: "start_container", op: runtime.Operation{Kind: string(plan.OpStartContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}}, fc: &fakeClient{startErr: providerErr}, wantErr: "start container \"hind.demo.consul.01\": provider failure", wantCalls: []string{"StartContainer"}},
		{name: "stop_container", op: runtime.Operation{Kind: string(plan.OpStopContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}}, fc: &fakeClient{stopErr: providerErr}, wantErr: "stop container \"hind.demo.consul.01\": provider failure", wantCalls: []string{"StopContainer"}},
		{name: "delete_container", op: runtime.Operation{Kind: string(plan.OpDeleteContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}}, fc: &fakeClient{deleteErr: providerErr}, wantErr: "delete container \"hind.demo.consul.01\": provider failure", wantCalls: []string{"StopContainer", "DeleteContainer"}},
		{name: "recreate_container_stop", op: runtime.Operation{Kind: string(plan.OpRecreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: baseSpec}, fc: &fakeClient{stopErr: providerErr}, wantErr: "stop container \"hind.demo.consul.01\" before recreate: provider failure", wantCalls: []string{"StopContainer"}},
		{name: "recreate_container_delete", op: runtime.Operation{Kind: string(plan.OpRecreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: baseSpec}, fc: &fakeClient{deleteErr: providerErr}, wantErr: "delete container \"hind.demo.consul.01\" before recreate: provider failure", wantCalls: []string{"StopContainer", "DeleteContainer"}},
		{name: "recreate_container_create", op: runtime.Operation{Kind: string(plan.OpRecreateContainer), Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}, Spec: baseSpec}, fc: &fakeClient{createErr: providerErr}, wantErr: "recreate container \"hind.demo.consul.01\": provider failure", wantCalls: []string{"StopContainer", "DeleteContainer", "CreateContainer"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.fc)
			_, err := a.Apply(context.Background(), tt.op)
			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
			assert.Equal(t, tt.wantCalls, tt.fc.calls)
		})
	}
}

func TestAdapterApply_UnsupportedOperation(t *testing.T) {
	a := New(&fakeClient{})
	_, err := a.Apply(context.Background(), runtime.Operation{Kind: "unsupported-op", Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"}})
	require.Error(t, err)
	assert.Equal(t, "unsupported operation kind \"unsupported-op\"", err.Error())
}

func TestAdapterApply_DeleteContainer_TolerantOfStopError(t *testing.T) {
	// Container may already be Exited; docker stop returns non-zero. The
	// delete must still proceed and succeed.
	fc := &fakeClient{stopErr: errors.New("Container ... is not running")}
	a := New(fc)

	_, err := a.Apply(context.Background(), runtime.Operation{
		Kind:     string(plan.OpDeleteContainer),
		Resource: runtime.ResourceRef{Kind: runtime.ResourceContainer, Name: "hind.demo.consul.01"},
	})
	require.NoError(t, err)
	wantCalls := []string{"StopContainer", "DeleteContainer"}
	assert.Equal(t, wantCalls, fc.calls)
}

func TestAdapterApply_CreateNetworkPropagatesProviderError(t *testing.T) {
	fc := &fakeClient{createNetworkErr: errors.New("create network failed")}
	a := New(fc)

	_, err := a.Apply(context.Background(), runtime.Operation{Kind: string(plan.OpCreateNetwork), Resource: runtime.ResourceRef{Kind: runtime.ResourceNetwork, Name: "hind.demo"}, Spec: &runtime.ResourceSpec{Network: domain.NetworkName("hind.demo")}})
	require.Error(t, err)
	assert.Equal(t, "create network \"hind.demo\": create network failed", err.Error())
}
