package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

type clusterStatusSnapshotService struct{}

// NewClusterStatusSnapshotService constructs a cluster status snapshot service.
func NewClusterStatusSnapshotService() ClusterStatusSnapshotService {
	return &clusterStatusSnapshotService{}
}

func (s *clusterStatusSnapshotService) Build(ctx context.Context, req ClusterStatusSnapshotRequest) (ClusterStatusSnapshotResult, error) {
	if ctx == nil {
		return ClusterStatusSnapshotResult{}, fmt.Errorf("context cannot be nil")
	}

	result := ClusterStatusSnapshotResult{TotalNodes: len(req.Inspect.Containers)}
	if len(req.Inspect.Containers) == 0 {
		result.Status = string(ClusterStatusNotFound)
		return result, nil
	}

	agg := domain.AggregateContainerStatus(req.Inspect.Containers)
	result.Status = string(agg)

	for _, container := range req.Inspect.Containers {
		if container.Status == domain.ContainerStatusRunning {
			result.RunningNodes++
		}
	}

	created, ok := oldestCreated(req.Inspect)
	if ok {
		result.Created = created
	}

	return result, nil
}

func oldestCreated(inspect InspectResult) (time.Time, bool) {
	oldestTime := time.Now()
	foundCreated := false
	for _, container := range inspect.Containers {
		if created, parseErr := parseCreatedTime(container.Created); parseErr == nil {
			if !foundCreated || created.Before(oldestTime) {
				oldestTime = created
				foundCreated = true
			}
		}
	}
	return oldestTime, foundCreated
}

func parseCreatedTime(created string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05 -0700 MST",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, created); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", created)
}
