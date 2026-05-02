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
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

const (
	// DefaultBuildTimeout is the default timeout for building a single image
	DefaultBuildTimeout = 15 * time.Minute
)

// flagpole holds all flags for the build command
type flagpole struct {
	timeout time.Duration
}

func NewCommand(logger *log.Logger, streams cmd.IOStreams) *cobra.Command {
	flags := &flagpole{}

	command := &cobra.Command{
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
			return runE(cmd.Context(), logger, streams, flags, args)
		},
	}

	command.Flags().DurationVar(&flags.timeout, "timeout", DefaultBuildTimeout, "Timeout for building a single image")
	// TODO: add cache/file cleanup/etc flags

	return command
}

func runE(ctx context.Context, logger *log.Logger, streams cmd.IOStreams, flags *flagpole, args []string) error {
	target := args[0]

	var kinds []release.ImageKind

	if target == "all" {
		kinds = release.Images()
	} else {
		kinds = []release.ImageKind{release.ImageKind(target)}
	}

	client := dockercli.New(logger)

	for _, k := range kinds {
		buildCtx, cancel := context.WithTimeout(ctx, flags.timeout)
		err := func() error {
			defer cancel()

			logger.WithField("timeout", flags.timeout).Debug("Building image with timeout")
			fmt.Fprintf(streams.ErrOut, "Building %s image...\n", k)

			builder, err := image.NewBuilder(logger, client, k)
			if err != nil {
				return fmt.Errorf("failed to create builder for %s: %w", k, err)
			}

			if err := builder.BuildImage(buildCtx); err != nil {
				return fmt.Errorf("failed to build %s image: %w", k, err)
			}

			fmt.Fprintf(streams.ErrOut, "Successfully built %s image\n", k)
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
