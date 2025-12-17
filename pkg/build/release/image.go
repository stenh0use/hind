package release

import (
	"fmt"
	"slices"
)

const (
	ImageRegistry   = "docker.io"
	ImageRepo       = "stenh0use"
	ImageNamePrefix = "hind"
)

type ImageKind string

const (
	Base        ImageKind = "debian"
	Consul      ImageKind = "consul"
	Nomad       ImageKind = "nomad"
	NomadClient ImageKind = "nomad-client"
	Vault       ImageKind = "vault"
)

func (i ImageKind) String() string {
	return string(i)
}

// Returns ImageRegistry/ImageRepo/ImageNamePrefix.ImageKind
//
// eg. docker.io/stenh0use/hind.consul
func (i ImageKind) ImageName() string {
	return fmt.Sprintf(
		"%s/%s/%s.%s",
		ImageRegistry, ImageRepo,
		ImageNamePrefix, i.String(),
	)
}

func Images() []ImageKind {
	// Images is build dependency order: consul -> nomad -> nomad-client/vault
	return []ImageKind{
		Consul,
		Nomad,
		NomadClient,
		Vault,
	}
}

// IsValidKind checks if the provided image name is valid.
// It returns true if the image name is in the list of valid images, false otherwise.
func IsValidKind(i string) bool {
	return slices.Contains(Images(), ImageKind(i))
}
