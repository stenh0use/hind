package cluster_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cmd/hind/internal/cluster"
)

func testLogger() *log.Logger {
	return &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
}

type fakeActive struct {
	stored string
	getErr error
	setErr error
	clrErr error
}

func (f *fakeActive) GetActive(_ context.Context) (string, error) { return f.stored, f.getErr }
func (f *fakeActive) SetActive(_ context.Context, n string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.stored = n
	return nil
}
func (f *fakeActive) ClearActive(_ context.Context) error {
	if f.clrErr != nil {
		return f.clrErr
	}
	f.stored = ""
	return nil
}

func TestResolveActive_ReturnsActiveWhenSet(t *testing.T) {
	got := cluster.ResolveActive(context.Background(), &fakeActive{stored: "prod"})
	assert.Equal(t, "prod", got)
}

func TestResolveActive_ReturnsDefaultWhenEmpty(t *testing.T) {
	got := cluster.ResolveActive(context.Background(), &fakeActive{})
	assert.Equal(t, "default", got)
}

func TestResolveActive_ReturnsDefaultOnError(t *testing.T) {
	got := cluster.ResolveActive(context.Background(), &fakeActive{getErr: errors.New("io")})
	assert.Equal(t, "default", got)
}

func TestClearActiveIfMatch_ClearsWhenNamesMatch(t *testing.T) {
	a := &fakeActive{stored: "staging"}
	err := cluster.ClearActiveIfMatch(context.Background(), a, "staging")
	require.NoError(t, err, "ClearActiveIfMatch")
	assert.Empty(t, a.stored, "stored = %q, want empty", a.stored)
}

func TestClearActiveIfMatch_NoopWhenNamesDiffer(t *testing.T) {
	a := &fakeActive{stored: "staging"}
	err := cluster.ClearActiveIfMatch(context.Background(), a, "other")
	require.NoError(t, err, "ClearActiveIfMatch")
	assert.Equal(t, "staging", a.stored, "stored = %q, want staging", a.stored)
}

func TestClearActiveIfMatch_NoopOnGetActiveError(t *testing.T) {
	a := &fakeActive{getErr: errors.New("io")}
	err := cluster.ClearActiveIfMatch(context.Background(), a, "x")
	require.NoError(t, err, "ClearActiveIfMatch")
}

func TestNewClusterServices_RejectsInvalidName(t *testing.T) {
	for _, name := range []string{"", "../etc", "a/b"} {
		_, err := cluster.NewClusterServices(testLogger(), name)
		require.Error(t, err, "NewClusterServices(%q) expected error", name)
	}
}

func TestNewClusterServices_ReturnsPopulatedComposite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	svc, err := cluster.NewClusterServices(testLogger(), "valid")
	require.NoError(t, err, "NewClusterServices")
	assert.NotNil(t, svc.Orchestration, "Orchestration is nil")
	assert.NotNil(t, svc.Active, "Active is nil")
}

// TestResolveClusterNameFromFS_ExplicitArgWins verifies that an explicit positional
// arg takes precedence over whatever is persisted as the active cluster.
func TestResolveClusterNameFromFS_ExplicitArgWins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	got := cluster.ResolveClusterNameFromFS(context.Background(), []string{"explicit"})
	assert.Equal(t, "explicit", got)
}

// TestResolveClusterNameFromFS_FallsBackToDefault verifies that when no arg is provided
// and no active cluster is persisted, the function returns "default".
func TestResolveClusterNameFromFS_FallsBackToDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // empty HOME → no active file
	got := cluster.ResolveClusterNameFromFS(context.Background(), nil)
	assert.Equal(t, "default", got)
}
