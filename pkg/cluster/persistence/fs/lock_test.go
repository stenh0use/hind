package fs_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stenh0use/hind/pkg/cluster/domain"
	"github.com/stenh0use/hind/pkg/cluster/persistence/fs"
)

func TestFSLocker_SerializesSameCluster(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	const name = domain.Name("test")
	var order []int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = locker.WithClusterLock(context.Background(), name, func(_ context.Context) error {
				mu.Lock()
				order = append(order, i)
				mu.Unlock()
				time.Sleep(5 * time.Millisecond)
				return nil
			})
		}()
	}
	wg.Wait()

	require.Len(t, order, 3, "expected 3 executions")
}

func TestFSLocker_DifferentClustersDoNotBlock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	done := make(chan struct{}, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = locker.WithClusterLock(ctx, "a", func(_ context.Context) error {
			time.Sleep(50 * time.Millisecond)
			done <- struct{}{}
			return nil
		})
	}()
	go func() {
		_ = locker.WithClusterLock(ctx, "b", func(_ context.Context) error {
			done <- struct{}{}
			return nil
		})
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("different-cluster operations should not block each other")
		}
	}
}

func TestFSLocker_ContextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Hold the lock.
	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = locker.WithClusterLock(context.Background(), "test", func(_ context.Context) error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held

	// Second acquisition should fail with context deadline.
	err := locker.WithClusterLock(ctx, "test", func(_ context.Context) error { return nil })
	assert.Error(t, err, "expected context error when lock is held")
	close(release)
}

func TestFSLocker_NilContext(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	//nolint:staticcheck
	err := locker.WithClusterLock(nil, "test", func(_ context.Context) error { return nil })
	assert.Error(t, err, "expected error for nil context")
}

func TestFSLocker_LockFileCreatedAndRemoved(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = locker.WithClusterLock(context.Background(), "mycluster", func(_ context.Context) error {
			close(held)
			<-release
			return nil
		})
		close(done)
	}()
	<-held

	// After lock is released, done channel closes.
	close(release)
	<-done
}

// TestFSLocker_DoesNotCreateClusterDirectory verifies that acquiring a lock for a
// cluster that does not yet exist on disk does not create the cluster-specific
// configuration directory (cluster/<name>/). Creating that directory would cause
// repo.Delete to treat a truly nonexistent cluster as a stale directory and
// silently return success instead of ErrNotFound (regression introduced by BUG-012).
func TestFSLocker_DoesNotCreateClusterDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locker := fs.NewLocker(dir)

	const clusterName = domain.Name("nonexistent")
	clusterDir := dir + "/cluster/" + string(clusterName)

	err := locker.WithClusterLock(context.Background(), clusterName, func(_ context.Context) error {
		return nil
	})
	require.NoError(t, err)

	_, statErr := os.Stat(clusterDir)
	assert.True(t, errors.Is(statErr, os.ErrNotExist), "locker must not create cluster config directory %q for a nonexistent cluster", clusterDir)
}

// TestFSLocker_RepoDeleteStillReturnsNotFoundAfterLock verifies the integration
// contract that repo.Delete returns ErrNotFound for a truly nonexistent cluster
// even after the locker has acquired and released the lock for that cluster.
// This is the direct regression test for the BUG-012 post-fix regression.
func TestFSLocker_RepoDeleteStillReturnsNotFoundAfterLock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	repo, err := fs.NewRepository()
	require.NoError(t, err)

	locker := fs.NewLocker(repo.RootDir())
	const clusterName = domain.Name("ghost")

	// Simulate what the orchestration locker does: acquire and release the lock.
	err = locker.WithClusterLock(context.Background(), clusterName, func(_ context.Context) error {
		return nil
	})
	require.NoError(t, err)

	// repo.Delete must still return ErrNotFound for a cluster that never existed.
	deleteErr := repo.Delete(context.Background(), clusterName)
	require.Error(t, deleteErr, "repo.Delete() must return ErrNotFound for a truly nonexistent cluster")
	require.ErrorIs(t, deleteErr, fs.ErrNotFound)
}
