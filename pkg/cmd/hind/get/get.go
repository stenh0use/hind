package get

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/provider"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

// DefaultGetTimeout is the default timeout for getting a cluster
const DefaultGetTimeout = 2 * time.Minute

type clusterManager interface {
	Get(ctx context.Context) (*cluster.ClusterInfo, error)
}

type clusterManagerFactory func(logger *log.Logger, name string) (clusterManager, error)

var newClusterManager clusterManagerFactory = func(logger *log.Logger, name string) (clusterManager, error) {
	return cluster.New(logger, name, dockercli.New(logger))
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

	state, err := manager.Get(getCtx)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Print cluster information
	fmt.Fprintf(streams.Out, "---\nCluster: %s\n", clusterName)
	fmt.Fprintf(streams.Out, "Status: %s\n", aggregateStatus(state))
	fmt.Fprintf(streams.Out, "Network: %s\n", state.Network.Name)

	if len(state.Containers) > 0 {
		w := tabwriter.NewWriter(streams.Out, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "\nNODE\tTYPE\tSTATE\tPORTS")

		for _, node := range state.Containers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				node.HostName,
				node.Image,
				node.Status,
				formatPorts(node.Ports),
			)
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("failed to flush output: %w", err)
		}
	}

	return nil
}

func aggregateStatus(state *cluster.ClusterInfo) string {
	if len(state.Containers) == 0 {
		return provider.NA.String()
	}

	hasRunning := false
	hasStopped := false
	hasError := false
	allRunning := true
	allStopped := true

	for _, container := range state.Containers {
		switch strings.ToLower(container.Status) {
		case provider.Running.String():
			hasRunning = true
			allStopped = false
		case provider.Stopped.String():
			hasStopped = true
			allRunning = false
		default:
			hasError = true
			allRunning = false
			allStopped = false
		}
	}

	switch {
	case allRunning:
		return provider.Running.String()
	case allStopped:
		return provider.Stopped.String()
	case hasError || (hasRunning && hasStopped):
		return provider.Error.String()
	default:
		return provider.Error.String()
	}
}

func formatPorts(ports []string) string {
	if len(ports) == 0 {
		return "-"
	}

	return strings.Join(ports, ", ")
}
