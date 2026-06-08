package domain

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/version"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "dev", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "whitespace", input: "   ", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "dotdot", input: "..", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
		{name: "backslash", input: "a\\b", wantErr: true},
		{name: "absolute", input: "/tmp/x", wantErr: true},
		{name: "traversal", input: "../x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildDefaultClusterShape(t *testing.T) {
	c, err := BuildDefaultCluster("slice1", version.HindVersion)
	require.NoError(t, err)
	require.NotEmpty(t, c.Nodes, "expected at least one node in default cluster")
	require.Equal(t, NetworkName("hind.slice1"), c.Network.Name)
	var serverCount, clientCount int
	for _, n := range c.Nodes {
		switch n.Role {
		case RoleServer:
			serverCount++
		case RoleClient:
			clientCount++
		}
	}
	assert.Greater(t, serverCount, 0, "expected at least one server node")
	assert.Greater(t, clientCount, 0, "expected at least one client node")
}

func TestSetClientCountPolicy(t *testing.T) {
	c, err := BuildDefaultCluster("slice1", version.HindVersion)
	require.NoError(t, err)

	err = SetClientCount(&c, 3, true)
	require.NoError(t, err, "SetClientCount() preserve error")
	assert.NotEmpty(t, clientIDs(c), "expected client ids")
	assert.Equal(t, "1,2,3", clientIDs(c))

	err = SetClientCount(&c, 2, false)
	require.NoError(t, err, "SetClientCount() compact error")
	assert.Equal(t, "1,2", clientIDs(c))
}

func TestValidateClusterFailures(t *testing.T) {
	c := Cluster{Name: "x", Network: Network{Name: "hind.x"}, Nodes: []Node{}}
	assert.Error(t, c.Validate(), "expected validation error for missing required nodes")
}

func clientIDs(c Cluster) string {
	out := ""
	for _, n := range c.Nodes {
		if n.Role != RoleClient {
			continue
		}
		if out != "" {
			out += ","
		}
		out += strconv.Itoa(n.ID)
	}
	return out
}

func TestAggregateContainerStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		containers []ContainerSummary
		want       ClusterStatus
	}{
		{
			name:       "empty slice returns not-found",
			containers: nil,
			want:       ClusterStatusNotFound,
		},
		{
			name: "all running returns running",
			containers: []ContainerSummary{
				{Status: ContainerStatusRunning},
				{Status: ContainerStatusRunning},
			},
			want: ClusterStatusRunning,
		},
		{
			name: "all stopped returns stopped",
			containers: []ContainerSummary{
				{Status: ContainerStatusStopped},
				{Status: ContainerStatusStopped},
			},
			want: ClusterStatusStopped,
		},
		{
			name: "mix of running and stopped returns partial",
			containers: []ContainerSummary{
				{Status: ContainerStatusRunning},
				{Status: ContainerStatusStopped},
			},
			want: ClusterStatusPartial,
		},
		{
			name: "unhealthy alone returns degraded",
			containers: []ContainerSummary{
				{Status: ContainerStatusUnhealthy},
			},
			want: ClusterStatusDegraded,
		},
		{
			name: "unknown alone returns degraded",
			containers: []ContainerSummary{
				{Status: ContainerStatusUnknown},
			},
			want: ClusterStatusDegraded,
		},
		{
			name: "unhealthy mixed with running returns degraded",
			containers: []ContainerSummary{
				{Status: ContainerStatusRunning},
				{Status: ContainerStatusUnhealthy},
			},
			want: ClusterStatusDegraded,
		},
		{
			name: "single running returns running",
			containers: []ContainerSummary{
				{Status: ContainerStatusRunning},
			},
			want: ClusterStatusRunning,
		},
		{
			name: "single stopped returns stopped",
			containers: []ContainerSummary{
				{Status: ContainerStatusStopped},
			},
			want: ClusterStatusStopped,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AggregateContainerStatus(tt.containers)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestContainerStatusConstants(t *testing.T) {
	assert.Equal(t, "running", string(ContainerStatusRunning))
	assert.Equal(t, "stopped", string(ContainerStatusStopped))
	assert.Equal(t, "unhealthy", string(ContainerStatusUnhealthy))
	assert.Equal(t, "unknown", string(ContainerStatusUnknown))
}

func TestClusterStatusConstants(t *testing.T) {
	assert.Equal(t, "running", string(ClusterStatusRunning))
	assert.Equal(t, "stopped", string(ClusterStatusStopped))
	assert.Equal(t, "degraded", string(ClusterStatusDegraded))
	assert.Equal(t, "partial", string(ClusterStatusPartial))
	assert.Equal(t, "not-found", string(ClusterStatusNotFound))
}
