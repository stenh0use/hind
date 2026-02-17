package set

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cluster"
	"github.com/stenh0use/hind/pkg/cmd"
)

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

			// Set the active cluster
			if err := cluster.SetActiveCluster(clusterName); err != nil {
				return fmt.Errorf("failed to set active cluster: %w", err)
			}

			fmt.Fprintf(streams.ErrOut, "Active cluster profile set to '%s'\n", clusterName)
			return nil
		},
	}

	return command
}
