package cluster

import (
	"fmt"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/config"
)

const (
	DefaultNomadServers  = 1
	DefaultConsulServers = 1
	DefaultNomadClients  = 1
	DefaultVaultServers  = 1
)

// StartResult indicates the outcome of a cluster start operation
type StartResult int

const (
	// StartResultCreated indicates a new cluster was created
	StartResultCreated StartResult = iota
	// StartResultResumed indicates an existing cluster was started
	StartResultResumed
	// StartResultAlreadyRunning indicates the cluster was already running
	StartResultAlreadyRunning
)

func newClusterConfig(name string, version string) (*config.Cluster, error) {
	v, err := release.Get(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	networkName := "hind." + name

	var nodes []config.Node

	for count := range DefaultConsulServers {
		consulServer := config.Node{
			Name:    fmt.Sprintf("hind.%s.consul.%.2d", name, count+1),
			Kind:    config.ConsulNode,
			Role:    config.Server,
			Network: networkName,
			Image: config.Image{
				Name: release.Consul.ImageName(),
				Tag:  v.Hind,
			},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE": "server",
			},
		}
		// expose the port only on the first instance
		if count == 0 {
			consulServer.Ports = []config.PortMapping{
				{
					HostPort:      8500,
					ContainerPort: 8500,
					Protocol:      "tcp",
				},
			}
		}
		nodes = append(nodes, consulServer)
	}

	for count := range DefaultNomadServers {
		nomadServer := config.Node{
			Name:    fmt.Sprintf("hind.%s.nomad.%.2d", name, count+1),
			Kind:    config.NomadNode,
			Role:    config.Server,
			Network: networkName,
			Image: config.Image{
				Name: release.Nomad.ImageName(),
				Tag:  v.Hind,
			},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE":     "client",
				"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", name, 1),
				"NOMAD_AGENT_MODE":      "server",
			},
		}
		// expose the port only on the first instance
		if count == 0 {
			nomadServer.Ports = []config.PortMapping{
				{
					HostPort:      4646,
					ContainerPort: 4646,
					Protocol:      "tcp",
				},
			}
		}
		nodes = append(nodes, nomadServer)
	}

	for count := range DefaultNomadClients {
		nomadClient := config.Node{
			Name:    fmt.Sprintf("hind.%s.client.%.2d", name, count+1),
			Kind:    config.NomadNode,
			Role:    config.Client,
			Network: networkName,
			Image: config.Image{
				Name: release.NomadClient.ImageName(),
				Tag:  v.Hind,
			},
			Devices: []string{"/dev/fuse"},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE":     "client",
				"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", name, 1),
				"NOMAD_AGENT_MODE":      "client",
			},
		}
		nodes = append(nodes, nomadClient)
	}

	for count := range DefaultVaultServers {
		vaultServer := config.Node{
			Name:    fmt.Sprintf("hind.%s.vault.%.2d", name, count+1),
			Kind:    config.VaultNode,
			Role:    config.Server,
			Network: networkName,
			Image: config.Image{
				Name: release.Vault.ImageName(),
				Tag:  v.Hind,
			},
			Ports: []config.PortMapping{
				{
					HostPort:      8200,
					ContainerPort: 8200,
					Protocol:      "tcp",
				},
			},
			Environment: map[string]string{
				"CONSUL_AGENT_MODE":     "client",
				"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", name, 1),
			},
		}
		// expose the port only on the first instance
		if count == 0 {
			vaultServer.Ports = []config.PortMapping{
				{
					HostPort:      8200,
					ContainerPort: 8200,
					Protocol:      "tcp",
				},
			}
		}
		nodes = append(nodes, vaultServer)
	}

	cluster := &config.Cluster{
		Name:    name,
		Nodes:   nodes,
		Network: config.Network{Name: "hind." + name},
		Version: v.Hind,
	}

	return cluster, nil
}
