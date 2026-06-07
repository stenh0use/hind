// Package naming provides pure image-naming primitives shared between the
// build layer and the cluster domain. It has no imports from provider, build,
// or domain packages, making it safe to import from either side without
// creating an import cycle.
package naming

import (
	"fmt"
	"slices"
)

const (
	// ImageRegistry is the container registry hosting hind images.
	ImageRegistry = "docker.io"
	// ImageRepo is the repository (user/organisation) under the registry.
	ImageRepo = "stenh0use"
	// ImageNamePrefix is the common prefix for all hind image names.
	ImageNamePrefix = "hind"
)

// ImageKind identifies a hind container image by its service role.
type ImageKind string

const (
	Base        ImageKind = "debian"
	Consul      ImageKind = "consul"
	Nomad       ImageKind = "nomad"
	NomadClient ImageKind = "nomad-client"
	Vault       ImageKind = "vault"
)

// String returns the string representation of the ImageKind.
func (i ImageKind) String() string {
	return string(i)
}

// ImageName returns the fully-qualified image reference for this kind:
// <registry>/<repo>/<prefix>.<kind>
func (i ImageKind) ImageName() string {
	return fmt.Sprintf("%s/%s/%s.%s", ImageRegistry, ImageRepo, ImageNamePrefix, i.String())
}

// Images returns the ordered list of buildable hind image kinds.
func Images() []ImageKind {
	return []ImageKind{Consul, Nomad, NomadClient, Vault}
}

// IsValidKind reports whether s is a recognised ImageKind.
func IsValidKind(s string) bool {
	return slices.Contains(Images(), ImageKind(s))
}
