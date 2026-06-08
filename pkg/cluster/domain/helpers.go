package domain

import (
	"fmt"

	"github.com/stenh0use/hind/pkg/naming"
)

const (
	DefaultConsulPort int32 = 8500
	DefaultNomadPort  int32 = 4646
	DefaultVaultPort  int32 = 8200
)

func newNode(clusterName string, networkName NetworkName, version string, kind NodeKind, role NodeRole, id int) Node {
	name := nodeNameFor(kind, clusterName, id)
	n := Node{
		Name:    name,
		Kind:    kind,
		Role:    role,
		ID:      id,
		Network: networkName,
	}

	switch kind {
	case KindConsul:
		n.Image = ImageRef{Name: naming.Consul.ImageName(), Tag: version}
		n.Environment = map[string]string{"CONSUL_AGENT_MODE": "server"}
		if id == 1 {
			n.Ports = []PortMapping{{HostPort: DefaultConsulPort, ContainerPort: DefaultConsulPort, Protocol: "tcp"}}
		}
	case KindNomad:
		n.Image = ImageRef{Name: naming.Nomad.ImageName(), Tag: version}
		n.Environment = map[string]string{
			"CONSUL_AGENT_MODE":     "client",
			"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", clusterName, 1),
			"NOMAD_AGENT_MODE":      "server",
		}
		if id == 1 {
			n.Ports = []PortMapping{{HostPort: DefaultNomadPort, ContainerPort: DefaultNomadPort, Protocol: "tcp"}}
		}
	case KindVault:
		n.Image = ImageRef{Name: naming.Vault.ImageName(), Tag: version}
		n.Environment = map[string]string{
			"CONSUL_AGENT_MODE":     "client",
			"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", clusterName, 1),
		}
		if id == 1 {
			n.Ports = []PortMapping{{HostPort: DefaultVaultPort, ContainerPort: DefaultVaultPort, Protocol: "tcp"}}
		}
	default:
		n.Image = ImageRef{Name: string(kind), Tag: version}
	}

	return n
}

func newNomadClientNode(clusterName string, networkName NetworkName, version string, id int) Node {
	n := clientNode(clusterName, networkName, version, id)
	n.Image = ImageRef{Name: naming.NomadClient.ImageName(), Tag: version}
	n.Devices = []string{"/dev/fuse"}
	n.Environment = map[string]string{
		"CONSUL_AGENT_MODE":     "client",
		"CONSUL_SERVER_ADDRESS": fmt.Sprintf("hind.%s.consul.%.2d", clusterName, 1),
		"NOMAD_AGENT_MODE":      "client",
	}
	n.Ports = nil
	return n
}

func clientNode(clusterName string, networkName NetworkName, version string, id int) Node {
	n := newNode(clusterName, networkName, version, KindNomad, RoleClient, id)
	n.Name = ClientNodeName(clusterName, id)
	return n
}

func newClient(c Cluster, id int) Node {
	return newNomadClientNode(string(c.Name), c.Network.Name, c.Version, id)
}

func nodeNameFor(kind NodeKind, clusterName string, id int) NodeName {
	switch kind {
	case KindConsul:
		return ConsulNodeName(clusterName, id)
	case KindNomad:
		return NomadServerNodeName(clusterName, id)
	case KindVault:
		return VaultNodeName(clusterName, id)
	default:
		return NodeName(clusterName)
	}
}
