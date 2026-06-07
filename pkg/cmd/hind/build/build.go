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
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/overrides"
	"github.com/stenh0use/hind/pkg/provider/dockercli"
)

const (
	// DefaultBuildTimeout is the default timeout for building a single image
	DefaultBuildTimeout = 15 * time.Minute
)

type flagpole struct {
	timeout           time.Duration
	baseVersion       string
	consulVersion     string
	nomadVersion      string
	vaultVersion      string
	containerdVersion string
	dockerceVersion   string
	cnipluginsVersion string
	ciliumVersion     string
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
				return err
			}
			return cobra.OnlyValidArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd, logger, streams, flags, args)
		},
	}

	command.Flags().DurationVar(&flags.timeout, "timeout", DefaultBuildTimeout, "Timeout for building a single image")
	command.Flags().StringVar(&flags.baseVersion, "base-version", "", "Override base package version for this build")
	command.Flags().StringVar(&flags.consulVersion, "consul-version", "", "Override consul package version for this build")
	command.Flags().StringVar(&flags.nomadVersion, "nomad-version", "", "Override nomad package version for this build")
	command.Flags().StringVar(&flags.vaultVersion, "vault-version", "", "Override vault package version for this build")
	command.Flags().StringVar(&flags.containerdVersion, "containerd-version", "", "Override containerd package version for this build")
	command.Flags().StringVar(&flags.dockerceVersion, "dockerce-version", "", "Override dockerce package version for this build")
	command.Flags().StringVar(&flags.cnipluginsVersion, "cniplugins-version", "", "Override cniplugins package version for this build")
	command.Flags().StringVar(&flags.ciliumVersion, "cilium-version", "", "Override cilium package version for this build")
	return command
}

// packageVersionOverrides builds the per-package release.Overrides for this
// invocation. The three HashiCorp services (Consul, Nomad, Vault) are sourced
// from the already-resolved overrides.Set (flag > env > default). The
// remaining build-infrastructure packages are sourced directly from flags
// because env-var support is out of scope for W-045.
func packageVersionOverrides(flags *flagpole, resolved overrides.Set) release.Overrides {
	return release.Overrides{
		Base:       strings.TrimSpace(flags.baseVersion),
		Consul:     resolved.Consul.Value,
		Nomad:      resolved.Nomad.Value,
		Vault:      resolved.Vault.Value,
		Containerd: strings.TrimSpace(flags.containerdVersion),
		DockerCe:   strings.TrimSpace(flags.dockerceVersion),
		CniPlugins: strings.TrimSpace(flags.cnipluginsVersion),
		Cilium:     strings.TrimSpace(flags.ciliumVersion),
	}
}

func runE(cobraCmd *cobra.Command, logger *log.Logger, streams cmd.IOStreams, flags *flagpole, args []string) error {
	// Resolve and validate HashiCorp version overrides BEFORE touching Docker
	// so invalid input fails fast with no side effects. Cobra's Changed()
	// distinguishes "flag unset" from "flag explicitly set to empty".
	resolved, err := overrides.Resolve(overrides.FlagInputs{
		NomadVersion:  flags.nomadVersion,
		ConsulVersion: flags.consulVersion,
		VaultVersion:  flags.vaultVersion,
		NomadSet:      cobraCmd.Flags().Changed("nomad-version"),
		ConsulSet:     cobraCmd.Flags().Changed("consul-version"),
		VaultSet:      cobraCmd.Flags().Changed("vault-version"),
	}, overrides.OSEnvLookup)
	if err != nil {
		return err
	}

	target := args[0]
	var kinds []image.ImageKind
	if target == "all" {
		kinds = image.Images()
	} else {
		kinds = []image.ImageKind{image.ImageKind(target)}
	}

	rel := release.Versions()
	if overrides := packageVersionOverrides(flags, resolved); overrides.HasAny() {
		rel = rel.WithOverrides(overrides)
	}

	// Emit attribution before any build output so users see the effective
	// versions for this invocation.
	for _, line := range overrides.RenderLines(resolved) {
		fmt.Fprintln(streams.ErrOut, line)
	}

	ctx := cobraCmd.Context()
	client := dockercli.New(logger)
	for _, k := range kinds {
		buildCtx, cancel := context.WithTimeout(ctx, flags.timeout)
		err := func() error {
			defer cancel()
			fmt.Fprintf(streams.ErrOut, "Building %s image...\n", k)
			builder, err := image.NewBuilderWithRelease(logger, client, k, rel)
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
