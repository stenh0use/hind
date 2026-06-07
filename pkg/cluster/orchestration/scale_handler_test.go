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

func TestScaleHandler_Scale_ContractTable(t *testing.T) {
	t.Parallel()

	baseCluster, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)

	emptySnap := runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}

	tests := []struct {
		name              string
		repoSetup         func() *fakeRepo
		snap              runtime.Snapshot
		inspErr           error
		applyErr          error
		waiterErr         error
		targetClientCount int
		wantErr           bool
		wantErrPhase      string
		wantClientCount   int // validated against persisted config on success
	}{
		{
			name: "scale up succeeds and persists new client count",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:              emptySnap,
			targetClientCount: 3,
			wantClientCount:   3,
		},
		{
			name: "scale to invalid count (0) returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:              emptySnap,
			targetClientCount: 0,
			wantErr:           true,
			wantErrPhase:      "scale: set client count",
		},
		{
			name: "repo load failure returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.loadErr = errors.New("disk error")
				return r
			},
			snap:              emptySnap,
			targetClientCount: 2,
			wantErr:           true,
			wantErrPhase:      "scale: load",
		},
		{
			name: "runtime inspect failure returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:              emptySnap,
			inspErr:           errors.New("docker unavailable"),
			targetClientCount: 2,
			wantErr:           true,
			wantErrPhase:      "scale: inspect",
		},
		{
			name: "apply failure returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:              emptySnap,
			applyErr:          errors.New("docker run failed"),
			targetClientCount: 2,
			wantErr:           true,
			wantErrPhase:      "scale: apply",
		},
		{
			name: "waiter failure returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:              emptySnap,
			waiterErr:         errors.New("timeout"),
			targetClientCount: 2,
			wantErr:           true,
			wantErrPhase:      "scale: wait",
		},
		{
			name: "repo save failure returns error",
			repoSetup: func() *fakeRepo {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				r.saveErr = errors.New("write failed")
				return r
			},
			snap:              emptySnap,
			targetClientCount: 2,
			wantErr:           true,
			wantErrPhase:      "scale: save",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repoSetup()
			rt := &fakeRuntime{snap: tt.snap, inspErr: tt.inspErr, applyErr: tt.applyErr}
			waiter := &fakeWaiter{err: tt.waiterErr}

			h, err := NewScaleHandler(ScaleHandlerOptions{
				Name:    "test",
				Repo:    repo,
				Runtime: rt,
				Waiter:  waiter,
			})
			require.NoError(t, err)

			err = h.Scale(context.Background(), tt.targetClientCount)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrPhase != "" {
					assert.True(t, strings.Contains(err.Error(), tt.wantErrPhase), "Scale() error = %q, want to contain %q", err.Error(), tt.wantErrPhase)
				}
				return
			}
			require.NoError(t, err)
			// Verify persisted client count matches target.
			saved, ok := repo.clusters["test"]
			require.True(t, ok, "Scale() did not persist cluster")
			clientCount := 0
			for _, n := range saved.Nodes {
				if n.Role == domain.RoleClient {
					clientCount++
				}
			}
			assert.Equal(t, tt.wantClientCount, clientCount)
		})
	}
}
