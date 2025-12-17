package release

const (
	latest string = "0.4.0"
)

// versions is the package release store containing all official hind versions.
var versions = New(
	latest,
	map[string]Info{
		"0.4.0": {
			Hind:       "0.4.0",
			Base:       "bullseye-slim",
			Consul:     "1.22.0",
			Nomad:      "1.10.5",
			Vault:      "1.21.0",
			Containerd: "1.7.27-1",
			DockerCe:   "28.5.1-1",
			CniPlugins: "1.3.0",
			Cilium:     "1.13.9",
		},
		"0.3.0": {
			Hind:       "0.3.0",
			Base:       "bullseye-slim",
			Consul:     "1.18.1",
			Nomad:      "1.7.6",
			Vault:      "1.15.4",
			Containerd: "1.6.31-1",
			DockerCe:   "26.0.1-1",
			CniPlugins: "1.3.0",
			Cilium:     "1.13.9",
		},
	},
)

// Package-level convenience functions that delegate to the releases datastore.

// Latest returns the latest release from the default store.
func Latest() Info {
	return versions.Latest()
}

// Get returns a specific release version from the default store.
func Get(version string) (Info, error) {
	return versions.Get(version)
}

// List returns all releases from the default store.
func List() []string {
	return versions.List()
}
