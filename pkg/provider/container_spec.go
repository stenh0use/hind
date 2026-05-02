package provider

import "github.com/stenh0use/hind/pkg/config"

type ContainerSpec struct {
	Name        string
	Network     string
	Image       config.Image
	Ports       []config.PortMapping
	Environment map[string]string
	Labels      config.Labels
	Devices     []string
}

func ContainerSpecFromNode(node config.Node) ContainerSpec {
	return ContainerSpec{
		Name:        node.Name,
		Network:     node.Network,
		Image:       node.Image,
		Ports:       node.Ports,
		Environment: node.Environment,
		Labels:      node.Labels,
		Devices:     node.Devices,
	}
}
