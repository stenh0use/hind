package rm

import (
	"context"
	"errors"
	"io"
	"os"
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

// saveTestCluster persists a minimal cluster config so repository-backed helpers can find the cluster by name.
func saveTestCluster(t *testing.T, name string) error {
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

// setActiveCluster sets the active cluster via repository for test setup.
func setActiveCluster(t *testing.T, name string) {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	err = repo.SetActive(context.Background(), name)
	require.NoError(t, err, "repo.SetActive(%q)", name)
}

// getActiveCluster reads the active cluster via repository for test assertions.
func getActiveCluster(t *testing.T) string {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	name, err := repo.GetActive(context.Background())
	require.NoError(t, err, "repo.GetActive()")
	return name
}

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

	assert.Equal(t, "rm [cluster-name]", command.Use)
	assert.Equal(t, "Remove a hind cluster", command.Short)
}

func TestDefaultTimeout(t *testing.T) {
	expected := 2 * time.Minute
	assert.Equal(t, expected, DefaultDeleteTimeout)
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

	assert.Equal(t, "2m0s", timeoutFlag.DefValue)
}

// stubDeleter is a no-op clusterDeleter used to bypass real Docker calls in tests.
type stubDeleter struct {
	deleteErr error
}

func (s *stubDeleter) Delete(_ context.Context) error { return s.deleteErr }

var _ clusterDeleter = (*stubDeleter)(nil)

// TestRunE_ClearsActiveClusterOnDelete verifies that when the cluster being removed
// is the currently active cluster, runE calls ClearActiveCluster so that subsequent
// commands fall back to the "default" cluster resolution path.
func TestRunE_ReturnsErrorWhenDeleteFails(t *testing.T) {
	orig := newClusterManagerFn
	newClusterManagerFn = func(_ *log.Logger, _ string) (rmServices, error) {
		return rmServices{Orchestration: &stubDeleter{deleteErr: context.DeadlineExceeded}, Active: &fakeActive{}}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, []string{"ghost"})
	require.Error(t, err, "runE expected error for missing cluster delete")
	assert.Contains(t, err.Error(), "failed to delete cluster")
}

func TestRunE_NotFoundMapping(t *testing.T) {
	orig := newClusterManagerFn
	tests := []struct {
		name      string
		err       error
		wantToken string
	}{
		{name: "typed not found", err: &orchestration.NotFoundError{Operation: "delete", Cluster: "ghost"}, wantToken: "cluster 'ghost' not found"},
		{name: "generic wrapped error", err: context.DeadlineExceeded, wantToken: "failed to delete cluster"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			newClusterManagerFn = func(_ *log.Logger, _ string) (rmServices, error) {
				return rmServices{Orchestration: &stubDeleter{deleteErr: tt.err}, Active: &fakeActive{}}, nil
			}
			defer func() { newClusterManagerFn = orig }()

			logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
			streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
			err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, []string{"ghost"})
			require.Error(t, err, "runE expected error")
			assert.Contains(t, err.Error(), tt.wantToken)
		})
	}
}

func TestRunEUsesActiveClusterWhenNoArg(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := saveTestCluster(t, "active-cluster")
	require.NoError(t, err, "saveTestCluster")

	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")

	err = repo.SetActive(context.Background(), "active-cluster")
	require.NoError(t, err, "repo.SetActive")

	orig := newClusterManagerFn
	var gotName string
	callCount := 0
	newClusterManagerFn = func(_ *log.Logger, clusterName string) (rmServices, error) {
		callCount++
		gotName = clusterName
		return rmServices{Orchestration: &stubDeleter{}, Active: &fakeActive{stored: "active-cluster"}}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err = runE(context.Background(), logger, streams, DefaultDeleteTimeout, nil)
	require.NoError(t, err, "runE returned unexpected error")

	assert.Equal(t, 1, callCount, "newClusterManagerFn called %d times, want 1", callCount)
	assert.Equal(t, "active-cluster", gotName)
}

func TestRunEFallsBackToDefaultWhenNoActiveClusterAndNoArg(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	orig := newClusterManagerFn
	var gotName string
	callCount := 0
	newClusterManagerFn = func(_ *log.Logger, clusterName string) (rmServices, error) {
		callCount++
		gotName = clusterName
		return rmServices{Orchestration: &stubDeleter{}, Active: &fakeActive{}}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, nil)
	require.NoError(t, err, "runE returned unexpected error")

	assert.Equal(t, 1, callCount, "newClusterManagerFn called %d times, want 1", callCount)
	assert.Equal(t, "default", gotName)
}

func TestRunE_ClearsActiveClusterOnDelete(t *testing.T) {
	// Redirect HOME so cluster state is isolated to this test.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	const clusterName = "my-cluster"

	// Pre-create the cluster config so SetActiveCluster accepts the name.
	err := saveTestCluster(t, clusterName)
	require.NoError(t, err, "saveTestCluster")

	// Set the cluster as active.
	setActiveCluster(t, clusterName)

	// Replace the factory with a stub so Delete() succeeds without Docker.
	fa := &fakeActive{stored: clusterName}
	orig := newClusterManagerFn
	newClusterManagerFn = func(_ *log.Logger, _ string) (rmServices, error) {
		return rmServices{Orchestration: &stubDeleter{}, Active: fa}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err = runE(context.Background(), logger, streams, DefaultDeleteTimeout, []string{clusterName})
	require.NoError(t, err, "runE returned unexpected error")

	// After deletion of the active cluster the active cluster file must be cleared.
	assert.Empty(t, fa.stored, "expected active cluster to be cleared (empty string), got %q", fa.stored)
}

func TestRm_DeleteFailureBeforeRemoval_PreservesActiveProfile(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	for _, name := range []string{"active-a", "other-b"} {
		err := saveTestCluster(t, name)
		require.NoError(t, err, "saveTestCluster(%s)", name)
	}
	setActiveCluster(t, "active-a")

	orig := newClusterManagerFn
	newClusterManagerFn = func(_ *log.Logger, _ string) (rmServices, error) {
		return rmServices{Orchestration: &stubDeleter{deleteErr: errors.New("delete boom")}, Active: &fakeActive{stored: "active-a"}}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, []string{"active-a"})
	require.Error(t, err, "expected delete failure")

	active := getActiveCluster(t)
	assert.Equal(t, "active-a", active, "expected active cluster preserved as %q, got %q", "active-a", active)
}

func TestRunE_PreservesActiveWhenRemovingNonActive(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	for _, name := range []string{"active-a", "other-b"} {
		err := saveTestCluster(t, name)
		require.NoError(t, err, "saveTestCluster(%s)", name)
	}
	// Test setup uses the repository directly; production code uses the seam.
	setActiveCluster(t, "active-a")

	orig := newClusterManagerFn
	newClusterManagerFn = func(_ *log.Logger, _ string) (rmServices, error) {
		return rmServices{Orchestration: &stubDeleter{}, Active: &fakeActive{stored: "active-a"}}, nil
	}
	defer func() { newClusterManagerFn = orig }()

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	err := runE(context.Background(), logger, streams, DefaultDeleteTimeout, []string{"other-b"})
	require.NoError(t, err, "runE returned unexpected error")

	active := getActiveCluster(t)
	assert.Equal(t, "active-a", active, "expected active cluster preserved as %q, got %q", "active-a", active)
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
