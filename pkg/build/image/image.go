// Package image defines Docker image specifications and build configurations
// for hind container images. It manages image metadata, dependencies, and
// build arguments for HashiCorp service containers.
package image

import (
	"fmt"
	"strings"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/version"
)

type Image struct {
	Name            string
	Kind            ImageKind
	Packages        []string
	BaseImage       ImageMeta
	Release         string
	ReleasePackages release.Packages
}

type ImageMeta struct {
	Name   string // Name of the image
	Digest string // Sha256 digest of the image
	Tag    string // Tag assigned to the image
	Pull   bool   // Denotes if we should pull the image or not when building
}

func BuildTargets() []string {
	targets := make([]string, 0, len(Images())+1)
	for _, t := range Images() {
		targets = append(targets, t.String())
	}
	return append(targets, "all")
}

func NewImage(i ImageKind) (Image, error) {
	return NewImageWithRelease(i, release.Versions())
}

// NewImageWithRelease constructs an image definition for the given image kind
// using the provided release metadata.
func NewImageWithRelease(i ImageKind, rel release.Packages) (Image, error) {
	switch i {
	case Consul:
		return newConsul(rel), nil
	case Nomad:
		return newNomad(rel), nil
	case NomadClient:
		return newNomadClient(rel), nil
	case Vault:
		return newVault(rel), nil
	default:
		return Image{}, fmt.Errorf("image '%s' is not a valid hind image", i)
	}
}

func newConsul(rel release.Packages) Image {
	return Image{
		Name:            "consul",
		Kind:            Consul,
		Packages:        []string{"consul"},
		ReleasePackages: rel,
		BaseImage: ImageMeta{
			Name: string(Base),
			Tag:  rel.Base,
			Pull: true,
		},
		Release: version.HindVersion,
	}
}

func newNomad(rel release.Packages) Image {
	return Image{
		Name:            "nomad",
		Kind:            Nomad,
		Packages:        []string{"consul", "nomad"},
		ReleasePackages: rel,
		BaseImage: ImageMeta{
			Name: Consul.ImageName(),
			Tag:  version.HindVersion,
			Pull: false,
		},
		Release: version.HindVersion,
	}
}
func newNomadClient(rel release.Packages) Image {
	return Image{
		Name:            "nomad-client",
		Kind:            NomadClient,
		Packages:        []string{"consul", "nomad", "dockerce", "containerd"},
		ReleasePackages: rel,
		BaseImage: ImageMeta{
			Name: Nomad.ImageName(),
			Tag:  version.HindVersion,
			Pull: false,
		},
		Release: version.HindVersion,
	}
}

func newVault(rel release.Packages) Image {
	return Image{
		Name:            "vault",
		Kind:            Vault,
		Packages:        []string{"consul", "vault"},
		ReleasePackages: rel,
		BaseImage: ImageMeta{
			Name: Consul.ImageName(),
			Tag:  version.HindVersion,
			Pull: false,
		},
		Release: version.HindVersion,
	}
}

// packagesToBuildArgs converts the image's package list to a map of build arg
// key-value pairs (e.g. CONSUL_VERSION -> "1.17.0").
func (i *Image) packagesToBuildArgs() (map[string]string, error) {
	return i.packagesToBuildArgsWithRelease(i.ReleasePackages)
}

func (i *Image) packagesToBuildArgsWithRelease(rel release.Packages) (map[string]string, error) {

	args := make(map[string]string, len(i.Packages))
	for _, name := range i.Packages {
		if version, err := rel.Package(name); err == nil {
			args[strings.ToUpper(name)+"_VERSION"] = version
		}
	}

	return args, nil
}

// buildArgs returns the full set of build arguments for the image, including
// package versions, the hind version, and the base image reference.
func (i *Image) buildArgs() (map[string]string, error) {
	args, err := i.packagesToBuildArgs()
	if err != nil {
		return nil, fmt.Errorf("failed to generate build args for image %s: %w", i.Name, err)
	}
	args["HIND_VERSION"] = i.Release
	args["BASE_IMAGE"] = fmt.Sprintf("%s:%s", i.BaseImage.Name, i.BaseImage.Tag)

	return args, nil
}
