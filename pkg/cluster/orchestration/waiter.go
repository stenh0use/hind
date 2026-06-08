package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

// Waiter blocks until all desired cluster nodes reach running status or the context expires.
type Waiter interface {
	WaitRunning(ctx context.Context, desired domain.Cluster) error
}

// RuntimeWaiter polls a Runtime port to check convergence.
type RuntimeWaiter struct {
	rt           runtime.Runtime
	pollInterval time.Duration
}

// NewRuntimeWaiter constructs a waiter backed by the given runtime adapter.
func NewRuntimeWaiter(rt runtime.Runtime, pollInterval time.Duration) *RuntimeWaiter {
	return &RuntimeWaiter{rt: rt, pollInterval: pollInterval}
}

// WaitRunning blocks until all nodes in desired are ContainerRunning or the context expires.
func (w *RuntimeWaiter) WaitRunning(ctx context.Context, desired domain.Cluster) error {
	for {
		snap, err := w.rt.Inspect(ctx, runtime.Selector{Cluster: string(desired.Name)})
		if err != nil {
			return fmt.Errorf("wait: inspect: %w", err)
		}
		allRunning := true
		failingNode := ""
		failingCause := ""
		for _, n := range desired.Nodes {
			c, ok := snap.Containers[n.Name]
			if !ok {
				allRunning = false
				failingNode = string(n.Name)
				failingCause = "container missing"
				break
			}
			if c.Status != runtime.ContainerRunning {
				allRunning = false
				failingNode = string(n.Name)
				failingCause = fmt.Sprintf("status=%s", c.Status)
				break
			}
		}
		if allRunning {
			return nil
		}
		timer := time.NewTimer(w.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("wait: node %s not running (%s): %w", failingNode, failingCause, ctx.Err())
		case <-timer.C:
		}
	}
}
