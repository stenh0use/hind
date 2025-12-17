package stop

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/cluster"
)

// DefaultStopTimeout is the default timeout for stopping a cluster
const DefaultStopTimeout = 30 * time.Second

// NewCommand creates the cluster stop command
func NewCommand(logger *log.Logger) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "stop [cluster-name]",
		Short: "Stop a hind cluster",
		Long: `Stop all containers in a hind cluster without deleting configuration.
The cluster can be resumed later with 'hind start'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var clusterName string
			if len(args) > 0 {
				clusterName = args[0]
			}
			return runE(cmd.Context(), logger, timeout, clusterName)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", DefaultStopTimeout, "Timeout for stopping the cluster")

	return cmd
}

func runE(ctx context.Context, logger *log.Logger, timeout time.Duration, clusterName string) error {
	// Get active cluster (for informational purposes only)
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
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create cluster manager
	clusterMgr, err := cluster.New(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Check if cluster config exists
	if !clusterMgr.ConfigFileExists() {
		return fmt.Errorf("cluster '%s' not found", clusterName)
	}

	// Execute stop operation
	if err := clusterMgr.Stop(stopCtx); err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	// Note: Unlike delete, we do NOT modify active cluster setting
	// The stopped cluster remains the active cluster for future start commands

	logger.Infof("cluster '%s' stopped successfully", clusterName)
	return nil
}
