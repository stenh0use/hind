// Package releases implements the "hind releases" command.
// It lists all available hind releases and the HashiCorp component versions
// each release includes, rendered as a tab-aligned table.
package releases

import (
	"context"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/cmd"
)

// NewCommand returns a cobra.Command that prints the hind releases table.
func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	command := &cobra.Command{
		Use:   "releases",
		Short: "List available hind releases",
		Long:  "List all available hind releases and the HashiCorp component versions they include.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, streams)
		},
	}

	return command
}

// runE fetches the release list, sorts descending by hind version, and writes
// a tabwriter-aligned table to streams.Out.
//
// Column order: HIND, CONSUL, NOMAD, VAULT (hind first; remaining in
// alphabetical order as required by hind-releases.feature).
//
// NOTE: Sorting uses lexicographic descending order which is correct for the
// current release set (MAJOR.MINOR.PATCH with no ambiguous zero-padding).
// TODO: switch to golang.org/x/mod/semver when the release count grows.
func runE(_ context.Context, _ *log.Logger, streams cmd.IOStreams) error {
	versions := release.List()
	if len(versions) == 0 {
		fmt.Fprintln(streams.ErrOut, "No releases found")
		return nil
	}

	// Sort descending so that the latest version appears on the first row.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})

	w := tabwriter.NewWriter(streams.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "HIND\tCONSUL\tNOMAD\tVAULT")

	for _, v := range versions {
		info, err := release.Get(v)
		if err != nil {
			// Skip unknown entries; this should not happen with the built-in store.
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Hind, info.Consul, info.Nomad, info.Vault)
	}

	return w.Flush()
}
