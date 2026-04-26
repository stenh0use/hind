package start

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
)

// DefaultStartTimeout is the default timeout for starting a cluster
const DefaultStartTimeout = 5 * time.Minute

// flagpole holds all flags for the start command
type flagpole struct {
	timeout time.Duration
	clients int
	verbose bool
}

// NewCommand creates the cluster start command
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}

	command := &cobra.Command{
		Use:   "start [cluster-name]",
		Short: "Start or create a hind cluster",
		Long:  "Start or create a hind cluster",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, cmd.Context(), logger, streams, flags, args)
		},
	}

	command.Flags().DurationVar(&flags.timeout, "timeout", DefaultStartTimeout, "Timeout for starting the cluster")
	command.Flags().IntVar(&flags.clients, "clients", 1, "Number of client nodes to create")
	command.Flags().BoolVar(&flags.verbose, "verbose", false, "Enable verbose output")

	return command
}

func runE(cmd *cobra.Command, ctx context.Context, logger *log.Logger, streams cmd.IOStreams, flags *flagpole, args []string) error {
	// Get cluster name from args or use default
	var clusterName string
	if len(args) > 0 {
		clusterName = args[0]
	}

	// If no cluster name provided, try to get active cluster, fall back to "default"
	if clusterName == "" {
		activeCluster, err := cluster.GetActiveCluster()
		if err != nil || activeCluster == "" {
			logger.Debugf("Failed to get active cluster")
			clusterName = "default"
		} else {
			clusterName = activeCluster
			logger.Debugf("Using active cluster: %s", clusterName)
		}
	}

	// Set log level based on verbose flag
	if flags.verbose {
		logger.Level = log.DebugLevel
		logger.Debug("Verbose mode enabled")
		logger.Debugf("Checking for existing cluster '%s'", clusterName)
	}

	// Create context with timeout
	startCtx, cancel := context.WithTimeout(ctx, flags.timeout)
	defer cancel()

	// Check if Docker daemon is accessible first
	logger.Debug("Checking Docker daemon accessibility")
	if err := checkDockerDaemon(startCtx, logger); err != nil {
		return fmt.Errorf("Docker daemon is not accessible: %w", err)
	}

	// Create cluster manager
	mgr, err := cluster.New(logger, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Start the cluster first (this loads config if exists, or uses defaults for new clusters)
	// For new clusters, we need to set client count before starting
	if !mgr.ConfigFileExists() && flags.clients != 1 {
		if err := mgr.SetClientCount(startCtx, flags.clients); err != nil {
			return fmt.Errorf("failed to set client count: %w", err)
		}
	}

	// Start the cluster (handles create, resume, and idempotent cases)
	result, err := mgr.Start(startCtx)
	if err != nil {
		return fmt.Errorf("failed to start cluster %q: %w", clusterName, err)
	}

	// If --clients flag was explicitly set for existing cluster, scale it
	if result == cluster.StartResultResumed && cmd.Flags().Changed("clients") {
		currentClientCount := mgr.CountClientNodes()
		if flags.clients != currentClientCount {
			logger.Debugf("Client count change requested: %d -> %d", currentClientCount, flags.clients)
			if err := mgr.Scale(startCtx, flags.clients); err != nil {
				return fmt.Errorf("failed to scale cluster: %w", err)
			}
		}
	}

	// Set this cluster as the active cluster
	if err := cluster.SetActiveCluster(clusterName); err != nil {
		logger.Warnf("Failed to set active cluster: %v", err)
		// Don't fail the command if we can't set the active cluster
	}

	// Display connection information only for newly created or resumed clusters
	if result != cluster.StartResultAlreadyRunning {
		displayConnectionInfo(streams, clusterName)
	}
	return nil
}

// checkDockerDaemon verifies the Docker daemon is accessible
func checkDockerDaemon(ctx context.Context, logger *log.Logger) error {
	// Create a temporary manager to test Docker connectivity
	// This is a lightweight check before we do any real work
	tempMgr, err := cluster.New(logger, "temp-check")
	if err != nil {
		return err
	}

	// Try to list containers to verify Docker daemon is accessible
	_, err = tempMgr.Provider().ListContainers(ctx, []string{})
	return err
}

// displayConnectionInfo shows the user how to connect to the cluster services
func displayConnectionInfo(streams cmd.IOStreams, clusterName string) {
	fmt.Fprintln(streams.ErrOut, "Connection information:")
	fmt.Fprintln(streams.ErrOut, "  Nomad:  http://localhost:4646")
	fmt.Fprintln(streams.ErrOut, "  Consul: http://localhost:8500")
	fmt.Fprintln(streams.ErrOut, "  Vault:  http://localhost:8200")
}
