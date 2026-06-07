package set

import (
	"context"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
)

type clusterSetter interface {
	List(ctx context.Context) (orchestration.ListResult, error)
	SetActive(ctx context.Context, name string) error
}

var newClusterSetterFn = func(logger *log.Logger) (clusterSetter, error) {
	svc, err := cluster.NewClusterServices(logger, "default")
	if err != nil {
		return nil, err
	}
	return setSetter{orch: svc.Orchestration, active: svc.Active}, nil
}

type setSetter struct {
	orch   orchestration.Service
	active persistence.ActiveRepository
}

func (s setSetter) List(ctx context.Context) (orchestration.ListResult, error) {
	return s.orch.List(ctx)
}

func (s setSetter) SetActive(ctx context.Context, name string) error {
	return s.active.SetActive(ctx, name)
}

var lookupProfileFn = func(ctx context.Context, svc clusterSetter, clusterName string) (bool, error) {
	clusters, err := svc.List(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, existing := range clusters.Names {
		if strings.EqualFold(existing, clusterName) {
			return true, nil
		}
	}
	return false, nil
}

// NewCommand creates the set command with subcommands
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	command := &cobra.Command{
		Use:   "set",
		Short: "Set hind configuration options",
		Long:  "Set various hind configuration options like the active cluster profile",
	}

	// Add subcommands
	command.AddCommand(newProfileCommand(logger, streams))

	return command
}

// newProfileCommand creates the 'set profile' subcommand
func newProfileCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	command := &cobra.Command{
		Use:   "profile [cluster-name]",
		Short: "Set the active cluster profile",
		Long:  "Set the active cluster profile to the specified cluster name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := args[0]

			svc, err := newClusterSetterFn(logger)
			if err != nil {
				return fmt.Errorf("failed to create cluster service: %w", err)
			}

			exists, err := lookupProfileFn(cmd.Context(), svc, clusterName)
			if err != nil {
				return fmt.Errorf("failed to lookup cluster %q: %w", clusterName, err)
			}
			if !exists {
				return fmt.Errorf("cluster %q not found", clusterName)
			}

			if err := svc.SetActive(cmd.Context(), clusterName); err != nil {
				return fmt.Errorf("failed to set active cluster: %w", err)
			}

			fmt.Fprintf(streams.Out, "Active cluster profile set to '%s'\n", clusterName)
			return nil
		},
	}

	return command
}
