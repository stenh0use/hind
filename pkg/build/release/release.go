// Package release manages hind release versions and package dependencies.
// It provides version information for HashiCorp services (Nomad, Consul, Vault)
// and associated tooling for each hind release.
package release

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

var (
	ErrUnknownPackage = errors.New("unknown package")
	ErrUnknownRelease = errors.New("unknown release")
)

// Info represents a specific hind release with all package versions.
type Info struct {
	Hind       string
	Base       string
	Consul     string
	Nomad      string
	Vault      string
	Containerd string
	DockerCe   string
	CniPlugins string
	Cilium     string
}

// GetPackage returns the version of a specific package from this release.
// Returns ErrUnknownPackage if the package name is not recognized.
func (i Info) GetPackage(name string) (string, error) {
	switch name {
	case "hind":
		return i.Hind, nil
	case "base":
		return i.Base, nil
	case "consul":
		return i.Consul, nil
	case "nomad":
		return i.Nomad, nil
	case "vault":
		return i.Vault, nil
	case "containerd":
		return i.Containerd, nil
	case "dockerce":
		return i.DockerCe, nil
	case "cniplugins":
		return i.CniPlugins, nil
	case "cilium":
		return i.Cilium, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownPackage, name)
	}
}

// Data manages release data and provides access to release information.
type Data struct {
	latest   string
	releases map[string]Info
}

// New creates a new Data instance with the given releases and latest version identifier.
func New(latest string, releases map[string]Info) *Data {
	return &Data{
		latest:   latest,
		releases: releases,
	}
}

// Latest returns the latest release information.
func (d *Data) Latest() Info {
	return d.releases[d.latest]
}

// Get returns release information for a specific version.
// Returns an error if the version does not exist.
func (d *Data) Get(release string) (Info, error) {
	info, ok := d.releases[release]
	if !ok {
		return Info{}, fmt.Errorf("%w: %s", ErrUnknownRelease, release)
	}
	return info, nil
}

// List returns all available releases by name.
func (d *Data) List() []string {
	return slices.Collect(maps.Keys(d.releases))
}
