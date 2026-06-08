package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
)

type countingLifecycle struct {
	mu              sync.Mutex
	activeByOp      map[string]int
	maxByOp         map[string]int
	activeMutations int
	maxMutations    int
	startDelay      time.Duration
	startCalled     int
	stopCalled      int
	scaleCalled     int
	deleteCalled    int
	waitGateByOp    map[string]chan struct{}
}

func (c *countingLifecycle) enter(op string) chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeByOp == nil {
		c.activeByOp = map[string]int{}
	}
	if c.maxByOp == nil {
		c.maxByOp = map[string]int{}
	}
	c.activeByOp[op]++
	if c.activeByOp[op] > c.maxByOp[op] {
		c.maxByOp[op] = c.activeByOp[op]
	}
	c.activeMutations++
	if c.activeMutations > c.maxMutations {
		c.maxMutations = c.activeMutations
	}
	if c.waitGateByOp == nil {
		return nil
	}
	return c.waitGateByOp[op]
}

func (c *countingLifecycle) leave(op string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.activeByOp[op]--
	c.activeMutations--
}

func (c *countingLifecycle) setWaitGate(op string, gate chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waitGateByOp == nil {
		c.waitGateByOp = map[string]chan struct{}{}
	}
	c.waitGateByOp[op] = gate
}

func (c *countingLifecycle) snapshot() (startCalled int, stopCalled int, deleteCalled int, scaleCalled int, maxMutations int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	startCalled = c.activeByOp["start"] + (c.maxByOp["start"] - c.activeByOp["start"])
	stopCalled = c.activeByOp["stop"] + (c.maxByOp["stop"] - c.activeByOp["stop"])
	deleteCalled = c.deleteCalled
	scaleCalled = c.activeByOp["scale"] + (c.maxByOp["scale"] - c.activeByOp["scale"])
	maxMutations = c.maxMutations
	return
}

func waitForMutationEntry(t *testing.T, life *countingLifecycle, op string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		life.mu.Lock()
		entered := life.activeByOp[op] > 0
		life.mu.Unlock()
		if entered {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.FailNow(t, "timed out waiting for "+op+" to enter")
}

func assertNoConcurrentMutations(t *testing.T, life *countingLifecycle) {
	t.Helper()
	life.mu.Lock()
	defer life.mu.Unlock()
	if life.maxMutations > 1 {
		require.FailNow(t, "max concurrent mutating ops exceeded", "got %d, want <= 1", life.maxMutations)
	}
}

func assertMutationsCanOverlap(t *testing.T, lifeA, lifeB *countingLifecycle) {
	t.Helper()
	lifeA.mu.Lock()
	maxA := lifeA.maxMutations
	lifeA.mu.Unlock()
	lifeB.mu.Lock()
	maxB := lifeB.maxMutations
	lifeB.mu.Unlock()
	assert.True(t, maxA >= 1 && maxB >= 1, "expected both services to execute mutating ops, got maxA=%d maxB=%d", maxA, maxB)
}

func waitOnGate(ctx context.Context, gate chan struct{}) error {
	if gate == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-gate:
		return nil
	}
}

func runWithDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func (c *countingLifecycle) Start(ctx context.Context, _ StartRequest) (domain.StartOutcome, error) {
	gate := c.enter("start")
	defer c.leave("start")
	c.mu.Lock()
	c.startCalled++
	delay := c.startDelay
	c.mu.Unlock()
	if err := waitOnGate(ctx, gate); err != nil {
		return domain.StartOutcome(-1), err
	}
	if err := runWithDelay(ctx, delay); err != nil {
		return domain.StartOutcome(-1), err
	}
	return domain.StartOutcomeCreated, nil
}

func (c *countingLifecycle) StopWithOptions(ctx context.Context, _ StopOptions) (StopResult, error) {
	gate := c.enter("stop")
	defer c.leave("stop")
	c.mu.Lock()
	c.stopCalled++
	delay := c.startDelay
	c.mu.Unlock()
	if err := waitOnGate(ctx, gate); err != nil {
		return StopResult{}, err
	}
	if err := runWithDelay(ctx, delay); err != nil {
		return StopResult{}, err
	}
	return StopResult{}, nil
}

func (c *countingLifecycle) Delete(ctx context.Context) error {
	gate := c.enter("delete")
	defer c.leave("delete")
	c.mu.Lock()
	c.deleteCalled++
	delay := c.startDelay
	c.mu.Unlock()
	if err := waitOnGate(ctx, gate); err != nil {
		return err
	}
	return runWithDelay(ctx, delay)
}

func (c *countingLifecycle) Scale(ctx context.Context, _ int) error {
	gate := c.enter("scale")
	defer c.leave("scale")
	c.mu.Lock()
	c.scaleCalled++
	delay := c.startDelay
	c.mu.Unlock()
	if err := waitOnGate(ctx, gate); err != nil {
		return err
	}
	return runWithDelay(ctx, delay)
}

var _ Lifecycle = (*countingLifecycle)(nil)
var _ Scaler = (*countingLifecycle)(nil)

type passthroughScaler struct{}

func (passthroughScaler) Scale(context.Context, int) error { return nil }

type passthroughInspector struct{}

func (passthroughInspector) Inspect(context.Context) (InspectResult, error) {
	return InspectResult{}, nil
}

type passthroughLister struct{}

func (passthroughLister) List(context.Context) (ListResult, error) { return ListResult{}, nil }

func TestServiceSameClusterMutationsAreSerialized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		holder string
		other  string
	}{
		{name: "start blocks stop", holder: "start", other: "stop"},
		{name: "stop blocks delete", holder: "stop", other: "delete"},
		{name: "delete blocks scale", holder: "delete", other: "scale"},
		{name: "scale blocks start", holder: "scale", other: "start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			life := &countingLifecycle{}
			holdGate := make(chan struct{})
			life.setWaitGate(tt.holder, holdGate)
			svc := NewService(Options{Lifecycle: life, Scale: life, Inspect: passthroughInspector{}, List: passthroughLister{}, Locker: NewInMemoryLocker(), Name: domain.Name("same")})

			errHolder := make(chan error, 1)
			errOther := make(chan error, 1)

			go func() { errHolder <- invokeMutation(svc, tt.holder) }()
			waitForMutationEntry(t, life, tt.holder)
			go func() { errOther <- invokeMutation(svc, tt.other) }()

			time.Sleep(25 * time.Millisecond)
			life.mu.Lock()
			otherActive := life.activeByOp[tt.other]
			life.mu.Unlock()
			assert.Equal(t, 0, otherActive, "%s started before %s released lock", tt.other, tt.holder)

			close(holdGate)
			require.NoError(t, <-errHolder, "holder %s err", tt.holder)
			require.NoError(t, <-errOther, "other %s err", tt.other)
			assertNoConcurrentMutations(t, life)
		})
	}
}

func TestServiceDifferentClusterMutationsCanRunInParallel(t *testing.T) {
	t.Parallel()

	lifeA := &countingLifecycle{}
	lifeB := &countingLifecycle{}
	gateA := make(chan struct{})
	gateB := make(chan struct{})
	lifeA.setWaitGate("start", gateA)
	lifeB.setWaitGate("delete", gateB)

	locker := NewInMemoryLocker()
	svcA := NewService(Options{Lifecycle: lifeA, Scale: lifeA, Inspect: passthroughInspector{}, List: passthroughLister{}, Locker: locker, Name: domain.Name("a")})
	svcB := NewService(Options{Lifecycle: lifeB, Scale: lifeB, Inspect: passthroughInspector{}, List: passthroughLister{}, Locker: locker, Name: domain.Name("b")})

	doneA := make(chan error, 1)
	doneB := make(chan error, 1)
	go func() { doneA <- invokeMutation(svcA, "start") }()
	go func() { doneB <- invokeMutation(svcB, "delete") }()

	waitForMutationEntry(t, lifeA, "start")
	waitForMutationEntry(t, lifeB, "delete")
	assertMutationsCanOverlap(t, lifeA, lifeB)

	close(gateA)
	close(gateB)
	require.NoError(t, <-doneA, "svcA mutating op err")
	require.NoError(t, <-doneB, "svcB mutating op err")
}

func invokeMutation(svc Service, op string) error {
	switch op {
	case "start":
		_, err := svc.Start(context.Background(), StartRequest{})
		return err
	case "stop":
		_, err := svc.StopWithOptions(context.Background(), StopOptions{})
		return err
	case "delete":
		return svc.Delete(context.Background())
	case "scale":
		return svc.Scale(context.Background(), 1)
	default:
		return errors.New("unknown operation")
	}
}

func TestServiceRejectsNilContext(t *testing.T) {
	t.Parallel()
	svc := NewService(Options{Lifecycle: &countingLifecycle{}, Scale: passthroughScaler{}, Inspect: passthroughInspector{}, List: passthroughLister{}, Locker: NewInMemoryLocker()})
	_, err := svc.Start(nil, StartRequest{})
	require.Error(t, err)
	assert.Equal(t, "context cannot be nil", err.Error())
}

func TestInMemoryLockerRespectsCancellation(t *testing.T) {
	t.Parallel()
	locker := NewInMemoryLocker()
	name := domain.Name("demo")
	hold := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = locker.WithClusterLock(context.Background(), name, func(context.Context) error {
			close(hold)
			<-release
			return nil
		})
	}()
	<-hold
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := locker.WithClusterLock(ctx, name, func(context.Context) error { return nil })
	close(release)
	require.ErrorIs(t, err, context.Canceled)
}

var _ persistence.Locker = (*InMemoryLocker)(nil)
