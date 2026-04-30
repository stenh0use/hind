# Bugs

Active bugs only. Closed entries (BUG-001..BUG-007, BUG-009, BUG-010) archived in `archive/bugs-2026-04-26.md` along with their resolution work-item links.

## BUG-008
- Description: Historical report that `hind get` panics for missing/non-existent cluster network (severity: high)
- Repro steps verified on 2026-04-30 (`refactor-cleanup`, HEAD `9b4062e`):
  1. `go -C /Users/james/dev/github/stenh0use/hind run ./cmd/hind get qa-nonexistent`
  2. `go -C /Users/james/dev/github/stenh0use/hind run ./cmd/hind get ../../etc`
- Observed result:
  - No panic reproduced.
  - Missing cluster returns controlled output (`Status: n/a`, empty `Network`) and exits `0`.
  - Malformed traversal name returns controlled error (`invalid cluster name ... cluster name cannot contain traversal segments`) and exits `1`.
- Expected result: command should never panic; errors should be controlled.
- Status: closed — not reproducible on current branch; previous panic appears resolved by integrated fixes since original BL-007-era observation.
- Linked work item: BL-007 (historical observation); closed by re-verification evidence on current branch.

## BUG-011
- Description: Team handoff/work-item runtime state drifted from actual repo and worktree state during BL-025/BL-024/BL-027 coordination (severity: medium)
- Status: **closed** — reconciled at 2026-04-30 session start. BL-025 marked Completed in work-items.md, handoff.md reset to clean state, stale worktree identified for removal.
- Linked work item: BL-025
