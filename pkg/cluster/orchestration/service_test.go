package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

func TestDefaultInspector_EmptySnapshot_ReturnsNotFoundError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		snap runtime.Snapshot
	}{
		{
			name: "nil network no containers",
			snap: runtime.Snapshot{Network: nil, Containers: nil},
		},
		{
			name: "nil network empty containers map",
			snap: runtime.Snapshot{Network: nil, Containers: map[domain.NodeName]runtime.ContainerResource{}},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			insp := defaultInspector{
				name: domain.Name("missing-cluster"),
				rt:   &fakeRuntime{snap: tc.snap},
			}
			_, err := insp.Inspect(context.Background())
			require.Error(t, err, "expected not-found error for empty snapshot")
			require.True(t, IsNotFound(err), "expected IsNotFound to be true, got: %v", err)
		})
	}
}

func TestDefaultInspector_NonEmptySnapshot_ReturnsResult(t *testing.T) {
	t.Parallel()
	containerName := domain.NodeName("hind.demo.server.01")
	snap := runtime.Snapshot{
		Network: &runtime.NetworkResource{Name: domain.NetworkName("hind.demo")},
		Containers: map[domain.NodeName]runtime.ContainerResource{
			containerName: {Name: containerName, Status: runtime.ContainerRunning},
		},
	}
	insp := defaultInspector{
		name: domain.Name("demo"),
		rt:   &fakeRuntime{snap: snap},
	}
	result, err := insp.Inspect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "hind.demo", result.NetworkName)
	assert.Len(t, result.Containers, 1)
}

type fakeLifecycle struct {
	startResult domain.StartOutcome
	startErr    error
	stopResult  StopResult
	stopErr     error
	deleteErr   error
}

func (f fakeLifecycle) Start(_ context.Context, _ StartRequest) (domain.StartOutcome, error) {
	return f.startResult, f.startErr
}
func (f fakeLifecycle) StopWithOptions(context.Context, StopOptions) (StopResult, error) {
	return f.stopResult, f.stopErr
}
func (f fakeLifecycle) Delete(context.Context) error { return f.deleteErr }

type fakeScaler struct{ err error }

func (f fakeScaler) Scale(context.Context, int) error { return f.err }

type fakeInspector struct {
	info InspectResult
	err  error
}

func (f fakeInspector) Inspect(context.Context) (InspectResult, error) { return f.info, f.err }

type fakeLister struct {
	names ListResult
	err   error
}

func (f fakeLister) List(context.Context) (ListResult, error) { return f.names, f.err }

func TestNewServiceDelegates(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")
	wantInfo := InspectResult{NetworkName: "hind.demo", Containers: []ContainerSummary{{Name: "c1", Status: domain.ContainerStatusRunning}}}
	svc := NewService(Options{
		Lifecycle: fakeLifecycle{startResult: domain.StartOutcomeResumed, stopResult: StopResult{StoppedCount: 1}},
		Scale:     fakeScaler{err: wantErr},
		Inspect:   fakeInspector{info: wantInfo},
		List:      fakeLister{names: ListResult{Names: []string{"a", "b"}}},
	})

	res, err := svc.Start(context.Background(), StartRequest{})
	require.NoError(t, err)
	assert.Equal(t, domain.StartOutcomeResumed, res)

	stopRes, err := svc.StopWithOptions(context.Background(), StopOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, stopRes.StoppedCount)

	err = svc.Delete(context.Background())
	require.NoError(t, err)

	err = svc.Scale(context.Background(), 3)
	require.ErrorIs(t, err, wantErr)

	info, err := svc.Inspect(context.Background())
	require.NoError(t, err)
	assert.Equal(t, wantInfo.NetworkName, info.NetworkName)
	assert.Len(t, info.Containers, 1)

	names, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, names.Names, 2)
}

func TestStartRequest_Validate_VersionTokenFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		req       StartRequest
		wantErr   bool
		errSubstr string // optional: substring expected in error
	}{
		{
			name:    "all empty (no overrides)",
			req:     StartRequest{},
			wantErr: false,
		},
		{
			name:    "canonical semver accepted",
			req:     StartRequest{NomadVersion: "1.9.5", ConsulVersion: "1.9.5+ent", VaultVersion: "1.9.5-beta.1"},
			wantErr: false,
		},
		{
			name:      "whitespace-only rejected as empty",
			req:       StartRequest{NomadVersion: "   "},
			wantErr:   true,
			errSubstr: "value must not be empty",
		},
		{
			name:      "leading underscore rejected",
			req:       StartRequest{ConsulVersion: "_1.0"},
			wantErr:   true,
			errSubstr: "--consul-version",
		},
		{
			name:      "leading slash rejected",
			req:       StartRequest{VaultVersion: "/1.0"},
			wantErr:   true,
			errSubstr: "--vault-version",
		},
		{
			name:      "embedded @ rejected (invalid format, not empty)",
			req:       StartRequest{NomadVersion: "@1.0"},
			wantErr:   true,
			errSubstr: "invalid format",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.req.Validate()
			if tc.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, errInvalidStartRequest)
				if tc.errSubstr != "" {
					assert.True(t, strings.Contains(err.Error(), tc.errSubstr), "error %q does not contain %q", err.Error(), tc.errSubstr)
				}
				return
			}
			require.NoError(t, err)
		})
	}
}
