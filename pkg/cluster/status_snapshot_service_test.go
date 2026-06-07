package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

func TestStatusSnapshot_Build(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	earlier := now.Add(-10 * time.Minute)

	tests := []struct {
		name    string
		inspect InspectResult
		status  string
		running int
		total   int
		created time.Time
	}{
		{
			name: "running healthy",
			inspect: InspectResult{Containers: []ContainerSummary{
				{Name: "n1", Status: domain.ContainerStatusRunning, Created: now.Format(time.RFC3339)},
				{Name: "n2", Status: domain.ContainerStatusRunning, Created: earlier.Format(time.RFC3339)},
			}},
			status:  "running",
			running: 2,
			total:   2,
			created: earlier,
		},
		{
			name: "stopped exited",
			inspect: InspectResult{Containers: []ContainerSummary{
				{Name: "n1", Status: domain.ContainerStatusStopped, Created: now.Format(time.RFC3339Nano)},
			}},
			status:  "stopped",
			running: 0,
			total:   1,
			created: now,
		},
		{
			name:    "missing containers",
			inspect: InspectResult{Containers: []ContainerSummary{}},
			status:  "not-found",
			running: 0,
			total:   0,
		},
		{
			name: "malformed created falls back to zero",
			inspect: InspectResult{Containers: []ContainerSummary{
				{Name: "n1", Status: domain.ContainerStatusRunning, Created: "bad-timestamp"},
			}},
			status:  "running",
			running: 1,
			total:   1,
			created: time.Time{},
		},
		{
			name: "mixed becomes partial",
			inspect: InspectResult{Containers: []ContainerSummary{
				{Name: "n1", Status: domain.ContainerStatusRunning, Created: now.Format(time.RFC3339)},
				{Name: "n2", Status: domain.ContainerStatusStopped, Created: now.Format(time.RFC3339)},
			}},
			status:  "partial",
			running: 1,
			total:   2,
			created: now,
		},
	}

	svc := NewClusterStatusSnapshotService()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Build(context.Background(), ClusterStatusSnapshotRequest{Inspect: tt.inspect})
			require.NoError(t, err)
			assert.Equal(t, tt.status, res.Status)
			assert.Equal(t, tt.running, res.RunningNodes)
			assert.Equal(t, tt.total, res.TotalNodes)
			if tt.created.IsZero() {
				assert.True(t, res.Created.IsZero(), "created expected zero, got %v", res.Created)
				return
			}
			assert.Equal(t, tt.created.Unix(), res.Created.Unix())
		})
	}
}

func TestStatusSnapshot_Build_NilContext(t *testing.T) {
	t.Parallel()
	svc := NewClusterStatusSnapshotService()
	_, err := svc.Build(nil, ClusterStatusSnapshotRequest{})
	require.Error(t, err, "expected error for nil context")
}

func TestClusterStatusSnapshotService_Build_AllStatusTransitions(t *testing.T) {
	t.Parallel()

	svc := NewClusterStatusSnapshotService()

	runningContainer := ContainerSummary{Name: "c1", Status: domain.ContainerStatusRunning}
	stoppedContainer := ContainerSummary{Name: "c2", Status: domain.ContainerStatusStopped}
	unknownContainer := ContainerSummary{Name: "c3", Status: domain.ContainerStatusUnknown}

	tests := []struct {
		name           string
		containers     []ContainerSummary
		wantStatus     string
		wantRunning    int
		wantTotalNodes int
	}{
		{
			name:           "no containers returns not-found",
			containers:     nil,
			wantStatus:     string(ClusterStatusNotFound),
			wantTotalNodes: 0,
		},
		{
			name:           "all running returns running",
			containers:     []ContainerSummary{runningContainer, {Name: "c2", Status: domain.ContainerStatusRunning}},
			wantStatus:     string(ClusterStatusRunning),
			wantRunning:    2,
			wantTotalNodes: 2,
		},
		{
			name:           "all stopped returns stopped",
			containers:     []ContainerSummary{stoppedContainer, {Name: "c3", Status: domain.ContainerStatusStopped}},
			wantStatus:     string(ClusterStatusStopped),
			wantRunning:    0,
			wantTotalNodes: 2,
		},
		{
			name:           "mix of running and stopped returns partial",
			containers:     []ContainerSummary{runningContainer, stoppedContainer},
			wantStatus:     string(ClusterStatusPartial),
			wantRunning:    1,
			wantTotalNodes: 2,
		},
		{
			name:           "unknown status returns degraded",
			containers:     []ContainerSummary{unknownContainer},
			wantStatus:     string(ClusterStatusDegraded),
			wantRunning:    0,
			wantTotalNodes: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := ClusterStatusSnapshotRequest{
				Inspect: InspectResult{Containers: tt.containers},
			}
			result, err := svc.Build(context.Background(), req)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantRunning, result.RunningNodes)
			assert.Equal(t, tt.wantTotalNodes, result.TotalNodes)
		})
	}
}

func TestClusterStatusConstants_MatchExpectedValues(t *testing.T) {
	assert.Equal(t, "running", string(ClusterStatusRunning))
	assert.Equal(t, "stopped", string(ClusterStatusStopped))
	assert.Equal(t, "degraded", string(ClusterStatusDegraded))
	assert.Equal(t, "partial", string(ClusterStatusPartial))
	assert.Equal(t, "not-found", string(ClusterStatusNotFound))
}
