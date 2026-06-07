package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/build/release"
	"github.com/stenh0use/hind/pkg/version"
)

func TestBuildArgs_ReleaseParity(t *testing.T) {
	rel := release.Versions()

	tests := []struct {
		name string
		kind ImageKind
		want map[string]string
	}{
		{
			name: "consul",
			kind: Consul,
			want: map[string]string{
				"CONSUL_VERSION": rel.Consul,
				"HIND_VERSION":   version.HindVersion,
				"BASE_IMAGE":     string(Base) + ":" + rel.Base,
			},
		},
		{
			name: "nomad",
			kind: Nomad,
			want: map[string]string{
				"CONSUL_VERSION": rel.Consul,
				"NOMAD_VERSION":  rel.Nomad,
				"HIND_VERSION":   version.HindVersion,
				"BASE_IMAGE":     Consul.ImageName() + ":" + version.HindVersion,
			},
		},
		{
			name: "vault",
			kind: Vault,
			want: map[string]string{
				"CONSUL_VERSION": rel.Consul,
				"VAULT_VERSION":  rel.Vault,
				"HIND_VERSION":   version.HindVersion,
				"BASE_IMAGE":     Consul.ImageName() + ":" + version.HindVersion,
			},
		},
		{
			name: "nomad-client",
			kind: NomadClient,
			want: map[string]string{
				"CONSUL_VERSION":     rel.Consul,
				"NOMAD_VERSION":      rel.Nomad,
				"DOCKERCE_VERSION":   rel.DockerCe,
				"CONTAINERD_VERSION": rel.Containerd,
				"HIND_VERSION":       version.HindVersion,
				"BASE_IMAGE":         Nomad.ImageName() + ":" + version.HindVersion,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img, err := NewImage(tc.kind)
			require.NoError(t, err)

			args, err := img.buildArgs()
			require.NoError(t, err)

			require.Len(t, args, len(tc.want), "buildArgs() arg count mismatch: got %v, want %v", args, tc.want)

			for k, wantV := range tc.want {
				gotV, ok := args[k]
				require.True(t, ok, "buildArgs() missing key %q; got keys: %v", k, mapKeys(args))
				assert.Equal(t, wantV, gotV, "buildArgs() value mismatch for %q", k)
			}
		})
	}
}

func TestPackagesToBuildArgs_ReleaseParity(t *testing.T) {
	rel := release.Versions()

	overriddenRel := rel.WithOverrides(release.Overrides{
		Consul:   "9.9.9",
		Nomad:    "8.8.8",
		Vault:    "7.7.7",
		DockerCe: "6.6.6",
	})

	tests := []struct {
		name string
		kind ImageKind
		want map[string]string
	}{
		{
			name: "consul override",
			kind: Consul,
			want: map[string]string{"CONSUL_VERSION": overriddenRel.Consul},
		},
		{
			name: "nomad override",
			kind: Nomad,
			want: map[string]string{"CONSUL_VERSION": overriddenRel.Consul, "NOMAD_VERSION": overriddenRel.Nomad},
		},
		{
			name: "vault override",
			kind: Vault,
			want: map[string]string{"CONSUL_VERSION": overriddenRel.Consul, "VAULT_VERSION": overriddenRel.Vault},
		},
		{
			name: "nomad-client override",
			kind: NomadClient,
			want: map[string]string{
				"CONSUL_VERSION":     overriddenRel.Consul,
				"NOMAD_VERSION":      overriddenRel.Nomad,
				"DOCKERCE_VERSION":   overriddenRel.DockerCe,
				"CONTAINERD_VERSION": overriddenRel.Containerd,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img, err := NewImageWithRelease(tc.kind, overriddenRel)
			require.NoError(t, err)

			args, err := img.packagesToBuildArgs()
			require.NoError(t, err)

			require.Len(t, args, len(tc.want), "packagesToBuildArgs() arg count mismatch: got %v, want %v", args, tc.want)

			for k, wantV := range tc.want {
				gotV, ok := args[k]
				require.True(t, ok, "packagesToBuildArgs() missing key %q; got keys: %v", k, mapKeys(args))
				assert.Equal(t, wantV, gotV, "packagesToBuildArgs() value mismatch for %q", k)
			}
		})
	}

	defaultTests := []struct {
		name string
		kind ImageKind
		want map[string]string
	}{
		{
			name: "consul",
			kind: Consul,
			want: map[string]string{"CONSUL_VERSION": rel.Consul},
		},
		{
			name: "nomad",
			kind: Nomad,
			want: map[string]string{"CONSUL_VERSION": rel.Consul, "NOMAD_VERSION": rel.Nomad},
		},
		{
			name: "vault",
			kind: Vault,
			want: map[string]string{"CONSUL_VERSION": rel.Consul, "VAULT_VERSION": rel.Vault},
		},
		{
			name: "nomad-client",
			kind: NomadClient,
			want: map[string]string{
				"CONSUL_VERSION":     rel.Consul,
				"NOMAD_VERSION":      rel.Nomad,
				"DOCKERCE_VERSION":   rel.DockerCe,
				"CONTAINERD_VERSION": rel.Containerd,
			},
		},
	}

	for _, tc := range defaultTests {
		t.Run(tc.name, func(t *testing.T) {
			img, err := NewImage(tc.kind)
			require.NoError(t, err)

			args, err := img.packagesToBuildArgs()
			require.NoError(t, err)

			require.Len(t, args, len(tc.want), "packagesToBuildArgs() arg count mismatch: got %v, want %v", args, tc.want)

			for k, wantV := range tc.want {
				gotV, ok := args[k]
				require.True(t, ok, "packagesToBuildArgs() missing key %q; got keys: %v", k, mapKeys(args))
				assert.Equal(t, wantV, gotV, "packagesToBuildArgs() value mismatch for %q", k)
			}
		})
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
