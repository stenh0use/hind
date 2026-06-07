// Package release manages hind release versions and package dependencies.
// It provides version information for HashiCorp services (Nomad, Consul, Vault)
// and associated tooling for each hind release.
package release

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnknownPackage = errors.New("unknown package")
)

var release = Packages{
	Base:       "bullseye-slim",
	Consul:     "1.22.0",
	Nomad:      "1.10.5",
	Vault:      "1.21.0",
	Containerd: "1.7.27-1",
	DockerCe:   "28.5.1-1",
	CniPlugins: "1.3.0",
	Cilium:     "1.13.9",
}

// Packages represents hind package versions used for image builds.
type Packages struct {
	Base       string
	Consul     string
	Nomad      string
	Vault      string
	Containerd string
	DockerCe   string
	CniPlugins string
	Cilium     string
}

// Versions returns the latest release info.
func Versions() Packages {
	return release
}

// Package returns the version of a specific package from this release.
// Returns ErrUnknownPackage if the package name is not recognized.
func (p Packages) Package(name string) (string, error) {
	switch strings.ToLower(name) {
	case "base":
		return p.Base, nil
	case "consul":
		return p.Consul, nil
	case "nomad":
		return p.Nomad, nil
	case "vault":
		return p.Vault, nil
	case "containerd":
		return p.Containerd, nil
	case "dockerce":
		return p.DockerCe, nil
	case "cniplugins":
		return p.CniPlugins, nil
	case "cilium":
		return p.Cilium, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownPackage, name)
	}
}

// Overrides is a typed package-version override container.
type Overrides struct {
	Base       string
	Consul     string
	Nomad      string
	Vault      string
	Containerd string
	DockerCe   string
	CniPlugins string
	Cilium     string
}

// WithOverrides returns a copy of this release info with explicit overrides applied.
func (p Packages) WithOverrides(o Overrides) Packages {
	updated := p
	if o.Base != "" {
		updated.Base = o.Base
	}
	if o.Consul != "" {
		updated.Consul = o.Consul
	}
	if o.Nomad != "" {
		updated.Nomad = o.Nomad
	}
	if o.Vault != "" {
		updated.Vault = o.Vault
	}
	if o.Containerd != "" {
		updated.Containerd = o.Containerd
	}
	if o.DockerCe != "" {
		updated.DockerCe = o.DockerCe
	}
	if o.CniPlugins != "" {
		updated.CniPlugins = o.CniPlugins
	}
	if o.Cilium != "" {
		updated.Cilium = o.Cilium
	}
	return updated
}

// HasAny reports whether at least one override field is set.
func (o Overrides) HasAny() bool {
	return o.Base != "" || o.Consul != "" || o.Nomad != "" || o.Vault != "" || o.Containerd != "" || o.DockerCe != "" || o.CniPlugins != "" || o.Cilium != ""
}
