package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

// fakeRuntime is a test double for runtime.Runtime.
type fakeRuntime struct {
	snap     runtime.Snapshot
	inspErr  error
	applyErr error
	applied  []runtime.Operation
}

func (f *fakeRuntime) Inspect(_ context.Context, _ runtime.Selector) (runtime.Snapshot, error) {
	return f.snap, f.inspErr
}

func (f *fakeRuntime) Apply(_ context.Context, op runtime.Operation) (runtime.OperationResult, error) {
	f.applied = append(f.applied, op)
	return runtime.OperationResult{Resource: op.Resource}, f.applyErr
}

// fakeRepo implements persistence.Repository in memory.
type fakeRepo struct {
	clusters map[domain.Name]domain.Cluster
	saveErr  error
	loadErr  error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{clusters: map[domain.Name]domain.Cluster{}}
}

func (r *fakeRepo) Load(_ context.Context, name domain.Name) (domain.Cluster, error) {
	if r.loadErr != nil {
		return domain.Cluster{}, r.loadErr
	}
	c, ok := r.clusters[name]
	if !ok {
		return domain.Cluster{}, fmt.Errorf("not found")
	}
	return c, nil
}

func (r *fakeRepo) Save(_ context.Context, c domain.Cluster) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.clusters[c.Name] = c
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, name domain.Name) error {
	if _, ok := r.clusters[name]; !ok {
		return fmt.Errorf("cluster not found")
	}
	delete(r.clusters, name)
	return nil
}

func (r *fakeRepo) Exists(_ context.Context, name domain.Name) (bool, error) {
	_, ok := r.clusters[name]
	return ok, nil
}

func (r *fakeRepo) List(_ context.Context) ([]domain.Name, error) {
	names := make([]domain.Name, 0, len(r.clusters))
	for n := range r.clusters {
		names = append(names, n)
	}
	return names, nil
}

// fakeWaiter is a Waiter that always succeeds or fails immediately.
type fakeWaiter struct{ err error }

func (f *fakeWaiter) WaitRunning(_ context.Context, _ domain.Cluster) error {
	return f.err
}

func TestLifecycleHandler_Start_NewCluster(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{
		Name:    "test",
		Repo:    repo,
		Runtime: rt,
		Waiter:  &fakeWaiter{},
	})
	require.NoError(t, err)

	result, err := h.Start(context.Background(), StartRequest{})
	require.NoError(t, err)
	assert.Equal(t, domain.StartOutcomeCreated, result)
	_, ok := repo.clusters["test"]
	assert.True(t, ok, "Start() did not persist cluster")
}

func TestLifecycleHandler_Start_ExistingRunningCluster(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	// Pre-populate repo with a running cluster.
	c, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)
	repo.clusters["test"] = c

	// Runtime shows all containers running.
	containers := map[domain.NodeName]runtime.ContainerResource{}
	for _, n := range c.Nodes {
		containers[n.Name] = runtime.ContainerResource{Name: n.Name, Status: runtime.ContainerRunning}
	}
	rt := &fakeRuntime{snap: runtime.Snapshot{
		Network:    &runtime.NetworkResource{Name: "hind.test"},
		Containers: containers,
	}}

	h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{}})
	require.NoError(t, err)
	result, err := h.Start(context.Background(), StartRequest{})
	require.NoError(t, err)
	assert.Equal(t, domain.StartOutcomeNoOp, result)
}

func TestLifecycleHandler_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{}})
	require.NoError(t, err)

	err = h.Delete(context.Background())
	require.Error(t, err, "Delete() should error for missing cluster")
	_, ok := err.(*NotFoundError)
	assert.True(t, ok, "Delete() expected *NotFoundError, got: %T (%v)", err, err)
}

func TestLifecycleHandler_Stop_NotFound(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{}})
	require.NoError(t, err)

	_, err = h.StopWithOptions(context.Background(), StopOptions{})
	require.Error(t, err, "StopWithOptions() should error for missing cluster")
	_, ok := err.(*NotFoundError)
	assert.True(t, ok, "StopWithOptions() expected *NotFoundError, got: %T (%v)", err, err)
}

func TestLifecycleHandler_Start_WaiterTimeout(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	wantErr := errors.New("timeout")
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{err: wantErr}})
	require.NoError(t, err)

	_, err = h.Start(context.Background(), StartRequest{})
	require.Error(t, err, "Start() should fail when waiter times out")
	msg := err.Error()
	assert.Contains(t, msg, "start: wait")
	assert.Contains(t, msg, "timeout")
}

func TestLifecycleHandler_StopOutcomeClassification(t *testing.T) {
	t.Parallel()

	baseCluster, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)

	buildSnapshot := func(status runtime.ContainerStatus) runtime.Snapshot {
		containers := map[domain.NodeName]runtime.ContainerResource{}
		for _, n := range baseCluster.Nodes {
			containers[n.Name] = runtime.ContainerResource{Name: n.Name, Status: status}
		}
		return runtime.Snapshot{Containers: containers}
	}

	tests := []struct {
		name        string
		snapshot    runtime.Snapshot
		applyErr    error
		opts        StopOptions
		wantOutcome StopOutcome
	}{
		{name: "already stopped", snapshot: buildSnapshot(runtime.ContainerStopped), wantOutcome: StopOutcomeAlreadyStopped},
		{name: "success", snapshot: buildSnapshot(runtime.ContainerRunning), wantOutcome: StopOutcomeSuccess},
		{name: "partial failure", snapshot: buildSnapshot(runtime.ContainerRunning), applyErr: errors.New("boom"), wantOutcome: StopOutcomePartialFailure},
		{name: "degraded pre-failed", snapshot: buildSnapshot(runtime.ContainerUnhealthy), opts: StopOptions{Force: true}, wantOutcome: StopOutcomeDegradedPreFailed},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newFakeRepo()
			repo.clusters["test"] = baseCluster
			rt := &fakeRuntime{snap: tt.snapshot, applyErr: tt.applyErr}
			h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{}})
			require.NoError(t, err)

			res, err := h.StopWithOptions(context.Background(), tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOutcome, res.Outcome)
		})
	}
}

func TestLifecycleHandler_Start_AppliesVersionOverrides(t *testing.T) {
	t.Parallel()
	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{Name: "test", Repo: repo, Runtime: rt, Waiter: &fakeWaiter{}})
	require.NoError(t, err)

	_, err = h.Start(context.Background(), StartRequest{NomadVersion: "1.9.0"})
	require.NoError(t, err)

	saved, ok := repo.clusters["test"]
	require.True(t, ok, "Start() did not persist cluster")
	for _, n := range saved.Nodes {
		if n.Kind == domain.KindNomad {
			assert.Equal(t, "1.9.0", n.Image.Tag, "Nomad node %s tag mismatch", n.Name)
		}
	}
}

func TestLifecycleHandler_Start_ContractTable(t *testing.T) {
	t.Parallel()

	// Build a cluster to use as a pre-existing persisted state.
	baseCluster, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)

	// allRunningSnap has network + all nodes running — triggers noop path.
	allRunningSnap := func() runtime.Snapshot {
		containers := map[domain.NodeName]runtime.ContainerResource{}
		for _, n := range baseCluster.Nodes {
			containers[n.Name] = runtime.ContainerResource{Name: n.Name, Status: runtime.ContainerRunning}
		}
		return runtime.Snapshot{Network: &runtime.NetworkResource{Name: "hind.test"}, Containers: containers}
	}()

	// allStoppedSnap has network + all nodes stopped — triggers resume path.
	allStoppedSnap := func() runtime.Snapshot {
		containers := map[domain.NodeName]runtime.ContainerResource{}
		for _, n := range baseCluster.Nodes {
			containers[n.Name] = runtime.ContainerResource{Name: n.Name, Status: runtime.ContainerStopped}
		}
		return runtime.Snapshot{Network: &runtime.NetworkResource{Name: "hind.test"}, Containers: containers}
	}()

	emptySnap := runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}

	tests := []struct {
		name         string
		repoSetup    func() persistence.Repository
		snap         runtime.Snapshot
		inspErr      error
		waiterErr    error
		wantOutcome  domain.StartOutcome
		wantErr      bool
		wantErrPhase string // substring expected in error message
	}{
		{
			name: "new cluster created",
			repoSetup: func() persistence.Repository {
				return newFakeRepo()
			},
			snap:        emptySnap,
			wantOutcome: domain.StartOutcomeCreated,
		},
		{
			name: "existing cluster all running is noop",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:        allRunningSnap,
			wantOutcome: domain.StartOutcomeNoOp,
		},
		{
			name: "existing cluster all stopped is resumed",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap:        allStoppedSnap,
			wantOutcome: domain.StartOutcomeResumed,
		},
		{
			name: "inspect failure returns error",
			repoSetup: func() persistence.Repository {
				return newFakeRepo()
			},
			snap:         emptySnap,
			inspErr:      errors.New("docker unavailable"),
			wantErr:      true,
			wantErrPhase: "start: inspect",
		},
		{
			name: "repo load failure when cluster exists returns error",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				// Populate clusters map so Exists() returns true, then inject loadErr
				// so the subsequent Load() fails. Both must be set: Exists uses the map
				// directly; Load checks loadErr before looking in the map.
				r.clusters["test"] = baseCluster
				r.loadErr = errors.New("disk error")
				return r
			},
			snap:         emptySnap,
			wantErr:      true,
			wantErrPhase: "start: load",
		},
		{
			name: "waiter timeout returns error",
			repoSetup: func() persistence.Repository {
				return newFakeRepo()
			},
			snap:         emptySnap,
			waiterErr:    errors.New("timeout"),
			wantErr:      true,
			wantErrPhase: "start: wait",
		},
		{
			name: "repo save failure after apply returns error",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.saveErr = errors.New("write failed")
				return r
			},
			snap:         emptySnap,
			wantErr:      true,
			wantErrPhase: "start: save",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repoSetup()
			rt := &fakeRuntime{snap: tt.snap, inspErr: tt.inspErr}
			waiter := &fakeWaiter{err: tt.waiterErr}

			h, err := NewLifecycleHandler(LifecycleHandlerOptions{
				Name:    "test",
				Repo:    repo,
				Runtime: rt,
				Waiter:  waiter,
			})
			require.NoError(t, err)

			outcome, err := h.Start(context.Background(), StartRequest{})

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrPhase != "" {
					assert.True(t, strings.Contains(err.Error(), tt.wantErrPhase), "Start() error = %q, want to contain %q", err.Error(), tt.wantErrPhase)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOutcome, outcome)
		})
	}
}

func TestLifecycleHandler_LogsLifecycleMessages(t *testing.T) {
	t.Parallel()
	mem := memory.New()
	logger := &log.Logger{Handler: mem, Level: log.InfoLevel}

	repo := newFakeRepo()
	rt := &fakeRuntime{snap: runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}}
	h, err := NewLifecycleHandler(LifecycleHandlerOptions{
		Name:    "test",
		Repo:    repo,
		Runtime: rt,
		Waiter:  &fakeWaiter{},
		Logger:  logger,
	})
	require.NoError(t, err)

	_, err = h.Start(context.Background(), StartRequest{})
	require.NoError(t, err)
	err = h.Delete(context.Background())
	require.NoError(t, err)

	want := []string{"starting cluster", "cluster created", "deleting cluster", "cluster deleted"}
	got := make([]string, 0, len(mem.Entries))
	for _, e := range mem.Entries {
		got = append(got, e.Message)
	}

	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
				break
			}
		}
		assert.True(t, found, "missing log message %q; got entries: %v", w, got)
	}
}

func TestLifecycleHandler_Delete_ContractTable(t *testing.T) {
	t.Parallel()

	baseCluster, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)

	containersRunning := func() map[domain.NodeName]runtime.ContainerResource {
		m := map[domain.NodeName]runtime.ContainerResource{}
		for _, n := range baseCluster.Nodes {
			m[n.Name] = runtime.ContainerResource{Name: n.Name, Status: runtime.ContainerRunning}
		}
		return m
	}()

	tests := []struct {
		name         string
		repoSetup    func() persistence.Repository
		snap         runtime.Snapshot
		inspErr      error
		applyErr     error
		wantErr      bool
		wantErrPhase string
		wantType     string // "not_found" or ""
	}{
		{
			name: "delete succeeds and removes persisted config",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap: runtime.Snapshot{
				Network:    &runtime.NetworkResource{Name: "hind.test"},
				Containers: containersRunning,
			},
		},
		{
			name: "delete cluster not found returns NotFoundError",
			repoSetup: func() persistence.Repository {
				return newFakeRepo()
			},
			snap:     runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}},
			wantErr:  true,
			wantType: "not_found",
		},
		{
			name: "delete proceeds when config missing but runtime artifacts exist",
			repoSetup: func() persistence.Repository {
				return newFakeRepo()
			},
			snap: runtime.Snapshot{
				Network: &runtime.NetworkResource{Name: "hind.test"},
				Containers: map[domain.NodeName]runtime.ContainerResource{
					"hind.test.consul.1": {Name: "hind.test.consul.1", Status: runtime.ContainerRunning},
				},
			},
		},
		{
			name: "delete inspect failure returns wrapped error",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap: runtime.Snapshot{
				Network:    &runtime.NetworkResource{Name: "hind.test"},
				Containers: containersRunning,
			},
			inspErr:      errors.New("docker inspect failed"),
			wantErr:      true,
			wantErrPhase: "delete: inspect",
		},
		{
			name: "delete apply failure returns error",
			repoSetup: func() persistence.Repository {
				r := newFakeRepo()
				r.clusters["test"] = baseCluster
				return r
			},
			snap: runtime.Snapshot{
				Network:    &runtime.NetworkResource{Name: "hind.test"},
				Containers: containersRunning,
			},
			applyErr:     errors.New("docker rm failed"),
			wantErr:      true,
			wantErrPhase: "delete: apply",
		},
		{
			name: "delete is idempotent: calling delete on already-deleted cluster returns not-found",
			repoSetup: func() persistence.Repository {
				return newFakeRepo() // empty repo: cluster already deleted
			},
			snap:     runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}},
			wantErr:  true,
			wantType: "not_found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repoSetup()
			rt := &fakeRuntime{snap: tt.snap, inspErr: tt.inspErr, applyErr: tt.applyErr}

			h, err := NewLifecycleHandler(LifecycleHandlerOptions{
				Name:    "test",
				Repo:    repo,
				Runtime: rt,
				Waiter:  &fakeWaiter{},
			})
			require.NoError(t, err)

			err = h.Delete(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantType == "not_found" {
					_, ok := err.(*NotFoundError)
					assert.True(t, ok, "Delete() error type = %T, want *NotFoundError", err)
				}
				if tt.wantErrPhase != "" {
					assert.True(t, strings.Contains(err.Error(), tt.wantErrPhase), "Delete() error = %q, want to contain %q", err.Error(), tt.wantErrPhase)
				}
				return
			}
			require.NoError(t, err)
			// Verify config was deleted from repo.
			exists, existsErr := repo.Exists(context.Background(), "test")
			require.NoError(t, existsErr)
			assert.False(t, exists, "Delete() did not remove cluster from repository")
		})
	}
}
