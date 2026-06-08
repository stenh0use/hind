package domain

type Name string

type NetworkName string

type NodeName string

type NodeKind string

type NodeRole string

type ImageRef struct {
	Name string
	Tag  string
}

type PortMapping struct {
	HostPort      int32
	ContainerPort int32
	Protocol      string
}

type Network struct {
	Name NetworkName
}

type Node struct {
	Name        NodeName
	Kind        NodeKind
	Role        NodeRole
	ID          int
	Network     NetworkName
	Image       ImageRef
	Ports       []PortMapping
	Devices     []string
	Environment map[string]string
}

type Cluster struct {
	Name    Name
	Version string
	Network Network
	Nodes   []Node
}

const (
	KindConsul NodeKind = "consul"
	KindNomad  NodeKind = "nomad"
	KindVault  NodeKind = "vault"

	RoleServer NodeRole = "server"
	RoleClient NodeRole = "client"
)
