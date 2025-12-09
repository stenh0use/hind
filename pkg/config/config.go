package config

type Labels map[string]string

type Cluster struct {
	// Name of the hind cluster
	Name string
	// List of Nodes in the cluster
	Nodes []Node
	// Network configuration for the cluster
	Network Network
	// Hind version
	Version string
}

type Network struct {
	// Name of the network
	Name string
	// Network driver eg. 'bridge'
	Driver string
	// Subnet in CIDR format that represents a network segment
	Subnet string
	// IPv4 or IPv6 Gateway for the subnet
	Gateway string
	// Labels map of key/value labels to apply
	Labels Labels
}

// Type of Node
type Kind string

const (
	ConsulNode Kind = "consul"
	NomadNode  Kind = "nomad"
	VaultNode  Kind = "vault"
)

func (k Kind) String() string {
	return string(k)
}

type Role string

const (
	Server Role = "server"
	Client Role = "client"
)

func (r Role) String() string {
	return string(r)
}

type Node struct {
	// Name given to the node
	Name string
	// Kind of Node, eg, consul, nomad, vault
	Kind Kind
	// Role the node functions as eg. server or client
	Role Role
	// Image associated with the node
	Image Image
	// Network name to attach the node to
	Network string
	// Environment variables to pass to the node
	Environment map[string]string
	// List of ports to publish
	Ports []PortMapping
	// List of volumes to attach to the container
	Volumes []Volume
	// List of devices to expose to the container
	Devices []string
	// Labels map of key/value labels to apply
	Labels Labels
}

type Image struct {
	// OCI Image repository eg. docker.io/stenh0use/hind.consul
	Name string
	// Image tag eg. 0.3.0
	Tag string
	// Sha256 digest of the container image
	Digest string
}

type PortMapping struct {
	// Address to listen to on the host machine
	ListenAddress string
	// Port to map to on the host machine
	HostPort int32
	// Port to map to inside the container
	ContainerPort int32
	// L4 Protocol TCP/UDP/SCTP
	Protocol string
}

type Volume struct {
	// Name of the docker volume
	Name string
	// Destination path to mount to in the container
	Destination string
	// Source of the volume, eg path on host or volume identifier
	Source string
	// Labels map of key/value labels to apply
	Labels Labels
}
