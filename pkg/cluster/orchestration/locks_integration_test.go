package orchestration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/orchestration"
	"github.com/stenh0use/hind/pkg/cluster/persistence/fs"
)

// integLifecycle is a Lifecycle+Scaler test double that tracks concurrency.
type integLifecycle struct {
	mu          sync.Mutex
	activeCount int
	maxActive   int
	startGate   chan struct{} // if set, Start blocks until it is closed
	startCalled int
	stopCalled  int
}

func (l *integLifecycle) enter() {
	l.mu.Lock()
	l.activeCount++
	if l.activeCount > l.maxActive {
		l.maxActive = l.activeCount
	}
	l.mu.Unlock()
}

func (l *integLifecycle) leave() {
	l.mu.Lock()
	l.activeCount--
	l.mu.Unlock()
}

func (l *integLifecycle) Start(ctx context.Context, _ orchestration.StartRequest) (domain.StartOutcome, error) {
	l.enter()
	defer l.leave()
	l.mu.Lock()
	l.startCalled++
	gate := l.startGate
	l.mu.Unlock()
	if gate != nil {
		select {
		case <-ctx.Done():
			return domain.StartOutcome(-1), ctx.Err()
		case <-gate:
		}
	}
	return domain.StartOutcomeCreated, nil
}

func (l *integLifecycle) StopWithOptions(ctx context.Context, _ orchestration.StopOptions) (orchestration.StopResult, error) {
	l.enter()
	defer l.leave()
	l.mu.Lock()
	l.stopCalled++
	l.mu.Unlock()
	return orchestration.StopResult{}, nil
}

func (l *integLifecycle) Delete(_ context.Context) error {
	l.enter()
	defer l.leave()
	return nil
}

func (l *integLifecycle) Scale(_ context.Context, _ int) error {
	l.enter()
	defer l.leave()
	return nil
}

func (l *integLifecycle) activeNow() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.activeCount
}

type nopInspector struct{}

func (nopInspector) Inspect(context.Context) (orchestration.InspectResult, error) {
	return orchestration.InspectResult{}, nil
}

type nopLister struct{}

func (nopLister) List(context.Context) (orchestration.ListResult, error) {
	return orchestration.ListResult{}, nil
}

// waitForActive polls until the lifecycle has at least n goroutines active,
// or the deadline is exceeded.
func waitForActive(t *testing.T, life *integLifecycle, op string, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if life.activeNow() >= n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d active goroutines in %s", n, op)
}

// TestFSLockerIntegration_OrchestratorSerializesSameCluster wires an
// orchestration.Service with an FSLocker and confirms that concurrent
// Start+Stop calls on the same cluster are serialized (never overlap).
func TestFSLockerIntegration_OrchestratorSerializesSameCluster(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	life := &integLifecycle{startGate: make(chan struct{})}
	svc := orchestration.NewService(orchestration.Options{
		Lifecycle: life,
		Scale:     life,
		Inspect:   nopInspector{},
		List:      nopLister{},
		Locker:    locker,
		Name:      domain.Name("same-cluster"),
	})

	errStart := make(chan error, 1)
	errStop := make(chan error, 1)

	// Start takes the lock and blocks on the gate.
	go func() {
		_, err := svc.Start(context.Background(), orchestration.StartRequest{})
		errStart <- err
	}()
	waitForActive(t, life, "start", 1)

	// Stop should queue behind Start.
	go func() {
		_, err := svc.StopWithOptions(context.Background(), orchestration.StopOptions{})
		errStop <- err
	}()

	time.Sleep(25 * time.Millisecond)
	n := life.activeNow()
	assert.LessOrEqual(t, n, 1, "activeNow = %d; stop must not enter while start holds lock", n)

	// Release the gate so Start finishes.
	close(life.startGate)

	require.NoError(t, <-errStart)
	require.NoError(t, <-errStop)

	life.mu.Lock()
	maxActive := life.maxActive
	life.mu.Unlock()
	assert.LessOrEqual(t, maxActive, 1, "max concurrent mutations = %d, want <= 1", maxActive)
}

// TestFSLockerIntegration_DifferentClustersRunConcurrently confirms that
// operations on different clusters are not blocked by each other.
func TestFSLockerIntegration_DifferentClustersRunConcurrently(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	lifeA := &integLifecycle{startGate: make(chan struct{})}
	lifeB := &integLifecycle{}

	svcA := orchestration.NewService(orchestration.Options{
		Lifecycle: lifeA,
		Scale:     lifeA,
		Inspect:   nopInspector{},
		List:      nopLister{},
		Locker:    locker,
		Name:      domain.Name("cluster-a"),
	})
	svcB := orchestration.NewService(orchestration.Options{
		Lifecycle: lifeB,
		Scale:     lifeB,
		Inspect:   nopInspector{},
		List:      nopLister{},
		Locker:    locker,
		Name:      domain.Name("cluster-b"),
	})

	doneA := make(chan error, 1)
	doneB := make(chan error, 1)

	go func() {
		_, err := svcA.Start(context.Background(), orchestration.StartRequest{})
		doneA <- err
	}()
	waitForActive(t, lifeA, "start", 1)

	// cluster-b Start should run without waiting for cluster-a.
	go func() {
		_, err := svcB.Start(context.Background(), orchestration.StartRequest{})
		doneB <- err
	}()

	// Wait for cluster-b to complete — it should not be blocked by cluster-a.
	select {
	case err := <-doneB:
		require.NoError(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("cluster-b Start blocked on cluster-a lock (different clusters must not block each other)")
	}

	close(lifeA.startGate)
	require.NoError(t, <-doneA)
}

// TestFSLockerIntegration_ContextCancellationWhileQueued confirms that a
// context cancelled while waiting for the cluster lock returns the context
// error promptly without executing the operation.
func TestFSLockerIntegration_ContextCancellationWhileQueued(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	life := &integLifecycle{startGate: make(chan struct{})}
	svc := orchestration.NewService(orchestration.Options{
		Lifecycle: life,
		Scale:     life,
		Inspect:   nopInspector{},
		List:      nopLister{},
		Locker:    locker,
		Name:      domain.Name("cancel-cluster"),
	})

	// Start holds the lock.
	go func() {
		_, _ = svc.Start(context.Background(), orchestration.StartRequest{})
	}()
	waitForActive(t, life, "start", 1)

	// Stop with a short deadline — should return before the gate opens.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := svc.StopWithOptions(ctx, orchestration.StopOptions{})
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled), "expected context error, got %v", err)

	close(life.startGate)
}
