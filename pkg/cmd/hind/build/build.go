// Package build implements the `build` command
package build

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/stenh0use/hind/pkg/build/image"
	"github.com/stenh0use/hind/pkg/build/release"
)

const (
	// DefaultBuildTimeout is the default timeout for building a single image
	DefaultBuildTimeout = 15 * time.Minute
)

func NewCommand(logger *log.Logger) *cobra.Command {
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:       fmt.Sprintf("build [%s]", strings.Join(image.BuildTargets(), "|")),
		Short:     "Build container images",
		Long:      "Build one or more hind container images. Use 'all' to build all images.",
		ValidArgs: image.BuildTargets(),
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return fmt.Errorf("accepts 1 arg, received %s", args)
			}
			return cobra.OnlyValidArgs(cmd, args)
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), logger, timeout, args)
		},
	}

	cmd.Flags().DurationVar(&timeout, "timeout", DefaultBuildTimeout, "Timeout for building a single image")
	// TODO: add cache/file cleanup/etc flags

	return cmd
}

func runE(ctx context.Context, logger *log.Logger, timeout time.Duration, args []string) error {
	target := args[0]

	var kinds []release.ImageKind

	if target == "all" {
		kinds = release.Images()
	} else {
		kinds = []release.ImageKind{release.ImageKind(target)}
	}

	for _, k := range kinds {
		// For single image build, use the specified timeout
		buildCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		logger.WithField("timeout", timeout).Debug("Building image with timeout")
		builder, err := image.NewBuilder(logger, k)
		if err != nil {
			return err
		}

		if err := builder.BuildImage(buildCtx); err != nil {
			return err
		}
	}

	return nil
}
