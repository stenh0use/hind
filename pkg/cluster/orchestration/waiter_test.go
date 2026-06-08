package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/runtime"
)

type sequenceRuntime struct {
	snaps []runtime.Snapshot
	idx   int
}

func (s *sequenceRuntime) Inspect(_ context.Context, _ runtime.Selector) (runtime.Snapshot, error) {
	if len(s.snaps) == 0 {
		return runtime.Snapshot{Containers: map[domain.NodeName]runtime.ContainerResource{}}, nil
	}
	i := s.idx
	if i >= len(s.snaps) {
		i = len(s.snaps) - 1
	}
	s.idx++
	return s.snaps[i], nil
}

func (s *sequenceRuntime) Apply(_ context.Context, _ runtime.Operation) (runtime.OperationResult, error) {
	return runtime.OperationResult{}, nil
}

func TestRuntimeWaiter_WaitRunning_TimeoutIncludesNodeAndCause(t *testing.T) {
	t.Parallel()

	c, err := domain.BuildDefaultCluster("test", "v0.0.0-test")
	require.NoError(t, err)

	rt := &sequenceRuntime{
		snaps: []runtime.Snapshot{{
			Containers: map[domain.NodeName]runtime.ContainerResource{},
		}},
	}

	w := NewRuntimeWaiter(rt, time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err = w.WaitRunning(ctx, c)
	require.Error(t, err, "WaitRunning() expected timeout error")
	msg := err.Error()
	assert.Contains(t, msg, "wait:")
	assert.Contains(t, msg, "node")
	assert.Contains(t, msg, "context deadline exceeded")
}
