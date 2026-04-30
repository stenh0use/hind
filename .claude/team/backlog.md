# Team Backlog — Active Items

This file now tracks active backlog only.

Closed items were moved to:
- `.claude/team/done/backlog-closed-2026-04-30.md`

## Active items

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
- **Canonical spec**: `.claude/team/hind/spec-BL-013.md`


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
- **Canonical spec**: `.claude/team/hind/spec-BL-014.md`

### BL-017 — Close hind-stop.feature behavior gaps (force/verbose/partial failure/idempotent contracts)
- **Priority**: P2
- **Size**: L
- **Source**: BL-015 audit (`.claude/team/hind/spec-BL-015.md`)
- **Problem**: `hind-stop.feature` scenarios for force stop, verbose progress, partial-stop warnings, already-stopped messaging, and unhealthy-container handling are not fully implemented.
- **Expected outcome**: `hind stop` behavior and tests match `hind-stop.feature` scenarios:
  - "Stop command is idempotent when cluster already stopped"
  - "Stop with force flag kills containers immediately"
  - "Stop with verbose flag shows detailed progress"

### BL-019 — Enforce default-cluster.feature profile-selection contracts
- **Priority**: P2
- **Size**: M
- **Source**: BL-015 audit (`.claude/team/hind/spec-BL-015.md`)
- **Problem**: active-profile commands do not enforce cluster-existence checks and delete/rm active-profile reset semantics are not aligned with the feature spec.
- **Expected outcome**: CLI behavior and tests match `default-cluster.feature` scenarios:
  - "hind set profile [name]" when cluster exists
  - "hind set profile [name]" when cluster does not exist
  - active-profile reset behavior on cluster removal command alignment (`delete` vs `rm`)
