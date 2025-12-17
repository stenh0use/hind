package set

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/cluster"
)

// NewCommand creates the set command with subcommands
func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set hind configuration options",
		Long:  "Set various hind configuration options like the active cluster profile",
	}

	// Add subcommands
	cmd.AddCommand(newProfileCommand(logger))

	return cmd
}

// newProfileCommand creates the 'set profile' subcommand
func newProfileCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile [cluster-name]",
		Short: "Set the active cluster profile",
		Long:  "Set the active cluster profile to the specified cluster name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := args[0]

			// Set the active cluster
			if err := cluster.SetActiveCluster(clusterName); err != nil {
				return fmt.Errorf("failed to set active cluster: %w", err)
			}

			logger.Infof("Active cluster profile set to '%s'", clusterName)
			return nil
		},
	}

	return cmd
}
