package stop

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

type clusterStopper interface {
	ConfigFileExists() bool
	StopWithOptions(ctx context.Context, opts cluster.StopOptions) (cluster.StopResult, error)
}

var getActiveClusterFn = cluster.GetActiveCluster
var newClusterManagerFn = func(logger *log.Logger, clusterName string) (clusterStopper, error) {
	return cluster.New(logger, clusterName, dockercli.New(logger))
}

// DefaultStopTimeout is the default timeout for stopping a cluster
const DefaultStopTimeout = 30 * time.Second

// NewCommand creates the cluster stop command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	var timeout time.Duration
	var force bool
	var verbose bool

	command := &cobra.Command{
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
			return runE(cmd.Context(), logger, streams, timeout, force, verbose, clusterName)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultStopTimeout, "Timeout for stopping the cluster")
	command.Flags().BoolVar(&force, "force", false, "Force stop running containers immediately")
	command.Flags().BoolVar(&verbose, "verbose", false, "Show detailed stop progress")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration, force bool, verbose bool, clusterName string) error {
	// Get active cluster (for informational purposes only)
	activeCluster, err := getActiveClusterFn()
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
	clusterMgr, err := newClusterManagerFn(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Check if cluster config exists
	if !clusterMgr.ConfigFileExists() {
		return fmt.Errorf("cluster '%s' not found", clusterName)
	}

	if verbose {
		fmt.Fprintf(streams.ErrOut, "Checking cluster '%s' status\n", clusterName)
	}

	result, err := clusterMgr.StopWithOptions(stopCtx, cluster.StopOptions{Force: force, Verbose: verbose})
	if verbose {
		for _, line := range result.VerboseLines {
			fmt.Fprintf(streams.ErrOut, "%s\n", line)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	for _, name := range result.Failures {
		fmt.Fprintf(streams.ErrOut, "Failed to stop container '%s'\n", name)
	}

	if result.FailedPreStopCount > 0 {
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' stopped (some containers were already failed)\n", clusterName)
		return nil
	}
	if result.AlreadyStopped() {
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' is already stopped\n", clusterName)
		return nil
	}
	if result.FailedCount > 0 {
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' partially stopped\n", clusterName)
		return fmt.Errorf("failed to stop %d container(s)", result.FailedCount)
	}
	if force {
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' force stopped\n", clusterName)
		return nil
	}

	fmt.Fprintf(streams.ErrOut, "Cluster '%s' stopped successfully\n", clusterName)
	return nil
}
