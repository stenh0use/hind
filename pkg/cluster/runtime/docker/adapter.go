package docker

import (
	"context"
	"fmt"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/plan"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
	"github.com/stenh0use/hind/pkg/provider"
)

// Adapter translates runtime-neutral operations to provider client calls.
type Adapter struct {
	client provider.Client
}

// New constructs a Docker runtime adapter.
func New(client provider.Client) *Adapter {
	return &Adapter{client: client}
}

// Inspect returns runtime-neutral snapshot state for the given selector.
func (a *Adapter) Inspect(ctx context.Context, selector runtime.Selector) (runtime.Snapshot, error) {
	snap := runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}
	if selector.Cluster == "" {
		return snap, nil
	}

	networkName := fmt.Sprintf("hind.%s", selector.Cluster)
	netInfo, err := a.client.InspectNetwork(ctx, networkName)
	if err != nil {
		return runtime.Snapshot{}, fmt.Errorf("inspect network %q: %w", networkName, err)
	}
	if netInfo != nil {
		snap.Network = &runtime.NetworkResource{Name: domain.NetworkName(netInfo.Name)}
	}

	containers, err := a.client.ListContainers(ctx, []string{fmt.Sprintf("label=hind.cluster=%s", selector.Cluster)})
	if err != nil {
		return runtime.Snapshot{}, fmt.Errorf("list containers for cluster %q: %w", selector.Cluster, err)
	}
	if len(containers) == 0 {
		containers, err = a.client.ListContainers(ctx, []string{fmt.Sprintf("name=hind.%s.", selector.Cluster)})
		if err != nil {
			return runtime.Snapshot{}, fmt.Errorf("list containers for cluster %q via name fallback: %w", selector.Cluster, err)
		}
	}
	for _, c := range containers {
		resource := toContainerResource(c)
		nodeName := domain.NodeName(c.Name)
		snap.Containers[nodeName] = resource
	}
	return snap, nil
}

// Apply executes one runtime operation.
func (a *Adapter) Apply(ctx context.Context, op runtime.Operation) (runtime.OperationResult, error) {
	result := runtime.OperationResult{Resource: op.Resource}
	switch plan.OperationKind(op.Kind) {
	case plan.OpCreateNetwork:
		if op.Spec == nil {
			return result, fmt.Errorf("create network requires spec")
		}
		_, err := a.client.CreateNetwork(ctx, provider.NetworkSpec{Name: string(op.Spec.Network), Labels: cloneMap(op.Spec.Labels)})
		if err != nil {
			return result, fmt.Errorf("create network %q: %w", op.Spec.Network, err)
		}
	case plan.OpDeleteNetwork:
		if err := a.client.DeleteNetwork(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("delete network %q: %w", op.Resource.Name, err)
		}
	case plan.OpCreateContainer:
		if op.Spec == nil {
			return result, fmt.Errorf("create container requires spec")
		}
		_, err := a.client.CreateContainer(ctx, toProviderContainerSpec(op.Resource.Name, op.Spec))
		if err != nil {
			return result, fmt.Errorf("create container %q: %w", op.Resource.Name, err)
		}
	case plan.OpStartContainer:
		if err := a.client.StartContainer(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("start container %q: %w", op.Resource.Name, err)
		}
	case plan.OpStopContainer:
		if err := a.client.StopContainer(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("stop container %q: %w", op.Resource.Name, err)
		}
	case plan.OpDeleteContainer:
		// Best-effort stop: the container may already be Exited, in which case
		// docker returns non-zero. Ignore the stop error and let the delete
		// surface the real failure (or succeed). Forced removal at the provider
		// layer guarantees idempotency even if the container restarts between
		// calls.
		_ = a.client.StopContainer(ctx, op.Resource.Name)
		if err := a.client.DeleteContainer(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("delete container %q: %w", op.Resource.Name, err)
		}
	case plan.OpRecreateContainer:
		if op.Spec == nil {
			return result, fmt.Errorf("recreate container requires spec")
		}
		if err := a.client.StopContainer(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("stop container %q before recreate: %w", op.Resource.Name, err)
		}
		if err := a.client.DeleteContainer(ctx, op.Resource.Name); err != nil {
			return result, fmt.Errorf("delete container %q before recreate: %w", op.Resource.Name, err)
		}
		_, err := a.client.CreateContainer(ctx, toProviderContainerSpec(op.Resource.Name, op.Spec))
		if err != nil {
			return result, fmt.Errorf("recreate container %q: %w", op.Resource.Name, err)
		}
	default:
		return result, fmt.Errorf("unsupported operation kind %q", op.Kind)
	}
	return result, nil
}
