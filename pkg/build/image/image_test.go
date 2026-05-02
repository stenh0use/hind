package image

import (
	"testing"

	"github.com/stenh0use/hind/pkg/build/release"
)

func TestBuildArgs_ReturnsMapWithExpectedKeys(t *testing.T) {
	img, err := NewImage(release.Consul)
	if err != nil {
		t.Fatalf("NewImage(Consul): %v", err)
	}

	args, err := img.buildArgs()
	if err != nil {
		t.Fatalf("buildArgs(): %v", err)
	}

	wantKeys := []string{"CONSUL_VERSION", "HIND_VERSION", "BASE_IMAGE"}
	for _, k := range wantKeys {
		if _, ok := args[k]; !ok {
			t.Errorf("buildArgs() missing key %q; got keys: %v", k, mapKeys(args))
		}
	}
}

func TestPackagesToBuildArgs_KnownPackages(t *testing.T) {
	img, err := NewImage(release.Nomad)
	if err != nil {
		t.Fatalf("NewImage(Nomad): %v", err)
	}

	args, err := img.packagesToBuildArgs()
	if err != nil {
		t.Fatalf("packagesToBuildArgs(): %v", err)
	}

	// Nomad image packages: consul, nomad — expect both version keys.
	wantKeys := []string{"CONSUL_VERSION", "NOMAD_VERSION"}
	for _, k := range wantKeys {
		if _, ok := args[k]; !ok {
			t.Errorf("packagesToBuildArgs() missing key %q; got keys: %v", k, mapKeys(args))
		}
	}

	// Verify values are non-empty.
	for k, v := range args {
		if v == "" {
			t.Errorf("packagesToBuildArgs() key %q has empty value", k)
		}
	}
}

// mapKeys returns the keys of a map for error messages.
func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
