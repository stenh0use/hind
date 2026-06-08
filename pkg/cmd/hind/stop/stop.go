package stop

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

type clusterStopper interface {
	StopWithOptions(ctx context.Context, opts orchestration.StopOptions) (orchestration.StopResult, error)
}

type stopServices struct {
	Orchestration clusterStopper
	Active        persistence.ActiveRepository
}

var newStopServicesFn = func(logger *log.Logger, clusterName string) (stopServices, error) {
	svc, err := cluster.NewClusterServices(logger, clusterName)
	if err != nil {
		return stopServices{}, err
	}
	return stopServices{Orchestration: svc.Orchestration, Active: svc.Active}, nil
}

// DefaultStopTimeout is the default timeout for stopping a cluster
const DefaultStopTimeout = 30 * time.Second

// NewCommand creates the cluster stop command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	var timeout time.Duration
	var force bool

	command := &cobra.Command{
		Use:   "stop [cluster-name]",
		Short: "Stop a hind cluster",
		Long: `Stop all containers in a hind cluster without deleting configuration.
The cluster can be resumed later with 'hind start'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, streams, timeout, force, args)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultStopTimeout, "Timeout for stopping the cluster")
	command.Flags().BoolVar(&force, "force", false, "Force stop running containers immediately")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration, force bool, args []string) error {
	clusterName := cluster.ResolveClusterNameFromFS(ctx, args)

	// Create context with timeout
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	svc, err := newStopServicesFn(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	result, err := svc.Orchestration.StopWithOptions(stopCtx, orchestration.StopOptions{Force: force})
	if err != nil {
		if orchestration.IsNotFound(err) {
			return fmt.Errorf("cluster '%s' not found", clusterName)
		}
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	for _, name := range result.StoppedContainers {
		fmt.Fprintf(streams.ErrOut, "Stopped container '%s'\n", name)
	}

	for _, name := range result.Failures {
		fmt.Fprintf(streams.ErrOut, "Failed to stop container '%s'\n", name)
	}

	switch result.Outcome {
	case orchestration.StopOutcomeDegradedPreFailed:
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' stopped (some containers were already failed)\n", clusterName)
		return nil
	case orchestration.StopOutcomeAlreadyStopped:
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' is already stopped\n", clusterName)
		return nil
	case orchestration.StopOutcomePartialFailure:
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' partially stopped\n", clusterName)
		return fmt.Errorf("failed to stop %d container(s)", result.FailedCount)
	case orchestration.StopOutcomeSuccess:
		if force {
			fmt.Fprintf(streams.ErrOut, "Cluster '%s' force stopped\n", clusterName)
			return nil
		}
		fmt.Fprintf(streams.ErrOut, "Cluster '%s' stopped successfully\n", clusterName)
		return nil
	default:
		return fmt.Errorf("failed to stop cluster: unknown stop outcome %q", result.Outcome)
	}
}
