package provider

import "github.com/stenh0use/hind/pkg/cluster/domain"

type ContainerInfo struct {
	ID       string
	Name     string
	Created  string
	HostName string
	Status   Status
	Image    string
	Ports    []string
	Labels   map[string]string
}

// ContainerSpec is the runtime-neutral spec for creating a container.
// Image is the full image reference ("name:tag" or "name").
type ContainerSpec struct {
	Name        string
	Network     string
	Image       string
	Ports       []domain.PortMapping
	Environment map[string]string
	Labels      map[string]string
	Devices     []string
}
