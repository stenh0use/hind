package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence"
)

// FSLocker serializes same-cluster operations with per-cluster in-process
// mutexes. A lock marker file is written under root/.locks/<name>.lock for
// external visibility while the lock is held.
//
// For single-process use the in-process mutex provides strong serialization.
// The on-disk marker is a best-effort visibility aid — it is not used for
// cross-process mutual exclusion.
//
// Lock markers are stored under root/.locks/ (not root/cluster/<name>/) so that
// the locker never creates the cluster-specific config directory as a side
// effect. Creating that directory would cause repo.Delete to treat a truly
// nonexistent cluster as a stale directory and silently return success.
type FSLocker struct {
	root string
	mu   sync.Mutex
	// locks maps cluster name to a buffered channel that acts as a semaphore.
	// A full channel (capacity 1) means the lock is available; empty means held.
	locks map[domain.Name]chan struct{}
}

// NewLocker creates a locker that stores lock markers under root.
// root should be the configuration root directory (e.g. ~/.config/hind).
func NewLocker(root string) *FSLocker {
	return &FSLocker{root: root, locks: make(map[domain.Name]chan struct{})}
}

func (l *FSLocker) semaphore(name domain.Name) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch, ok := l.locks[name]
	if !ok {
		ch = make(chan struct{}, 1)
		ch <- struct{}{} // initially available
		l.locks[name] = ch
	}
	return ch
}

func (l *FSLocker) lockPath(name domain.Name) string {
	return filepath.Join(l.root, ".locks", string(name)+".lock")
}

// WithClusterLock runs fn while holding the named cluster lock.
// Acquisition blocks until either the lock becomes available or ctx is cancelled.
func (l *FSLocker) WithClusterLock(ctx context.Context, name domain.Name, fn func(context.Context) error) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	ch := l.semaphore(name)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
	}
	defer func() { ch <- struct{}{} }()

	// Write a lock marker for external visibility.
	lockFile := l.lockPath(name)
	_ = os.MkdirAll(filepath.Dir(lockFile), 0o755)
	_ = os.WriteFile(lockFile, []byte("locked"), 0o600)
	defer func() { _ = os.Remove(lockFile) }()

	return fn(ctx)
}

var _ persistence.Locker = (*FSLocker)(nil)
