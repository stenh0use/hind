package set

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	persistencefs "github.com/stenh0use/hind/pkg/cluster/persistence/fs"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/version"
)

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

func setActiveCluster(t *testing.T, name string) {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	err = repo.SetActive(context.Background(), name)
	require.NoError(t, err, "repo.SetActive() error")
}

func getActiveCluster(t *testing.T) string {
	t.Helper()
	repo, err := persistencefs.NewRepository()
	require.NoError(t, err, "NewRepository() error")
	active, err := repo.GetActive(context.Background())
	require.NoError(t, err, "repo.GetActive() error")
	return active
}

var _ persistence.ActiveRepository = (*persistencefs.Repository)(nil)

func TestSetProfile_SetsActiveForExistingCluster(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	testClusterName := "test-cluster"
	saveTestCluster(t, testClusterName)

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile", testClusterName})

	err := command.Execute()
	require.NoError(t, err, "Command execution failed")

	activeCluster := getActiveCluster(t)
	assert.Equal(t, testClusterName, activeCluster)
}

func TestSetProfile_NoActive_MissingTarget_ReturnsNotFoundAndRemainsUnset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	command := NewCommand(logger, streams)
	clusterName := "missing"
	command.SetArgs([]string{"profile", clusterName})

	err := command.Execute()
	require.Error(t, err, "expected error for missing cluster")

	errMsg := err.Error()
	assert.True(t, strings.Contains(errMsg, "cluster") && strings.Contains(errMsg, "not found") && strings.Contains(errMsg, clusterName),
		"expected not-found contract substrings in %q", errMsg)

	active := getActiveCluster(t)
	assert.Empty(t, active, "expected no active cluster, got %q", active)
}

func TestSetProfile_MissingTarget_PreservesExistingActive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	activeCluster := "active-a"
	saveTestCluster(t, activeCluster)
	setActiveCluster(t, activeCluster)

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile", "missing"})

	err := command.Execute()
	require.Error(t, err, "expected error for missing cluster")

	active := getActiveCluster(t)
	assert.Equal(t, activeCluster, active, "expected active cluster %q to be preserved, got %q", activeCluster, active)
}

func TestSetProfile_LookupRuntimeError_ReturnsOperationalErrorAndPreservesActive(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	activeCluster := "active-a"
	saveTestCluster(t, activeCluster)
	setActiveCluster(t, activeCluster)

	origLookup := lookupProfileFn
	lookupProfileFn = func(_ context.Context, _ clusterSetter, _ string) (bool, error) {
		return false, errors.New("disk failure")
	}
	t.Cleanup(func() { lookupProfileFn = origLookup })

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile", "any"})

	err := command.Execute()
	require.Error(t, err, "expected operational lookup error")

	errMsg := err.Error()
	assert.True(t, strings.Contains(errMsg, "failed") && strings.Contains(errMsg, "cluster"),
		"expected operational error context in %q", errMsg)
	assert.NotContains(t, errMsg, "not found", "unexpected logical not-found contract for operational error: %q", errMsg)

	active := getActiveCluster(t)
	assert.Equal(t, activeCluster, active, "expected active cluster %q to be preserved, got %q", activeCluster, active)
}

func TestSetProfile_UsesManagerGetNormalizationParity(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Create a cluster with exact name to test lookup matches it.
	saveTestCluster(t, "mixed-name")

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile", "mixed-name"})

	err := command.Execute()
	require.NoError(t, err, "expected normalization-parity success")
}

func TestSetProfileCommand_NoArgs(t *testing.T) {
	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{
		Out:    io.Discard,
		ErrOut: io.Discard,
	}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile"})

	err := command.Execute()
	require.Error(t, err, "Expected error when no cluster name provided, got nil")
}

func TestSetProfile_SuccessOutputUsesStdoutOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	clusterName := "stream-cluster"
	saveTestCluster(t, clusterName)

	logger := &log.Logger{Handler: discard.New()}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	streams := cmd.IOStreams{Out: &stdout, ErrOut: &stderr}

	command := NewCommand(logger, streams)
	command.SetArgs([]string{"profile", clusterName})
	err := command.Execute()
	require.NoError(t, err, "Command execution failed")

	assert.Contains(t, stdout.String(), "Active cluster profile set to '")
	assert.Empty(t, stderr.String(), "expected no success output on stderr, got %q", stderr.String())
}

func TestSetProfile_LookupReceivesCommandContext(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	clusterName := "context-cluster"
	saveTestCluster(t, clusterName)

	type ctxKey string
	const key ctxKey = "trace-id"
	const value = "set-profile-context"
	cmdCtx := context.WithValue(context.Background(), key, value)

	origLookup := lookupProfileFn
	lookupProfileFn = func(ctx context.Context, _ clusterSetter, _ string) (bool, error) {
		got := ctx.Value(key)
		assert.Equal(t, value, got, "expected lookup context value %q, got %v", value, got)
		return true, nil
	}
	t.Cleanup(func() { lookupProfileFn = origLookup })

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	command := NewCommand(logger, streams)
	command.SetContext(cmdCtx)
	command.SetArgs([]string{"profile", clusterName})

	err := command.Execute()
	require.NoError(t, err, "command execution failed")
}

func TestSetProfile_CanceledContextPropagatesToLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	origLookup := lookupProfileFn
	lookupProfileFn = func(ctx context.Context, _ clusterSetter, _ string) (bool, error) {
		return false, ctx.Err()
	}
	t.Cleanup(func() { lookupProfileFn = origLookup })

	logger := &log.Logger{Handler: discard.New()}
	streams := cmd.IOStreams{Out: io.Discard, ErrOut: io.Discard}
	command := NewCommand(logger, streams)
	command.SetContext(ctx)
	command.SetArgs([]string{"profile", "ignored"})

	err := command.Execute()
	require.Error(t, err, "expected error when command context is canceled")
	assert.ErrorIs(t, err, context.Canceled, "expected wrapped context cancellation error, got %v", err)
}
