package version

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/stenh0use/hind/pkg/build/release"
)

// Version is the CLI version. Set at build time with -ldflags.
var Version = release.Latest().Hind

// Commit is the git commit hash. Set at build time with -ldflags.
var Commit = ""

// isRelease returns true if the version string looks like a release (vMAJOR.MINOR.PATCH)
func isRelease(version string) bool {
	return strings.HasPrefix(version, "v") && len(strings.Split(version, ".")) == 3
}

// DisplayVersion returns the version string for display
func DisplayVersion() string {
	base := fmt.Sprintf("hind %s %s %s/%s", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if !isRelease(Version) && Commit != "" {
		base += fmt.Sprintf(" (commit %s)", Commit)
	}
	return base
}

// NewCommand returns a new cobra.Command for version
func NewCommand(logger *log.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the hind CLI version",
		Long:  "Prints the hind CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), DisplayVersion())
		},
	}
	return cmd
}
