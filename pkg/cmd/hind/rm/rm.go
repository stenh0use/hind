package rm

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/cluster"
)

// DefaultDeleteTimeout is the default timeout for destroying a cluster
const DefaultDeleteTimeout = 2 * time.Minute

// NewCommand creates the cluster delete command
func NewCommand(logger *log.Logger) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "rm [cluster-name]",
		Short: "Remove a hind cluster",
		Long:  "Remove a hind cluster and delete all its resources",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var clusterName string
			if len(args) > 0 {
				clusterName = args[0]
			}
			return runE(cmd.Context(), logger, timeout, clusterName)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", DefaultDeleteTimeout, "Timeout for destroying the cluster")

	return cmd
}

func runE(ctx context.Context, logger *log.Logger, timeout time.Duration, clusterName string) error {
	// Check if this is the active cluster (before any changes)
	activeCluster, err := cluster.GetActiveCluster()
	if err != nil {
		logger.Debugf("Failed to get active cluster: %v", err)
	}

	// If no cluster name provided, use active cluster or fall back to "default"
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

	// Create cluster configuration
	clusterMgr, err := cluster.New(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	if err := clusterMgr.Delete(deleteCtx); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// If the deleted cluster was the active cluster, clear the active cluster setting
	if activeCluster == clusterName {
		if err := cluster.ClearActiveCluster(); err != nil {
			logger.Warnf("Failed to clear active cluster: %v", err)
		}
	}

	logger.Infof("cluster '%s' deleted successfully", clusterName)
	return nil
}
