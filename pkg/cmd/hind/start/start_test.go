package start

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	persistencefs "github.com/stenh0use/hind/pkg/cluster/persistence/fs"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/cmd/hind/internal/overrides"
	"github.com/stenh0use/hind/pkg/version"
)

// setEnvForTest mirrors the helper in the overrides package: it
// unsets all three version env vars first, then sets only the keys present
// in vars. Do NOT use t.Setenv("X", "") for "unset" — that sets X to "".
func setEnvForTest(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, name := range []string{overrides.EnvNomad, overrides.EnvConsul, overrides.EnvVault} {
		require.NoError(t, os.Unsetenv(name))
		t.Cleanup(func() { _ = os.Unsetenv(name) })
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}
}

// saveTestCluster persists a minimal cluster config so repository-backed helpers
// can find the cluster by name.
func saveTestCluster(t *testing.T, name string) {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	c, err := domain.BuildDefaultCluster(name, version.HindVersion)
	require.NoError(t, err, "BuildDefaultCluster() error")
	err = repo.Save(context.Background(), c)
	require.NoError(t, err, "repo.Save() error")
}

// fakeActive is a minimal in-memory ActiveRepository for tests.
type fakeActive struct {
	stored string
	getErr error
}

func (f *fakeActive) GetActive(_ context.Context) (string, error) { return f.stored, f.getErr }
func (f *fakeActive) SetActive(_ context.Context, n string) error {
	f.stored = n
	return nil
}
func (f *fakeActive) ClearActive(_ context.Context) error {
	f.stored = ""
	return nil
}

// stubStartManager is a no-op clusterStarter used to bypass real Docker calls in tests.
type stubStartManager struct {
	startOutcome domain.StartOutcome
	clientCount  int
	startErr     error
	scaleErr     error
	startCalls   *int
	scaleCalls   *int
}

var lastCreatedStartManager *stubStartManager
var lastCreatedActive *fakeActive

func (s *stubStartManager) Start(_ context.Context, _ orchestration.StartRequest) (domain.StartOutcome, error) {
	if s.startCalls != nil {
		*s.startCalls = *s.startCalls + 1
	}
	if s.startErr != nil {
		return domain.StartOutcomeCreated, s.startErr
	}
	return s.startOutcome, nil
}

func (s *stubStartManager) Scale(_ context.Context, _ int) error {
	if s.scaleCalls != nil {
		*s.scaleCalls = *s.scaleCalls + 1
	}
	return s.scaleErr
}

var _ clusterStarter = (*stubStartManager)(nil)

// capturingStartManager records the StartRequest passed to Start.
type capturingStartManager struct {
	captured *orchestration.StartRequest
	outcome  domain.StartOutcome
}

func (c *capturingStartManager) Start(_ context.Context, req orchestration.StartRequest) (domain.StartOutcome, error) {
	*c.captured = req
	return c.outcome, nil
}
func (c *capturingStartManager) Scale(_ context.Context, _ int) error { return nil }

var _ clusterStarter = (*capturingStartManager)(nil)

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

	assert.Equal(t, "start [cluster-name]", command.Use)
	assert.Equal(t, "Start or create a hind cluster", command.Short)
}

func TestDefaultTimeout(t *testing.T) {
	expected := 5 * time.Minute
	assert.Equal(t, expected, DefaultStartTimeout)
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

	timeoutFlag := command.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag, "Expected 'timeout' flag to exist")
	assert.Equal(t, "5m0s", timeoutFlag.DefValue)

	clientsFlag := command.Flags().Lookup("clients")
	require.NotNil(t, clientsFlag, "Expected 'clients' flag to exist")
	assert.Equal(t, "1", clientsFlag.DefValue)

	verboseFlag := command.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag, "Expected 'verbose' flag to exist")

	assert.Nil(t, command.Flags().Lookup("version"), "Expected 'version' flag to be absent")

	nomadVersionFlag := command.Flags().Lookup("nomad-version")
	require.NotNil(t, nomadVersionFlag, "Expected 'nomad-version' flag to exist")

	consulVersionFlag := command.Flags().Lookup("consul-version")
	require.NotNil(t, consulVersionFlag, "Expected 'consul-version' flag to exist")

	vaultVersionFlag := command.Flags().Lookup("vault-version")
	require.NotNil(t, vaultVersionFlag, "Expected 'vault-version' flag to exist")
}

func TestVersionOverrideFlagParsing(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	tests := []struct {
		name              string
		args              []string
		wantNomadVersion  string
		wantConsulVersion string
		wantVaultVersion  string
	}{
		{
			name:              "no override flags leaves all versions unset",
			args:              []string{},
			wantNomadVersion:  "",
			wantConsulVersion: "",
			wantVaultVersion:  "",
		},
		{
			name:              "single nomad override sets only nomad",
			args:              []string{"--nomad-version", "1.9.0"},
			wantNomadVersion:  "1.9.0",
			wantConsulVersion: "",
			wantVaultVersion:  "",
		},
		{
			name:              "multiple overrides capture all specified values",
			args:              []string{"--nomad-version", "1.9.0", "--consul-version", "1.19.2", "--vault-version", "1.18.0"},
			wantNomadVersion:  "1.9.0",
			wantConsulVersion: "1.19.2",
			wantVaultVersion:  "1.18.0",
		},
		{
			name:              "duplicate nomad override uses last value",
			args:              []string{"--nomad-version", "1.8.4", "--nomad-version", "1.9.0"},
			wantNomadVersion:  "1.9.0",
			wantConsulVersion: "",
			wantVaultVersion:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCommand(logger, streams)
			err := command.Flags().Parse(tt.args)
			require.NoError(t, err, "Parse() error")

			nomadVersion, err := command.Flags().GetString("nomad-version")
			require.NoError(t, err, "GetString(nomad-version)")
			assert.Equal(t, tt.wantNomadVersion, nomadVersion)

			consulVersion, err := command.Flags().GetString("consul-version")
			require.NoError(t, err, "GetString(consul-version)")
			assert.Equal(t, tt.wantConsulVersion, consulVersion)

			vaultVersion, err := command.Flags().GetString("vault-version")
			require.NoError(t, err, "GetString(vault-version)")
			assert.Equal(t, tt.wantVaultVersion, vaultVersion)
		})
	}
}

// TestRunE_SetsActiveCluster verifies that after a successful start runE calls
// SetActive so that the started cluster becomes the active profile.
func TestRunE_SetsActiveCluster(t *testing.T) {
	// Redirect HOME so cluster state is isolated to this test.
	t.Setenv("HOME", t.TempDir())
	// Isolate version env vars.
	setEnvForTest(t, nil)

	const clusterName = "test-start-cluster"

	// Pre-create the cluster config so SetActiveCluster accepts the name.
	saveTestCluster(t, clusterName)

	// Stub checkDockerDaemonFn so runE does not require a live Docker daemon.
	origDockerCheck := checkDockerDaemonFn
	checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
	defer func() { checkDockerDaemonFn = origDockerCheck }()

	// Stub newStartServicesFn so mgr.Start() does not require Docker.
	origManagerFn := newStartServicesFn
	newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
		mgr := &stubStartManager{startOutcome: domain.StartOutcomeCreated}
		fa := &fakeActive{}
		lastCreatedStartManager = mgr
		lastCreatedActive = fa
		return startServices{Orchestration: mgr, Active: fa}, nil
	}
	defer func() { newStartServicesFn = origManagerFn }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	flags := &flagpole{
		timeout: DefaultStartTimeout,
		clients: 1,
	}

	// Build a minimal cobra command to satisfy the cmd parameter expected by runE.
	cobraCmd := NewCommand(logger, streams)

	err := runE(cobraCmd, context.Background(), logger, streams, flags, []string{clusterName})
	require.NoError(t, err, "runE returned unexpected error")

	// After a successful start the active cluster must be set to the started cluster name via SetActive.
	require.NotNil(t, lastCreatedActive, "expected active to be created")
	assert.Equal(t, clusterName, lastCreatedActive.stored)
	lastCreatedStartManager = nil
	lastCreatedActive = nil
}

func TestStart_FailureBeforeCompletion_DoesNotUpdateActiveProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	setEnvForTest(t, nil)

	active := "active-a"
	for _, name := range []string{active, "target"} {
		saveTestCluster(t, name)
	}
	// Test setup uses the repository directly; production code uses the seam.
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	err = repo.SetActive(context.Background(), active)
	require.NoError(t, err, "repo.SetActive")

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	cobraCmd := NewCommand(logger, streams)

	tests := []struct {
		name   string
		flags  *flagpole
		args   []string
		setup  func()
		assert func(error)
	}{
		{
			name:  "DockerCheckFailure",
			flags: &flagpole{timeout: DefaultStartTimeout, clients: 1},
			args:  []string{"target"},
			setup: func() {
				checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return errors.New("daemon down") }
				newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
					return startServices{Orchestration: &stubStartManager{startOutcome: domain.StartOutcomeCreated}, Active: &fakeActive{}}, nil
				}
			},
			assert: func(runErr error) {
				require.Error(t, runErr, "expected docker check failure")
			},
		},
		{
			name:  "StartFailure",
			flags: &flagpole{timeout: DefaultStartTimeout, clients: 1},
			args:  []string{"target"},
			setup: func() {
				checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
				newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
					return startServices{Orchestration: &stubStartManager{startErr: errors.New("start boom")}, Active: &fakeActive{}}, nil
				}
			},
			assert: func(runErr error) {
				require.Error(t, runErr, "expected start failure")
			},
		},
		{
			name:  "ScaleFailure",
			flags: &flagpole{timeout: DefaultStartTimeout, clients: 3},
			args:  []string{"target"},
			setup: func() {
				cobraCmd.Flags().Set("clients", "3") //nolint:errcheck
				checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
				newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
					return startServices{Orchestration: &stubStartManager{startOutcome: domain.StartOutcomeResumed, clientCount: 1, scaleErr: errors.New("scale boom")}, Active: &fakeActive{}}, nil
				}
			},
			assert: func(runErr error) {
				require.Error(t, runErr, "expected scale failure")
			},
		},
	}

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkDockerDaemonFn = origDocker
			newStartServicesFn = origFactory
			tt.setup()
			runErr := runE(cobraCmd, context.Background(), logger, streams, tt.flags, tt.args)
			tt.assert(runErr)
			assertRepo, assertRepoErr := persistencefs.NewRepository()
			require.NoError(t, assertRepoErr, "NewRepository() error")
			got, gerr := assertRepo.GetActive(context.Background())
			require.NoError(t, gerr, "GetActive")
			assert.Equal(t, active, got, "expected active cluster %q to remain unchanged, got %q", active, got)
		})
	}
}

func TestRunE_VersionOverrideValidationFailures(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	tests := []struct {
		name           string
		flags          *flagpole
		flagsToSet     map[string]string // simulate cobra.Flags().Changed()
		envVars        map[string]string
		errSubstrings  []string
		wantStartCalls int
	}{
		{
			name:           "whitespace-only nomad flag fails validation, no Start",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "   \t  "},
			flagsToSet:     map[string]string{"nomad-version": "   \t  "},
			errSubstrings:  []string{"--nomad-version", "must not be empty"},
			wantStartCalls: 0,
		},
		{
			name:           "malformed consul flag fails validation, no Start",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1, consulVersion: "bad/value"},
			flagsToSet:     map[string]string{"consul-version": "bad/value"},
			errSubstrings:  []string{"--consul-version", "invalid format", "bad/value"},
			wantStartCalls: 0,
		},
		{
			name:           "mixed valid + invalid flags fails validation, no Start",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0", vaultVersion: "bad/value"},
			flagsToSet:     map[string]string{"nomad-version": "1.9.0", "vault-version": "bad/value"},
			errSubstrings:  []string{"--vault-version", "invalid format"},
			wantStartCalls: 0,
		},
		{
			name:           "whitespace-padded flag value is trimmed and accepted",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "  1.9.0  "},
			flagsToSet:     map[string]string{"nomad-version": "  1.9.0  "},
			wantStartCalls: 1,
		},
		{
			name:           "empty NOMAD_VERSION env var fails validation with env attribution",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1},
			envVars:        map[string]string{overrides.EnvNomad: "   "},
			errSubstrings:  []string{"NOMAD_VERSION", "must not be empty"},
			wantStartCalls: 0,
		},
		{
			name:           "malformed CONSUL_VERSION env var fails with env attribution",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1},
			envVars:        map[string]string{overrides.EnvConsul: "@bad"},
			errSubstrings:  []string{"CONSUL_VERSION", "invalid format", "@bad"},
			wantStartCalls: 0,
		},
		{
			name:           "valid VAULT_VERSION env var is accepted",
			flags:          &flagpole{timeout: DefaultStartTimeout, clients: 1},
			envVars:        map[string]string{overrides.EnvVault: "1.18.0"},
			wantStartCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Scope env to this test; always clear all three to avoid leakage.
			setEnvForTest(t, tt.envVars)

			startCalls := 0
			checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
			newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
				return startServices{
					Orchestration: &stubStartManager{
						startOutcome: domain.StartOutcomeCreated,
						startCalls:   &startCalls,
					},
					Active: &fakeActive{},
				}, nil
			}

			// Fresh command per test so cobra.Flags().Changed() reflects only this case.
			localCmd := NewCommand(logger, streams)
			for name, val := range tt.flagsToSet {
				err := localCmd.Flags().Set(name, val)
				require.NoError(t, err)
			}

			err := runE(localCmd, context.Background(), logger, streams, tt.flags, []string{"test"})
			if len(tt.errSubstrings) > 0 {
				require.Error(t, err, "expected validation error")
				for _, want := range tt.errSubstrings {
					assert.Contains(t, err.Error(), want)
				}
			} else {
				require.NoError(t, err, "runE returned unexpected error")
			}
			assert.Equal(t, tt.wantStartCalls, startCalls)
		})
	}
}

func TestRunE_OrchestrationBoundaries(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)        //nolint:errcheck
	defer os.Setenv("HOME", oldHome) //nolint:errcheck
	setEnvForTest(t, nil)

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var errOut bytes.Buffer
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: &errOut}

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	tests := []struct {
		name                 string
		args                 []string
		setClientsFlag       bool
		flags                *flagpole
		stub                 *stubStartManager
		expectScaleCalls     int
		expectStartCalls     int
		expectConnectionInfo bool
		expectOverridesInfo  bool
	}{
		{
			name:                 "new cluster creates successfully",
			args:                 []string{"new-cluster"},
			flags:                &flagpole{timeout: DefaultStartTimeout, clients: 1},
			stub:                 &stubStartManager{startOutcome: domain.StartOutcomeCreated},
			expectScaleCalls:     0,
			expectStartCalls:     1,
			expectConnectionInfo: true,
		},
		{
			name:                 "resumed cluster with explicit clients flag and different count scales after start",
			args:                 []string{"existing-cluster"},
			setClientsFlag:       true,
			flags:                &flagpole{timeout: DefaultStartTimeout, clients: 4},
			stub:                 &stubStartManager{startOutcome: domain.StartOutcomeResumed, clientCount: 1},
			expectScaleCalls:     1,
			expectStartCalls:     1,
			expectConnectionInfo: true,
		},
		{
			name:                 "noop cluster suppresses connection output",
			args:                 []string{"running-cluster"},
			flags:                &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0"},
			stub:                 &stubStartManager{startOutcome: domain.StartOutcomeNoOp},
			expectScaleCalls:     0,
			expectStartCalls:     1,
			expectConnectionInfo: false,
			expectOverridesInfo:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errOut.Reset()
			startCalls := 0
			scaleCalls := 0
			stub := *tt.stub
			stub.startCalls = &startCalls
			stub.scaleCalls = &scaleCalls

			checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
			newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
				return startServices{Orchestration: &stub, Active: &fakeActive{}}, nil
			}

			cobraCmd := NewCommand(logger, streams)
			if tt.setClientsFlag {
				err := cobraCmd.Flags().Set("clients", "4")
				require.NoError(t, err, "failed to set clients flag")
			}
			// Wire version flags so Changed() is set correctly for any flagpole version values.
			if tt.flags.nomadVersion != "" {
				require.NoError(t, cobraCmd.Flags().Set("nomad-version", tt.flags.nomadVersion))
			}
			if tt.flags.consulVersion != "" {
				require.NoError(t, cobraCmd.Flags().Set("consul-version", tt.flags.consulVersion))
			}
			if tt.flags.vaultVersion != "" {
				require.NoError(t, cobraCmd.Flags().Set("vault-version", tt.flags.vaultVersion))
			}

			err := runE(cobraCmd, context.Background(), logger, streams, tt.flags, tt.args)
			require.NoError(t, err, "runE returned unexpected error")

			assert.Equal(t, tt.expectStartCalls, startCalls, "Start calls = %d, want %d", startCalls, tt.expectStartCalls)
			assert.Equal(t, tt.expectScaleCalls, scaleCalls, "Scale calls = %d, want %d", scaleCalls, tt.expectScaleCalls)

			out := errOut.String()
			hasConnectionInfo := strings.Contains(out, "Connection information:")
			assert.Equal(t, tt.expectConnectionInfo, hasConnectionInfo,
				"connection info output = %t, want %t; output=%q", hasConnectionInfo, tt.expectConnectionInfo, out)
			hasOverridesInfo := strings.Contains(out, "Version overrides:")
			assert.Equal(t, tt.expectOverridesInfo, hasOverridesInfo,
				"version override output = %t, want %t; output=%q", hasOverridesInfo, tt.expectOverridesInfo, out)
		})
	}
}

func TestRunE_DisplaysVersionOverrides(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}

	tests := []struct {
		name            string
		flags           *flagpole
		flagsToSet      map[string]string // flag name -> value; causes Changed()=true
		envVars         map[string]string // only vars that should be present
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:            "no overrides emits no override lines",
			flags:           &flagpole{timeout: DefaultStartTimeout, clients: 1},
			wantNotContains: []string{"Version overrides:", "nomad:", "consul:", "vault:"},
		},
		{
			name:       "single override emits only that package",
			flags:      &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0"},
			flagsToSet: map[string]string{"nomad-version": "1.9.0"},
			wantContains: []string{
				"Version overrides:",
				"  nomad: 1.9.0",
			},
			wantNotContains: []string{"  consul:", "  vault:"},
		},
		{
			name:  "multiple overrides emit deterministic order",
			flags: &flagpole{timeout: DefaultStartTimeout, clients: 1, vaultVersion: "1.18.0", nomadVersion: "1.9.0", consulVersion: "1.19.2"},
			flagsToSet: map[string]string{
				"nomad-version":  "1.9.0",
				"consul-version": "1.19.2",
				"vault-version":  "1.18.0",
			},
			wantContains: []string{
				"Version overrides:",
				"  nomad: 1.9.0",
				"  consul: 1.19.2",
				"  vault: 1.18.0",
			},
		},
		{
			name:    "env-sourced overrides include (from env) attribution",
			flags:   &flagpole{timeout: DefaultStartTimeout, clients: 1},
			envVars: map[string]string{overrides.EnvNomad: "1.9.0", overrides.EnvConsul: "1.19.2"},
			wantContains: []string{
				"Version overrides:",
				"  nomad: 1.9.0 (from env)",
				"  consul: 1.19.2 (from env)",
			},
			wantNotContains: []string{"  vault:"},
		},
		{
			name:       "flag-sourced overrides do not include (from env) suffix",
			flags:      &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0"},
			flagsToSet: map[string]string{"nomad-version": "1.9.0"},
			wantContains: []string{
				"Version overrides:",
				"  nomad: 1.9.0",
			},
			wantNotContains: []string{"(from env)", "  consul:", "  vault:"},
		},
		{
			name:       "mixed flag and env emits suffix only on env-sourced line",
			flags:      &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0"},
			flagsToSet: map[string]string{"nomad-version": "1.9.0"},
			envVars:    map[string]string{overrides.EnvConsul: "1.19.2"},
			wantContains: []string{
				"Version overrides:",
				"  nomad: 1.9.0",
				"  consul: 1.19.2 (from env)",
			},
			wantNotContains: []string{"  vault:", "nomad: 1.9.0 (from env)"},
		},
	}

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset env per case.
			setEnvForTest(t, tt.envVars)

			errBuf := &bytes.Buffer{}
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: errBuf}
			cobraCmd := NewCommand(logger, streams)
			// Set flags so Cobra Changed() returns true.
			for name, val := range tt.flagsToSet {
				require.NoError(t, cobraCmd.Flags().Set(name, val))
			}

			checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
			newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
				return startServices{Orchestration: &stubStartManager{startOutcome: domain.StartOutcomeCreated}, Active: &fakeActive{}}, nil
			}

			err := runE(cobraCmd, context.Background(), logger, streams, tt.flags, []string{"test"})
			require.NoError(t, err, "runE returned unexpected error")

			out := errBuf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, out, want)
			}
			for _, unwanted := range tt.wantNotContains {
				assert.NotContains(t, out, unwanted)
			}

			if tt.name == "multiple overrides emit deterministic order" {
				nomadIdx := strings.Index(out, "  nomad: 1.9.0")
				consulIdx := strings.Index(out, "  consul: 1.19.2")
				vaultIdx := strings.Index(out, "  vault: 1.18.0")
				assert.True(t, nomadIdx != -1 && consulIdx != -1 && vaultIdx != -1,
					"override lines missing from output %q", out)
				assert.True(t, nomadIdx < consulIdx && consulIdx < vaultIdx,
					"override order was not nomad->consul->vault in output %q", out)
			}
		})
	}
}

// TestRunE_SingleWiresClusterService verifies that the factory is called exactly
// once regardless of whether the resolved name differs from "default".
func TestRunE_SingleWiresClusterService(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	setEnvForTest(t, nil)

	origDockerCheck := checkDockerDaemonFn
	checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
	defer func() { checkDockerDaemonFn = origDockerCheck }()

	origManagerFn := newStartServicesFn
	defer func() { newStartServicesFn = origManagerFn }()

	tests := []struct {
		name         string
		args         []string
		activeStored string
	}{
		{name: "explicit arg", args: []string{"explicit-cluster"}, activeStored: ""},
		{name: "no arg falls back to default", args: nil, activeStored: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
				callCount++
				return startServices{
					Orchestration: &stubStartManager{startOutcome: domain.StartOutcomeCreated},
					Active:        &fakeActive{stored: tt.activeStored},
				}, nil
			}

			logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
			flags := &flagpole{timeout: DefaultStartTimeout, clients: 1}
			cobraCmd := NewCommand(logger, streams)

			_ = runE(cobraCmd, context.Background(), logger, streams, flags, tt.args)

			assert.Equal(t, 1, callCount, "newStartServicesFn called %d times, want 1", callCount)
		})
	}
}

func TestClusterNameExtraction(t *testing.T) {
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
		{
			name:      "no args",
			args:      []string{},
			wantError: false,
		},
		{
			name:      "one arg",
			args:      []string{"test-cluster"},
			wantError: false,
		},
		{
			name:      "too many args",
			args:      []string{"cluster1", "cluster2"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := NewCommand(logger, streams)
			command.SetArgs(tt.args)
			err := command.Args(command, tt.args)
			if (err != nil) != tt.wantError {
				t.Errorf("Args validation error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRunE_FlagBeatsEnvVar(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	setEnvForTest(t, map[string]string{overrides.EnvNomad: "1.8.0"}) // should be ignored

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	var capturedReq orchestration.StartRequest
	checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error { return nil }
	newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
		return startServices{
			Orchestration: &capturingStartManager{captured: &capturedReq, outcome: domain.StartOutcomeCreated},
			Active:        &fakeActive{},
		}, nil
	}

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var errBuf bytes.Buffer
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: &errBuf}

	cobraCmd := NewCommand(logger, streams)
	require.NoError(t, cobraCmd.Flags().Set("nomad-version", "1.9.0"))

	flags := &flagpole{timeout: DefaultStartTimeout, clients: 1, nomadVersion: "1.9.0"}
	err := runE(cobraCmd, context.Background(), logger, streams, flags, []string{"test"})
	require.NoError(t, err)

	assert.Equal(t, "1.9.0", capturedReq.NomadVersion, "flag must win over env")
	out := errBuf.String()
	assert.Contains(t, out, "  nomad: 1.9.0")
	assert.NotContains(t, out, "nomad: 1.9.0 (from env)")
}

func TestRunE_InvalidEnvVarDoesNotCallDocker(t *testing.T) {
	setEnvForTest(t, map[string]string{overrides.EnvNomad: "@bad"})

	origDocker := checkDockerDaemonFn
	origFactory := newStartServicesFn
	defer func() {
		checkDockerDaemonFn = origDocker
		newStartServicesFn = origFactory
	}()

	dockerCalls := 0
	factoryCalls := 0
	checkDockerDaemonFn = func(_ context.Context, _ *log.Logger) error {
		dockerCalls++
		return nil
	}
	newStartServicesFn = func(_ *log.Logger, _ string) (startServices, error) {
		factoryCalls++
		return startServices{Orchestration: &stubStartManager{}, Active: &fakeActive{}}, nil
	}

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	cobraCmd := NewCommand(logger, streams)
	flags := &flagpole{timeout: DefaultStartTimeout, clients: 1}

	err := runE(cobraCmd, context.Background(), logger, streams, flags, []string{"test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NOMAD_VERSION")
	assert.Equal(t, 0, dockerCalls, "Docker daemon must not be probed on invalid env override")
	assert.Equal(t, 0, factoryCalls, "service factory must not be invoked on invalid env override")
}
