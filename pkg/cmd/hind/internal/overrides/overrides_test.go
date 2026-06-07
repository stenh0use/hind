package overrides

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setEnvForTest sets only the env vars in the provided map; all three version
// env vars are first unset so each test starts from a clean state.
//
// IMPORTANT: do NOT call t.Setenv("X", "") for "unset" — that sets X to the
// empty string, which the resolver treats as a present-but-empty value and
// rejects. Use os.Unsetenv for absent keys.
func setEnvForTest(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, name := range []string{EnvNomad, EnvConsul, EnvVault} {
		require.NoError(t, os.Unsetenv(name))
		t.Cleanup(func() { _ = os.Unsetenv(name) })
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestResolve_Precedence(t *testing.T) {
	tests := []struct {
		name       string
		flagNomad  string
		flagConsul string
		flagVault  string
		envVars    map[string]string
		wantNomad  Resolved
		wantConsul Resolved
		wantVault  Resolved
	}{
		{
			name:       "no flags no env yields default sources",
			wantNomad:  Resolved{Value: "", Source: SourceDefault},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "env only sets env source",
			envVars:    map[string]string{EnvNomad: "1.9.0"},
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceEnv},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "flag only sets flag source",
			flagNomad:  "1.9.0",
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceFlag},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "flag wins over env",
			flagNomad:  "1.9.0",
			envVars:    map[string]string{EnvNomad: "1.8.0"},
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceFlag},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "per-package independence",
			flagNomad:  "1.9.0",
			envVars:    map[string]string{EnvConsul: "1.19.2"},
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceFlag},
			wantConsul: Resolved{Value: "1.19.2", Source: SourceEnv},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "env value gets trimmed",
			envVars:    map[string]string{EnvNomad: "  1.9.0  "},
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceEnv},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
		{
			name:       "flag value gets trimmed",
			flagNomad:  "  1.9.0  ",
			wantNomad:  Resolved{Value: "1.9.0", Source: SourceFlag},
			wantConsul: Resolved{Value: "", Source: SourceDefault},
			wantVault:  Resolved{Value: "", Source: SourceDefault},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvForTest(t, tt.envVars)
			in := FlagInputs{
				NomadVersion:  tt.flagNomad,
				ConsulVersion: tt.flagConsul,
				VaultVersion:  tt.flagVault,
				NomadSet:      tt.flagNomad != "",
				ConsulSet:     tt.flagConsul != "",
				VaultSet:      tt.flagVault != "",
			}
			got, err := Resolve(in, OSEnvLookup)
			require.NoError(t, err)
			assert.Equal(t, tt.wantNomad, got.Nomad)
			assert.Equal(t, tt.wantConsul, got.Consul)
			assert.Equal(t, tt.wantVault, got.Vault)
		})
	}
}

func TestResolve_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		flagNomad      string
		flagNomadSet   bool
		envVars        map[string]string
		wantErrSubstrs []string
	}{
		{
			name:           "flag empty after trim attributes to flag",
			flagNomad:      "   ",
			flagNomadSet:   true,
			wantErrSubstrs: []string{"--nomad-version", "must not be empty"},
		},
		{
			name:           "env whitespace-only value attributes to env var name",
			envVars:        map[string]string{EnvConsul: "   "},
			wantErrSubstrs: []string{"CONSUL_VERSION", "must not be empty"},
		},
		{
			name:           "env empty string value attributes to env var name",
			envVars:        map[string]string{EnvNomad: ""},
			wantErrSubstrs: []string{"NOMAD_VERSION", "must not be empty"},
		},
		{
			name:           "flag malformed attributes to flag with reason",
			flagNomad:      "bad/value",
			flagNomadSet:   true,
			wantErrSubstrs: []string{"--nomad-version", "invalid format", "bad/value"},
		},
		{
			name:           "env malformed attributes to env var with reason",
			envVars:        map[string]string{EnvNomad: "@bad"},
			wantErrSubstrs: []string{"NOMAD_VERSION", "invalid format", "@bad"},
		},
		{
			name:           "flag and env both set: flag wins, env ignored even if invalid",
			flagNomad:      "1.9.0",
			flagNomadSet:   true,
			envVars:        map[string]string{EnvNomad: "@bad"},
			wantErrSubstrs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvForTest(t, tt.envVars)
			in := FlagInputs{NomadVersion: tt.flagNomad, NomadSet: tt.flagNomadSet}
			_, err := Resolve(in, OSEnvLookup)
			if tt.wantErrSubstrs == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, s := range tt.wantErrSubstrs {
				assert.Contains(t, err.Error(), s)
			}
		})
	}
}

func TestRenderLines_Attribution(t *testing.T) {
	got := RenderLines(Set{
		Nomad:  Resolved{Value: "1.9.0", Source: SourceFlag},
		Consul: Resolved{Value: "1.19.2", Source: SourceEnv},
		Vault:  Resolved{Value: "", Source: SourceDefault},
	})
	require.Len(t, got, 3)
	assert.Equal(t, "Version overrides:", got[0])
	assert.Equal(t, "  nomad: 1.9.0", got[1])
	assert.Equal(t, "  consul: 1.19.2 (from env)", got[2])
}

func TestRenderLines_NoOverridesReturnsNoLines(t *testing.T) {
	got := RenderLines(Set{
		Nomad:  Resolved{Value: "", Source: SourceDefault},
		Consul: Resolved{Value: "", Source: SourceDefault},
		Vault:  Resolved{Value: "", Source: SourceDefault},
	})
	assert.Empty(t, got)
}
