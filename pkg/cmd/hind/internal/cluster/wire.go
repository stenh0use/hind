// Package cluster provides command-layer composition utilities shared by
// all hind subcommands. It owns the canonical wiring of cluster service
// dependencies so that individual command packages do not repeat that logic.
package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	persistencefs "github.com/stenh0use/hind/pkg/cluster/persistence/fs"
	dockerruntime "github.com/stenh0use/hind/pkg/cluster/runtime/docker"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

const (
	defaultStartTimeout = 30 * time.Second
	defaultPollInterval = time.Second
)

// ClusterServices is the composite returned by NewClusterServices. Commands
// consume Orchestration for lifecycle/scale/inspect/list operations and Active
// for active-cluster persistence.
type ClusterServices struct {
	Orchestration orchestration.Service
	Active        persistence.ActiveRepository
}

// NewClusterServices wires orchestration + active repository for a named
// cluster. It is the single composition seam shared by the start, list, get,
// rm, stop, and set commands.
func NewClusterServices(logger *log.Logger, clusterName string) (ClusterServices, error) {
	if logger == nil {
		return ClusterServices{}, fmt.Errorf("logger cannot be nil")
	}
	if err := domain.ValidateName(clusterName); err != nil {
		return ClusterServices{}, fmt.Errorf("invalid cluster name %q: %w", clusterName, err)
	}

	name := domain.Name(clusterName)
	client := dockercli.New(logger)
	rt := dockerruntime.New(client)

	repo, err := persistencefs.NewRepository()
	if err != nil {
		return ClusterServices{}, fmt.Errorf("wire: persistence repository: %w", err)
	}

	lcHandler, err := orchestration.NewLifecycleHandler(orchestration.LifecycleHandlerOptions{
		Name:         name,
		Repo:         repo,
		Active:       repo,
		Runtime:      rt,
		Logger:       logger,
		StartTimeout: defaultStartTimeout,
		PollInterval: defaultPollInterval,
	})
	if err != nil {
		return ClusterServices{}, fmt.Errorf("wire: lifecycle handler: %w", err)
	}

	scaleHandler, err := orchestration.NewScaleHandler(orchestration.ScaleHandlerOptions{
		Name:         name,
		Repo:         repo,
		Runtime:      rt,
		StartTimeout: defaultStartTimeout,
		PollInterval: defaultPollInterval,
	})
	if err != nil {
		return ClusterServices{}, fmt.Errorf("wire: scale handler: %w", err)
	}

	locker := persistencefs.NewLocker(repo.RootDir())

	orchSvc := orchestration.NewService(orchestration.Options{
		Lifecycle: lcHandler,
		Scale:     scaleHandler,
		Locker:    locker,
		Name:      name,
		Repo:      repo,
		Runtime:   rt,
	})

	return ClusterServices{Orchestration: orchSvc, Active: repo}, nil
}

// ResolveActive returns the persisted active cluster name, or "default" when
// no active cluster is set or when reading fails. It is the canonical default-
// resolution rule for commands that take an optional cluster argument.
func ResolveActive(ctx context.Context, active persistence.ActiveRepository) string {
	name, err := active.GetActive(ctx)
	if err != nil || name == "" {
		return "default"
	}
	return name
}

// ClearActiveIfMatch clears the active cluster only when the current active
// cluster equals name. No-op when name does not match or no active cluster is
// set; it never returns an error in those cases.
func ClearActiveIfMatch(ctx context.Context, active persistence.ActiveRepository, name string) error {
	current, err := active.GetActive(ctx)
	if err != nil || current != name {
		return nil
	}
	return active.ClearActive(ctx)
}
