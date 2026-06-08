package get

import (
	"context"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
)

// DefaultGetTimeout is the default timeout for getting a cluster
const DefaultGetTimeout = 2 * time.Minute

type clusterManager interface {
	Inspect(ctx context.Context) (orchestration.InspectResult, error)
}

type clusterManagerFactory func(logger *log.Logger, name string) (clusterManager, error)

var newClusterManager clusterManagerFactory = func(logger *log.Logger, name string) (clusterManager, error) {
	svc, err := cluster.NewClusterServices(logger, name)
	if err != nil {
		return nil, err
	}
	return svc.Orchestration, nil
}

// NewCommand creates the cluster delete command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	var timeout time.Duration

	command := &cobra.Command{
		Use:   "get [cluster-name]",
		Short: "Get a hind cluster details",
		Long:  "Get the details of a hind cluster and all it's resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, streams, timeout, args)
		},
	}

	command.Flags().DurationVar(&timeout, "timeout", DefaultGetTimeout, "Timeout for getting the state of the cluster")

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, timeout time.Duration, args []string) error {
	clusterName := args[0]

	// Create context with timeout
	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	manager, err := newClusterManager(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	state, err := manager.Inspect(getCtx)
	if err != nil {
		if orchestration.IsNotFound(err) {
			return fmt.Errorf("cluster '%s' not found", clusterName)
		}
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Print cluster information
	fmt.Fprintf(streams.Out, "---\nCluster: %s\n", clusterName)
	fmt.Fprintf(streams.Out, "Status: %s\n", domain.AggregateContainerStatus(state.Containers))
	fmt.Fprintf(streams.Out, "Network: %s\n", state.NetworkName)

	if len(state.Containers) > 0 {
		w := tabwriter.NewWriter(streams.Out, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "\nNODE\tTYPE\tSTATE")

		for _, node := range state.Containers {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				node.HostName,
				node.Image,
				string(node.Status),
			)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}
	}

	return nil
}
