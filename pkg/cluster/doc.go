// Package cluster aggregates cluster-related subpackages and provides the
// cluster status-snapshot service consumed by the `hind list` command.
//
// Lifecycle and orchestration live in pkg/cluster/orchestration. Wiring lives
// in pkg/cmd/hind/internal/wire (sharedresolver). Persistence lives in
// pkg/cluster/persistence (port) and pkg/cluster/persistence/fs (adapter).
//
// Dependency graph:
//
//	pkg/cmd/hind/*       -> pkg/cluster/orchestration, pkg/cluster (status snapshot),
//	                        pkg/cluster/persistence, pkg/cluster/persistence/fs,
//	                        pkg/cluster/runtime/docker
//	pkg/cluster          -> pkg/cluster/domain, pkg/provider (status snapshot only)
//	orchestration        -> domain, plan, runtime (ports), persistence (ports)
//	plan                 -> domain, runtime (DTOs only)
//	runtime/docker       -> runtime (ports+DTOs), pkg/provider
//	persistence/fs       -> persistence (ports), domain, pkg/file
//
// domain and plan have no adapter dependencies.
package cluster
