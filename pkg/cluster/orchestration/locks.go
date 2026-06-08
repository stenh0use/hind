package orchestration

import (
	"context"
	"fmt"
	"sync"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
)

// InMemoryLocker serializes same-cluster operations with per-cluster locks.
type InMemoryLocker struct {
	mu    sync.Mutex
	locks map[domain.Name]chan struct{}
}

// NewInMemoryLocker creates a process-local per-cluster locker.
func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{locks: make(map[domain.Name]chan struct{})}
}

func (l *InMemoryLocker) lockChan(name domain.Name) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch, ok := l.locks[name]
	if !ok {
		ch = make(chan struct{}, 1)
		ch <- struct{}{}
		l.locks[name] = ch
	}
	return ch
}

// WithClusterLock runs fn while holding the named cluster lock.
func (l *InMemoryLocker) WithClusterLock(ctx context.Context, name domain.Name, fn func(context.Context) error) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ch := l.lockChan(name)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
	}
	defer func() { ch <- struct{}{} }()
	return fn(ctx)
}

var _ persistence.Locker = (*InMemoryLocker)(nil)
