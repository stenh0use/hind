package list

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cmd"
	"github.com/stenh0use/hind/pkg/file"
)

// fakeActive is a minimal in-memory ActiveRepository for tests.
type fakeActive struct {
	stored    string
	activeErr error
}

func (f *fakeActive) GetActive(_ context.Context) (string, error) { return f.stored, f.activeErr }
func (f *fakeActive) SetActive(_ context.Context, n string) error {
	f.stored = n
	return nil
}
func (f *fakeActive) ClearActive(_ context.Context) error {
	f.stored = ""
	return nil
}

var _ persistence.ActiveRepository = (*fakeActive)(nil)

type stubOrchListService struct {
	result orchestration.ListResult
	err    error
}

func (s *stubOrchListService) List(_ context.Context) (orchestration.ListResult, error) {
	if s.err != nil {
		return orchestration.ListResult{}, s.err
	}
	return s.result, nil
}

func withRunEStubs(t *testing.T, orchSvc clusterListService, activeCluster string, activeErr error, statuses map[string]*clusterStatus, statusErrs map[string]error) {
	t.Helper()
	oldListSvc := newClusterListServices
	oldStatusFn := getClusterStatusFn

	newClusterListServices = func(_ *log.Logger) (listServices, error) {
		return listServices{
			Orchestration: orchSvc,
			Active:        &fakeActive{stored: activeCluster, activeErr: activeErr},
		}, nil
	}
	getClusterStatusFn = func(_ context.Context, _ *log.Logger, clusterName string, _ time.Duration) (*clusterStatus, error) {
		if statusErrs != nil {
			if err, ok := statusErrs[clusterName]; ok {
				return nil, err
			}
		}
		if statuses != nil {
			if status, ok := statuses[clusterName]; ok {
				return status, nil
			}
		}
		return nil, fmt.Errorf("missing status for %s", clusterName)
	}

	t.Cleanup(func() {
		newClusterListServices = oldListSvc
		getClusterStatusFn = oldStatusFn
	})
}

func TestRunE_NoClustersOnFirstRunWhenConfigDirMissing(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	streams := cmd.IOStreams{In: strings.NewReader(""), Out: &stdout, ErrOut: &stderr}

	withRunEStubs(t, &stubOrchListService{result: orchestration.ListResult{Names: nil}}, "", nil, nil, nil)

	err := runE(
		context.Background(),
		logger,
		streams,
		DefaultListTimeout,
	)
	require.NoError(t, err, "runE() returned error on first-run missing config dir")

	assert.Contains(t, stderr.String(), "No clusters found")
	assert.Empty(t, stdout.String(), "expected no stdout table output")
}

func TestRunE_TableFormattingAndActiveMarkerCompatibility(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	streams := cmd.IOStreams{In: strings.NewReader(""), Out: &stdout, ErrOut: &stderr}

	withRunEStubs(t,
		&stubOrchListService{result: orchestration.ListResult{Names: []string{"alpha", "beta"}}},
		"beta",
		nil,
		map[string]*clusterStatus{
			"alpha": {Status: "running", RunningNodes: 2, TotalNodes: 2},
			"beta":  {Status: "not-found"},
		},
		nil,
	)

	err := runE(
		context.Background(),
		logger,
		streams,
		DefaultListTimeout,
	)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "ACTIVE")
	assert.Contains(t, output, "NODES")
	assert.Contains(t, output, "alpha")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "2/2")
	assert.Contains(t, output, "beta")
	assert.Contains(t, output, "*")
	assert.Contains(t, output, "not-found")
	assert.Contains(t, output, "-")
	assert.Empty(t, stderr.String(), "expected empty stderr on successful table output")
}

func TestRunE_StatusErrorFallsBackToErrorRowCompatibility(t *testing.T) {
	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	var stdout bytes.Buffer
	streams := cmd.IOStreams{In: strings.NewReader(""), Out: &stdout, ErrOut: &bytes.Buffer{}}

	withRunEStubs(t,
		&stubOrchListService{result: orchestration.ListResult{Names: []string{"alpha"}}},
		"",
		nil,
		nil,
		map[string]error{"alpha": errors.New("boom")},
	)

	err := runE(
		context.Background(),
		logger,
		streams,
		DefaultListTimeout,
	)
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "alpha")
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "-")
}

func TestFormatCreatedTime_JustNow(t *testing.T) {
	now := time.Now().Add(-30 * time.Second)
	result := formatCreatedTime(now)

	assert.Equal(t, "just now", result)
}

func TestFormatCreatedTime_Minutes(t *testing.T) {
	past := time.Now().Add(-5 * time.Minute)
	result := formatCreatedTime(past)

	assert.Equal(t, "5m ago", result)
}

func TestFormatCreatedTime_Hours(t *testing.T) {
	past := time.Now().Add(-3 * time.Hour)
	result := formatCreatedTime(past)

	assert.Equal(t, "3h ago", result)
}

func TestFormatCreatedTime_Days(t *testing.T) {
	past := time.Now().Add(-2 * 24 * time.Hour)
	result := formatCreatedTime(past)

	assert.Equal(t, "2d ago", result)
}

func TestFormatCreatedTime_AbsoluteDate(t *testing.T) {
	past := time.Now().Add(-10 * 24 * time.Hour)
	result := formatCreatedTime(past)

	expected := past.Format("2006-01-02")
	assert.Equal(t, expected, result)
}

func TestFormatCreatedTime_ZeroTime(t *testing.T) {
	zeroTime := time.Time{}
	result := formatCreatedTime(zeroTime)

	assert.Equal(t, "unknown", result)
}

type failAfterWriter struct {
	writesLeft int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.writesLeft <= 0 {
		return 0, errors.New("write failed")
	}
	w.writesLeft--
	return len(p), nil
}

func TestRunE_ReturnsErrorWhenTableFlushFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Use the same path layout as persistence/fs: $HOME/.config/hind/cluster/<name>
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "failed to get user home dir")

	rootDir := filepath.Clean(filepath.Join(homeDir, ".config", "hind"))
	fm, err := file.New(rootDir)
	require.NoError(t, err, "failed to create file manager")

	err = fm.EnsureDir(filepath.Join("cluster", "flush-cluster"))
	require.NoError(t, err, "failed to create test cluster dir")

	logger := &log.Logger{Handler: discard.New(), Level: log.ErrorLevel}
	streams := cmd.IOStreams{In: strings.NewReader(""), Out: &failAfterWriter{writesLeft: 1}, ErrOut: &bytes.Buffer{}}

	err = runE(context.Background(), logger, streams, DefaultListTimeout)
	require.Error(t, err, "expected flush/write error, got nil")
}
