# RE-001 References

This file contains evidence and supporting context for backlog items in `.claude/team/backlog.md`.

## R-001: Nil network panic in cluster state retrieval
- Source reviews:
  - QA handoff (`.claude/team/hind/handoff.md`)
  - Staff handoff (`.claude/team/hind/handoff.md`)
  - QA bug entry: BUG-001 (`.claude/team/hind/bugs.md`)
- Evidence:
  - `pkg/cluster/manager.go:248-253`
  - `pkg/cmd/hind/get/get.go:51-53`
  - `pkg/cmd/hind/list/list.go:125-127`
- Notes:
  - Staff marked this as a critical correctness blocker and requested changes before sign-off.
  - QA classified this as high severity and reproducible via missing network path.

## R-002: Path traversal / root escape in file manager and cluster name inputs
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
  - QA bug entry: BUG-007 (`.claude/team/hind/bugs.md`)
- Evidence:
  - `pkg/file/file.go:250-255`
  - `pkg/file/file.go:261-273`
  - `pkg/cluster/manager.go:55`
  - `pkg/cmd/hind/start/start.go:31-53`
- Notes:
  - Staff classified this as critical security/correctness work.
  - QA identified concrete traversal repro path and expected root confinement.

## R-003: Stop/read flows use stale in-memory defaults instead of persisted topology
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
  - QA bug entry: BUG-002 (`.claude/team/hind/bugs.md`)
- Evidence:
  - `pkg/cluster/manager.go:38-56`
  - `pkg/cluster/manager.go:140-149`
  - `pkg/cluster/manager.go:246-267`
  - `pkg/cmd/hind/stop/stop.go:63-76`
- Notes:
  - Staff direction: separate default-config initialization from persisted-config loading for read/stop correctness.
  - QA repro shows scaled clients can remain running after stop.

## R-004: Swallowed provider inspect errors in stop/delete paths
- Source reviews:
  - QA bug entry: BUG-003 (`.claude/team/hind/bugs.md`)
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `pkg/cluster/manager.go:157-165`
  - `pkg/cluster/manager.go:208-214`
  - `pkg/cluster/manager.go:227-233`
  - `pkg/provider/dockercli/container.go:194-203`
- Notes:
  - Both reviewers called out weak error propagation and false-success risk.

## R-005: `start --version` flag/documentation contract drift
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `pkg/cmd/hind/start/start.go:20`
  - `pkg/cmd/hind/start/start.go:40`
  - `README.md:121-124`
- Notes:
  - Staff direction: either implement end-to-end version selection or remove the user-facing contract.

## R-006: Cluster status mapping mismatch (`exited` vs `stopped`)
- Source reviews:
  - QA bug entry: BUG-004 (`.claude/team/hind/bugs.md`)
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `pkg/provider/dockercli/container.go:275-280`
  - `pkg/cmd/hind/list/list.go:154-182`
  - `pkg/provider/status.go:6-10`
- Notes:
  - Causes user-visible status misclassification.

## R-007: `hind get` output correctness issues
- Source reviews:
  - QA bug entry: BUG-005 (`.claude/team/hind/bugs.md`)
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `pkg/cmd/hind/get/get.go:58-71`
- Notes:
  - Hardcoded status output and formatting mismatch degrade reliability of CLI output.

## R-008: First-run `hind list` fails when config dir absent
- Source reviews:
  - QA bug entry: BUG-006 (`.claude/team/hind/bugs.md`)
- Evidence:
  - `pkg/cluster/cluster.go:33-35`
  - `pkg/cmd/hind/list/list.go:51-55`
- Notes:
  - Should return empty-state UX (`No clusters found`) instead of error.

## R-009: Provider/data-structure shaping and boundary cleanup
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `pkg/provider/status.go:16-20`
  - `pkg/provider/dockercli/container.go:224-240`
  - `pkg/provider/dockercli/container.go:212-219`
- Notes:
  - Staff direction: clarify DTO boundaries (inspect vs list fidelity), avoid ambiguous required/optional fields.

## R-010: Test depth and coverage in critical paths
- Source reviews:
  - QA handoff (`.claude/team/hind/handoff.md`)
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - Review observations cite command tests concentrated on constructor/flags and thinner behavioral assertions.
  - Commands executed during review: `go test ./...`, `go test ./... -cover`, `go test ./... -race`.
- Notes:
  - Staff direction: prioritize behavior/error-path tests for lifecycle commands and provider failure semantics.

## R-011: Documentation/comments drift and stale expectations
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - `README.md:160`
  - `pkg/cmd/hind/get/get.go:19`
- Notes:
  - Staff direction: align docs/comments with actual runtime behavior and supported paths.

## R-012: Architectural strengths to preserve while refactoring
- Source reviews:
  - Staff handoff (`.claude/team/hind/handoff.md`)
- Evidence:
  - Layering: `pkg/cmd`, `pkg/cluster`, `pkg/provider`, `pkg/build`
  - IO abstraction: `pkg/cmd/iostreams.go:7-30`
  - Reconcile flow: `pkg/cluster/reconcile.go`
- Notes:
  - Preserve these patterns while addressing defects and modularity changes.

## R-026: `hind build` "path must be relative" error (BUG-009)
- Source reviews:
  - Bug entry: BUG-009 (`.claude/team/hind/bugs.md#bug-009`)
- **Root cause**: `pkg/build/image/files/files.go:42` sets `i.buildDir` to an absolute path via `file.JoinPath(homeDir, buildBaseDir, buildSubDir, i.name)` where `homeDir` comes from `os.UserHomeDir()` (returns absolute). When `WriteFiles()` calls `i.manager.EnsureDir(i.buildDir)` at line 68, it passes this absolute path to `EnsureDir` which now rejects it (BL-002 added `validatePath` that calls `filepath.IsAbs` and returns error).
- Evidence:
  - Root issue: `pkg/build/image/files/files.go:42` — `i.buildDir = file.JoinPath(homeDir, buildBaseDir, buildSubDir, i.name)` produces absolute path
  - Call site: `pkg/build/image/files/files.go:68` — `i.manager.EnsureDir(i.buildDir)` passes absolute path
  - Validation: `pkg/file/file.go:328-329` — `if filepath.IsAbs(trimmed) { return errors.New("path must be relative") }`
- Fix approach: Pass relative path to EnsureDir instead of absolute, OR use `Manager` root directly without re-validating pre-constructed paths.
- Notes:
  - This was a latent bug—BL-002's stricter validation exposed it.
  - HIGH severity: `hind build` completely broken for all targets.
