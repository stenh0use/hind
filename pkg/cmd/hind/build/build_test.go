package build

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/overrides"
)

func TestNewCommand(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	require.NotNil(t, command, "NewCommand() returned nil")

	// Verify Use contains expected text
	assert.NotEmpty(t, command.Use, "Expected Use to be non-empty")

	assert.Equal(t, "Build container images", command.Short)
}

func TestDefaultTimeout(t *testing.T) {
	expected := 15 * time.Minute
	assert.Equal(t, expected, DefaultBuildTimeout)
}

func TestCommandFlags(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	// Check if timeout flag exists
	timeoutFlag := command.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag, "Expected 'timeout' flag to exist")

	assert.Equal(t, "15m0s", timeoutFlag.DefValue)
}

func TestCommandArgs(t *testing.T) {
	logger := &log.Logger{
		Handler: discard.New(),
		Level:   log.ErrorLevel,
	}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{name: "no args", args: []string{}, wantError: true},
		{name: "one valid arg", args: []string{"all"}, wantError: false},
		{name: "too many args", args: []string{"nomad", "consul"}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCommand(logger, streams)
			command.SetArgs(tt.args)
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError && err != nil {
				assert.False(t, strings.Contains(err.Error(), "[") || strings.Contains(err.Error(), "]"),
					"expected human-readable arg-count error without raw slice formatting, got %q", err.Error())
			}
		})
	}
}

// setBuildEnvForTest mirrors the helper in overrides: unset all three
// env vars first, then set only the keys present in vars. Do NOT call
// t.Setenv("X", "") to mean "unset" — that sets X to the empty string and
// the resolver will treat it as present-but-empty and reject it.
func setBuildEnvForTest(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, name := range []string{overrides.EnvNomad, overrides.EnvConsul, overrides.EnvVault} {
		require.NoError(t, os.Unsetenv(name))
		t.Cleanup(func() { _ = os.Unsetenv(name) })
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

func TestRunE_InvalidEnvVarFailsFastBeforeAnyBuild(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		wantErrSubstrs []string
	}{
		{
			name:           "empty NOMAD_VERSION",
			envVars:        map[string]string{overrides.EnvNomad: ""},
			wantErrSubstrs: []string{"NOMAD_VERSION", "must not be empty"},
		},
		{
			name:           "whitespace CONSUL_VERSION",
			envVars:        map[string]string{overrides.EnvConsul: "   "},
			wantErrSubstrs: []string{"CONSUL_VERSION", "must not be empty"},
		},
		{
			name:           "malformed VAULT_VERSION",
			envVars:        map[string]string{overrides.EnvVault: "@bad"},
			wantErrSubstrs: []string{"VAULT_VERSION", "invalid format", "@bad"},
		},
		{
			name: "one valid one invalid still fails (no partial build)",
			envVars: map[string]string{
				overrides.EnvNomad:  "1.9.0",
				overrides.EnvConsul: "@bad",
			},
			wantErrSubstrs: []string{"CONSUL_VERSION", "invalid format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBuildEnvForTest(t, tt.envVars)

			logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
			var errBuf bytes.Buffer
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: &errBuf}

			command := NewCommand(logger, streams)
			command.SetArgs([]string{"nomad"})
			command.SetContext(context.Background())

			err := command.Execute()
			require.Error(t, err)
			for _, s := range tt.wantErrSubstrs {
				assert.Contains(t, err.Error(), s)
			}
			assert.NotContains(t, errBuf.String(), "Building",
				"no image build should have started before validation failure")
		})
	}
}

func TestRunE_NoOverridesEmitsNoVersionOverridesHeader(t *testing.T) {
	setBuildEnvForTest(t, nil)

	flags := &flagpole{timeout: time.Second}

	// Verify packageVersionOverrides emits a zero Overrides struct when nothing set.
	ov := packageVersionOverrides(flags, overrides.Set{})
	assert.False(t, ov.HasAny(), "expected no overrides when neither flags nor env are set")
}

func TestPackageVersionOverrides_MergesResolvedHashiCorpVersions(t *testing.T) {
	flags := &flagpole{
		baseVersion:       "bookworm-slim",
		containerdVersion: "1.7.99-1",
	}
	resolved := overrides.Set{
		Nomad:  overrides.Resolved{Value: "1.9.0", Source: overrides.SourceEnv},
		Consul: overrides.Resolved{Value: "1.19.2", Source: overrides.SourceFlag},
		Vault:  overrides.Resolved{Value: "", Source: overrides.SourceDefault},
	}

	ov := packageVersionOverrides(flags, resolved)

	assert.Equal(t, "1.9.0", ov.Nomad)
	assert.Equal(t, "1.19.2", ov.Consul)
	assert.Empty(t, ov.Vault, "default source must leave Vault unset so release default wins")
	assert.Equal(t, "bookworm-slim", ov.Base)
	assert.Equal(t, "1.7.99-1", ov.Containerd)
}

func TestRunE_EmitsAttributionFromEnvAndOmitsFromFlag(t *testing.T) {
	setBuildEnvForTest(t, map[string]string{
		overrides.EnvConsul: "1.19.2",
	})

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var errBuf bytes.Buffer
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: &errBuf}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"nomad", "--nomad-version", "1.9.0"})
	command.SetContext(context.Background())

	// Build will fail at the Docker boundary in CI without a daemon — that's
	// fine; attribution is written before BuildImage runs.
	_ = command.Execute()

	out := errBuf.String()
	assert.Contains(t, out, "Version overrides:")
	assert.Contains(t, out, "  nomad: 1.9.0\n", "flag-sourced override must not carry (from env)")
	assert.Contains(t, out, "  consul: 1.19.2 (from env)")
	assert.NotContains(t, out, "nomad: 1.9.0 (from env)")
}
