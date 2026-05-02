# BL-014 Spec — Release versioning requirements with discoverable versions

Status: approved discovery/spec output (2026-04-30)
Source work item: BL-014

## Scope completed
- Discovery/spec only completed for BL-014; no product-code edits were made.
- Requirements were defined for dependency version sources, refresh behavior, version catalog/selection schema boundaries, CLI UX, and validation/error handling.

## Supported dependency/version sources + refresh strategy
- Supported sources (in precedence order):
  1. Explicit user-selected version set (persisted selection state)
  2. Repository-managed default catalog snapshot (deterministic baseline)
  3. Optional remote source(s) per dependency family (HashiCorp and non-HashiCorp)
- Refresh behavior:
  - No implicit network refresh on normal CLI invocation by default.
  - Explicit refresh path required (dedicated command and/or `--refresh` flag).
  - Cache records must include source, retrieval timestamp, and staleness metadata.
  - Offline mode must continue with local snapshot/cache and surface stale-data warning context.

## Schema/API requirements: available versions vs selected versions
- Separate models are required:
  - Available versions catalog (source-of-truth candidates)
  - Selected versions set (user intent for active build/runtime inputs)
- Available versions schema must include:
  - Dependency key (normalized)
  - Version string (normalized/parsed form)
  - Source provenance (`default`, `remote`, `cache`)
  - Retrieved timestamp / freshness metadata
  - Optional compatibility annotations for cross-component constraints
- Selected versions schema must include:
  - Dependency key
  - Selected version
  - Selection scope (global vs project/local if both supported)
  - Selection source (`user`, `default-fallback`) and timestamp
- API boundary requirements in `pkg/build/release`:
  - Read available versions by dependency (and aggregate list)
  - Read effective selected versions
  - Set/update selected version with validation against available catalog and compatibility rules
  - Refresh available catalog through explicit action and return refresh status metadata

## CLI UX requirements for listing and choosing versions
- Read UX:
  - Provide `hind versions list` (or equivalent) with dependency, version, source, and freshness/staleness visibility.
  - Support narrowing output by dependency key.
- Write UX:
  - Provide `hind versions select <dependency> <version>` (or equivalent) for explicit user selection.
  - If multi-scope selection exists, expose scope flag and default scope behavior.
  - After selection, print effective configured value and source to confirm applied state.
- UX consistency:
  - CLI text should clearly distinguish "available" from "selected/effective" versions.
  - Stale cache/offline state should be visible but non-fatal unless user requested strict fresh mode.

## Validation and error behavior for unsupported version inputs
- Required validation failures:
  - Unknown dependency key
  - Unsupported/unknown version for a known dependency
  - Invalid version format (including unsupported aliases)
  - Incompatible version combinations where compatibility constraints are declared
- Error response requirements:
  - Return actionable messages with next step (e.g., list available versions, run refresh, correct dependency key).
  - Preserve deterministic non-zero exits for invalid user input.
  - Avoid silent fallback to defaults when user explicitly requested an unsupported version.

## Risks, open questions, and implementation guardrails
- Risks:
  - Source divergence (remote vs repo snapshot) can produce confusing effective state unless provenance is surfaced.
  - Compatibility matrix ownership must be explicit to avoid ad hoc validation spread across CLI handlers.
- Open questions to resolve during implementation planning:
  - Canonical remote endpoints and trust/update policy per dependency family.
  - Persistence location/format for selected versions (project config vs user config).
  - Whether strict freshness mode is required in CI workflows.
- Guardrails:
  - Keep version parsing/validation centralized in `pkg/build/release`.
  - Keep CLI command layer presentation-only; do not duplicate validation logic in command handlers.

## Staff verdict
- Verdict: approved
- Reason: BL-014 acceptance criteria are fully satisfied as discovery/spec output with concrete requirements for source/refresh strategy, schema/API boundaries, CLI UX, and unsupported-input validation semantics.
- Next role: engineer converts this spec into an implementation plan and task breakdown; QA validates stale/offline/error-path behavior before closure.
