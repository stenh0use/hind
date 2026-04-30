# Bugs

Active bugs only. Closed entries (BUG-001..BUG-007, BUG-009, BUG-010) archived in `archive/bugs-2026-04-26.md` along with their resolution work-item links.

## BUG-008
- Description: `hind get` can still panic for missing/non-existent cluster network in BL-007 validation worktree (severity: high)
- Repro steps or triggering condition:
  1. Run `go run ./cmd/hind get qa-nonexistent`
  2. (Also reproducible with malformed name) `go run ./cmd/hind get ../../etc`
- Observed result: process panics with nil-pointer dereference in `pkg/cluster/manager.go` (`state.Network = *networkInfo`)
- Expected result: command should return a controlled user-facing error (for example cluster/network not found) and never panic
- Status: open (needs re-verification on current `refactor-cleanup` HEAD `6d7bd34` — BL-001 was supposed to address the same nil-pointer path, and BL-013 has since refactored manager construction; the panic site may have moved or been resolved)
- Linked work item: BL-007 (originally observed); re-verify candidate for BL-009 scope

## BUG-011
- Description: Team handoff/work-item runtime state drifted from actual repo and worktree state during BL-025/BL-024/BL-027 coordination (severity: medium)
- Status: **closed** — reconciled at 2026-04-30 session start. BL-025 marked Completed in work-items.md, handoff.md reset to clean state, stale worktree identified for removal.
- Linked work item: BL-025
