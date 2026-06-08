package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersions(t *testing.T) {
	got := Versions()
	require.NotEmpty(t, got.Consul, "Versions().Consul should not be empty")
	require.NotEmpty(t, got.Nomad, "Versions().Nomad should not be empty")
}

func TestVersions_GetPackage(t *testing.T) {
	rel := Versions()

	tests := []struct {
		name        string
		packageName string
		expectError bool
		expected    string
	}{
		{name: "get consul version", packageName: "consul", expected: rel.Consul},
		{name: "get consul version case insensitive", packageName: "Consul", expected: rel.Consul},
		{name: "get nomad version", packageName: "nomad", expected: rel.Nomad},
		{name: "get vault version", packageName: "vault", expected: rel.Vault},
		{name: "get invalid package", packageName: "invalid", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rel.Package(tt.packageName)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrUnknownPackage)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestVersions_WithOverrides(t *testing.T) {
	rel := Versions()
	overrides := Overrides{
		Nomad:  "1.11.0",
		Consul: "1.23.0",
	}

	updated := rel.WithOverrides(overrides)

	require.Equal(t, "1.11.0", updated.Nomad)
	require.Equal(t, "1.23.0", updated.Consul)
	require.Equal(t, rel.Vault, updated.Vault)
}

func TestOverrides_HasAny(t *testing.T) {
	tests := []struct {
		name      string
		overrides Overrides
		want      bool
	}{
		{name: "empty", overrides: Overrides{}, want: false},
		{name: "nomad only", overrides: Overrides{Nomad: "1.0.0"}, want: true},
		{name: "consul only", overrides: Overrides{Consul: "1.0.0"}, want: true},
		{name: "vault only", overrides: Overrides{Vault: "1.0.0"}, want: true},
		{name: "base only", overrides: Overrides{Base: "bullseye"}, want: true},
		{name: "multiple", overrides: Overrides{Nomad: "1.0.0", Consul: "2.0.0"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.overrides.HasAny()
			require.Equal(t, tt.want, got)
		})
	}
}
