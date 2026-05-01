package rm

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

// DefaultDeleteTimeout is the default timeout for destroying a cluster
const DefaultDeleteTimeout = 2 * time.Minute

// clusterDeleter is the minimal interface required by runE to delete a cluster.
// It is satisfied by *cluster.Manager and can be replaced in tests to avoid Docker.
type clusterDeleter interface {
	Delete(ctx context.Context) error
}

// newClusterManagerFn is the factory used to create a clusterDeleter for a given cluster
// name. Tests may replace this variable to inject a stub without a real Docker daemon.
var newClusterManagerFn = func(logger *log.Logger, clusterName string) (clusterDeleter, error) {
	return cluster.New(logger, clusterName, dockercli.New(logger))
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
			var clusterName string
			if len(args) > 0 {
				clusterName = args[0]
			}
			return runE(cmd.Context(), logger, streams, timeout, clusterName)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultDeleteTimeout, "Timeout for destroying the cluster")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration, clusterName string) error {
	// Check if this is the active cluster (before any changes)
	activeCluster, err := cluster.GetActiveCluster()
	if err != nil {
		logger.Debugf("Failed to get active cluster: %v", err)
	}

	// If no cluster name provided, use active cluster or fall back to "default".
	if clusterName == "" {
		if activeCluster == "" {
			clusterName = "default"
		} else {
			clusterName = activeCluster
			logger.Debugf("Using active cluster: %s", clusterName)
		}
	}

	// Create context with timeout
	deleteCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create cluster manager via factory seam (replaceable in tests)
	clusterMgr, err := newClusterManagerFn(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	if err := clusterMgr.Delete(deleteCtx); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// If the deleted cluster was the active cluster, clear the active cluster setting.
	// Clearing removes the active file entirely, leaving no active cluster (empty string).
	// All command resolution paths treat an empty/missing active cluster as "default", so
	// clearing is semantically equivalent to resetting to "default" without writing that
	// literal value to disk (which would conflict with a real cluster named "default").
	if activeCluster == clusterName {
		if err := cluster.ClearActiveCluster(); err != nil {
			logger.Warnf("Failed to clear active cluster: %v", err)
		}
	}

	fmt.Fprintf(streams.ErrOut, "Cluster '%s' deleted successfully\n", clusterName)
	return nil
}
