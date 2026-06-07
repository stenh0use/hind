package start

import (
	"context"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/overrides"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

// DefaultStartTimeout is the default timeout for starting a cluster
const DefaultStartTimeout = 5 * time.Minute

// clusterStarter is the minimal interface required by runE.
type clusterStarter interface {
	Start(ctx context.Context, req orchestration.StartRequest) (domain.StartOutcome, error)
	Scale(ctx context.Context, targetClientCount int) error
}

// startServices bundles dependencies for runE.
type startServices struct {
	Orchestration clusterStarter
	Active        persistence.ActiveRepository
}

var newStartServicesFn = func(logger *log.Logger, clusterName string) (startServices, error) {
	svc, err := cluster.NewClusterServices(logger, clusterName)
	if err != nil {
		return startServices{}, err
	}
	return startServices{Orchestration: svc.Orchestration, Active: svc.Active}, nil
}

// flagpole holds all flags for the start command
type flagpole struct {
	timeout       time.Duration
	clients       int
	verbose       bool
	nomadVersion  string
	consulVersion string
	vaultVersion  string
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
	command.Flags().StringVar(&flags.nomadVersion, "nomad-version", "", "One-shot Nomad package version override for this start invocation")
	command.Flags().StringVar(&flags.consulVersion, "consul-version", "", "One-shot Consul package version override for this start invocation")
	command.Flags().StringVar(&flags.vaultVersion, "vault-version", "", "One-shot Vault package version override for this start invocation")

	return command
}

// checkDockerDaemonFn is the function used to verify Docker daemon accessibility.
// Tests may replace this variable to bypass the real Docker check.
var checkDockerDaemonFn = checkDockerDaemon

func configureLogging(logger *log.Logger, flags *flagpole, clusterName string) {
	if !flags.verbose {
		return
	}
	logger.Level = log.DebugLevel
	logger.Debug("Verbose mode enabled")
	logger.Debugf("Checking for existing cluster '%s'", clusterName)
}

func createStartServices(ctx context.Context, logger *log.Logger, clusterName string) (startServices, error) {
	logger.Debug("Checking Docker daemon accessibility")
	if err := checkDockerDaemonFn(ctx, logger); err != nil {
		return startServices{}, fmt.Errorf("Docker daemon is not accessible: %w", err)
	}

	svc, err := newStartServicesFn(logger, clusterName)
	if err != nil {
		return startServices{}, fmt.Errorf("failed to create cluster manager: %w", err)
	}

	return svc, nil
}

func startOrResumeCluster(cmd *cobra.Command, ctx context.Context, logger *log.Logger, svc startServices, flags *flagpole, resolved overrides.Set, clusterName string) (domain.StartOutcome, error) {
	req := orchestration.StartRequest{
		ClientCount:   flags.clients,
		NomadVersion:  resolved.Nomad.Value,
		ConsulVersion: resolved.Consul.Value,
		VaultVersion:  resolved.Vault.Value,
	}
	outcome, err := svc.Orchestration.Start(ctx, req)
	if err != nil {
		return domain.StartOutcomeCreated, fmt.Errorf("failed to start cluster %q: %w", clusterName, err)
	}

	// Post-start scale if resumed and --clients flag explicitly changed the count.
	if outcome == domain.StartOutcomeResumed && cmd.Flags().Changed("clients") {
		logger.Debugf("Client count change requested to: %d", flags.clients)
		if err := svc.Orchestration.Scale(ctx, flags.clients); err != nil {
			return domain.StartOutcomeCreated, fmt.Errorf("failed to scale cluster: %w", err)
		}
		return domain.StartOutcomeScaled, nil
	}

	return outcome, nil
}

func finalizeStartOutput(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, svc startServices, resolved overrides.Set, clusterName string, outcome domain.StartOutcome) {
	if err := svc.Active.SetActive(ctx, clusterName); err != nil {
		logger.Warnf("Failed to set active cluster: %v", err)
	}

	if outcome != domain.StartOutcomeAlreadyRunning && outcome != domain.StartOutcomeNoOp {
		displayConnectionInfo(streams, clusterName)
	}

	displayVersionOverrides(streams, resolved)
}

func runE(cmd *cobra.Command, ctx context.Context, logger *log.Logger, streams cmd.IOStreams, flags *flagpole, args []string) error {
	clusterName := cluster.ResolveClusterNameFromFS(ctx, args)

	configureLogging(logger, flags, clusterName)

	// Resolve and validate version overrides BEFORE touching Docker so invalid
	// input fails fast with no side effects. Cobra's Changed() distinguishes
	// "flag unset" from "flag explicitly set to empty".
	resolved, err := overrides.Resolve(overrides.FlagInputs{
		NomadVersion:  flags.nomadVersion,
		ConsulVersion: flags.consulVersion,
		VaultVersion:  flags.vaultVersion,
		NomadSet:      cmd.Flags().Changed("nomad-version"),
		ConsulSet:     cmd.Flags().Changed("consul-version"),
		VaultSet:      cmd.Flags().Changed("vault-version"),
	}, overrides.OSEnvLookup)
	if err != nil {
		return err
	}

	startCtx, cancel := context.WithTimeout(ctx, flags.timeout)
	defer cancel()

	svc, err := createStartServices(startCtx, logger, clusterName)
	if err != nil {
		return err
	}

	if flags.verbose {
		logger.Debugf("Resolved cluster name: %s", clusterName)
	}

	outcome, err := startOrResumeCluster(cmd, startCtx, logger, svc, flags, resolved, clusterName)
	if err != nil {
		return err
	}

	finalizeStartOutput(ctx, logger, streams, svc, resolved, clusterName, outcome)
	return nil
}

// displayVersionOverrides writes one header + one line per overridden package
// to streams.ErrOut, with `(from env)` attribution for env-sourced packages.
// When no package is overridden, writes nothing.
func displayVersionOverrides(streams cmd.IOStreams, r overrides.Set) {
	for _, line := range overrides.RenderLines(r) {
		fmt.Fprintln(streams.ErrOut, line)
	}
}

// checkDockerDaemon verifies the Docker daemon is accessible
func checkDockerDaemon(ctx context.Context, logger *log.Logger) error {
	client := dockercli.New(logger)
	_, err := client.ListContainers(ctx, []string{})
	return err
}

// displayConnectionInfo shows the user how to connect to the cluster services
func displayConnectionInfo(streams cmd.IOStreams, clusterName string) {
	fmt.Fprintln(streams.ErrOut, "Connection information:")
	fmt.Fprintf(streams.ErrOut, "  Nomad:  http://localhost:%d\n", domain.DefaultNomadPort)
	fmt.Fprintf(streams.ErrOut, "  Consul: http://localhost:%d\n", domain.DefaultConsulPort)
	fmt.Fprintf(streams.ErrOut, "  Vault:  http://localhost:%d\n", domain.DefaultVaultPort)
}
