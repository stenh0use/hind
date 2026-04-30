# Team Backlog — Active Items

This file now tracks active backlog only.

Closed items were moved to:
- `.claude/team/done/backlog-closed-2026-04-30.md`

## Active items

### BL-012 — Preserve proven architecture patterns during refactors
- **Priority**: P3
- **Size**: S
- **Source**: Staff
- **Maps to QA bugs**: none direct
- **Problem**: quality fixes may accidentally erode strong current architecture traits.
- **Reviewer direction to preserve**: maintain clear layering, IOStreams abstraction, and reconcile-plan execution model.
- **Expected outcome**: defects reduced without degrading modularity and maintainability.
- **References**: [R-012](./refs.md#r-012-architectural-strengths-to-preserve-while-refactoring)

### BL-013 — Define migration requirements from `internal/docker` to `pkg/provider` in image builds
- **Priority**: P2
- **Size**: M
- **Source**: User
- **Problem**: image build logic currently relies on `internal/docker` paths that should be migrated behind `pkg/provider` interfaces.
- **Expected outcome**: documented, scoped migration requirements for moving image build runtime interactions from `internal/docker` usage to `pkg/provider` abstractions.
- **Acceptance criteria**:
  - identify all `internal/docker` usages and related runtime interactions in image build flows.
  - define the provider interfaces/adapters needed to replace each usage.
  - estimate migration work by component/call path, including sequencing and blockers.
  - produce migration guidance for non-conforming call paths and test updates.

### BL-014 — Define release versioning requirements with discoverable versions
- **Priority**: P1
- **Size**: L
- **Source**: User
- **Problem**: release versioning requirements for HashiCorp and other dependencies are not fully defined, and users need a way to select explicit versions.
- **Expected outcome**: requirements for version modeling, available-version tracking, and CLI/version selection behavior in `pkg/build/release`.
- **Acceptance criteria**:
  - define supported dependency/version sources and refresh strategy.
  - define schema/API for available versions and selected versions.
  - define CLI UX for listing and choosing versions.
  - document validation/error behavior for unsupported version inputs.

### BL-015 — Audit feature specs versus implementation status
- **Priority**: P2
- **Size**: M
- **Source**: User
- **Problem**: status of feature specs under `.claude/team/features/` is unknown.
- **Expected outcome**: implementation matrix for all feature specs and backlog coverage for any missing work.
- **Scope**: `hind-releases.feature`, `hind-build.feature`, `default-cluster.feature`, `hind-start.feature`, `hind-stop.feature`.
- **Acceptance criteria**:
  - assess each feature spec as implemented, partially implemented, or not implemented.
  - add backlog follow-up items for any gaps found.
  - link each follow-up item back to the specific feature spec and scenario(s).
