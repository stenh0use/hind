package get

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/cluster"
)

// DefaultGetTimeout is the default timeout for getting a cluster
const DefaultGetTimeout = 2 * time.Minute

// NewCommand creates the cluster delete command
func NewCommand(logger *log.Logger) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:   "get [cluster-name]",
		Short: "Get a hind cluster details",
		Long:  "Get the details of a hind cluster and all it's resources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, timeout, args)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", DefaultGetTimeout, "Timeout for getting the state of the cluster")

	return cmd
}

func runE(ctx context.Context, logger *log.Logger, timeout time.Duration, args []string) error {
	clusterName := args[0]

	// Create context with timeout
	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create cluster configuration
	cluster, err := cluster.New(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	state, err := cluster.Get(getCtx)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Print cluster information
	fmt.Printf("---\nCluster: %s\n", cluster.Config().Name)
	fmt.Printf("Status: created\n")
	fmt.Printf("Network: %s\n", state.Network.Name)

	if len(state.Containers) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "\nNODE\tTYPE\tSTATE\tPORTS")

		for _, node := range state.Containers {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				node.HostName,
				node.Image,
				node.Status,
				node.Ports,
			)
		}
		w.Flush()
	}

	return nil
}
