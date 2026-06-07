package naming

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageKind_ImageName(t *testing.T) {
	cases := []struct {
		name string
		kind ImageKind
		want string
	}{
		{"Consul", Consul, "docker.io/stenh0use/hind.consul"},
		{"Nomad", Nomad, "docker.io/stenh0use/hind.nomad"},
		{"NomadClient hyphenated", NomadClient, "docker.io/stenh0use/hind.nomad-client"},
		{"Vault", Vault, "docker.io/stenh0use/hind.vault"},
		{"Base produces debian", Base, "docker.io/stenh0use/hind.debian"},
		{"arbitrary literal still formats", ImageKind("postgres"), "docker.io/stenh0use/hind.postgres"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, tc.kind.ImageName())
		})
	}
}

func TestIsValidKind(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"consul valid", "consul", true},
		{"nomad valid", "nomad", true},
		{"nomad-client valid", "nomad-client", true},
		{"vault valid", "vault", true},
		{"debian (Base) not in Images()", "debian", false},
		{"empty string invalid", "", false},
		{"unrecognised string invalid", "postgres", false},
		{"case-sensitive: Consul invalid", "Consul", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, IsValidKind(tc.input))
		})
	}
}
