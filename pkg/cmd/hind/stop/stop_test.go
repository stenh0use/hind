package stop

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	"github.com/stenh0use/hind/pkg/version"
)

func testLogger() *log.Logger {
	return &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
}

func saveStopTestCluster(t *testing.T, name string) error {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	if err != nil {
		return err
	}
	c, err := domain.BuildDefaultCluster(name, version.HindVersion)
	if err != nil {
		return err
	}
	return repo.Save(context.Background(), c)
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

type fakeStopManager struct {
	result       orchestration.StopResult
	err          error
	receivedOpts orchestration.StopOptions
}

func (f *fakeStopManager) StopWithOptions(_ context.Context, opts orchestration.StopOptions) (orchestration.StopResult, error) {
	f.receivedOpts = opts
	if f.err != nil {
		return orchestration.StopResult{}, f.err
	}
	return f.result, nil
}

func withRunEStubs(t *testing.T, active string, build func() clusterStopper) {
	t.Helper()
	oldNewSvc := newStopServicesFn
	newStopServicesFn = func(_ *log.Logger, _ string) (stopServices, error) {
		mgr := build()
		fa := &fakeActive{stored: active}
		return stopServices{Orchestration: mgr, Active: fa}, nil
	}
	t.Cleanup(func() {
		newStopServicesFn = oldNewSvc
	})
}

func TestRunEMessageContracts(t *testing.T) {
	tests := []struct {
		name              string
		force             bool
		result            orchestration.StopResult
		wantSummaryToken  string
		wantContains      []string
		wantExitCode      int
		wantForceOpt      bool
		wantOrderedTokens []string
	}{
		{
			name:             "already stopped",
			result:           orchestration.StopResult{Outcome: orchestration.StopOutcomeAlreadyStopped, AlreadyStoppedCount: 2},
			wantSummaryToken: "already stopped",
			wantContains:     []string{"Cluster 'default'", "already stopped"},
			wantExitCode:     0,
		},
		{
			name:             "force stopped",
			force:            true,
			result:           orchestration.StopResult{Outcome: orchestration.StopOutcomeSuccess, StoppedCount: 2},
			wantSummaryToken: "force stopped",
			wantContains:     []string{"Cluster 'default'", "force stopped"},
			wantExitCode:     0,
			wantForceOpt:     true,
		},
		{
			name:             "partial failure warnings",
			result:           orchestration.StopResult{Outcome: orchestration.StopOutcomePartialFailure, StoppedCount: 1, FailedCount: 1, Failures: []string{"n2"}},
			wantSummaryToken: "partially stopped",
			wantContains:     []string{"Failed to stop container 'n2'", "partially stopped"},
			wantExitCode:     1,
		},
		{
			name:             "force with partial failure still errors",
			force:            true,
			result:           orchestration.StopResult{Outcome: orchestration.StopOutcomePartialFailure, StoppedCount: 1, FailedCount: 1, Failures: []string{"n2"}},
			wantSummaryToken: "partially stopped",
			wantContains:     []string{"Failed to stop container 'n2'", "partially stopped"},
			wantExitCode:     1,
			wantForceOpt:     true,
		},
		{
			// W-017 AC5 fallback assertion: command-layer coverage must keep
			// health-complication wording explicit when top-level e2e is absent.
			name: "W-017 AC5 fallback: unhealthy pre-failed keeps health token and exit semantics",
			result: orchestration.StopResult{
				Outcome:             orchestration.StopOutcomeDegradedPreFailed,
				AlreadyStoppedCount: 1,
				FailedPreStopCount:  1,
			},
			wantSummaryToken: "already failed",
			wantContains: []string{
				"Cluster 'default'",
				"already failed",
			},
			wantExitCode: 0,
		},
		{
			// W-017 AC5 fallback assertion: equivalent token coverage for
			// unresponsive containers at command layer.
			name: "W-017 AC5 fallback: unresponsive pre-failed keeps health token and exit semantics",
			result: orchestration.StopResult{
				Outcome:             orchestration.StopOutcomeDegradedPreFailed,
				AlreadyStoppedCount: 1,
				FailedPreStopCount:  1,
			},
			wantSummaryToken: "already failed",
			wantContains: []string{
				"Cluster 'default'",
				"already failed",
			},
			wantExitCode: 0,
		},
		{
			name:             "stopped container output",
			result:           orchestration.StopResult{Outcome: orchestration.StopOutcomeSuccess, StoppedCount: 1, StoppedContainers: []string{"n1"}},
			wantSummaryToken: "stopped successfully",
			wantContains: []string{
				"Stopped container 'n1'",
				"Cluster 'default' stopped successfully",
			},
			wantOrderedTokens: []string{
				"Stopped container 'n1'",
				"stopped successfully",
			},
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &fakeStopManager{result: tt.result}
			withRunEStubs(t, "", func() clusterStopper { return mgr })
			errBuf := &bytes.Buffer{}
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: errBuf}

			err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, tt.force, nil)
			if tt.wantExitCode == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to stop 1 container(s)")
			}

			out := errBuf.String()
			for _, token := range tt.wantContains {
				assert.Contains(t, out, token, "output missing token %q", token)
			}
			assert.Equal(t, 1, strings.Count(out, tt.wantSummaryToken),
				"summary token %q count want=1 output=%q", tt.wantSummaryToken, out)
			if len(tt.wantOrderedTokens) > 0 {
				last := -1
				for _, token := range tt.wantOrderedTokens {
					idx := strings.Index(out, token)
					assert.NotEqual(t, -1, idx, "ordered token missing %q in output %q", token, out)
					assert.Greater(t, idx, last, "token %q appeared out of order in output %q", token, out)
					last = idx
				}
			}
			assert.Equal(t, tt.wantForceOpt, mgr.receivedOpts.Force)
		})
	}
}

// W-017 fallback traceability: this command-level suite is the approved
// fallback when /test/e2e harness coverage is absent.
func TestRunEStopErrorHardAbortNoSummaryLine(t *testing.T) {
	mgr := &fakeStopManager{err: errors.New("boom")}
	withRunEStubs(t, "", func() clusterStopper { return mgr })
	errBuf := &bytes.Buffer{}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: errBuf}

	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop cluster")

	out := errBuf.String()
	assert.False(t,
		strings.Contains(out, "stopped successfully") ||
			strings.Contains(out, "partially stopped") ||
			strings.Contains(out, "already stopped") ||
			strings.Contains(out, "force stopped") ||
			strings.Contains(out, "already failed"),
		"expected no final summary line on hard abort path, got output %q", out)
}

func TestRunEClusterNotFound(t *testing.T) {
	mgr := &fakeStopManager{err: &orchestration.NotFoundError{Operation: "stop", Cluster: "default"}}
	withRunEStubs(t, "", func() clusterStopper { return mgr })
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cluster 'default' not found")
}

func TestRunEUsesActiveClusterWhenNoArg(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := saveStopTestCluster(t, "active-cluster")
	require.NoError(t, err, "saveStopTestCluster")

	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	err = repo.SetActive(context.Background(), "active-cluster")
	require.NoError(t, err, "repo.SetActive")

	oldNewSvc := newStopServicesFn
	var gotName string
	callCount := 0
	newStopServicesFn = func(_ *log.Logger, clusterName string) (stopServices, error) {
		callCount++
		gotName = clusterName
		return stopServices{
			Orchestration: &fakeStopManager{result: orchestration.StopResult{Outcome: orchestration.StopOutcomeAlreadyStopped, AlreadyStoppedCount: 1}},
			Active:        &fakeActive{stored: "active-cluster"},
		}, nil
	}
	t.Cleanup(func() {
		newStopServicesFn = oldNewSvc
	})

	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	err = runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "newStopServicesFn called %d times, want 1", callCount)
	assert.Equal(t, "active-cluster", gotName)
}

func TestRunEFallsBackToDefaultWhenNoActiveClusterAndNoArg(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	oldNewSvc := newStopServicesFn
	var gotName string
	callCount := 0
	newStopServicesFn = func(_ *log.Logger, clusterName string) (stopServices, error) {
		callCount++
		gotName = clusterName
		return stopServices{
			Orchestration: &fakeStopManager{result: orchestration.StopResult{Outcome: orchestration.StopOutcomeAlreadyStopped, AlreadyStoppedCount: 1}},
			Active:        &fakeActive{},
		}, nil
	}
	t.Cleanup(func() {
		newStopServicesFn = oldNewSvc
	})

	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	err := runE(context.Background(), testLogger(), streams, DefaultStopTimeout, false, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "newStopServicesFn called %d times, want 1", callCount)
	assert.Equal(t, "default", gotName)
}

func TestNewCommand(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	require.NotNil(t, command, "NewCommand() returned nil")

	assert.Equal(t, "stop [cluster-name]", command.Use)
	assert.Equal(t, "Stop a hind cluster", command.Short)
}

func TestDefaultTimeout(t *testing.T) {
	expected := 30 * time.Second
	assert.Equal(t, expected, DefaultStopTimeout)
}

func TestCommandFlags(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)

	// Check flags exist
	timeoutFlag := command.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag, "Expected 'timeout' flag to exist")
	forceFlag := command.Flags().Lookup("force")
	require.NotNil(t, forceFlag, "Expected 'force' flag to exist")

	assert.Equal(t, "30s", timeoutFlag.DefValue)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestCommandArgs(t *testing.T) {
	logger := testLogger()
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	// Test with valid number of args (0 or 1)
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
