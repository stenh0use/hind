package cluster

import (
	"context"

	persistencefs "github.com/stenh0use/hind/pkg/cluster/persistence/fs"
)

// ResolveClusterNameFromFS resolves the target cluster name using only the
// filesystem persistence layer — no Docker client or orchestration wiring is
// required. It is the preferred entry point for commands that have not yet
// constructed their full cluster service.
//
// Resolution order:
//  1. First positional arg (if provided).
//  2. Active cluster from ~/.config/hind/active (if set and the cluster exists).
//  3. "default".
func ResolveClusterNameFromFS(ctx context.Context, args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	repo, err := persistencefs.NewRepository()
	if err != nil {
		return "default"
	}
	return ResolveActive(ctx, repo)
}
