package cluster

import (
	"context"
	"time"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

// InspectResult is a runtime-neutral typed inspect response, re-exported from
// pkg/cluster/domain for ergonomic use by status snapshot consumers.
type InspectResult = domain.InspectResult

// ContainerSummary is a runtime-neutral typed container descriptor.
type ContainerSummary = domain.ContainerSummary

// ClusterStatusSnapshotService derives typed status rows for cluster list rendering adapters.
type ClusterStatusSnapshotService interface {
	Build(ctx context.Context, req ClusterStatusSnapshotRequest) (ClusterStatusSnapshotResult, error)
}

// ClusterStatusSnapshotRequest configures snapshot generation from inspect data.
type ClusterStatusSnapshotRequest struct {
	Inspect InspectResult
}

// ClusterStatusSnapshotResult contains derived status data for a single cluster.
type ClusterStatusSnapshotResult struct {
	Status       string
	RunningNodes int
	TotalNodes   int
	Created      time.Time
}

// Re-export domain cluster status constants so that cluster-package tests and
// consumers can reference them without a qualified import.
const (
	ClusterStatusRunning  = domain.ClusterStatusRunning
	ClusterStatusStopped  = domain.ClusterStatusStopped
	ClusterStatusDegraded = domain.ClusterStatusDegraded
	ClusterStatusPartial  = domain.ClusterStatusPartial
	ClusterStatusNotFound = domain.ClusterStatusNotFound
)
