package image

import "github.com/stenh0use/hind/pkg/naming"

// Re-export naming primitives so callers within the build layer can use the
// image package as a single import point.

const (
	ImageRegistry   = naming.ImageRegistry
	ImageRepo       = naming.ImageRepo
	ImageNamePrefix = naming.ImageNamePrefix
)

// ImageKind is an alias for naming.ImageKind so that existing code within
// this package compiles without change.
type ImageKind = naming.ImageKind

const (
	Base        = naming.Base
	Consul      = naming.Consul
	Nomad       = naming.Nomad
	NomadClient = naming.NomadClient
	Vault       = naming.Vault
)

// Images returns the ordered list of buildable hind image kinds.
func Images() []ImageKind { return naming.Images() }

// IsValidKind reports whether s is a recognised ImageKind.
func IsValidKind(s string) bool { return naming.IsValidKind(s) }
