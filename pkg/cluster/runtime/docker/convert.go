package docker

import (
	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
	"github.com/stenh0use/hind/pkg/provider"
)

func toContainerStatus(s provider.Status) runtime.ContainerStatus {
	switch s {
	case provider.Running:
		return runtime.ContainerRunning
	case provider.Stopped:
		return runtime.ContainerStopped
	case provider.Error:
		return runtime.ContainerUnhealthy
	default:
		return runtime.ContainerUnknown
	}
}

func toContainerResource(c provider.ContainerInfo) runtime.ContainerResource {
	return runtime.ContainerResource{
		Name:        domain.NodeName(c.Name),
		Status:      toContainerStatus(c.Status),
		SpecHash:    c.Labels["hind.specHash"],
		Image:       c.Image,
		Created:     c.Created,
		HostName:    c.HostName,
		ID:          c.ID,
		Environment: map[string]string{},
		Ports:       nil,
		Devices:     nil,
		Network:     "",
		Labels:      cloneMap(c.Labels),
	}
}

func toProviderContainerSpec(name string, spec *runtime.ResourceSpec) provider.ContainerSpec {
	return provider.ContainerSpec{
		Name:        name,
		Network:     string(spec.Network),
		Image:       spec.Image,
		Ports:       append([]domain.PortMapping(nil), spec.Ports...),
		Environment: cloneMap(spec.Environment),
		Labels:      cloneMap(spec.Labels),
		Devices:     append([]string(nil), spec.Devices...),
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
