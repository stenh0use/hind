package cluster

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/version"
)

// newDomainClientNode returns the domain client node at the given sequence number
// from a freshly constructed scratch cluster.
func newDomainClientNode(clusterName, networkName, version string, nodeNumber int) domain.Node {
	scratch := domain.Cluster{
		Name:    domain.Name(clusterName),
		Version: version,
		Network: domain.Network{Name: domain.NetworkName(networkName)},
		Nodes:   []domain.Node{},
	}
	_ = domain.SetClientCount(&scratch, nodeNumber, false)
	if len(scratch.Nodes) == 0 {
		return domain.Node{}
	}
	return scratch.Nodes[nodeNumber-1]
}

func TestNewNomadClientNode_ReturnsConsistentClientConfig(t *testing.T) {
	node := newDomainClientNode("demo", "hind.demo", "1.8.0", 7)

	assert.Equal(t, domain.NodeName("hind.demo.client.07"), node.Name)
	assert.Equal(t, domain.KindNomad, node.Kind)
	assert.Equal(t, domain.RoleClient, node.Role)
	assert.Equal(t, 7, node.ID)
	assert.Equal(t, domain.NetworkName("hind.demo"), node.Network)
	assert.Equal(t, "1.8.0", node.Image.Tag)
	assert.Equal(t, []string{"/dev/fuse"}, node.Devices)
	assert.Equal(t, "client", node.Environment["CONSUL_AGENT_MODE"])
	assert.Equal(t, "hind.demo.consul.01", node.Environment["CONSUL_SERVER_ADDRESS"])
	assert.Equal(t, "client", node.Environment["NOMAD_AGENT_MODE"])
}

func TestNewClusterConfig_UsesClientNodeFactory(t *testing.T) {
	dc, err := domain.BuildDefaultCluster("test", version.HindVersion)
	require.NoError(t, err)

	var clients []domain.Node
	for _, node := range dc.Nodes {
		if node.Role == domain.RoleClient {
			clients = append(clients, node)
		}
	}
	require.Len(t, clients, 1, "client node count mismatch")

	expected := newDomainClientNode("test", "hind.test", dc.Version, 1)
	client := clients[0]
	assert.Equal(t, expected.Name, client.Name)
	assert.Equal(t, expected.Image, client.Image)
	assert.True(t, slices.Equal(client.Devices, expected.Devices), "client.Devices = %v, want %v", client.Devices, expected.Devices)
	assert.True(t, slices.Equal(client.Ports, expected.Ports), "client.Ports = %v, want %v", client.Ports, expected.Ports)
	assert.Len(t, client.Environment, len(expected.Environment))
	for key, wantValue := range expected.Environment {
		assert.Equal(t, wantValue, client.Environment[key], "environment[%q] mismatch", key)
	}
}
