package runtime

import "github.com/stenh0use/hind/pkg/cluster/domain"

type ContainerStatus string

const (
	ContainerRunning   ContainerStatus = "running"
	ContainerStopped   ContainerStatus = "stopped"
	ContainerUnhealthy ContainerStatus = "unhealthy"
	ContainerUnknown   ContainerStatus = "unknown"
)

type NetworkResource struct {
	Name domain.NetworkName
}

type ContainerResource struct {
	Name        domain.NodeName
	Status      ContainerStatus
	SpecHash    string
	Image       string
	ID          string
	Created     string
	HostName    string
	Environment map[string]string
	Ports       []domain.PortMapping
	Devices     []string
	Network     domain.NetworkName
	Labels      map[string]string
}

type Snapshot struct {
	Network    *NetworkResource
	Containers map[domain.NodeName]ContainerResource
	Orphans    []ContainerResource
}

type ResourceKind string

const (
	ResourceNetwork   ResourceKind = "network"
	ResourceContainer ResourceKind = "container"
)

type ResourceRef struct {
	Kind ResourceKind
	Name string
}

type ResourceSpec struct {
	Image       string
	Environment map[string]string
	Ports       []domain.PortMapping
	Devices     []string
	Network     domain.NetworkName
	Labels      map[string]string
	SpecHash    string
}
