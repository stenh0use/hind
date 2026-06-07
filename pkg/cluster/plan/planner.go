package plan

import (
	"fmt"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

func hasDrift(specHash string, c runtime.ContainerResource) bool {
	return specHash != "" && c.SpecHash != "" && specHash != c.SpecHash
}

func toResourceSpec(clusterName domain.Name, n domain.Node) *ResourceSpec {
	ports := make([]int32, 0, len(n.Ports))
	for _, p := range n.Ports {
		ports = append(ports, p.ContainerPort)
	}
	labels := map[string]string{"hind.cluster": string(clusterName)}
	return &ResourceSpec{
		Image:       fmt.Sprintf("%s:%s", n.Image.Name, n.Image.Tag),
		Environment: cloneMap(n.Environment),
		Ports:       ports,
		Devices:     append([]string(nil), n.Devices...),
		Network:     string(n.Network),
		Labels:      labels,
	}
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func Build(desired domain.Cluster, actual runtime.Snapshot, goal Goal) (Plan, error) {
	p := Plan{Goal: goal}
	id := 1
	add := func(kind OperationKind, ref ResourceRef, spec *ResourceSpec) {
		p.Operations = append(p.Operations, Operation{ID: OperationID(fmt.Sprintf("op-%d", id)), Kind: kind, Resource: ref, Spec: spec})
		id++
	}

	switch goal {
	case GoalStart:
		networkCreated := false
		if actual.Network == nil {
			add(OpCreateNetwork, ResourceRef{Kind: ResourceNetwork, Name: string(desired.Network.Name)}, &ResourceSpec{Network: string(desired.Network.Name)})
			networkCreated = true
		}
		for _, n := range desired.Nodes {
			c, ok := actual.Containers[n.Name]
			if !ok {
				add(OpCreateContainer, ResourceRef{Kind: ResourceContainer, Name: string(n.Name)}, toResourceSpec(desired.Name, n))
				if networkCreated {
					deps := []OperationID{p.Operations[0].ID}
					p.Operations[len(p.Operations)-1].DependsOn = deps
				}
				continue
			}
			switch c.Status {
			case runtime.ContainerRunning:
				continue
			case runtime.ContainerStopped:
				add(OpStartContainer, ResourceRef{Kind: ResourceContainer, Name: string(n.Name)}, nil)
			default:
				add(OpRecreateContainer, ResourceRef{Kind: ResourceContainer, Name: string(n.Name)}, toResourceSpec(desired.Name, n))
			}
		}
	case GoalStop:
		for _, n := range desired.Nodes {
			if c, ok := actual.Containers[n.Name]; ok && c.Status == runtime.ContainerRunning {
				add(OpStopContainer, ResourceRef{Kind: ResourceContainer, Name: string(n.Name)}, nil)
			}
		}
	case GoalDelete:
		deleteContainerIDs := make([]OperationID, 0, len(desired.Nodes))
		for _, n := range desired.Nodes {
			if _, ok := actual.Containers[n.Name]; ok {
				add(OpDeleteContainer, ResourceRef{Kind: ResourceContainer, Name: string(n.Name)}, nil)
				deleteContainerIDs = append(deleteContainerIDs, p.Operations[len(p.Operations)-1].ID)
			}
		}
		if actual.Network != nil {
			add(OpDeleteNetwork, ResourceRef{Kind: ResourceNetwork, Name: string(actual.Network.Name)}, nil)
			p.Operations[len(p.Operations)-1].DependsOn = append([]OperationID(nil), deleteContainerIDs...)
		}
	}
	p.Noop = len(p.Operations) == 0
	if err := Validate(p); err != nil {
		return Plan{}, err
	}
	return p, nil
}
