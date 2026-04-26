# Team Backlog — RE-001 (Staff + QA Consolidation)

This backlog consolidates the completed Staff Engineer and QA Engineer reviews for work item `RE-001`, preserving reviewer intent, severity judgments, and implementation direction.

- Staff verdict: **changes requested** (critical correctness/security blockers before sign-off).
- QA outcome: **7 actionable defects** (BUG-001..BUG-007) with reproductions and expected behavior.

Reference index: `.claude/team/refs.md`

## Prioritization model
- **Priority**: P0 (immediate), P1 (next), P2 (important follow-up), P3 (quality/cleanup)
- **Size**: S / M / L (estimated remediation effort)
- **Source**: Staff, QA, or Both

---

## P0 — Immediate blockers (must address before quality sign-off)

### BL-001 — Prevent nil-pointer panic in cluster state retrieval
- **Priority**: P0
- **Size**: S
- **Source**: Both
- **Maps to QA bugs**: BUG-001
- **Problem**: `Manager.Get` can dereference a nil network pointer and crash (`hind get`/`hind list` paths).
- **Why now**: Staff marked as critical correctness blocker; QA confirmed reproducible crash behavior.
- **Expected outcome**: no panic path; explicit not-found/error semantics.
- **References**: [R-001](./refs.md#r-001-nil-network-panic-in-cluster-state-retrieval)

### BL-002 — Enforce path confinement (block traversal/root escape)
- **Priority**: P0
- **Size**: M
- **Source**: Both
- **Maps to QA bugs**: BUG-007
- **Problem**: file/path handling accepts patterns that can escape configured root boundaries.
- **Why now**: Staff classified as critical security/correctness; QA supplied traversal trigger conditions.
- **Expected outcome**: reject traversal/absolute escapes for user-controlled names; root-constrained resolution.
- **References**: [R-002](./refs.md#r-002-path-traversal--root-escape-in-file-manager-and-cluster-name-inputs)

### BL-013 — Fix `hind build` "path must be relative" error (BUG-009)
- **Priority**: P0
- **Size**: S
- **Source**: QA
- **Maps to QA bugs**: BUG-009
- **Problem**: `hind build <target>` fails with "path must be relative" because WriteFiles passes absolute `buildDir` to EnsureDir which now rejects absolute paths (latent bug exposed by BL-002's stricter validation).
- **Why now**: HIGH severity, `hind build` completely broken.
- **Expected outcome**: `hind build <target>` templates and builds images successfully.
- **References**: [BUG-009](./hind/bugs.md#bug-009); [Root cause](./refs.md#r-026)

---

## P1 — High-value correctness and contract fixes

### BL-003 — Load persisted cluster config consistently for read/stop operations
- **Priority**: P1
- **Size**: M
- **Source**: Both
- **Maps to QA bugs**: BUG-002
- **Problem**: stop/get/list behavior can rely on in-memory defaults rather than persisted topology.
- **Reviewer direction to preserve**: separate default-config creation from persisted-config loading semantics.
- **Expected outcome**: scaled/updated cluster topology is correctly honored in lifecycle operations.
- **References**: [R-003](./refs.md#r-003-stopread-flows-use-stale-in-memory-defaults-instead-of-persisted-topology)

### BL-004 — Fix inspect error propagation in stop/delete flows
- **Priority**: P1
- **Size**: S
- **Source**: Both
- **Maps to QA bugs**: BUG-003
- **Problem**: inspect failures can be interpreted as not-found, creating false-success lifecycle outcomes.
- **Reviewer direction to preserve**: normalize provider error semantics and avoid swallowing infrastructure failures.
- **Expected outcome**: explicit not-found vs failure handling with reliable command outcomes.
- **References**: [R-004](./refs.md#r-004-swallowed-provider-inspect-errors-in-stopdelete-paths)

### BL-005 — Resolve `start --version` contract drift
- **Priority**: P1
- **Size**: S
- **Source**: Staff
- **Maps to QA bugs**: none
- **Problem**: user-facing flag and docs indicate version behavior that is not wired through execution.
- **Reviewer direction to preserve**: either implement full behavior or remove contract until supported.
- **Expected outcome**: CLI contract and documentation accurately reflect runtime behavior.
- **References**: [R-005](./refs.md#r-005-start---version-flagdocumentation-contract-drift)

---

## P2 — User-visible reliability and model quality improvements

### BL-006 — Normalize status mapping (`exited`/`stopped`) in list aggregation
- **Priority**: P2
- **Size**: S
- **Source**: Both
- **Maps to QA bugs**: BUG-004
- **Problem**: status classification can incorrectly show `partial` for stopped clusters.
- **Expected outcome**: consistent lifecycle status interpretation across provider and command layers.
- **References**: [R-006](./refs.md#r-006-cluster-status-mapping-mismatch-exited-vs-stopped)

### BL-007 — Correct `hind get` status/ports rendering
- **Priority**: P2
- **Size**: S
- **Source**: Both
- **Maps to QA bugs**: BUG-005
- **Problem**: output contains hardcoded status and formatting artifacts.
- **Expected outcome**: accurate, human-readable cluster details output.
- **References**: [R-007](./refs.md#r-007-hind-get-output-correctness-issues)

### BL-008 — Make first-run `hind list` return empty-state success
- **Priority**: P2
- **Size**: S
- **Source**: QA
- **Maps to QA bugs**: BUG-006
- **Problem**: missing config dir causes command failure instead of graceful empty list behavior.
- **Expected outcome**: first-run UX prints `No clusters found` without error.
- **References**: [R-008](./refs.md#r-008-first-run-hind-list-fails-when-config-dir-absent)

### BL-009 — Tighten provider/data-structure shaping and boundary clarity
- **Priority**: P2
- **Size**: M
- **Source**: Staff
- **Maps to QA bugs**: partial overlap with BUG-004/BUG-005 behavior
- **Problem**: mixed DTO fidelity and ambiguous field expectations across inspect/list paths.
- **Reviewer direction to preserve**: clarify model boundaries and optional/required semantics.
- **Expected outcome**: cleaner interfaces and fewer downstream interpretation bugs.
- **References**: [R-009](./refs.md#r-009-providerdata-structure-shaping-and-boundary-cleanup)

---

## P3 — Professionalization and sustainment work

### BL-010 — Deepen behavioral/error-path test coverage in critical command/provider flows
- **Priority**: P3
- **Size**: M
- **Source**: Both
- **Maps to QA bugs**: supports all BUG-001..BUG-007 regression prevention
- **Problem**: tests are relatively thin on behavior and failure semantics in key lifecycle paths.
- **Reviewer direction to preserve**: prioritize regression tests around panic-safety, scaling stop behavior, and provider failure handling.
- **Expected outcome**: stronger defect prevention confidence and less regression churn.
- **References**: [R-010](./refs.md#r-010-test-depth-and-coverage-in-critical-paths)

### BL-011 — Align docs/comments with actual runtime behavior
- **Priority**: P3
- **Size**: S
- **Source**: Staff
- **Maps to QA bugs**: none direct
- **Problem**: stale or mismatched comments/docs create confusion about current behavior.
- **Expected outcome**: docs and in-code comments match current implementation and contracts.
- **References**: [R-011](./refs.md#r-011-documentationcomments-drift-and-stale-expectations)

### BL-012 — Preserve proven architecture patterns during refactors
- **Priority**: P3
- **Size**: S
- **Source**: Staff
- **Maps to QA bugs**: none direct
- **Problem**: quality fixes may accidentally erode strong current architecture traits.
- **Reviewer direction to preserve**: maintain clear layering, IOStreams abstraction, and reconcile-plan execution model.
- **Expected outcome**: defects reduced without degrading modularity and maintainability.
- **References**: [R-012](./refs.md#r-012-architectural-strengths-to-preserve-while-refactoring)

---

## QA bug index (required inclusion)

- BUG-001 → BL-001
- BUG-002 → BL-003
- BUG-003 → BL-004
- BUG-004 → BL-006
- BUG-005 → BL-007
- BUG-006 → BL-008
- BUG-007 → BL-002
- BUG-009 → BL-013
- BUG-009 → BL-013

Source of bug details: `.claude/team/hind/bugs.md`

## Context preservation notes

The following reviewer context is intentionally preserved in prioritization:

1. **Staff engineer gate**: “changes requested” until critical panic and path-confinement issues are resolved.
2. **QA severity framing**: seven actionable defects remain open and are all represented in this backlog.
3. **Combined direction**: prioritize correctness/safety first, then lifecycle semantics, then UX/reporting, then sustainment.
4. **Do not regress strengths**: keep existing architectural boundaries and IO/reconcile patterns intact while remediating.
