package hind

import (
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/build"
	"github.com/stenh0use/hind/pkg/cmd/hind/get"
	"github.com/stenh0use/hind/pkg/cmd/hind/list"
	"github.com/stenh0use/hind/pkg/cmd/hind/rm"
	"github.com/stenh0use/hind/pkg/cmd/hind/set"
	"github.com/stenh0use/hind/pkg/cmd/hind/start"
	"github.com/stenh0use/hind/pkg/cmd/hind/stop"
	"github.com/stenh0use/hind/pkg/cmd/hind/version"
)

// NewCommand returns a new cobra.Command implementing the root command for hind
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "hind",
		Short: "hind is a tool for running hashistack clusters in docker",
		Long: strings.Join([]string{
			"hind allows you to define and run multi-node hashistack",
			"(nomad, consul, vault) based clusters in docker for testing",
			"and development.",
		}, " "),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.DisplayVersion(),
	}

	// Set IO streams for the root command
	rootCmd.SetOut(streams.Out)
	rootCmd.SetErr(streams.ErrOut)

	// Add subcommands
	rootCmd.AddCommand(build.NewCommand(logger, streams))
	rootCmd.AddCommand(get.NewCommand(logger, streams))
	rootCmd.AddCommand(list.NewCommand(logger, streams))
	rootCmd.AddCommand(rm.NewCommand(logger, streams))
	rootCmd.AddCommand(set.NewCommand(logger, streams))
	rootCmd.AddCommand(start.NewCommand(logger, streams))
	rootCmd.AddCommand(stop.NewCommand(logger, streams))
	rootCmd.AddCommand(version.NewCommand(logger, streams))

	return rootCmd
}
