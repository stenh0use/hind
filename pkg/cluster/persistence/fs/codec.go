package fs

import (
	"encoding/json"
	"fmt"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

const (
	clusterAPIVersion = "hind.dev/v1alpha1"
	clusterKind       = "Cluster"
)

type recordEnvelope struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   recordMetadata  `json:"metadata"`
	Spec       json.RawMessage `json:"spec"`
	Status     any             `json:"status,omitempty"`
}

type recordMetadata struct {
	Name string `json:"name"`
}

// wireCluster is the on-disk representation of domain.Cluster.
// Field tags are lowercase to keep the JSON schema stable and decoupled
// from Go field naming.
type wireCluster struct {
	Name    string      `json:"name"`
	Version string      `json:"version,omitempty"`
	Network wireNetwork `json:"network"`
	Nodes   []wireNode  `json:"nodes,omitempty"`
}

type wireNetwork struct {
	Name string `json:"name"`
}

type wireNode struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Role        string            `json:"role"`
	ID          int               `json:"id"`
	Network     string            `json:"network"`
	Image       wireImage         `json:"image"`
	Ports       []wirePort        `json:"ports,omitempty"`
	Devices     []string          `json:"devices,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

type wireImage struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

type wirePort struct {
	HostPort      int32  `json:"hostPort"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

func decodeCluster(data []byte) (domain.Cluster, error) {
	// Try envelope shape first.
	var env recordEnvelope
	if err := json.Unmarshal(data, &env); err == nil && env.Kind != "" && len(env.Spec) > 0 {
		wc, err := unmarshalWire(env.Spec)
		if err != nil {
			return domain.Cluster{}, fmt.Errorf("decode envelope spec: %w", err)
		}
		if wc.Name == "" {
			wc.Name = env.Metadata.Name
		}
		return wireToDomain(wc), nil
	}
	// Bare wireCluster fallback (legacy compatibility with existing fixtures).
	wc, err := unmarshalWire(data)
	if err != nil {
		return domain.Cluster{}, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	return wireToDomain(wc), nil
}

func unmarshalWire(data []byte) (wireCluster, error) {
	var wc wireCluster
	if err := json.Unmarshal(data, &wc); err != nil {
		return wireCluster{}, err
	}
	if wc.Name == "" {
		return wireCluster{}, fmt.Errorf("invalid cluster config: name is required")
	}
	return wc, nil
}

func encodeCluster(c domain.Cluster) ([]byte, error) {
	wc := domainToWire(c)
	spec, err := json.Marshal(wc)
	if err != nil {
		return nil, fmt.Errorf("encode envelope spec: %w", err)
	}
	env := recordEnvelope{
		APIVersion: clusterAPIVersion,
		Kind:       clusterKind,
		Metadata:   recordMetadata{Name: wc.Name},
		Spec:       spec,
	}
	return json.MarshalIndent(env, "", "  ")
}

func domainToWire(c domain.Cluster) wireCluster {
	nodes := make([]wireNode, 0, len(c.Nodes))
	for _, n := range c.Nodes {
		ports := make([]wirePort, 0, len(n.Ports))
		for _, p := range n.Ports {
			ports = append(ports, wirePort{HostPort: p.HostPort, ContainerPort: p.ContainerPort, Protocol: p.Protocol})
		}
		nodes = append(nodes, wireNode{
			Name:        string(n.Name),
			Kind:        string(n.Kind),
			Role:        string(n.Role),
			ID:          n.ID,
			Network:     string(n.Network),
			Image:       wireImage{Name: n.Image.Name, Tag: n.Image.Tag},
			Ports:       ports,
			Devices:     n.Devices,
			Environment: n.Environment,
		})
	}
	return wireCluster{
		Name:    string(c.Name),
		Version: c.Version,
		Network: wireNetwork{Name: string(c.Network.Name)},
		Nodes:   nodes,
	}
}

func wireToDomain(wc wireCluster) domain.Cluster {
	nodes := make([]domain.Node, 0, len(wc.Nodes))
	for _, n := range wc.Nodes {
		ports := make([]domain.PortMapping, 0, len(n.Ports))
		for _, p := range n.Ports {
			ports = append(ports, domain.PortMapping{HostPort: p.HostPort, ContainerPort: p.ContainerPort, Protocol: p.Protocol})
		}
		nodes = append(nodes, domain.Node{
			Name:        domain.NodeName(n.Name),
			Kind:        domain.NodeKind(n.Kind),
			Role:        domain.NodeRole(n.Role),
			ID:          n.ID,
			Network:     domain.NetworkName(n.Network),
			Image:       domain.ImageRef{Name: n.Image.Name, Tag: n.Image.Tag},
			Ports:       ports,
			Devices:     n.Devices,
			Environment: n.Environment,
		})
	}
	return domain.Cluster{
		Name:    domain.Name(wc.Name),
		Version: wc.Version,
		Network: domain.Network{Name: domain.NetworkName(wc.Network.Name)},
		Nodes:   nodes,
	}
}
