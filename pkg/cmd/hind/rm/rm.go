package rm

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
)

// DefaultDeleteTimeout is the default timeout for destroying a cluster
const DefaultDeleteTimeout = 2 * time.Minute

// clusterDeleter is the minimal interface required by runE to delete a cluster.
type clusterDeleter interface {
	Delete(ctx context.Context) error
}

// rmServices bundles dependencies for runE.
type rmServices struct {
	Orchestration clusterDeleter
	Active        persistence.ActiveRepository
}

// newClusterManagerFn is the factory used to create rmServices for a given cluster
// name. Tests may replace this variable to inject a stub without a real Docker daemon.
var newClusterManagerFn = func(logger *log.Logger, clusterName string) (rmServices, error) {
	svc, err := cluster.NewClusterServices(logger, clusterName)
	if err != nil {
		return rmServices{}, err
	}
	return rmServices{Orchestration: svc.Orchestration, Active: svc.Active}, nil
}

// NewCommand creates the cluster delete command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	var timeout time.Duration

	command := &cobra.Command{
		Use:   "rm [cluster-name]",
		Short: "Remove a hind cluster",
		Long:  "Remove a hind cluster and delete all its resources",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, streams, timeout, args)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultDeleteTimeout, "Timeout for destroying the cluster")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration, args []string) error {
	clusterName := cluster.ResolveClusterNameFromFS(ctx, args)

	// Create context with timeout.
	deleteCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	svc, err := newClusterManagerFn(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	if err := svc.Orchestration.Delete(deleteCtx); err != nil {
		if orchestration.IsNotFound(err) {
			return fmt.Errorf("cluster '%s' not found", clusterName)
		}
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Seam enforces: clear active only when deleted name matches current active.
	if err := cluster.ClearActiveIfMatch(ctx, svc.Active, clusterName); err != nil {
		logger.Warnf("Failed to clear active cluster: %v", err)
	}

	fmt.Fprintf(streams.ErrOut, "Cluster '%s' deleted successfully\n", clusterName)
	return nil
}
