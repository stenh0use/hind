package persistence

import (
	"context"

	"github.com/stenh0use/hind/pkg/cluster/domain"
)

// Repository defines persistence operations for cluster configuration records.
type Repository interface {
	Load(ctx context.Context, name domain.Name) (domain.Cluster, error)
	Save(ctx context.Context, c domain.Cluster) error
	Delete(ctx context.Context, name domain.Name) error
	Exists(ctx context.Context, name domain.Name) (bool, error)
	List(ctx context.Context) ([]domain.Name, error)
}

// ActiveRepository defines persistence operations for tracking the active cluster name.
type ActiveRepository interface {
	GetActive(ctx context.Context) (string, error)
	SetActive(ctx context.Context, clusterName string) error
	ClearActive(ctx context.Context) error
}

// Locker serializes same-cluster mutations.
type Locker interface {
	WithClusterLock(ctx context.Context, name domain.Name, fn func(context.Context) error) error
}
