// Package image defines Docker image specifications and build configurations
// for hind container images. It manages image metadata, dependencies, and
// build arguments for HashiCorp service containers.
package image

import (
	"fmt"
	"strings"

	"github.com/stenh0use/hind/pkg/build/image/internal/docker"
	"github.com/stenh0use/hind/pkg/build/release"
)

type Image struct {
	Name      string
	Kind      release.ImageKind
	Packages  []string
	BaseImage ImageMeta
	Release   string
}

type ImageMeta struct {
	Name   string // Name of the image
	Digest string // Sha256 digest of the image
	Tag    string // Tag assigned to the image
	Pull   bool   // Denotes if we should pull the image or not when building
}

func BuildTargets() []string {
	targets := make([]string, 0, len(release.Images())+1)
	for _, t := range release.Images() {
		targets = append(targets, t.String())
	}
	return append(targets, "all")
}

func NewImage(i release.ImageKind) (Image, error) {
	rel := release.Latest()
	switch i {
	case release.Consul:
		return newConsul(rel), nil
	case release.Nomad:
		return newNomad(rel), nil
	case release.NomadClient:
		return newNomadClient(rel), nil
	case release.Vault:
		return newVault(rel), nil
	default:
		return Image{}, fmt.Errorf("image '%s' is not a valid hind image", i)
	}
}

func newConsul(rel release.Info) Image {
	return Image{
		Name:     "consul",
		Kind:     release.Consul,
		Packages: []string{"consul"},
		BaseImage: ImageMeta{
			Name: string(release.Base),
			Tag:  rel.Base,
			Pull: true,
		},
		Release: rel.Hind,
	}
}

func newNomad(rel release.Info) Image {
	return Image{
		Name:     "nomad",
		Kind:     release.Nomad,
		Packages: []string{"consul", "nomad"},
		BaseImage: ImageMeta{
			Name: release.Consul.ImageName(),
			Tag:  rel.Hind,
			Pull: false,
		},
		Release: rel.Hind,
	}
}

func newNomadClient(rel release.Info) Image {
	return Image{
		Name:     "nomad-client",
		Kind:     release.NomadClient,
		Packages: []string{"consul", "nomad", "dockerce", "containerd"},
		BaseImage: ImageMeta{
			Name: release.Nomad.ImageName(),
			Tag:  rel.Hind,
			Pull: false,
		},
		Release: rel.Hind,
	}
}

func newVault(rel release.Info) Image {
	return Image{
		Name:     "vault",
		Kind:     release.Vault,
		Packages: []string{"consul", "vault"},
		BaseImage: ImageMeta{
			Name: release.Consul.ImageName(),
			Tag:  rel.Hind,
			Pull: false,
		},
		Release: rel.Hind,
	}
}

func (i *Image) packagesToBuildArgs() ([]docker.BuildArg, error) {
	rel, err := release.Get(i.Release)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %w", i.Release, err)
	}

	args := make([]docker.BuildArg, 0, len(i.Packages))
	for _, name := range i.Packages {
		if version, err := rel.GetPackage(name); err == nil {
			args = append(args, docker.BuildArg{
				Arg:   strings.ToUpper(name) + "_VERSION",
				Value: version,
			})
		}
	}

	return args, nil
}

func (i *Image) buildArgs() ([]docker.BuildArg, error) {
	args, err := i.packagesToBuildArgs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate build args for image %s: %w", i.Name, err)
	}
	args = append(args, docker.BuildArg{
		Arg:   "HIND_VERSION",
		Value: i.Release,
	})
	args = append(args, docker.BuildArg{
		Arg:   "BASE_IMAGE",
		Value: fmt.Sprintf("%s:%s", i.BaseImage.Name, i.BaseImage.Tag),
	})

	return args, nil
}
