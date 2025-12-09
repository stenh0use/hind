package hind

import (
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

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
func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
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
	// Add subcommands
	cmd.AddCommand(build.NewCommand(logger))
	cmd.AddCommand(get.NewCommand(logger))
	cmd.AddCommand(list.NewCommand(logger))
	cmd.AddCommand(rm.NewCommand(logger))
	cmd.AddCommand(set.NewCommand(logger))
	cmd.AddCommand(start.NewCommand(logger))
	cmd.AddCommand(stop.NewCommand(logger))
	cmd.AddCommand(version.NewCommand(logger))
	return cmd
}
